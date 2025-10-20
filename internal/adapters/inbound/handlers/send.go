package handlers

import websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"

type HandlerSendMessage struct{}

func NewHandlerSendMessage(router *websocket.HandlerChain) *HandlerSendMessage {
	// router.HandleFunc()

	return &HandlerSendMessage{}
}
