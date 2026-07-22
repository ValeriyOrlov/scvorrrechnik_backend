package service

import (
	"context"
	"fmt"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) SearchUsers(ctx context.Context, query string, excludeID uint, limit int) ([]model.User, error) {
	if query == "" {
		return nil, fmt.Errorf("search query is empty")
	}
	users, err := s.userRepo.SearchByUsername(ctx, query, excludeID, limit)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	return users, nil
}

func (s *UserService) GetPublicKey(ctx context.Context, userID uint) (string, error) {
	return s.userRepo.GetPublicKey(ctx, userID)
}

func (s *UserService) SaveBackup(ctx context.Context, userID uint, backup string) error {
	return s.userRepo.SaveEncryptedBackup(ctx, userID, backup)
}

func (s *UserService) GetBackup(ctx context.Context, userID uint) (string, error) {
	return s.userRepo.GetEncryptedBackup(ctx, userID)
}
