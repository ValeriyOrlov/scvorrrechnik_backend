package ws

import (
	"encoding/json"
	"log"
	"sync"
)

type Hub struct {
	//карта: UserID -> список соединений, сгруппированных по userID
	clients map[uint][]*Client
	// каналы для управления подключениями
	register   chan *Client
	unregister chan *Client

	mu sync.RWMutex //защищает clients

	onlineUsers map[uint]bool
	broadcast   chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[uint][]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		onlineUsers: make(map[uint]bool),
		broadcast:   make(chan []byte, 256),
	}
}

// Run запускает главный цикл Hub. Должен вызываться в отдельной горутине
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = append(h.clients[client.userID], client)
			h.onlineUsers[client.userID] = true
			h.mu.Unlock()
			h.broadcastStatus(client.userID, true)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.userID]; ok {
				for i, c := range clients {
					if c == client {
						h.clients[client.userID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				//если у пользователя нет соединений, удаляем ключ
				if len(h.clients[client.userID]) == 0 {
					delete(h.clients, client.userID)
					delete(h.onlineUsers, client.userID)
					h.mu.Unlock()
					h.broadcastStatus(client.userID, false)
				} else {
					h.mu.Unlock()
				}
			} else {
				h.mu.Unlock()
			}
		}
	}
}

func (h *Hub) broadcastStatus(userID uint, online bool) {
	log.Printf("Broadcast status: user=%d, online=%v", userID, online)

	status := map[string]interface{}{
		"type":    "user_status",
		"user_id": userID,
		"online":  online,
	}
	data, _ := json.Marshal(status)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, clients := range h.clients {
		for _, client := range clients {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

// BroadcastToChat отправляет сообщение всем участникам чата, кроме отправителя
func (h *Hub) BroadcastToChat(chatID uint, senderID uint, message []byte, memberIDs []uint) {
	log.Printf("BroadcastToChat: chat=%d, sender=%d, payload=%s", chatID, senderID, string(message))
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range memberIDs {
		if userID == senderID {
			continue //отправитель получает подтверждение через REST-ответ
		}
		if clients, ok := h.clients[userID]; ok {
			for _, client := range clients {
				select {
				case client.send <- message:
				default:
					//Буфер переполнен - закрываем канал и удаляем клиента
					close(client.send)
					h.Unregister(client)
				}
			}
		}
	}
}

func (h *Hub) Register(client *Client) {
	log.Printf("Registering client for user %d", client.userID)
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) OnlineUsers() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]uint, 0, len(h.onlineUsers))
	for id := range h.onlineUsers {
		ids = append(ids, id)
	}
	return ids
}

func (h *Hub) BroadcastTyping(chatID, senderID uint, memberIDs []uint) {
	payload := map[string]interface{}{
		"type":    "typing",
		"chat_id": chatID,
		"user_id": senderID,
	}
	log.Printf("BroadcastTyping: chat=%d, sender=%d, members=%v", chatID, senderID, memberIDs)
	data, _ := json.Marshal(payload)
	h.BroadcastToChat(chatID, senderID, data, memberIDs)
}

func (h *Hub) BroadcastReadReceipt(chatID, userID uint, lastReadMessageID uint, memberIDs []uint) {
	payload := map[string]interface{}{
		"type":                 "messages_read",
		"chat_id":              chatID,
		"user_id":              userID,
		"last_read_message_id": lastReadMessageID,
	}
	data, _ := json.Marshal(payload)
	h.BroadcastToChat(chatID, userID, data, memberIDs)
}
