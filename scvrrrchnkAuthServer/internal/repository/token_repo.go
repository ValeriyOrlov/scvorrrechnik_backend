package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/model"
	"gorm.io/gorm"
)

var ErrTokenNotFound = errors.New("token not found")

type TokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByToken(ctx context.Context, tokenStr string) (*model.RefreshToken, error)
	DeleteByUserID(ctx context.Context, userID uint) error
	DeleteByToken(ctx context.Context, tokenStr string) error
}

type GormTokenRepo struct {
	db *gorm.DB
}

func NewGormTokenRepo(db *gorm.DB) *GormTokenRepo {
	return &GormTokenRepo{db: db}
}

func (r *GormTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	result := r.db.WithContext(ctx).Create(token)
	if result.Error != nil {
		return fmt.Errorf("create token in repository: %w", result.Error)
	}
	return nil
}

func (r *GormTokenRepo) FindByToken(ctx context.Context, tokenStr string) (*model.RefreshToken, error) {
	token := model.RefreshToken{}
	result := r.db.WithContext(ctx).Where("token = ?", tokenStr).First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("find token by token: %w", result.Error)
	}
	return &token, nil
}

func (r *GormTokenRepo) DeleteByUserID(ctx context.Context, userID uint) error {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.RefreshToken{})
	if result.Error != nil {
		return fmt.Errorf("delete user by id: %w", result.Error)
	}
	return nil
}

func (r *GormTokenRepo) DeleteByToken(ctx context.Context, tokenStr string) error {
	result := r.db.WithContext(ctx).Where("token = ?", tokenStr).Delete(&model.RefreshToken{})
	if result.Error != nil {
		return fmt.Errorf("delete token by token string: %w", result.Error)
	}
	return nil
}
