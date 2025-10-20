package main

import (
	"log/slog"
	"os"

	"github.com/DENFNC/devPractice/infrastructure/config"
	"github.com/DENFNC/devPractice/infrastructure/logger"
	kvstore "github.com/DENFNC/devPractice/infrastructure/store/kv-store"
	inbound_http "github.com/DENFNC/devPractice/internal/app/http"
)

const configPath = "config/config.yaml"

func main() {
	cfg := config.LoadConfig(configPath)
	log := initLogger()

	store := kvstore.NewRedis(&cfg.RedisConfig, log)

	server := inbound_http.NewServer(log, store, &cfg.HTTPConfig)

	if err := server.Start(); err != nil {
		log.Error("Failed start server",
			slog.String("err", err.Error()),
		)
	}
}

func initLogger() *slog.Logger {
	logHandler := logger.NewPrettyHandler(os.Stdout, logger.PrettyHandlerOptions{})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
	return logger
}
