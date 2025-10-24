// Package kafka реализует инфраструктурный адаптер для взаимодействия с
// брокером Apache Kafka, предоставляя унифицированные обертки над продюсером,
// консьюмером и созданием топиков.
package kafka

import (
	"context"
	"log/slog"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/segmentio/kafka-go"
)

// Kafka инкапсулирует состояния продюсера и консьюмера для единого доступа
// к возможностям брокера из прикладного кода.
type Kafka struct {
	name     string
	consumer *kafka.Reader
	producer *kafka.Writer
	deps     *KafkaDeps
}

// KafkaDeps хранит основные зависимости структуры Kafka:
// логгер и конфигурация кафки.
type KafkaDeps struct {
	Log *slog.Logger
	Cfg *config.KafkaConfig
}

// NewKafka создает и наполняет структуру кафки базовыми полями:
// имя компонента и его зависимости.
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

// Метод Name возвращает имя компонента
func (k *Kafka) Name() string { return k.name }

// Метод Start принимает контекст, проверяет доступность соединения,
// создает и добавляет в структуру кафки набор продюсер/консьюмер.
// Логгирует и возвращает ошибку при наличии таковой. При успешном
// запуске сообщит о нем и вернет nil.
func (k *Kafka) Start(ctx context.Context) error {
	if err := ensureKafkaConnection(ctx, k.deps.Cfg.Network, k.deps.Cfg.Address); err != nil {
		k.deps.Log.Error(
			"Kafka connection failed",
			slog.String("network", k.deps.Cfg.Network),
			slog.String("address", k.deps.Cfg.Address),
			slog.String("error", err.Error()),
		)
		return err
	}

	k.consumer = createReader(k.deps.Cfg.Address, k.deps.Cfg.TestTopic, k.deps.Cfg.GroupID)
	k.producer = createWriter(k.deps.Cfg.Address, k.deps.Cfg.TestTopic)

	k.deps.Log.Info(
		"Connected to Kafka",
		slog.String("network", k.deps.Cfg.Network),
		slog.String("address", k.deps.Cfg.Address),
		slog.String("group_id", k.deps.Cfg.GroupID),
		slog.String("topic", k.deps.Cfg.TestTopic),
	)
	return nil
}

// Метод Stop принимает контекст, останавливает работу продюсера
// и консьюмера, возвращает ошибку при неудаче закрытия
// соединения. Если все прошло по плану, возвращает nil.
func (k *Kafka) Stop(ctx context.Context) error {
	if k.consumer != nil {
		if err := k.consumer.Close(); err != nil {
			k.deps.Log.Error(
				"Failed to close Kafka consumer connection",
				slog.String("address", k.deps.Cfg.Address),
				slog.String("group_id", k.deps.Cfg.GroupID),
				slog.String("topic", k.deps.Cfg.TestTopic),
				slog.String("error", err.Error()),
			)
			return err
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
			return err
		}
	}

	k.deps.Log.Info(
		"Kafka connections closed",
		slog.String("address", k.deps.Cfg.Address),
		slog.String("group_id", k.deps.Cfg.GroupID),
		slog.String("topic", k.deps.Cfg.TestTopic),
	)
	return nil
}

// ensureKafkaConnection проверяет соединение кафки в указанной сети
// и по указанному адресу. Возвращает ошибку - результат попытки
// закрыть соединение по передаваемым параметрам.
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
		k.deps.Log.Error("kafka write message failed", "err", err)
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
			k.deps.Log.Error("kafka read message failed", "err", err)
			break
		}
		k.deps.Log.Info("kafka message received",
			"topic", m.Topic,
			"offset", m.Offset,
			"value", string(m.Value),
		)

		if err := k.consumer.CommitMessages(ctx, m); err != nil {
			k.deps.Log.Error("kafka commit message failed",
				"topic", m.Topic,
				"offset", m.Offset,
				"err", err,
			)
			break
		}
		k.deps.Log.Info("kafka message committed",
			"topic", m.Topic,
			"offset", m.Offset,
		)
	}
}
