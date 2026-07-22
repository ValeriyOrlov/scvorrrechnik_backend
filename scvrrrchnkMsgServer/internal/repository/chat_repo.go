package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"gorm.io/gorm"
)

var (
	ErrChatAlreadyExists = errors.New("chat already exists")
	ErrChatNotFound      = errors.New("chat not found")
)

type ChatRepository interface {
	CreateChat(ctx context.Context, chat *model.Chat) error
	AddMembers(ctx context.Context, members []model.ChatMember) error
	GetUserChats(ctx context.Context, userID uint) ([]model.Chat, error)
	GetChatByID(ctx context.Context, chatID, userID uint) (model.Chat, error)
	FindPrivateChat(ctx context.Context, userID1, userUD2 uint) (model.Chat, error)
	AddMembersToChat(ctx context.Context, chatID uint, members []model.ChatMember) error
	RemoveMember(ctx context.Context, chatID, userID uint) error
	MarkAsRead(ctx context.Context, chatID, userID, lastReadMessageID uint) error
}

type GormChatRepo struct {
	db *gorm.DB
}

func NewGormChatRepo(db *gorm.DB) *GormChatRepo {
	return &GormChatRepo{db: db}
}

func (r *GormChatRepo) CreateChat(ctx context.Context, chat *model.Chat) error {
	result := r.db.WithContext(ctx).Create(chat)
	if result.Error != nil {
		return fmt.Errorf("create chat error: %w", result.Error)
	}
	return nil
}

func (r *GormChatRepo) AddMembers(ctx context.Context, members []model.ChatMember) error {
	if len(members) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).Create(&members)
	if result.Error != nil {
		return fmt.Errorf("add members: %w", result.Error)
	}
	return nil
}

func (r *GormChatRepo) GetUserChats(ctx context.Context, userID uint) ([]model.Chat, error) {
	var chats []model.Chat
	result := r.db.WithContext(ctx).
		Preload("Members.User").
		Joins("JOIN chat_members ON chat_members.chat_id = chats.id").
		Where("chat_members.user_id = ?", userID).
		Distinct().
		Find(&chats)
	if result.Error != nil {
		return nil, fmt.Errorf("get user chats: %w", result.Error)
	}
	for i := range chats {
		var lastMsg model.Message
		err := r.db.WithContext(ctx).
			Where("chat_id = ?", chats[i].ID).
			Order("created_at DESC").
			First(&lastMsg).Error
		if err == nil {
			r.db.WithContext(ctx).Preload("Sender").
				First(&lastMsg, lastMsg.ID)
			chats[i].LastMessage = &lastMsg
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("WARN: get last message for chat %d: %v", chats[i].ID, err)
		}
		// если ErrRecordNotFound - оставляем LastMessage = nil
	}
	return chats, nil
}

func (r *GormChatRepo) GetChatByID(ctx context.Context, chatID, userID uint) (model.Chat, error) {
	var chat model.Chat
	result := r.db.WithContext(ctx).
		Preload("Members.User").
		Joins("JOIN chat_members ON chat_members.chat_id = chats.id").
		Where("chat_members.user_id = ? AND chat_members.chat_id = ?", userID, chatID).
		First(&chat)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return model.Chat{}, ErrChatNotFound
		}
		return model.Chat{}, fmt.Errorf("get chat by id: %w", result.Error)
	}
	return chat, nil
}

func (r *GormChatRepo) FindPrivateChat(ctx context.Context, userID1, userID2 uint) (model.Chat, error) {
	var chat model.Chat
	err := r.db.WithContext(ctx).
		Preload("Members.User").
		Joins("JOIN chat_members cm1 ON cm1.chat_id = chats.id AND cm1.user_id = ?", userID1).
		Joins("JOIN chat_members cm2 ON cm2.chat_id = chats.id AND cm2.user_id = ?", userID2).
		Where("chats.type = ?", "private").
		First(&chat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Chat{}, ErrChatNotFound
	}
	return chat, err
}

func (r *GormChatRepo) AddMembersToChat(ctx context.Context, chatID uint, members []model.ChatMember) error {
	if len(members) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).Create(&members)
	return result.Error
}

func (r *GormChatRepo) RemoveMember(ctx context.Context, chatID, userID uint) error {
	result := r.db.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Delete(&model.ChatMember{})
	if result.Error != nil {
		return fmt.Errorf("remove member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrChatNotFound
	}
	return nil
}

func (r *GormChatRepo) MarkAsRead(ctx context.Context, chatID, userID, lastReadMessageID uint) error {
	result := r.db.WithContext(ctx).
		Model(&model.ChatMember{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Update("last_read_message_id", lastReadMessageID)
	if result.Error != nil {
		return fmt.Errorf("mark as read: %w", result.Error)
	}
	return nil
}
