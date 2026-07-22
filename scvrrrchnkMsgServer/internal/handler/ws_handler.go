package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/repository"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/service"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/ws"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type WSHandler struct {
	hub         *ws.Hub
	userRepo    repository.UserRepository
	chatService *service.ChatService
	msgService  *service.MessageService
	jwtSecret   string
}

func NewWSHandler(
	hub *ws.Hub,
	userRepo repository.UserRepository,
	chatService *service.ChatService,
	msgService *service.MessageService,
	jwtSecret string,
) *WSHandler {
	return &WSHandler{
		hub:         hub,
		userRepo:    userRepo,
		chatService: chatService,
		msgService:  msgService,
		jwtSecret:   jwtSecret,
	}
}

func (h *WSHandler) HandleWebSocket() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Проверяем токен на HTTP-уровне
		tokenStr := c.Query("token")
		if tokenStr == "" {
			return c.Status(401).JSON(fiber.Map{"error": "token required"})
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(h.jwtSecret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token claims"})
		}
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token payload"})
		}
		userID := uint(userIDFloat)
		username, _ := claims["username"].(string)

		// 2. Гарантируем существование пользователя
		user, err := h.userRepo.FindOrCreate(c.Context(), userID, username)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "user sync failed"})
		}

		// 3. Апгрейдим до WebSocket только после успешной проверки
		return websocket.New(func(conn *websocket.Conn) {
			log.Printf("WS client created for user %d", user.ID)

			client := ws.NewClient(h.hub, conn, user.ID, func(message []byte, userID uint) {
				log.Printf("Callback received message: %s", string(message))
				var msg struct {
					Type    string `json:"type"`
					ChatID  uint   `json:"chat_id"`
					Content string `json:"content"`
				}
				if err := json.Unmarshal(message, &msg); err != nil {
					log.Printf("invalid message: %v", err)
					return
				}
				log.Printf("Decoded: type=%s, chatID=%d", msg.Type, msg.ChatID)
				if msg.Type == "chat_message" && msg.ChatID > 0 && msg.Content != "" {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if _, err := h.msgService.SendMessage(ctx, userID, msg.ChatID, msg.Content, "", "", "", "", ""); err != nil {
						log.Printf("send message error: %v", err)
					}
				}
				if msg.Type == "typing" && msg.ChatID > 0 {
					log.Println("Entering typing block")
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					chat, err := h.chatService.GetChatByID(ctx, msg.ChatID, userID)
					if err == nil {
						h.hub.BroadcastTyping(msg.ChatID, userID, chat.MemberIDs())
					}
				}
			})
			h.hub.Register(client)
			log.Println("Starting WS pumps")

			go client.WritePump()
			client.ReadPump() // блокируемся, пока клиент не отключится
		})(c)
	}
}
