package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/DENFNC/devPractice/internal/adapters/outbound/logger"
	kvstore "github.com/DENFNC/devPractice/internal/adapters/outbound/store/kv-store"
	"github.com/DENFNC/devPractice/internal/app"
)

const configPath = "config/config.yaml"

func main() {
	cfg := config.LoadConfig(configPath)
	log := initLogger()

	store := kvstore.NewRedis(cfg.RedisConfig, log)

	app := app.New(&app.AppDeps{
		Log:   log,
		Cfg:   cfg,
		Store: store,
	})

	app.StartAsync()

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT)

	done := make(chan struct{})

	go func() {
		select {
		case <-sig:
			log.Info("Shutting down server...")
			app.Shutdown()
			close(done)
			return
		case <-time.After(30 * time.Second):
			log.Info("Shutting down server due to timeout...")
			os.Exit(1)
		}
	}()

	<-done
}

func initLogger() *slog.Logger {
	logHandler := logger.NewPrettyHandler(os.Stdout, logger.PrettyHandlerOptions{})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
	return logger
}
