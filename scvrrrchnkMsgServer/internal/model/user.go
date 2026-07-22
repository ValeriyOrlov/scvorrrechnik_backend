package model

type User struct {
	ID              uint         `gorm:"primaryKey" json:"id"`
	Username        string       `gorm:"not null" json:"username"`
	ChatMemberships []ChatMember `gorm:"foreignKey:UserID" json:"-"`
	PublicKey       string       `gorm:"type:text" json:"public_key,omitempty`
	EncryptedBackup string       `gorm:"type:text" json:"-"`
}
