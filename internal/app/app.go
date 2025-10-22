package app

import (
	"log/slog"

	websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/DENFNC/devPractice/internal/app/happ"
)

type App struct {
	deps *AppDeps
	happ *happ.HTTPServer
}

type AppDeps struct {
	Log *slog.Logger
	Cfg *config.Config

	Store websocket.SessionStore
}

func New(deps *AppDeps) *App {
	hserver := happ.New(&happ.ServerDeps{
		Log:   deps.Log,
		Cfg:   deps.Cfg.HTTPConfig,
		Store: deps.Store,
	})

	return &App{
		deps: deps,
		happ: hserver,
	}
}

func (a *App) StartAsync() {
	if a.deps.Cfg.AppConfig.WebSocket {
		go a.happ.MustStart()
	}
}

func (a *App) Shutdown() error {
	if a.happ != nil {
		if err := a.happ.Stop(); err != nil {
			return nil
		}
	}
	return nil
}
