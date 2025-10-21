package kafka

import (
	"context"
	"log/slog"

	"github.com/DENFNC/devPractice/infrastructure/config"
)

const configPath = "config/config.yaml"

func TestKafka(log *slog.Logger) error {
	cfg := config.LoadConfig(configPath)
	k := NewKafka(&cfg.KafkaConfig, log)
	ctx := context.Background()

	go StartConsuming(k.consumer)

	msg := []byte("Hey there, this is my message")
	err := WriteMessage(k.producer, ctx, msg)
	if err != nil {
		return err
	}
	return nil
}
