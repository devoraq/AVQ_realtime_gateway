package handlers

import (
	"net/http"

	websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
)

type SendHandler struct{}

func NewSendHandler(mux *http.ServeMux) *SendHandler {
	handler := &SendHandler{}
	{
		mux.HandleFunc("realtime/ws/", nil)
	}

	return handler
}

func (sh *SendHandler) HandleSendMessage(session *websocket.Session, env websocket.Envelope) {}
