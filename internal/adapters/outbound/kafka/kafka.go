package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/DENFNC/devPractice/pkg/retry"
	"github.com/segmentio/kafka-go"
)

// Kafka управляет жизненным циклом соединений с Kafka: создаёт/закрывает
// продюсера и консюмера, проверяет доступность брокера и предоставляет
// базовые операции отправки/чтения сообщений.
//
// Пример:
//
//	kfk := kafka.NewKafka(&kafka.KafkaDeps{Cfg: cfg, Log: log})
//	if err := kfk.Start(ctx); err != nil {
//		return err
//	}
//	defer kfk.Stop(ctx)
//
//	kfk.AddDeliveryHandler(func(ctx context.Context, payload []byte) error {
//		var msg Message
//		return json.Unmarshal(payload, &msg)
//	})
//	go kfk.StartConsuming(ctx)
//
//	if err := kfk.WriteMessage(ctx, []byte(`"ping"`)); err != nil {
//		return err
//	}
//
// Компонент потокобезопасен в части основных операций, однако зарегистрированные
// обработчики выполняются последовательно в порядке добавления.
type Kafka struct {
	name     string
	router   *Router
	consumer *kafka.Reader
	producer *kafka.Writer
	deps     *KafkaDeps
}

// KafkaDeps содержит зависимости рантайма для Kafka-адаптера: логгер и
// конфигурацию подключения к брокеру.
//
// Пример:
//
//	deps := &kafka.KafkaDeps{
//		Cfg: cfg.KafkaConfig,
//		Log: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
//	}
//
//nolint:revive //! осознанно оставляем имя KafkaDeps
type KafkaDeps struct {
	Cfg *config.KafkaConfig
	Log *slog.Logger
}

// NewKafka валидирует переданные зависимости и возвращает экземпляр адаптера.
// Паника возникает, если отсутствует конфигурация или логгер.
//
// Пример:
//
//	cfg := config.MustLoad()
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	kfk := kafka.NewKafka(&kafka.KafkaDeps{Cfg: cfg.KafkaConfig, Log: logger})
func NewKafka(deps *KafkaDeps) *Kafka {
	if deps == nil || deps.Cfg == nil {
		panic("Kafka config cannot be nil")
	}
	if deps.Log == nil {
		panic("Logger cannot be nil")
	}

	router := NewRouter()

	return &Kafka{
		name:   "kafka",
		router: router,
		deps:   deps,
	}
}

// Name возвращает символьный идентификатор компонента.
//
// Пример:
//
//	fmt.Println(kfk.Name()) // kafka
func (k *Kafka) Name() string { return k.name }

// Handle регистрирует обработчик событий для указанного топика Kafka.
func (k *Kafka) Handle(topic string, handler Handler) {
	if handler == nil {
		panic("kafka handler cannot be nil")
	}
	k.router.Handle(topic, handler)
}

