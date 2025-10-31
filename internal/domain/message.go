// Package domain содержит базовые сущности чатового взаимодействия.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Message описывает минимальное доменное сообщение чата.
type Message struct {
	ID        uuid.UUID
	With      string
	To        string
	Content   string
	CreatedAt int64
}

// NewMessage создаёт новое сообщение с временной меткой и уникальным идентификатором.
func NewMessage(from, to, content string) *Message {
	uid, _ := uuid.NewV7()

	return &Message{
		ID:        uid,
		With:      from,
		To:        to,
		Content:   content,
		CreatedAt: time.Now().Unix(),
	}
}
