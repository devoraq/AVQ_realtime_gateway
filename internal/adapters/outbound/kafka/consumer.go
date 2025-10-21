package kafka

import (
	"context"
	"log"

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

// Начинает чтение топика для консьюмера, которого получила на вход.
// Запускать как горутину.
//
// Пример использования:
//
// consumer := createReader(cfg.Address, cfg.MyTopic, cfg.MyGroupID, 69)
// go StartConsuming(consumer)
func StartConsuming(r *kafka.Reader) {
	ctx := context.Background()
	for {
		m, err := r.FetchMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v\n", err)
			break
		}
		log.Printf("New message at topic/offset [%v/%v]: %s\n",
			m.Topic, m.Offset, string(m.Value))

		if err := r.CommitMessages(ctx, m); err != nil {
			log.Printf("Error committing message: %v\n", err)
			break
		}
		log.Printf("Committed message at topic/offset [%v/%v]\n",
			m.Topic, m.Offset)
	}
}