// Start устанавливает соединение с брокером (health-check), инициализирует
// консюмера и продюсера и логирует параметры подключения.
//
// Пример:
//
//	if err := kfk.Start(ctx); err != nil {
//		return fmt.Errorf("kafka start: %w", err)
//	}
func (k *Kafka) Start(ctx context.Context) error {
	if err := ensureKafkaConnection(ctx, k.deps.Cfg.Network, k.deps.Cfg.Address); err != nil {
		k.deps.Log.Debug(
			"Kafka connection failed",
			slog.String("network", k.deps.Cfg.Network),
			slog.String("address", k.deps.Cfg.Address),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%w: %w", ErrEnsureConnection, err)
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
//
// Пример:
//
//	defer func() {
//		if err := kfk.Stop(ctx); err != nil {
//			log.Warn("kafka stop", "err", err)
//		}
//	}()
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
//
// Пример:
//
//	payload := []byte(`{"id":"123","event":"created"}`)
//	if err := kfk.WriteMessage(ctx, payload); err != nil {
//		return fmt.Errorf("publish: %w", err)
//	}
func (k *Kafka) WriteMessage(ctx context.Context, msg []byte) error {
	err := k.producer.WriteMessages(ctx,
		kafka.Message{
			Key:   nil,
			Value: msg,
		},
	)
	if err != nil {
		k.deps.Log.Error("kafka write message failed", "err", fmt.Errorf("%w: %w", ErrWriteMessage, err))
		return fmt.Errorf("%w: %w", ErrWriteMessage, err)
	}
	return nil
}

// StartConsuming запускает непрерывное чтение сообщений из топика с коммитом
// оффсетов. Останавливается при отмене контекста. При временных ошибках чтения
// делает паузы и продолжает работу, используя стратегии повторных попыток,
// заданные в конфигурации.
//
// Пример:
//
//	go func() {
//		kfk.StartConsuming(ctx)
//	}()
//	// ... позже cancel() остановит цикл чтения и метод завершится.
func (k *Kafka) StartConsuming(ctx context.Context) {
	defer func() {
		if err := k.consumer.Close(); err != nil {
			k.deps.Log.Warn("consumer close failed", "err", err)
		}
	}() // безопасное закрытие

	backoff := retry.NewBackoff(&k.deps.Cfg.FetchBackoff)

	for {
		if ctx.Err() != nil {
			k.deps.Log.Debug("Kafka consumer stopped", "err", ctx.Err())
			return
		}

		msg, err := k.fetch(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				k.deps.Log.Debug("Kafka consumer context canceled")
				return
			}
			k.deps.Log.Error("fetch failed", "err", fmt.Errorf("%w: %w", ErrFetchMessage, err))
			backoff.Sleep(ctx)
			continue
		}
		backoff.Reset()

		if err := k.router.Dispatch(ctx, msg); err != nil {
			// Если обработка упала — НЕ коммитим, чтобы переиграть позже.
			k.deps.Log.Error("handler failed", "err", err, "topic", msg.Topic, "offset", msg.Offset)
			//! либо ретраим локально с ограничением, либо отдаем в DLQ.
			//! if dlqErr := k.toDLQ(ctx, msg, err); dlqErr != nil {
			//! 	k.deps.Log.Error("dlq failed", "err", dlqErr)
			//! 	// Без DLQ оставляем без коммита → повторная доставка.
			//! }
			continue
		}

		if err := k.commitWithRetry(ctx, msg); err != nil {
			k.deps.Log.Error("commit failed", "err", fmt.Errorf("%w: %w", ErrCommitMessage, err),
				"topic", msg.Topic, "offset", msg.Offset)
			// Не удалось зафиксировать — сообщение придет снова (at-least-once).
			continue
		}
		k.deps.Log.Debug("message committed", "topic", msg.Topic, "offset", msg.Offset)
	}
}

func (k *Kafka) fetch(ctx context.Context) (kafka.Message, error) {
	m, err := k.consumer.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("error fetch: %w", err)
	}
	k.deps.Log.Debug("message received", "topic", m.Topic, "offset", m.Offset, "size", len(m.Value))
	return m, nil
}

func (k *Kafka) commitWithRetry(ctx context.Context, m kafka.Message) error {
	backoff := retry.NewBackoff(&k.deps.Cfg.CommitBackoff)
	attempts := backoff.Attempts()
	for i := 0; i < attempts; i++ {
		if err := k.consumer.CommitMessages(ctx, m); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			k.deps.Log.Warn("commit retry", "attempt", i+1, "err", err)
			backoff.Sleep(ctx)
			continue
		}
		return nil
	}
	return fmt.Errorf("commit retries exceeded: %w", ErrCommitMessage)
}

// func (k *Kafka) toDLQ(ctx context.Context, m kafka.Message, cause error) error {
// 	if k.dlq == nil {
// 		return nil
// 	}
// 	return k.dlq.Publish(ctx, m, cause)
// }
