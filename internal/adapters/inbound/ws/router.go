// Package ws предоставляет инфраструктурный слой входящих адаптеров
// для работы с WebSocket-соединениями, включая маршрутизацию Envelope-сообщений
// и управление сессиями.
package ws

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrNoRouteMatched возвращается, когда для входящего сообщения не найден
// зарегистрированный обработчик. Это сигнал инфраструктуре, что клиент
// отправил неподдерживаемый тип сообщения.
var ErrNoRouteMatched = errors.New("websocket: no route matched envelope")

// HandlerFunc представляет функцию обработки входящего сообщения конкретного
// типа. Функция получает контекст, актуальную сессию и Envelope и должна
// вернуть ошибку, если обработка завершилась неудачно.
type HandlerFunc func(ctx context.Context, s *Session, env Envelope) error

// Envelope описывает базовый контракт входящего сообщения WebSocket.
// Поле Payload содержит JSON-представление конкретного события, которое
// десериализуется обработчиком.
type Envelope struct {
	Type string `json:"type"`
	// RequestID uuid.UUID       `json:"id"`
	// Timestamp int64           `json:"timestamp"`
	Payload json.RawMessage `json:"payload"`
}

// Router задает контракт маршрутизатора, который ищет подходящий обработчик
// для полученного сообщения. Реализация должна возвращать ErrNoRouteMatched,
// если ни одна функция не зарегистрирована на запрошенный тип.
type Router interface {
	Route(ctx context.Context, s *Session, env Envelope) error
}

// HandlerChain реализует простейший роутер, сопоставляющий тип сообщения
// зарегистрированному обработчику.
//
// Пример:
//
//	router := websocket.NewHandlerChain()
//	router.HandleFunc("send_message", sendMessageHandler)
//	if err := router.Route(ctx, session, env); err != nil {
//		// обработка ошибки
//	}
type HandlerChain struct {
	handlers map[string]HandlerFunc
}

// NewHandlerChain создает пустую цепочку обработчиков и возвращает ее
// вызывающему коду.
func NewHandlerChain() *HandlerChain {
	return &HandlerChain{
		handlers: make(map[string]HandlerFunc),
	}
}

// HandleFunc регистрирует обработчик для заданного типа сообщения.
// Повторная регистрация для того же типа перезапишет существующий обработчик.
func (c *HandlerChain) HandleFunc(messageType string, handler HandlerFunc) {
	if messageType == "" || handler == nil {
		return
	}

	c.handlers[messageType] = handler
}

// Route вызывает подходящий обработчик, либо возвращает ErrNoRouteMatched.
// Функция подходит для прямого использования в Session.ReadLoop.
func (c *HandlerChain) Route(ctx context.Context, s *Session, env Envelope) error {
	if handler, ok := c.handlers[env.Type]; ok {
		return handler(ctx, s, env)
	}

	return ErrNoRouteMatched
}
