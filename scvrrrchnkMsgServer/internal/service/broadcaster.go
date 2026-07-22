package service

type MessageBroadcaster interface {
	BroadcastToChat(chatID uint, senderID uint, message []byte, memberIDs []uint)
}
