package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

// Создает нового продюсера, записывает в передаваемый на входе топик.
// Запуск происходит в инициализации кафки
func createWriter(address, topic string) *kafka.Writer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(address),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return w
}

// Пишет сообщение в топик продюсера, которого получает на вход.
//
// Пример использования:
//
// msg := []byte("Some message as a byte slice")
// err := WriteMessage(producer, ctx, msg)
// if err != nil {*обработка ошибки*}
func WriteMessage(w *kafka.Writer, ctx context.Context, msg []byte) error {
	err := w.WriteMessages(ctx,
		kafka.Message{
			Key:   nil,
			Value: []byte(msg),
		},
	)
	if err != nil {
		log.Printf("Failed to write message: %v", err)
		return err
	}
	return nil
}
