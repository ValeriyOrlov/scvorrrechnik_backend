package model

import "time"

type ChatMember struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"not null;index:idx_chat_user,unique" json:"user_id"`
	ChatID            uint      `gorm:"not null;index:idx_chat_user,unique" json:"chat_id"`
	Role              string    `gorm:"default:'member'" json:"role"` //"admin", "member"
	LastReadMessageID *uint     `json:"last_read_message_id"`         //nullable, указатель, чтобы можно было NULL
	JoinedAt          time.Time `json:"joined_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE "json:"user"`
	Chat Chat `gorm:"foreignKey:ChatID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}
