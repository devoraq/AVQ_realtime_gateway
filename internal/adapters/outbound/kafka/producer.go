package kafka

import (
	"context"
	"log/slog"

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

// WriteMessage отправляет переданное сообщение через подготовленный продюсер,
// фиксируя ошибки в журнале и возвращая их вызывающему коду.
//
// Пример использования:
//
// msg := []byte("Some message as a byte slice")
// err := WriteMessage(ctx, log, producer, msg)
// if err != nil {*обработка ошибки*}
func WriteMessage(ctx context.Context, log *slog.Logger, w *kafka.Writer, msg []byte) error {
	err := w.WriteMessages(ctx,
		kafka.Message{
			Key:   nil,
			Value: []byte(msg),
		},
	)
	if err != nil {
		log.Error("kafka write message failed", "err", err)
		return err
	}
	return nil
}
