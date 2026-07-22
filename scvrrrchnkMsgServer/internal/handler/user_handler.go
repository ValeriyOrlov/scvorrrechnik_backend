package handler

import (
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userRepo repository.UserRepository
}

func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// PUT /api/users/me/key
func (h *UserHandler) UpdatePublicKey(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}

	var req struct {
		PublicKey string `json:"public_key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.PublicKey == "" {
		return c.Status(400).JSON(fiber.Map{"error": "public_key is required"})
	}

	err := h.userRepo.UpdatePublicKey(c.Context(), currentUser.ID, req.PublicKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update public key"})
	}
	return c.SendStatus(200)
}

func (h *UserHandler) GetPublicKey(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	publicKey, err := h.userRepo.GetPublicKey(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "public key not found"})
	}
	return c.JSON(fiber.Map{"public_key": publicKey})
}

// PUT /api/users/me/backup
func (h *UserHandler) SaveBackup(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}
	var req struct {
		EncryptedBackup string `json:"encrypted_backup"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := h.userRepo.SaveEncryptedBackup(c.Context(), currentUser.ID, req.EncryptedBackup); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save backup"})
	}
	return c.SendStatus(200)
}

// GET /api/users/me/backup
func (h *UserHandler) GetBackup(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}
	backup, err := h.userRepo.GetEncryptedBackup(c.Context(), currentUser.ID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "backup not found"})
	}
	return c.JSON(fiber.Map{"encrypted_backup": backup})
}
