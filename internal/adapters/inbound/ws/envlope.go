package websocket

import (
	"encoding/json"
)

type TypeMessage string

const (
	EnvMessage TypeMessage = "message"
)

type Envelope struct {
	Type TypeMessage `json:"type"`
	// RequestID uuid.UUID       `json:"id"`
	// Timestamp int64           `json:"timestamp"`
	Payload json.RawMessage `json:"payload"`
}
