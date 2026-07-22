package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/repository"
	"gorm.io/gorm"
)

type MessageService struct {
	messageRepo repository.MessageRepository
	chatRepo    repository.ChatRepository
	db          *gorm.DB
	broadcaster MessageBroadcaster
}

func NewMessageService(
	messageRepo repository.MessageRepository,
	chatRepo repository.ChatRepository,
	db *gorm.DB,
	broadcaster MessageBroadcaster,
) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
		db:          db,
		broadcaster: broadcaster,
	}
}

func (s *MessageService) SendMessage(
	ctx context.Context,
	senderID, chatID uint,
	content string,
	encryptedContent, encryptedKeySender, encryptedKeyRecipient, iv, authTag string,
) (*model.Message, error) {
	_, err := s.chatRepo.GetChatByID(ctx, chatID, senderID)
	if err != nil {
		return nil, fmt.Errorf("access denied or chat not found: %w", err)
	}

	msg := &model.Message{
		ChatID:                chatID,
		SenderID:              senderID,
		Content:               content,
		EncryptedContent:      encryptedContent,
		EncryptedKeySender:    encryptedKeySender,
		EncryptedKeyRecipient: encryptedKeyRecipient,
		IV:                    iv,
		AuthTag:               authTag,
	}
	if err := s.messageRepo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	var fullMsg model.Message
	if err := s.db.WithContext(ctx).Preload("Sender").First(&fullMsg, msg.ID).Error; err != nil {
		return nil, fmt.Errorf("load new message with sender: %w", err)
	}

	// broadcast с типом chat_message
	chat, err := s.chatRepo.GetChatByID(ctx, chatID, senderID)
	if err == nil {
		payload := map[string]interface{}{
			"type":    "chat_message",
			"message": fullMsg,
		}
		resp, _ := json.Marshal(payload)
		s.broadcaster.BroadcastToChat(chatID, senderID, resp, chat.MemberIDs())
	}

	return &fullMsg, nil
}

func (s *MessageService) GetChatHistory(ctx context.Context, userID, chatID uint, limit, offset int) ([]model.Message, error) {
	_, err := s.chatRepo.GetChatByID(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("access denied or chat not found: %w", err)
	}
	return s.messageRepo.GetMessagesByChatID(ctx, chatID, limit, offset)
}

func (s *MessageService) UpdateMessage(
	ctx context.Context,
	senderID, chatID, messageID uint,
	content string, // новый открытый текст (может быть пустым)
	encContent, encKeySender, encKeyRecipient, iv, authTag string, // зашифрованные поля
) (*model.Message, error) {
	// 1. Обновляем сообщение в репозитории (только для автора)
	msg, err := s.messageRepo.UpdateMessage(ctx, messageID, senderID, content)
	if err != nil {
		return nil, fmt.Errorf("update message: %w", err)
	}

	// 2. Если пришли зашифрованные данные, обновляем их в объекте
	if encContent != "" {
		msg.EncryptedContent = encContent
		msg.EncryptedKeySender = encKeySender
		msg.EncryptedKeyRecipient = encKeyRecipient
		msg.IV = iv
		msg.AuthTag = authTag
		// Очищаем открытый текст, чтобы сервер его не хранил
		msg.Content = ""
		// Явно сохраняем изменения в базе (кроме уже изменённого контента)
		if err := s.messageRepo.UpdateEncryptedFields(ctx, messageID, encContent, encKeySender, encKeyRecipient, iv, authTag); err != nil {
			return nil, fmt.Errorf("update encrypted fields: %w", err)
		}
	}

	// 3. Подгружаем отправителя
	var fullMsg model.Message
	if err := s.db.WithContext(ctx).Preload("Sender").First(&fullMsg, msg.ID).Error; err != nil {
		return nil, fmt.Errorf("load updated message: %w", err)
	}

	// 4. Рассылаем событие
	chat, err := s.chatRepo.GetChatByID(ctx, chatID, senderID)
	if err == nil {
		payload := map[string]interface{}{
			"type":    "message_updated",
			"message": fullMsg,
		}
		data, _ := json.Marshal(payload)
		log.Printf(">>> BROADCAST: chat=%d, sender=%d, payload=%s", chatID, senderID, string(data))
		s.broadcaster.BroadcastToChat(chatID, senderID, data, chat.MemberIDs())
		log.Println(">>> BROADCAST SENT")
	}

	return &fullMsg, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, senderID, chatID, messageID uint) error {
	if err := s.messageRepo.DeleteMessage(ctx, messageID, senderID); err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	// Рассылаем событие
	chat, err := s.chatRepo.GetChatByID(ctx, chatID, senderID)
	if err == nil {
		payload := map[string]interface{}{
			"type":       "message_deleted",
			"message_id": messageID,
			"chat_id":    chatID,
		}
		data, _ := json.Marshal(payload)
		log.Printf(">>> BROADCAST: chat=%d, sender=%d, payload=%s", chatID, senderID, string(data))

		s.broadcaster.BroadcastToChat(chatID, senderID, data, chat.MemberIDs())
		log.Println(">>> BROADCAST SENT")

	}
	return nil
}

func (s *MessageService) GetMessageByID(ctx context.Context, id uint) (*model.Message, error) {
	msg, err := s.messageRepo.GetMessageByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get message by ID: %w", err)
	}
	return &msg, nil
}
