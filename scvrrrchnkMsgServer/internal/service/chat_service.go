package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/repository"
	"gorm.io/gorm"
)

type ChatService struct {
	chatRepo    repository.ChatRepository
	userRepo    repository.UserRepository
	chatKeyRepo repository.ChatKeyRepository
	db          *gorm.DB
}

func NewChatService(
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	chatKeyRepo repository.ChatKeyRepository,
	db *gorm.DB,
) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		userRepo:    userRepo,
		chatKeyRepo: chatKeyRepo,
		db:          db,
	}
}

func (s *ChatService) CreateChat(ctx context.Context, creatorID uint, chatType string, chatName string, memberIDs []uint) (*model.Chat, error) {
	// добавляем создателя в учестники и убираем дубликаты
	memberSet := make(map[uint]bool)
	memberSet[creatorID] = true
	for _, id := range memberIDs {
		memberSet[id] = true
	}
	uniqueIDs := make([]uint, 0, len(memberSet))
	for id := range memberSet {
		uniqueIDs = append(uniqueIDs, id)
	}

	if chatType == "private" {
		// определяем ID собеседника
		var otherID uint
		for _, id := range uniqueIDs {
			if id != creatorID {
				otherID = id
				break
			}
		}
		// проверяем, существует ли уже приватный чат
		existingChat, err := s.chatRepo.FindPrivateChat(ctx, creatorID, otherID)
		if err == nil {
			return &existingChat, nil
		}
		if !errors.Is(err, repository.ErrChatNotFound) {
			return nil, fmt.Errorf("find private chat: %w", err)
		}
		// если не найден - продолжаем создание
	}

	// запускаем транзакцию
	chat := &model.Chat{Type: chatType, ChatName: chatName}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// создаем временные репозитории от транзакционного соединения
		txChatRepo := repository.NewGormChatRepo(tx)
		txUserRepo := repository.NewGormUserRepo(tx)
		// создаем чат
		if err := txChatRepo.CreateChat(ctx, chat); err != nil {
			return fmt.Errorf("create chat: %w", err)
		}
		// загружаем (или создаём) каждого участника и сохраняем в мапе
		userMap := make(map[uint]model.User)
		for _, userID := range uniqueIDs {
			if err := txUserRepo.EnsureUserExists(ctx, userID); err != nil {
				return fmt.Errorf("ensure user %d: %w", userID, err)
			}
		}
		// загружаем актуальные данные пользователей
		for _, userID := range uniqueIDs {
			var user model.User
			if err := tx.First(&user, userID).Error; err != nil {
				return fmt.Errorf("load user %d: %w", userID, err)
			}
			userMap[userID] = user
		}
		// формируем участников
		members := make([]model.ChatMember, 0, len(uniqueIDs))
		for _, userID := range uniqueIDs {
			role := "member"
			if userID == creatorID {
				role = "admin"
			}
			members = append(members, model.ChatMember{
				UserID:   userID,
				ChatID:   chat.ID,
				Role:     role,
				JoinedAt: time.Now(),
				User:     userMap[userID], // <-- заполняем пользователя
			})
		}
		// добавляем участников
		if err := txChatRepo.AddMembers(ctx, members); err != nil {
			return fmt.Errorf("add members: %w", err)
		}

		chat.Members = members
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("create chat transaction: %w", err)
	}
	return chat, nil
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]model.Chat, error) {
	chats, err := s.chatRepo.GetUserChats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user chats: %w", err)
	}
	return chats, nil
}

func (s *ChatService) GetChatByID(ctx context.Context, chatID, userID uint) (model.Chat, error) {
	chat, err := s.chatRepo.GetChatByID(ctx, chatID, userID)
	if err != nil {
		return model.Chat{}, fmt.Errorf("get chat by ID: %w", err)
	}
	return chat, nil
}

func (s *ChatService) AddMembersToChat(ctx context.Context, chatID, requesterID uint, memberIDs []uint) error {
	// Проверяем, что запрашивающий является участником чата
	chat, err := s.chatRepo.GetChatByID(ctx, chatID, requesterID)
	if err != nil {
		return fmt.Errorf("access denied or chat not found: %w", err)
	}

	// Собираем ID существующих участников
	existingIDs := make(map[uint]bool)
	for _, m := range chat.Members {
		existingIDs[m.UserID] = true
	}

	var newMembers []model.ChatMember
	for _, userID := range memberIDs {
		if existingIDs[userID] {
			continue
		}
		// Гарантируем существование пользователя
		if _, err := s.userRepo.FindOrCreate(ctx, userID, ""); err != nil {
			return fmt.Errorf("ensure user %d: %w", userID, err)
		}
		newMembers = append(newMembers, model.ChatMember{
			UserID:   userID,
			ChatID:   chatID,
			Role:     "member",
			JoinedAt: time.Now(),
		})
	}

	if len(newMembers) == 0 {
		return nil // все уже в чате
	}

	return s.chatRepo.AddMembersToChat(ctx, chatID, newMembers)
}

func (s *ChatService) LeaveChat(ctx context.Context, chatID, userID uint) error {
	// Проверяем, что пользователь – участник
	_, err := s.chatRepo.GetChatByID(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("you are not a member of this chat: %w", err)
	}

	// Удаляем запись участника
	return s.chatRepo.RemoveMember(ctx, chatID, userID)
}

func (s *ChatService) MarkAsRead(ctx context.Context, chatID, userID, lastReadMessageID uint) error {
	_, err := s.chatRepo.GetChatByID(ctx, chatID, userID) // проверяем членство
	if err != nil {
		return fmt.Errorf("access denied: %w", err)
	}
	return s.chatRepo.MarkAsRead(ctx, chatID, userID, lastReadMessageID)
}

func (s *ChatService) SaveRoomKey(ctx context.Context, chatID, userID uint, encryptedKey string) error {
	return s.chatKeyRepo.SaveEncryptedKey(ctx, chatID, userID, encryptedKey)
}

func (s *ChatService) GetRoomKey(ctx context.Context, chatID, userID uint) (string, error) {
	return s.chatKeyRepo.GetEncryptedKey(ctx, chatID, userID)
}
