package migrations

import (
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/model"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{}, &model.RefreshToken{})
}
