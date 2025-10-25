package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/segmentio/kafka-go"
)

// Kafka manages Kafka producer and consumer lifecycle.
type Kafka struct {
	name     string
	consumer *kafka.Reader
	producer *kafka.Writer
	deps     *KafkaDeps
}

// KafkaDeps contains runtime dependencies for the Kafka adapter.
//
//nolint:revive
type KafkaDeps struct {
	Log *slog.Logger
	Cfg *config.KafkaConfig
}

// NewKafka validates dependencies and prepares the adapter instance.
func NewKafka(deps *KafkaDeps) *Kafka {
	if deps.Cfg == nil {
		panic("Kafka config cannot be nil")
	}
	if deps.Log == nil {
		panic("Logger cannot be nil")
	}

	return &Kafka{
		name: "kafka",
		deps: deps,
	}
}

// Name returns component identifier.
func (k *Kafka) Name() string { return k.name }

// Start establishes producer and consumer connections and verifies broker availability.
func (k *Kafka) Start(ctx context.Context) error {
	if err := ensureKafkaConnection(ctx, k.deps.Cfg.Network, k.deps.Cfg.Address); err != nil {
		k.deps.Log.Debug(
			"Kafka connection failed",
			slog.String("network", k.deps.Cfg.Network),
			slog.String("address", k.deps.Cfg.Address),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("ensure kafka connection: %w", err)
	}

	k.consumer = createReader(k.deps.Cfg.Address, k.deps.Cfg.TestTopic, k.deps.Cfg.GroupID)
	k.producer = createWriter(k.deps.Cfg.Address, k.deps.Cfg.TestTopic)

	k.deps.Log.Debug(
		"Connected to Kafka",
		slog.String("network", k.deps.Cfg.Network),
		slog.String("address", k.deps.Cfg.Address),
		slog.String("group_id", k.deps.Cfg.GroupID),
		slog.String("topic", k.deps.Cfg.TestTopic),
	)
	return nil
}

// Stop gracefully closes consumer and producer connections.
func (k *Kafka) Stop(_ context.Context) error {
	if k.consumer != nil {
		if err := k.consumer.Close(); err != nil {
			k.deps.Log.Error(
				"Failed to close Kafka consumer connection",
				slog.String("address", k.deps.Cfg.Address),
				slog.String("group_id", k.deps.Cfg.GroupID),
				slog.String("topic", k.deps.Cfg.TestTopic),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("close kafka consumer: %w", err)
		}
	}

	if k.producer != nil {
		if err := k.producer.Close(); err != nil {
			k.deps.Log.Error(
				"Failed to close Kafka producer connection",
				slog.String("address", k.deps.Cfg.Address),
				slog.String("topic", k.deps.Cfg.TestTopic),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("close kafka producer: %w", err)
		}
	}

	k.deps.Log.Debug(
		"Kafka connections closed",
		slog.String("address", k.deps.Cfg.Address),
		slog.String("group_id", k.deps.Cfg.GroupID),
		slog.String("topic", k.deps.Cfg.TestTopic),
	)
	return nil
}

func ensureKafkaConnection(ctx context.Context, network, address string) error {
	conn, err := kafka.DialContext(ctx, network, address)
	if err != nil {
		return fmt.Errorf("dial kafka broker %s://%s: %w", network, address, err)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close kafka connection: %w", err)
	}
	return nil
}

// WriteMessage pushes a message to the configured Kafka topic.
func (k *Kafka) WriteMessage(ctx context.Context, msg []byte) error {
	err := k.producer.WriteMessages(ctx,
		kafka.Message{
			Key:   nil,
			Value: msg,
		},
	)
	if err != nil {
		k.deps.Log.Error("kafka write message failed", "err", err)
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

// StartConsuming continuously reads messages and commits offsets.
func (k *Kafka) StartConsuming(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			k.deps.Log.Debug("Kafka consumer stopped by context")
			return
		default:
		}

		m, err := k.consumer.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				k.deps.Log.Debug("Kafka consumer context canceled")
				return
			}
			k.deps.Log.Error("kafka read message failed", "err", err)

			time.Sleep(5 * time.Second)
			continue
		}
		k.deps.Log.Debug("kafka message received",
			"topic", m.Topic,
			"offset", m.Offset,
			"value", string(m.Value),
		)

		if err := k.consumer.CommitMessages(ctx, m); err != nil {
			k.deps.Log.Debug("kafka commit message failed",
				"topic", m.Topic,
				"offset", m.Offset,
				"err", err,
			)
			time.Sleep(2 * time.Second)
			continue
		}
		k.deps.Log.Debug("kafka message committed",
			"topic", m.Topic,
			"offset", m.Offset,
		)
	}
}
