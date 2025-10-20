package inbound_http

import (
	"fmt"
	"net/http"

	websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
)

type Server struct {
	address string
	server  *http.Server
}

func NewServer(address string, mux *http.ServeMux) *Server {
	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	gw := websocket.NewGateway()

	mux.HandleFunc("/realtime/ws", gw.HandleWS)

	return &Server{
		address: address,
		server:  server,
	}
}

func (s *Server) Start() error {
	fmt.Println("старт сервера")
	if err := s.server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
