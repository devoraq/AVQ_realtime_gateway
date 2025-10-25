// Package main boots the realtime gateway.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/DENFNC/devPractice/internal/adapters/outbound/logger"
	"github.com/DENFNC/devPractice/internal/app"
)

const configPath = "config/config.yaml"

func main() {
	cfg := config.LoadConfig(configPath)
	log := initLogger()

	app := app.New(&app.Deps{
		Log: log,
		Cfg: cfg,
	})

	app.StartAsync()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)

	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Error(
			"Error shutdown app",
			slog.String("err", err.Error()),
		)
		os.Exit(1)
	}
}

func initLogger() *slog.Logger {
	logHandler := logger.NewPrettyHandler(os.Stdout, logger.PrettyHandlerOptions{})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
	return logger
}
