package model

import "time"

type Message struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	ChatID                uint      `gorm:"not null;index:idx_chat_created,priority:1" json:"chat_id"`
	SenderID              uint      `gorm:"not null" json:"sender_id"`
	Content               string    `gorm:"type:text" json:"content"` // только для обратной совместимости
	EncryptedContent      string    `gorm:"type:text" json:"encrypted_content,omitempty"`
	EncryptedKeySender    string    `gorm:"type:text" json:"encrypted_key_sender,omitempty"`
	EncryptedKeyRecipient string    `gorm:"type:text" json:"encrypted_key_recipient,omitempty"`
	IV                    string    `gorm:"type:text" json:"iv,omitempty"`
	AuthTag               string    `gorm:"type:text" json:"auth_tag,omitempty"`
	CreatedAt             time.Time `gorm:"index:idx_chat_created,priority:2" json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at,omitempty"`
	Sender                User      `gorm:"foreignKey:SenderID;references:ID" json:"sender"`
	Chat                  Chat      `gorm:"foreignKey:ChatID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}
