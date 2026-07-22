package repository

import (
	"context"
	"fmt"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"gorm.io/gorm"
)

type ChatKeyRepository interface {
	SaveEncryptedKey(ctx context.Context, chatID, userID uint, encryptedKey string) error
	GetEncryptedKey(ctx context.Context, chatID, userID uint) (string, error)
	DeleteEncryptedKey(ctx context.Context, chatID, userID uint) error
}

type GormChatKeyRepo struct {
	db *gorm.DB
}

func NewGormChatKeyRepo(db *gorm.DB) *GormChatKeyRepo {
	return &GormChatKeyRepo{db: db}
}

func (r *GormChatKeyRepo) SaveEncryptedKey(ctx context.Context, chatID, userID uint, encryptedKey string) error {
	ck := model.ChatKey{
		ChatID:       chatID,
		UserID:       userID,
		EncryptedKey: encryptedKey,
	}
	//Upsert: обновить, если уже существует
	result := r.db.WithContext(ctx).Where("chat_id = ? AND user_id = ?", chatID, userID).Assign(ck).FirstOrCreate(&ck)
	return result.Error
}

func (r *GormChatKeyRepo) GetEncryptedKey(ctx context.Context, chatID, userID uint) (string, error) {
	var ck model.ChatKey
	err := r.db.WithContext(ctx).Where("chat_id = ? AND user_id = ?", chatID, userID).First(&ck).Error
	if err != nil {
		return "", fmt.Errorf("encrypted key not found: %w", err)
	}
	return ck.EncryptedKey, nil
}

func (r *GormChatKeyRepo) DeleteEncryptedKey(ctx context.Context, chatID, userID uint) error {
	return r.db.WithContext(ctx).Where("chat_id = ? AND user_id = ?", chatID, userID).Delete(&model.ChatKey{}).Error
}
