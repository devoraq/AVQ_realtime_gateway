// Package dto содержит структуры данных для обмена между слоями приложения.
package dto

import (
	"github.com/google/uuid"
)

// MessageCreatedEvent используется при получении сообщения от клиента.
type MessageCreatedEvent struct {
	From    uuid.UUID `json:"with"`
	To      uuid.UUID `json:"to"`
	Content string    `json:"content"`
	// ClientReqID
}
