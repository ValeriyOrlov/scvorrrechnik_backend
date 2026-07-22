package model

type ChatKey struct {
	ID           uint   `gorm:"primaryKey"`
	ChatID       uint   `gorm:"not null;uniqueIndex:idx_chat_user"`
	UserID       uint   `gorm:"not null;uniqueIndex:idx_chat_user"`
	EncryptedKey string `gorm:"type:text;not null"`
}
