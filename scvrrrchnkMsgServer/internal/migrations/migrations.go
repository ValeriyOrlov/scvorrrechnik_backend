package migrations

import (
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{}, &model.Chat{}, &model.ChatMember{}, &model.Message{}, &model.ChatKey{})
}
