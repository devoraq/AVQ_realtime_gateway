// Package happ hosts the HTTP gateway server.
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

// HTTPServer wraps net/http.Server configuration.
type HTTPServer struct {
	log    *slog.Logger
	server *http.Server
}

// ServerDeps aggregates constructor dependencies.
type ServerDeps struct {
	Log *slog.Logger
	Cfg *config.HTTPConfig

	Store websocket.SessionStore
}

// New wires HTTP handlers and returns configured server instance.
func New(deps *ServerDeps) *HTTPServer {
	const op = "Server.NewServer"
	log := deps.Log.With("op", op)

	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr:    deps.Cfg.Address,
		// Protect from Slowloris-style attacks.
		ReadHeaderTimeout: 10 * time.Second,
	}

	router := websocket.NewHandlerChain()

	gw := websocket.NewGateway(deps.Store, router)

	mux.HandleFunc("/realtime/ws", gw.HandleWS)

	log.Info(
		"Successful HTTP upgraded to WebSocket",
		slog.String("address", server.Addr),
	)

	return &HTTPServer{
		log:    log,
		server: server,
	}
}

// MustStart starts the server and panics when it fails.
func (s *HTTPServer) MustStart() {
	if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.log.Error(
			"HTTP server error to start",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
}

// Start listens for incoming HTTP connections.
func (s *HTTPServer) Start() error {
	s.log.Info("HTTP server starting",
		slog.String("address", s.server.Addr),
	)
	if err := s.server.ListenAndServe(); err != nil {
		return fmt.Errorf("http listen and serve: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *HTTPServer) Stop(ctx context.Context) error {
	defer s.log.Info("HTTP server stopping")
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}
	return nil
}
