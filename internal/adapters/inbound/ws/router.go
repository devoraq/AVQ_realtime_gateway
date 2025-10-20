package websocket

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrNoRouteMatched = errors.New("websocket: no route matched envelope")

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

type Router interface {
	Route(ctx context.Context, s *Session, env Envelope) error
}

type HandlerChain struct {
	handlers map[TypeMessage]HandlerFunc
}

type HandlerFunc func(ctx context.Context, s *Session, env Envelope) error

func NewHandlerChain() *HandlerChain {
	return &HandlerChain{
		handlers: make(map[TypeMessage]HandlerFunc),
	}
}

func (c *HandlerChain) HandleFunc(messageType TypeMessage, handler HandlerFunc) {
	if messageType == "" || handler == nil {
		return
	}

	c.handlers[messageType] = handler
}

func (c *HandlerChain) Route(ctx context.Context, s *Session, env Envelope) error {
	if handler, ok := c.handlers[env.Type]; ok {
		return handler(ctx, s, env)
	}

	return ErrNoRouteMatched
}
