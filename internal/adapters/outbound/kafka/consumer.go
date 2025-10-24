package kafka

import (
	"github.com/segmentio/kafka-go"
)

// Создает нового консьюмера, читает передаваемый на входе топик.
// Запуск происходит в инициализации кафки
func createReader(address, topic, groupID string) *kafka.Reader {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{address},
		Topic:   topic,
		GroupID: groupID,
	})
	return r
}
