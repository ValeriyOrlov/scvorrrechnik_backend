package handler

import (
	"errors"
	"log"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/model"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/repository"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/service"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/ws"
	"github.com/gofiber/fiber/v2"
)

type CreateChatRequest struct {
	Type      string `json:"type"`
	ChatName  string `json:"chat_name"`
	MemberIDs []uint `json:"member_ids"`
}

type ChatHandler struct {
	chatService *service.ChatService
	hub         *ws.Hub
	userService *service.UserService
	userRepo    repository.UserRepository
}

// публичные ключи участников
type memberKey struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
}

func NewChatHandler(chatService *service.ChatService, hub *ws.Hub, userService *service.UserService, userRepo repository.UserRepository) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		hub:         hub,
		userService: userService,
		userRepo:    userRepo,
	}
}

func (h *ChatHandler) CreateChat(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found in context"})
	}
	var req CreateChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "create chat request parse error"})
	}
	if req.Type != "private" && req.Type != "group" {
		return c.Status(400).JSON(fiber.Map{"error": "create chat request parse error"})
	}
	ctx := c.Context()
	chat, err := h.chatService.CreateChat(ctx, currentUser.ID, req.Type, req.ChatName, req.MemberIDs)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "create chat error"})
	}
	return c.Status(fiber.StatusCreated).JSON(chat)
}

func (h *ChatHandler) GetUserChats(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found in context"})
	}
	ctx := c.Context()
	userChats, err := h.chatService.GetUserChats(ctx, currentUser.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "get user chats error"})
	}
	return c.Status(fiber.StatusOK).JSON(userChats)
}

func (h *ChatHandler) GetChatByID(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found in context"})
	}
	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "chat id parse error"})

	}
	ctx := c.Context()
	currentChat, err := h.chatService.GetChatByID(ctx, uint(chatID), currentUser.ID)
	if err != nil {
		if errors.Is(err, repository.ErrChatNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "chat not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "get chat by id error"})
	}
	return c.Status(fiber.StatusOK).JSON(currentChat)
}

func (h *ChatHandler) GetOnlineUsers(c *fiber.Ctx) error {
	online := h.hub.OnlineUsers()
	return c.JSON(fiber.Map{"online": online})
}

func (h *ChatHandler) SearchUsers(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found in context"})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query is required"})
	}

	users, err := h.userService.SearchUsers(c.Context(), query, currentUser.ID, 20)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "search failed"})
	}
	return c.JSON(users)
}

func (h *ChatHandler) AddMembers(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}

	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}

	var req struct {
		MemberIDs []uint `json:"member_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.chatService.AddMembersToChat(c.Context(), uint(chatID), currentUser.ID, req.MemberIDs); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "members added"})
}

func (h *ChatHandler) LeaveChat(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}

	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}

	if err := h.chatService.LeaveChat(c.Context(), uint(chatID), currentUser.ID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "left chat"})
}

func (h *ChatHandler) MarkAsRead(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}
	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}
	var req struct {
		LastReadMessageID uint `json:"last_read_message_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := h.chatService.MarkAsRead(c.Context(), uint(chatID), currentUser.ID, req.LastReadMessageID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to mark as read"})
	}
	// Получаем список участников для рассылки
	chat, err := h.chatService.GetChatByID(c.Context(), uint(chatID), currentUser.ID)
	if err == nil {
		h.hub.BroadcastReadReceipt(uint(chatID), currentUser.ID, req.LastReadMessageID, chat.MemberIDs())
	}
	return c.SendStatus(200)
}

// GET /api/chats/:id/keys
func (h *ChatHandler) GetChatKeys(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}
	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}

	// Проверяем, что запрашивающий является участником чата
	chat, err := h.chatService.GetChatByID(c.Context(), uint(chatID), currentUser.ID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": "access denied"})
	}

	// Собираем публичные ключи участников
	keys := make([]memberKey, 0, len(chat.Members))
	for _, m := range chat.Members {
		user, err := h.userRepo.FindByID(c.Context(), m.UserID)
		if err == nil && user.PublicKey != "" {
			keys = append(keys, memberKey{
				UserID:    user.ID,
				Username:  user.Username,
				PublicKey: user.PublicKey,
			})
		}
	}
	return c.JSON(keys)
}

// POST /api/chats/:id/room-key
func (h *ChatHandler) SaveRoomKey(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}
	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}

	var req struct {
		UserID       uint   `json:"user_id"`
		EncryptedKey string `json:"encrypted_key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// Проверяем, что текущий пользователь является участником чата
	_, err = h.chatService.GetChatByID(c.Context(), uint(chatID), currentUser.ID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": "access denied"})
	}

	// Проверяем, что целевой пользователь также участник (или создатель)
	_, err = h.chatService.GetChatByID(c.Context(), uint(chatID), req.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "target user is not a member"})
	}

	if err := h.chatService.SaveRoomKey(c.Context(), uint(chatID), req.UserID, req.EncryptedKey); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save room key"})
	}
	return c.SendStatus(200)
}

// GET /api/chats/:id/room-key
func (h *ChatHandler) GetRoomKey(c *fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(model.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "user not found"})
	}
	chatID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid chat id"})
	}
	encryptedKey, err := h.chatService.GetRoomKey(c.Context(), uint(chatID), currentUser.ID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "room key not found"})
	}
	log.Printf("GetRoomKey: chat=%d, user=%d, found=%v", chatID, currentUser.ID, encryptedKey != "")
	return c.JSON(fiber.Map{"encrypted_key": encryptedKey})
}
