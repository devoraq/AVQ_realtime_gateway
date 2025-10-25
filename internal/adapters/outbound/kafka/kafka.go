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

// Kafka управляет жизненным циклом соединений с Kafka:
// создаёт/закрывает продюсера и консюмера, проверяет доступность брокера
// и предоставляет базовые операции отправки/чтения сообщений.
type Kafka struct {
	name     string
	consumer *kafka.Reader
	producer *kafka.Writer
	deps     *KafkaDeps
}

// KafkaDeps содержит зависимости рантайма для Kafka-адаптера:
// логгер и конфигурацию подключения к брокеру.
//
//nolint:revive // осознанно оставляем имя KafkaDeps
type KafkaDeps struct {
	Log *slog.Logger
	Cfg *config.KafkaConfig
}

// NewKafka валидирует переданные зависимости и возвращает экземпляр адаптера.
// Паника возникает, если отсутствует конфигурация или логгер.
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

// Name возвращает символьный идентификатор компонента.
func (k *Kafka) Name() string { return k.name }

// Start устанавливает соединение с брокером (health-check),
// инициализирует консюмера и продюсера и логирует параметры подключения.
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

// Stop корректно закрывает соединения консюмера и продюсера,
// логируя ошибки закрытия при их возникновении.
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

// ensureKafkaConnection выполняет проверку доступности брокера:
// открывает и закрывает TCP-соединение к адресу Kafka.
// Не экспортируется намеренно.
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

// WriteMessage публикует одно сообщение в настроенный Kafka-топик.
// Возвращает ошибку с обёрткой при сбое записи.
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

// StartConsuming запускает непрерывное чтение сообщений из топика
// с коммитом оффсетов. Останавливается при отмене контекста.
// При временных ошибках чтения делает паузы и продолжает работу.
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
