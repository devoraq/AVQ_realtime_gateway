// Package app отвечает за сборку приложения и управление его жизненным циклом.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/inbound/handlers"
	"github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/DENFNC/devPractice/internal/adapters/outbound/kafka"
	kvstore "github.com/DENFNC/devPractice/internal/adapters/outbound/store/kv-store"
	"github.com/DENFNC/devPractice/internal/app/happ"
	"github.com/DENFNC/devPractice/internal/usecases"
)

// App объединяет инфраструктурные адаптеры с HTTP-сервером и управляет их жизненным циклом.
type App struct {
	deps *Deps

	happ      *happ.HTTPServer
	container *Container
	kafka     *kafka.Kafka

	startOnce    sync.Once
	shutdownOnce sync.Once
	wg           sync.WaitGroup

	consumerCancel context.CancelFunc
}

// Deps описывает зависимости, необходимые для сборки приложения.
type Deps struct {
	Log *slog.Logger
	Cfg *config.Config
}

// New собирает компоненты, запускает инфраструктурные адаптеры и возвращает готовый экземпляр.
func New(deps *Deps) *App {
	container, store, kfk := initInfrastructure(deps)

	router, consumer := initMessaging(deps, store, kfk)

	hserver := happ.New(&happ.ServerDeps{
		Log:    deps.Log,
		Cfg:    deps.Cfg.HTTPConfig,
		Store:  store,
		Router: router,
	})

	app := &App{
		deps:           deps,
		container:      container,
		happ:           hserver,
		kafka:          kfk,
		consumerCancel: consumer.cancel,
	}

	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		kfk.StartConsuming(consumer.ctx)
	}()

	return app
}

type consumerControl struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// StartAsync запускает HTTP-сервер в отдельной горутине и гарантирует однократный старт.
func (a *App) StartAsync() {
	a.startOnce.Do(func() {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.happ.MustStart()
		}()
	})
}

// Shutdown корректно останавливает HTTP-сервер и все зарегистрированные компоненты.
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
		if a.consumerCancel != nil {
			a.consumerCancel()
		}
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

func initInfrastructure(deps *Deps) (*Container, *kvstore.Redis, *kafka.Kafka) {
	container := NewContainer(deps.Log, deps.Cfg)

	store := kvstore.NewRedis(&kvstore.RedisDeps{
		Log: deps.Log,
		Cfg: deps.Cfg.RedisConfig,
	})
	kfk := kafka.NewKafka(&kafka.KafkaDeps{
		Log: deps.Log,
		Cfg: deps.Cfg,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container.Add(store, kfk)
	if err := container.StartAll(ctx); err != nil {
		deps.Log.Error("Failed to start infrastructure components after multiple retries", slog.String("error", err.Error()))
		panic(fmt.Errorf("start components: %w", err))
	}

	return container, store, kfk
}

func initMessaging(
	deps *Deps,
	store *kvstore.Redis,
	kfk *kafka.Kafka,
) (*ws.HandlerChain, consumerControl) {
	router := ws.NewHandlerChain()
	notifier := ws.NewNotifier(store)

	usecase := usecases.NewMessageUsecase(kfk, notifier)
	kfk.Handle(deps.Cfg.TestTopic, usecase.HandleDelivery)

	handlers.NewSendMessageHandler(&handlers.MessageHandlerDeps{
		Usecase: usecase,
		Router:  router,
		Store:   store,
	})

	ctx, cancel := context.WithCancel(context.Background())

	return router, consumerControl{ctx: ctx, cancel: cancel}
}
