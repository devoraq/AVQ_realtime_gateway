// Package app provides application wiring and lifecycle orchestration.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/DENFNC/devPractice/internal/adapters/outbound/kafka"
	kvstore "github.com/DENFNC/devPractice/internal/adapters/outbound/store/kv-store"
	"github.com/DENFNC/devPractice/internal/app/happ"
)

// App wires infrastructure adapters alongside the HTTP server and orchestrates
// their lifecycle.
type App struct {
	deps *Deps

	happ      *happ.HTTPServer
	container *Container

	startOnce    sync.Once
	shutdownOnce sync.Once
	wg           sync.WaitGroup
}

// Deps describes dependencies required to construct the application.
type Deps struct {
	Log *slog.Logger
	Cfg *config.Config
}

// New assembles every component, eagerly starts infrastructure adapters and
// returns a ready-to-run application instance.
func New(deps *Deps) *App {
	container := NewContainer(deps.Log, deps.Cfg.RetryConfig)

	store := kvstore.NewRedis(&kvstore.RedisDeps{
		Log: deps.Log,
		Cfg: deps.Cfg.RedisConfig,
	})

	kfk := kafka.NewKafka(&kafka.KafkaDeps{
		Log: deps.Log,
		Cfg: deps.Cfg.KafkaConfig,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container.Add(store, kfk)
	if err := container.StartAll(ctx); err != nil {
		deps.Log.Error("Failed to start infrastructure components after multiple retries", slog.String("error", err.Error()))
		panic(fmt.Errorf("start components: %w", err))
	}

	hserver := happ.New(&happ.ServerDeps{
		Log:   deps.Log,
		Cfg:   deps.Cfg.HTTPConfig,
		Store: store,
	})

	return &App{
		deps:      deps,
		container: container,
		happ:      hserver,
	}
}

// StartAsync runs the HTTP server in a goroutine and ensures it starts once.
func (a *App) StartAsync() {
	a.startOnce.Do(func() {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.happ.MustStart()
		}()
	})
}

// Shutdown gracefully stops the HTTP server and all registered components.
func (a *App) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("app shutdown: context is nil")
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("app shutdown: %w", ctx.Err())
	default:
	}

	var errs []error
	a.shutdownOnce.Do(func() {
		if err := a.happ.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
		if err := a.container.StopAll(ctx); err != nil {
			errs = append(errs, err)
		}
		a.wg.Wait()
	})

	return errors.Join(errs...)
}
