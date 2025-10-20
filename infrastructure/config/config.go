package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTPConfig  `yaml:"http"`
	RedisConfig `yaml:"redis"`
}

type HTTPConfig struct {
	Address string `yaml:"address"`
}

type RedisConfig struct {
	Address  string `yaml:"address" required:"true"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	Timeout  time.Duration
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
