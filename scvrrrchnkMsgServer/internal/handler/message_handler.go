package handler

import (
	"errors"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/repository"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/service"
	"github.com/gofiber/fiber/v2"
)

type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

// POST /api/chats/:id/messages
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found in context"})
	}
	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}

	var req struct {
		Content               string `json:"content"`
		EncryptedContent      string `json:"encrypted_content"`
		EncryptedKeySender    string `json:"encrypted_key_sender"`
		EncryptedKeyRecipient string `json:"encrypted_key_recipient"`
		IV                    string `json:"iv"`
		AuthTag               string `json:"auth_tag"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Content == "" && req.EncryptedContent == "" {
		return c.Status(400).JSON(fiber.Map{"error": "content or encrypted content must be provided"})
	}

	msg, err := h.messageService.SendMessage(
		c.Context(),
		currentUser.ID,
		uint(chatID),
		req.Content,
		req.EncryptedContent,
		req.EncryptedKeySender,
		req.EncryptedKeyRecipient,
		req.IV,
		req.AuthTag,
	)
	if err != nil {
		if errors.Is(err, repository.ErrChatNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "chat not found or access denied"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "send message failed"})
	}

	return c.Status(fiber.StatusCreated).JSON(msg)
}

// GET /api/chats/:id/messages?limit=50&offset=0
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found in context"})
	}
	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	messages, err := h.messageService.GetChatHistory(c.Context(), currentUser.ID, uint(chatID), limit, offset)
	if err != nil {
		if errors.Is(err, repository.ErrChatNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "chat not found or access denied"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "get messages failed"})
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}

func (h *MessageHandler) UpdateMessage(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}
	chatID, err := c.ParamsInt("chatId")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}
	messageID, err := c.ParamsInt("messageId")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid message id"})
	}
	var req struct {
		Content               string `json:"content"`
		EncryptedContent      string `json:"encrypted_content"`
		EncryptedKeySender    string `json:"encrypted_key_sender"`
		EncryptedKeyRecipient string `json:"encrypted_key_recipient"`
		IV                    string `json:"iv"`
		AuthTag               string `json:"auth_tag"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	msg, err := h.messageService.UpdateMessage(
		c.Context(),
		currentUser.ID,
		uint(chatID),
		uint(messageID),
		req.Content,
		req.EncryptedContent,
		req.EncryptedKeySender,
		req.EncryptedKeyRecipient,
		req.IV,
		req.AuthTag,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(msg)
}

func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}
	chatID, err := c.ParamsInt("chatId")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}
	messageID, err := c.ParamsInt("messageId")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid message id"})
	}
	if err := h.messageService.DeleteMessage(c.Context(), currentUser.ID, uint(chatID), uint(messageID)); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

func (h *MessageHandler) GetMessageByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	msg, err := h.messageService.GetMessageByID(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "message not found"})
	}
	return c.JSON(msg)
}
