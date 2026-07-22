package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

type UserRepository interface {
	FindOrCreate(ctx context.Context, userID uint, username string) (model.User, error)
	FindByID(ctx context.Context, id uint) (model.User, error)
	FindByUsername(ctx context.Context, username string) (model.User, error)
	SearchByUsername(ctx context.Context, query string, excludeID uint, limit int) ([]model.User, error)
	EnsureUserExists(ctx context.Context, userID uint) error
	UpdatePublicKey(ctx context.Context, userID uint, publicKey string) error
	GetPublicKey(ctx context.Context, userID uint) (string, error)
	SaveEncryptedBackup(ctx context.Context, userID uint, backup string) error
	GetEncryptedBackup(ctx context.Context, userID uint) (string, error)
}

type GormUserRepo struct {
	db *gorm.DB
}

func NewGormUserRepo(db *gorm.DB) *GormUserRepo {
	return &GormUserRepo{db: db}
}

func (r *GormUserRepo) FindByID(ctx context.Context, id uint) (model.User, error) {
	user := model.User{}
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("find user by id: %w", result.Error)
	}
	return user, nil
}

func (r *GormUserRepo) FindOrCreate(ctx context.Context, userID uint, username string) (model.User, error) {
	if username == "" {
		username = fmt.Sprintf("user_%d", userID)
	}
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ?", userID).
		Assign(model.User{ID: userID, Username: username}).
		FirstOrCreate(&user).Error
	if err != nil {
		return model.User{}, fmt.Errorf("find or create user: %w", err)
	}
	return user, nil
}

func (r *GormUserRepo) FindByUsername(ctx context.Context, username string) (model.User, error) {
	user := model.User{}
	result := r.db.WithContext(ctx).Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("find user by username: %w", result.Error)
	}
	return user, nil
}

func (r *GormUserRepo) SearchByUsername(ctx context.Context, query string, excludeID uint, limit int) ([]model.User, error) {
	var users []model.User
	result := r.db.WithContext(ctx).
		Where("username LIKE ? AND id != ?", "%"+query+"%", excludeID).
		Limit(limit).
		Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("search users: %w", result.Error)
	}
	return users, nil
}

func (r *GormUserRepo) EnsureUserExists(ctx context.Context, userID uint) error {
	var existing model.User
	err := r.db.WithContext(ctx).First(&existing, userID).Error
	if err == nil {
		return nil // уже существует, ничего не делаем
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find user by id: %w", err)
	}
	// Создаём с временным именем
	tempName := fmt.Sprintf("user_%d", userID)
	user := model.User{
		ID:       userID,
		Username: tempName,
	}
	return r.db.WithContext(ctx).Create(&user).Error
}

func (r *GormUserRepo) UpdatePublicKey(ctx context.Context, userID uint, publicKey string) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("public_key", publicKey)
	if result.Error != nil {
		return fmt.Errorf("update public key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *GormUserRepo) GetPublicKey(ctx context.Context, userID uint) (string, error) {
	var user model.User
	err := r.db.WithContext(ctx).Select("public_key").First(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("get public key: %w", err)
	}
	return user.PublicKey, nil
}

func (r *GormUserRepo) SaveEncryptedBackup(ctx context.Context, userID uint, backup string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).
		Update("encrypted_backup", backup).Error
}

func (r *GormUserRepo) GetEncryptedBackup(ctx context.Context, userID uint) (string, error) {
	var user model.User
	err := r.db.WithContext(ctx).Select("encrypted_backup").First(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("get encrypted backup: %w", err)
	}
	return user.EncryptedBackup, nil
}
