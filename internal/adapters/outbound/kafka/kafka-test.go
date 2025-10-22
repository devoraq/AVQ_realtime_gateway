package kafka

import (
	"context"
	"log/slog"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
)

const configPath = "config/config.yaml"

// TestKafka выполняет проверку жизнеспособности адаптера: создает соединение
// с брокером, запускает консьюмера и публикует тестовое сообщение в
// конфигурационный топик.
func TestKafka(log *slog.Logger) error {
	cfg := config.LoadConfig(configPath)
	k := NewKafka(cfg.KafkaConfig, log)
	ctx := context.Background()

	go StartConsuming(ctx, log, k.consumer)

	msg := []byte("Hey there, this is my message")
	err := WriteMessage(ctx, log, k.producer, msg)
	if err != nil {
		return err
	}
	return nil
}
