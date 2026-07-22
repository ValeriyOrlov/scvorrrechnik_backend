package db

import (
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresDB(cfg string, log *logrus.Logger) (*gorm.DB, error) {
	gormLogger := &GormLogrusAdapter{Logger: log, Level: logger.Info}
	db, err := gorm.Open(postgres.Open(cfg), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, err
	}
	return db, nil
}
