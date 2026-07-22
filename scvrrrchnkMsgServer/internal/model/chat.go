package model

type Chat struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Type        string       `gorm:"not null" json:"type"`
	ChatName    string       `gorm:"default:''" json:"chat_name"` //для групп
	Members     []ChatMember `gorm:"foreignKey:ChatID" json:"members"`
	Messages    []Message    `gorm:"foreignKey:ChatID" json:"-"`
	LastMessage *Message     `gorm:"foreignKey:ChatID" json:"last_message,omitempty"`
}

func (c *Chat) MemberIDs() []uint {
	ids := make([]uint, 0, len(c.Members))
	for _, m := range c.Members {
		ids = append(ids, m.UserID)
	}
	return ids
}
