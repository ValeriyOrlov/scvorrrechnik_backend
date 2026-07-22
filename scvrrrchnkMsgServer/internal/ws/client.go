package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client представляет одно WebSocket-соединение
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    uint
	onMessage func(message []byte, userID uint)
}

// NewClient создаёт клиента (вызывается из хендлера)
func NewClient(hub *Hub, conn *websocket.Conn, userID uint, onMessage func(message []byte, userID uint)) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		userID:    userID,
		onMessage: onMessage,
	}
}

// ReadPump читает сообщения из WebSocket и обрабатывает их
func (c *Client) ReadPump() {
	log.Printf("ReadPump started for user %d", c.userID)

	defer func() {
		log.Printf("ReadPump exited for user %d", c.userID)

		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		log.Printf("ReadPump waiting for message from user %d", c.userID)
		_, message, err := c.conn.ReadMessage()
		if err == nil {
			log.Printf("ReadPump received message from user %d: %s", c.userID, string(message))
			if c.onMessage != nil {
				c.onMessage(message, c.userID)
			}
		}
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg struct {
			Type    string `json:"type"`
			ChatID  uint   `json:"chat_id"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("invalid message: %v", err)
			continue
		}

		if msg.Type == "chat_message" && msg.ChatID > 0 && msg.Content != "" {
			if c.onMessage != nil {
				c.onMessage(message, c.userID)
			}
		}
	}
}

func (c *Client) WritePump() {
	log.Printf("WritePump started for user %d", c.userID)

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Printf("ReadPump exited for user %d", c.userID)

		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
