// Package usecases содержит бизнес-логику работы с сообщениями.
package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DENFNC/devPractice/internal/domain"
	"github.com/DENFNC/devPractice/internal/dto"
	"github.com/DENFNC/devPractice/internal/events"
)

const deliveredMessageType = "message_delivered"

// Eventbus описывает шину, через которую публикуются сообщения.
type Eventbus interface {
	WriteMessage(ctx context.Context, msg []byte) error
}

// Notifier уведомляет получателя через активные сессии.
type Notifier interface {
	Notify(ctx context.Context, userID string, messageType string, payload any) error
}

// MessageUsecase инкапсулирует бизнес-логику отправки сообщений.
type MessageUsecase struct {
	eventbus Eventbus
	notifier Notifier
}

// NewMessageUsecase конструирует usecase с необходимыми зависимостями.
func NewMessageUsecase(bus Eventbus, notifier Notifier) *MessageUsecase {
	return &MessageUsecase{
		eventbus: bus,
		notifier: notifier,
	}
}

// SendMessage валидирует DTO, конструирует доменную модель и публикует её в шину.
func (uc *MessageUsecase) SendMessage(ctx context.Context, dto *dto.MessageCreatedEvent) error {
	if dto == nil {
		return errors.New("message dto is nil")
	}

	message := domain.NewMessage(dto.From.String(), dto.To.String(), dto.Content)
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if err := uc.eventbus.WriteMessage(ctx, payload); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}

// HandleDelivery вызывается после подтверждения Kafka и отправляет сообщение получателю.
func (uc *MessageUsecase) HandleDelivery(ctx context.Context, event events.Message) error {
	if uc.notifier == nil {
		return errors.New("notifier is not configured")
	}

	var message domain.Message
	if err := json.Unmarshal(event.Value, &message); err != nil {
		return fmt.Errorf("unmarshal delivered message: %w", err)
	}

	if message.To == "" {
		return errors.New("delivered message recipient is empty")
	}

	if err := uc.notifier.Notify(ctx, message.To, deliveredMessageType, message); err != nil {
		return fmt.Errorf("notify recipient: %w", err)
	}

	return nil
}
