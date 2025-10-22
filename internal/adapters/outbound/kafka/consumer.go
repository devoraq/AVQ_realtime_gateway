package kafka

import (
	"context"
	"log/slog"

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

// StartConsuming запускает бесконечный цикл чтения сообщений для переданного
// консьюмера. Подходит для исполнения в отдельной горутине и логирует
// результаты обработки каждого сообщения.
//
// Пример использования:
//
// consumer := createReader(cfg.Address, cfg.MyTopic, cfg.MyGroupID, 69)
// go StartConsuming(ctx, log, consumer)
func StartConsuming(ctx context.Context, log *slog.Logger, r *kafka.Reader) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			log.Error("kafka read message failed", "err", err)
			break
		}
		log.Info("kafka message received",
			"topic", m.Topic,
			"offset", m.Offset,
			"value", string(m.Value),
		)

		if err := r.CommitMessages(ctx, m); err != nil {
			log.Error("kafka commit message failed",
				"topic", m.Topic,
				"offset", m.Offset,
				"err", err,
			)
			break
		}
		log.Info("kafka message committed",
			"topic", m.Topic,
			"offset", m.Offset,
		)
	}
}
