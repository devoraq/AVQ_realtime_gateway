// Package usecases contains application business logic.
package usecases

import (
	"context"
	"errors"
)

// Eventbus defines a transport for delivering chat messages.
type Eventbus interface {
	WriteMessage(ctx context.Context, msg []byte) error
}

// ChatUsecase dispatches chat events to the underlying event bus.
type ChatUsecase struct {
	eventbus Eventbus
}

// NewChatUsecase wires the use case with provided event bus.
func NewChatUsecase(bus Eventbus) *ChatUsecase {
	return &ChatUsecase{eventbus: bus}
}

// SendMessage delivers chat messages to the event bus.
func (uc *ChatUsecase) SendMessage(userID string, message string) error {
	_ = userID
	_ = message

	if uc.eventbus == nil {
		return errors.New("chat usecase: eventbus is nil")
	}

	return errors.New("chat usecase send message not implemented")
}
