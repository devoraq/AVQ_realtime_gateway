package app

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	kvstore "github.com/DENFNC/devPractice/internal/adapters/outbound/store/kv-store"
	"github.com/DENFNC/devPractice/internal/app/happ"
)

type App struct {
	deps *AppDeps

	happ      *happ.HTTPServer
	container *Container

	startOnce    sync.Once
	shutdownOnce sync.Once
	wg           sync.WaitGroup
}

type AppDeps struct {
	Log *slog.Logger
	Cfg *config.Config
}

func New(deps *AppDeps) *App {
	container := NewContainer()

	store := kvstore.NewRedis(&kvstore.RedisDeps{
		Log: deps.Log,
		Cfg: deps.Cfg.RedisConfig,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	container.Add(store)
	container.StartAll(ctx)

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

func (a *App) StartAsync() {
	a.startOnce.Do(func() {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.happ.MustStart()
		}()
	})
}

func (a *App) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("app shutdown: context is nil")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
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
