package kafka

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/segmentio/kafka-go"
)

const maxRetries = 5

type Kafka struct {
	consumer *kafka.Reader
	producer *kafka.Writer
	log      *slog.Logger
}

// Создает и наполняет структуру кафки консьюмером, продюсером и логгером.
// Запускает топики
func NewKafka(cfg *config.KafkaConfig, log *slog.Logger) *Kafka {
	if cfg == nil {
		panic("Kafka config cannot be nil")
	}
	if log == nil {
		panic("Logger cannot be nil")
	}

	go NewTopic(cfg.Network, cfg.Address, cfg.TestTopic, 0)
	consumer := createReader(cfg.Address, cfg.TestTopic, cfg.GroupID)
	producer := createWriter(cfg.Address, cfg.TestTopic)

	return &Kafka{
		consumer: consumer,
		producer: producer,
		log:      log,
	}
}

// Создает новый топик, принимает на вход параметры конфигурации кафки:
// сеть, адрес, название топика. Партиция указывается вручную.
// Запускать как горутину
func NewTopic(network, address, topic string, partition int) {
	for i := 0; i < maxRetries; i++ {
		_, err := kafka.DialLeader(context.Background(), network, address, topic, partition)
		if err == nil {
			break
		}
		log.Printf("Error creating Kafka connection (attempt %d/%d): %v", i+1, maxRetries, err)
		if i == maxRetries-1 {
			panic("Failed to connect to Kafka after all attempts")
		}
		time.Sleep(time.Second)
	}
	log.Printf("Topic %s created successfuly on %s", topic, address)
}
