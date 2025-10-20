package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
)

type SendPayload struct {
	DialogID string `json:"dialogId"`
	Text     string `json:"text"`
}

type SendCommand struct {
	DialogID string
	Text     string
	SenderID uuid.UUID
}

type MessageService interface {
	Send(ctx context.Context, cmd SendCommand) error
}

func HandleSend(ctx context.Context, session *websocket.Session, env websocket.Envelope, service MessageService) error {
	if service == nil {
		return errors.New("handlers: message service is nil")
	}

	if len(env.Payload) == 0 {
		return fmt.Errorf("handlers: empty payload")
	}

	var payload SendPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("handlers: decode send payload: %w", err)
	}

	if payload.DialogID == "" {
		return fmt.Errorf("handlers: dialogId is required")
	}

	if payload.Text == "" {
		return fmt.Errorf("handlers: text is required")
	}

	cmd := SendCommand{
		DialogID: payload.DialogID,
		Text:     payload.Text,
		SenderID: session.UserID,
	}

	return service.Send(ctx, cmd)
}

type LoggingMessageService struct {
	logger *slog.Logger
}

func NewLoggingMessageService(logger *slog.Logger) *LoggingMessageService {
	if logger == nil {
		logger = slog.Default()
	}

	return &LoggingMessageService{
		logger: logger,
	}
}

func (s *LoggingMessageService) Send(ctx context.Context, cmd SendCommand) error {
	s.logger.With(
		slog.String("dialog_id", cmd.DialogID),
		slog.String("sender_id", cmd.SenderID.String()),
	).InfoContext(ctx, "handled send message", slog.String("text", cmd.Text))
	return nil
}
