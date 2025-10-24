// Package kafka реализует инфраструктурный адаптер для взаимодействия с
// брокером Apache Kafka, предоставляя унифицированные обертки над продюсером,
// консьюмером и созданием топиков.
package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/segmentio/kafka-go"
)

const maxRetries = 5

// Kafka инкапсулирует состояния продюсера и консьюмера для единого доступа
// к возможностям брокера из прикладного кода.
type Kafka struct {
	consumer *kafka.Reader
	producer *kafka.Writer
	log      *slog.Logger
}

// Создает и наполняет структуру кафки консьюмером, продюсером и логгером.
// Запускает топики
// NewKafka принимает конфигурацию брокера и логгер, проверяет доступность
// соединения, создает служебный топик и возвращает готовый к работе набор
// продюсер/консьюмер. При ошибке подключения функция логирует сбой и
// возвращает nil.
func NewKafka(cfg *config.KafkaConfig, log *slog.Logger) *Kafka {
	if cfg == nil {
		panic("Kafka config cannot be nil")
	}
	if log == nil {
		panic("Logger cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ensureKafkaConnection(ctx, cfg.Network, cfg.Address); err != nil {
		log.Error("kafka connection check failed", "err", err)
		return nil
	}

	consumer := createReader(cfg.Address, cfg.TestTopic, cfg.GroupID)
	producer := createWriter(cfg.Address, cfg.TestTopic)

	return &Kafka{
		consumer: consumer,
		producer: producer,
		log:      log,
	}
}

func ensureKafkaConnection(ctx context.Context, network, address string) error {
	conn, err := kafka.DialContext(ctx, network, address)
	if err != nil {
		return err
	}
	return conn.Close()
}

// WriteMessage отправляет переданное сообщение через подготовленный продюсер,
// фиксируя ошибки в журнале и возвращая их вызывающему коду.
//
// Пример использования:
//
// msg := []byte("Some message as a byte slice")
// err := WriteMessage(ctx, msg)
// if err != nil {*обработка ошибки*}
func (k *Kafka) WriteMessage(ctx context.Context, msg []byte) error {
	err := k.producer.WriteMessages(ctx,
		kafka.Message{
			Key:   nil,
			Value: msg,
		},
	)
	if err != nil {
		k.log.Error("kafka write message failed", "err", err)
		return err
	}
	return nil
}

// StartConsuming запускает бесконечный цикл чтения сообщений для переданного
// консьюмера. Подходит для исполнения в отдельной горутине и логирует
// результаты обработки каждого сообщения.
//
// Пример использования:
//
// go StartConsuming(ctx)
func (k *Kafka) StartConsuming(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		m, err := k.consumer.FetchMessage(ctx)
		if err != nil {
			k.log.Error("kafka read message failed", "err", err)
			break
		}
		k.log.Info("kafka message received",
			"topic", m.Topic,
			"offset", m.Offset,
			"value", string(m.Value),
		)

		if err := k.consumer.CommitMessages(ctx, m); err != nil {
			k.log.Error("kafka commit message failed",
				"topic", m.Topic,
				"offset", m.Offset,
				"err", err,
			)
			break
		}
		k.log.Info("kafka message committed",
			"topic", m.Topic,
			"offset", m.Offset,
		)
	}
}
