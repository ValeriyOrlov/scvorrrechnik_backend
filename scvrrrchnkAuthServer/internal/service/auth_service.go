package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/model"
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrWeakPassword        = errors.New("min 8 characters")
	ErrShortUsername       = errors.New("min 3 characters")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
)

type AuthService struct {
	userRepo   repository.UserRepository
	tokenRepo  repository.TokenRepository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtSecret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, email, username, password string) (*model.User, error) {
	if email == "" || !strings.Contains(email, "@") {
		return nil, ErrInvalidEmail
	}
	if len(username) < 3 {
		return nil, ErrShortUsername
	}
	if len(password) < 8 {
		return nil, ErrWeakPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	newUser, err := s.userRepo.Create(ctx, email, username, string(hashedPassword))
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("registration error: %w", err)
	}
	return &newUser, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("find user by email: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("comparing hash and password: %w", err)
	}

	accessToken, refreshToken, err := s.createTokenPair(ctx, user.ID, user.Username)
	if err != nil {
		return "", "", fmt.Errorf("create token pair error: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshTokenStr string) (newAccess, newRefresh string, err error) {
	token, err := jwt.Parse(refreshTokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", "", ErrInvalidRefreshToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", ErrInvalidRefreshToken
	}
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return "", "", ErrInvalidRefreshToken
	}
	userID := uint(userIDFloat)

	// Ищем в базе
	_, err = s.tokenRepo.FindByToken(ctx, refreshTokenStr)
	if errors.Is(err, repository.ErrTokenNotFound) {
		return "", "", ErrInvalidRefreshToken
	}
	if err != nil {
		return "", "", fmt.Errorf("refresh: find token: %w", err)
	}

	// Удаляем старый
	if err := s.tokenRepo.DeleteByToken(ctx, refreshTokenStr); err != nil {
		return "", "", fmt.Errorf("refresh: delete old token: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", "", ErrUserNotFound
		}
		return "", "", fmt.Errorf("find user by id error: %w", err)
	}

	accessToken, refreshToken, err := s.createTokenPair(ctx, userID, user.Username)
	if err != nil {
		return "", "", fmt.Errorf("create token pair error: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) createTokenPair(ctx context.Context, userID uint, username string) (string, string, error) {
	accessClaims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(s.accessTTL).Unix(),
		"iat":      time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	jti := uuid.New().String()
	refreshExpAt := time.Now().Add(s.refreshTTL)
	refreshClaims := jwt.MapClaims{
		"user_id": userID,
		"exp":     refreshExpAt.Unix(),
		"iat":     time.Now().Unix(),
		"jti":     jti,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefresh, err := refreshToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	refreshModel := model.RefreshToken{
		UserID:    userID,
		Token:     signedRefresh,
		ExpiresAt: refreshExpAt,
	}

	err = s.tokenRepo.Create(ctx, &refreshModel)
	if err != nil {
		return "", "", fmt.Errorf("adding refresh token to bd: %w", err)

	}
	return signedAccess, signedRefresh, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	_, err := s.tokenRepo.FindByToken(ctx, refreshTokenStr)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil
		}
		return fmt.Errorf("logout error: %w", err)
	}
	if err := s.tokenRepo.DeleteByToken(ctx, refreshTokenStr); err != nil {
		return err
	}
	return nil
}
