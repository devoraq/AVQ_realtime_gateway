// Package config предоставляет структуры конфигурации для инфраструктурных
// адаптеров и сервисов, а также утилиту загрузки параметров из YAML-файла.
// Пакет служит единой точкой инициализации настроек, чтобы остальные слои
// приложения могли полагаться на предварительно валидированные значения.
package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config агрегирует все секции конфигурационного файла приложения.
// Каждое вложенное поле отвечает за конкретный инфраструктурный компонент.
type Config struct {
	*AppConfig   `yaml:"app"`
	*HTTPConfig  `yaml:"http"`
	*RedisConfig `yaml:"redis"`
	*KafkaConfig `yaml:"kafka"`
	*RetryConfig `yaml:"retry"`
}

// AppConfig описывает параметры верхнеуровневого приложения
type AppConfig struct {
}

// HTTPConfig хранит настройки HTTP-сервера, включая bind-адрес, который
// используется адаптерами входящего трафика.
type HTTPConfig struct {
	Address string `yaml:"address"`
}

// RedisConfig инкапсулирует параметры подключения к Redis, такие как адрес,
// пароль, номер базы данных и таймаут, используемые кеш-адаптером.
type RedisConfig struct {
	Address  string        `yaml:"address" required:"true"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	Timeout  time.Duration `yaml:"timeout"`
}

// KafkaConfig содержит настройки брокера Kafka, необходимые для инициализации
// продюсеров, консьюмеров и управления топиками.
type KafkaConfig struct {
	Address   string `yaml:"address"`
	TestTopic string `yaml:"test-topic"`
	GroupID   string `yaml:"group-id"`
	Network   string `yaml:"network"`
}

// RetryConfig определяет параметры для механизма повторных попыток.
type RetryConfig struct {
	Attempts int           `yaml:"attempts"`
	Initial  time.Duration `yaml:"initial"`
	Max      time.Duration `yaml:"max"`
	Factor   float64       `yaml:"factor"`
	Jitter   bool          `yaml:"jitter"`
}

func LoadConfig(path string) *Config {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || info.IsDir() {
		log.Fatalf("Config file not found: %s", path)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	return &cfg
}
