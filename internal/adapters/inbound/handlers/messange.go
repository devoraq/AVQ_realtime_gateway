// Package handlers содержит реализацию входящих обработчиков WebSocket,
// преобразующих события транспортного слоя в вызовы доменных usecase'ов.
// Каждый обработчик привязывается к конкретному типу входящего сообщения
// посредством роутера из пакета websocket.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ws "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
	"github.com/DENFNC/devPractice/internal/dto"
)

// SessionStore описывает абстракцию для хранения активных WebSocket-сессий.
// Типичная реализация использует Redis или in-memory map.
type SessionStore interface {
	Add(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Remove(ctx context.Context, keys ...string) error
}

// MessageType описывает тип входящего WebSocket-сообщения, который используется
// маршрутизатором для выбора подходящего обработчика.
type MessageType string

const (
	// MessageTypeSend соответствует событиям отправки текста в чат.
	MessageTypeSend MessageType = "send_message"
)

// MessageUsecase задает контракт доменной логики чата, которой пользуются
// входящие обработчики. Реализация должна инкапсулировать бизнес-правила,
// валидацию и работу с хранилищем.
type MessageUsecase interface {
	SendMessage(ctx context.Context, dto *dto.MessageCreatedEvent) error
}

// MessageHandler обрабатывает события WebSocket и делегирует работу доменной
// логике через MessageUsecase. Обработчик не знает о деталях транспорта или
// формата сообщения, полагаясь на Envelope из websocket.
type MessageHandler struct {
	usecase MessageUsecase
}

// MessageHandlerDeps описывает зависимости обработчика чата и используется при
// сборке цепочки обработчиков.
type MessageHandlerDeps struct {
	Usecase MessageUsecase
	Router  *ws.HandlerChain
	Store   SessionStore
}

// NewSendMessageHandler регистрирует обработчик события отправки сообщения в чат.
func NewSendMessageHandler(deps *MessageHandlerDeps) *MessageHandler {
	h := &MessageHandler{
		usecase: deps.Usecase,
	}

	{
		deps.Router.HandleFunc(string(MessageTypeSend), h.SendMessage)
	}

	return h
}

// SendMessage обрабатывает входящие конверты типа send_message.
func (h *MessageHandler) SendMessage(ctx context.Context, _ *ws.Session, env ws.Envelope) error {
	var dto dto.MessageCreatedEvent
	if err := json.Unmarshal(env.Payload, &dto); err != nil {
		return fmt.Errorf("decode send_message payload: %w", err)
	}
	if err := h.usecase.SendMessage(ctx, &dto); err != nil {
		return fmt.Errorf("usecase send message: %w", err)
	}
	return nil
}
