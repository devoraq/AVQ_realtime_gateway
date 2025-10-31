package kafka

import (
	"context"
	"fmt"

	"github.com/DENFNC/devPractice/internal/events"
	"github.com/segmentio/kafka-go"
)

// Handler описывает функцию, обрабатывающую событие Kafka.
type Handler func(context.Context, events.Message) error

// Router управляет маршрутизацией событий Kafka внутри адаптера.
type Router struct {
	handlers map[string]Handler
}

// NewRouter создаёт пустой маршрутизатор для Kafka-сообщений.
func NewRouter() *Router { return &Router{make(map[string]Handler)} }

// Handle регистрирует обработчик для указанного топика.
func (r *Router) Handle(topic string, h Handler) {
	r.handlers[topic] = h
}

// Dispatch преобразует kafka.Message в events.Message и передаёт его обработчику.
func (r *Router) Dispatch(ctx context.Context, msg kafka.Message) error {
	h, ok := r.handlers[msg.Topic]
	if !ok {
		return fmt.Errorf("no handler for topic %s", msg.Topic)
	}
	return h(ctx, toEventMessage(msg))
}

func toEventMessage(msg kafka.Message) events.Message {
	return events.Message{
		Topic:   msg.Topic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: headersToMap(msg.Headers),
	}
}

func headersToMap(headers []kafka.Header) map[string][]byte {
	if len(headers) == 0 {
		return nil
	}

	result := make(map[string][]byte, len(headers))
	for _, header := range headers {
		result[header.Key] = header.Value
	}

	return result
}
