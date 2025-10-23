package happ

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
)

type HTTPServer struct {
	log    *slog.Logger
	server *http.Server
}

type ServerDeps struct {
	Log *slog.Logger
	Cfg *config.HTTPConfig

	Store websocket.SessionStore
}

func New(deps *ServerDeps) *HTTPServer {
	const op = "Server.NewServer"
	log := deps.Log.With("op", op)

	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr:    deps.Cfg.Address,
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

func (s *HTTPServer) MustStart() {
	if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.log.Error(
			"HTTP server error to start",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
}

func (s *HTTPServer) Start() error {
	s.log.Info("HTTP server starting",
		slog.String("address", s.server.Addr),
	)
	if err := s.server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	defer s.log.Info("HTTP server stopping")
	return s.server.Shutdown(ctx)
}
