package handlers

import (
	"context"
	"fmt"

	websocket "github.com/DENFNC/devPractice/internal/adapters/inbound/ws"
)

type MessageType string

const (
	MessageTypeSend MessageType = "send_message"
)

type ChatUsecase interface {
	SendMessage(userID string, message string) error
}

type ChatHandler struct {
	usecase ChatUsecase
}

type ChatHandlerDeps struct {
	Usecase ChatUsecase
	Router  *websocket.HandlerChain
}

func NewSendMessageHandler(deps *ChatHandlerDeps) *ChatHandler {
	h := &ChatHandler{
		usecase: deps.Usecase,
	}

	{
		deps.Router.HandleFunc(string(MessageTypeSend), h.SendMessage)
	}

	return h
}

func (h *ChatHandler) SendMessage(ctx context.Context, s *websocket.Session, env websocket.Envelope) error {
	fmt.Println(env)
	// TODO: implement send message logic
	return nil
}
