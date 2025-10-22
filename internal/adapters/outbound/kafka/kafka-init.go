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

	go NewTopic(cfg.Network, cfg.Address, cfg.TestTopic, 0, log)
	consumer := createReader(cfg.Address, cfg.TestTopic, cfg.GroupID)
	producer := createWriter(cfg.Address, cfg.TestTopic)

	return &Kafka{
		consumer: consumer,
		producer: producer,
		log:      log,
	}
}

// NewTopic создает указанный топик или проверяет его наличие, предпринимая
// несколько попыток подключения к брокеру. При достижении лимита ретраев
// фиксирует ошибку в логах.
func NewTopic(network, address, topic string, partition int, log *slog.Logger) {
	for i := 0; i < maxRetries; i++ {
		_, err := kafka.DialLeader(context.Background(), network, address, topic, partition)
		if err == nil {
			break
		}
		log.Error("kafka leader dial failed", "attempt", i+1, "max_attempts", maxRetries, "topic", topic, "err", err)
		if i == maxRetries-1 {
			log.Error("kafka leader dial exhausted", "topic", topic, "err", err)
			return
		}
		time.Sleep(time.Second)
	}
	log.Info("kafka topic ensured", "topic", topic, "address", address)
}

func ensureKafkaConnection(ctx context.Context, network, address string) error {
	conn, err := kafka.DialContext(ctx, network, address)
	if err != nil {
		return err
	}
	return conn.Close()
}
