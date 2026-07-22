package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"gorm.io/gorm"
)

var ErrMessageNotFound = errors.New("message not found")

type MessageRepository interface {
	CreateMessage(ctx context.Context, message *model.Message) error
	GetMessagesByChatID(ctx context.Context, chatID uint, limit, offset int) ([]model.Message, error)
	UpdateMessage(ctx context.Context, messageID uint, senderID uint, newContent string) (model.Message, error)
	DeleteMessage(ctx context.Context, messageID uint, senderID uint) error
	GetMessageByID(ctx context.Context, id uint) (model.Message, error)
	UpdateEncryptedFields(ctx context.Context, messageID uint, encContent, encKeySender, encKeyRecipient, iv, authTag string) error
}

type GormMessageRepo struct {
	db *gorm.DB
}

func NewGormMessageRepo(db *gorm.DB) *GormMessageRepo {
	return &GormMessageRepo{db: db}
}

func (r *GormMessageRepo) CreateMessage(ctx context.Context, message *model.Message) error {
	result := r.db.WithContext(ctx).Create(message)
	if result.Error != nil {
		return fmt.Errorf("create message error: %w", result.Error)
	}
	return nil
}

func (r *GormMessageRepo) GetMessagesByChatID(ctx context.Context, chatID uint, limit, offset int) ([]model.Message, error) {
	var messages []model.Message
	query := r.db.WithContext(ctx).
		Preload("Sender").
		Where("chat_id = ?", chatID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)

	result := query.Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("get messages by chat id: %w", result.Error)
	}
	return messages, nil
}

func (r *GormMessageRepo) UpdateMessage(ctx context.Context, messageID uint, senderID uint, newContent string) (model.Message, error) {
	var msg model.Message
	err := r.db.WithContext(ctx).
		Where("id = ? AND sender_id = ?", messageID, senderID).
		First(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Message{}, ErrMessageNotFound
		}
		return model.Message{}, fmt.Errorf("find message to update: %w", err)
	}
	msg.Content = newContent
	// GORM автоматически обновит UpdatedAt благодаря тегу UpdatedAt
	if err := r.db.WithContext(ctx).Save(&msg).Error; err != nil {
		return model.Message{}, fmt.Errorf("update message: %w", err)
	}
	return msg, nil
}

func (r *GormMessageRepo) DeleteMessage(ctx context.Context, messageID, senderID uint) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND sender_id = ?", messageID, senderID).
		Delete(&model.Message{})
	if result.Error != nil {
		return fmt.Errorf("delete message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *GormMessageRepo) GetMessageByID(ctx context.Context, id uint) (model.Message, error) {
	var msg model.Message
	err := r.db.WithContext(ctx).Preload("Sender").First(&msg, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Message{}, ErrMessageNotFound
	}
	return msg, err
}

func (r *GormMessageRepo) UpdateEncryptedFields(ctx context.Context, messageID uint, encContent, encKeySender, encKeyRecipient, iv, authTag string) error {
	return r.db.WithContext(ctx).Model(&model.Message{}).Where("id = ?", messageID).Updates(map[string]interface{}{
		"encrypted_content":       encContent,
		"encrypted_key_sender":    encKeySender,
		"encrypted_key_recipient": encKeyRecipient,
		"iv":                      iv,
		"auth_tag":                authTag,
	}).Error
}
