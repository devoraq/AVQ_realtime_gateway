package inbound_http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/DENFNC/devPractice/infrastructure/config"
	"github.com/DENFNC/devPractice/internal/adapters/inbound/handlers"
	websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
)

type Server struct {
	log    *slog.Logger
	server *http.Server
}

func NewServer(log *slog.Logger, store websocket.SessionStore, cfg *config.HTTPConfig) *Server {
	const op = "Server.NewServer"
	log = log.With("op", op)

	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr:    cfg.Address,
	}

	sendService := handlers.NewLoggingMessageService(log.With("handler", "send"))

	router := websocket.NewHandlerChain()
	router.HandleFunc(websocket.EnvMessage, func(ctx context.Context, session *websocket.Session, env websocket.Envelope) error {
		return handlers.HandleSend(ctx, session, env, sendService)
	})

	gw := websocket.NewGateway(store, router)

	mux.HandleFunc("/realtime/ws", gw.HandleWS)
	log.Info(
		"Successful HTTP upgraded to WebSocket",
		slog.String("address", server.Addr),
	)

	return &Server{
		log:    log,
		server: server,
	}
}

func (s *Server) Start() error {
	s.log.Info("HTTP server starting",
		slog.String("address", s.server.Addr),
	)
	if err := s.server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
