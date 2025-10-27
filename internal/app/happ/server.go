// Package happ размещает HTTP-шлюз приложения.
package happ

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
)

// HTTPServer инкапсулирует конфигурацию net/http.Server.
type HTTPServer struct {
	log    *slog.Logger
	server *http.Server
}

// ServerDeps агрегирует зависимости, необходимые для создания сервера.
type ServerDeps struct {
	Log    *slog.Logger
	Cfg    *config.HTTPConfig
	Router websocket.Router
	Store  websocket.SessionStore
}

// New настраивает HTTP-хендлеры и возвращает готовый сервер.
func New(deps *ServerDeps) *HTTPServer {
	const op = "Server.NewServer"
	log := deps.Log.With("op", op)

	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr:    deps.Cfg.Address,
		// Защита от Slowloris-подобных атак.
		ReadHeaderTimeout: 10 * time.Second,
	}

	gw := websocket.NewGateway(deps.Store, deps.Router)

	mux.HandleFunc("/realtime/chat", gw.HandleWS)

	log.Info(
		"Successful HTTP upgraded to WebSocket",
		slog.String("address", server.Addr),
	)

	return &HTTPServer{
		log:    log,
		server: server,
	}
}

// MustStart запускает сервер и паникует при ошибке запуска.
func (s *HTTPServer) MustStart() {
	if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.log.Error(
			"HTTP server error to start",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
}

// Start слушает входящие HTTP-подключения.
func (s *HTTPServer) Start() error {
	s.log.Info("HTTP server starting",
		slog.String("address", s.server.Addr),
	)
	if err := s.server.ListenAndServe(); err != nil {
		return fmt.Errorf("http listen and serve: %w", err)
	}
	return nil
}

// Stop корректно завершает работу HTTP-сервера.
func (s *HTTPServer) Stop(ctx context.Context) error {
	defer s.log.Info("HTTP server stopping")
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}
	return nil
}
