package handler

import (
	"errors"

	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/service"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService *service.AuthService
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// mapServiceError преобразует ошибки сервиса в коды состояния HTTP
func mapServiceError(err error) (int, string) {
	switch {
	case errors.Is(err, service.ErrInvalidEmail),
		errors.Is(err, service.ErrInvalidInput),
		errors.Is(err, service.ErrShortUsername),
		errors.Is(err, service.ErrWeakPassword):
		return fiber.StatusBadRequest, err.Error()
	case errors.Is(err, service.ErrUserAlreadyExists):
		return fiber.StatusConflict, err.Error()
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrUserNotFound),
		errors.Is(err, service.ErrInvalidRefreshToken):
		return fiber.StatusUnauthorized, err.Error()
	default:
		return fiber.StatusInternalServerError, "internal error"
	}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		status, msg := mapServiceError(err)
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}
	ctx := c.Context()
	user, err := h.authService.Register(ctx, req.Email, req.Username, req.Password)
	if err != nil {
		status, msg := mapServiceError(err)
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}
	return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		status, msg := mapServiceError(err)
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}
	ctx := c.Context()
	accessToken, refreshToken, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		status, msg := mapServiceError(err)
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
	})
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		status, msg := mapServiceError(err)
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	accessToken, refreshToken, err := h.authService.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		status, msg := mapServiceError(err)
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		status, msg := mapServiceError(err)
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	if err := h.authService.Logout(c.Context(), req.RefreshToken); err != nil {
		status, msg := mapServiceError(err)
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "logged out"})
}
