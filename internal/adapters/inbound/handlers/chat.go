// Package handlers содержит реализацию входящих обработчиков WebSocket,
// преобразующих события транспортного слоя в вызовы доменных usecase'ов.
// Каждый обработчик привязывается к конкретному типу входящего сообщения
// посредством роутера из пакета websocket.
package handlers

import (
	"context"
	"errors"
	"time"

	ws "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
)

// SessionStore описывает абстракцию для хранения активных WebSocket-сессий.
// Типичная реализация использует Redis или in-memory map.
type SessionStore interface {
	Add(ctx context.Context, key string, value any, expiration time.Duration) error
	Remove(ctx context.Context, keys ...string) error
	ScanKeys(ctx context.Context, match string, step int64) (map[string]string, error)
}

// MessageType описывает тип входящего WebSocket-сообщения, который используется
// маршрутизатором для выбора подходящего обработчика.
type MessageType string

const (
	// MessageTypeSend соответствует событиям отправки текста в чат.
	MessageTypeSend MessageType = "send_message"
)

// ChatUsecase задает контракт доменной логики чата, которой пользуются
// входящие обработчики. Реализация должна инкапсулировать бизнес-правила,
// валидацию и работу с хранилищем.
type ChatUsecase interface {
	SendMessage(userID string, message string) error
}

// ChatHandler обрабатывает события WebSocket и делегирует работу доменной
// логике через ChatUsecase. Обработчик не знает о деталях транспорта или
// формата сообщения, полагаясь на Envelope из websocket.
type ChatHandler struct {
	usecase ChatUsecase
}

// ChatHandlerDeps описывает зависимости обработчика чата и используется при
// сборке цепочки обработчиков.
type ChatHandlerDeps struct {
	Usecase ChatUsecase
	Router  *ws.HandlerChain
	Store   SessionStore
}

// NewSendMessageHandler registers the handler for sending chat messages.
func NewSendMessageHandler(deps *ChatHandlerDeps) *ChatHandler {
	h := &ChatHandler{
		usecase: deps.Usecase,
	}

	{
		deps.Router.HandleFunc(string(MessageTypeSend), h.SendMessage)
	}

	return h
}

// SendMessage handles incoming send_message envelopes.
func (h *ChatHandler) SendMessage(ctx context.Context, s *ws.Session, env ws.Envelope) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_ = s
	_ = env

	return errors.New("chat handler not implemented yet")
}
