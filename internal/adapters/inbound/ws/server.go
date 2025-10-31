package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

var (
	sessionsMu  sync.RWMutex
	sessionPool = make(map[string]*Session)
)

// Session хранит метаданные WebSocket-подключения.
type Session struct {
	ID     uuid.UUID
	UserID uuid.UUID

	conn   net.Conn
	router Router
}

// NewSession создаёт сессию и генерирует служебные идентификаторы для трассировки.
func NewSession(conn net.Conn, router Router) (*Session, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	userID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}

	return &Session{
		ID:     id,
		UserID: userID,
		conn:   conn,
		router: router,
	}, nil
}

func registerSession(session *Session) {
	if session == nil {
		return
	}

	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	sessionPool[session.ID.String()] = session
}

func unregisterSession(sessionID string) {
	if sessionID == "" {
		return
	}

	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	delete(sessionPool, sessionID)
}

// SendToSession отправляет JSON-сообщение конкретной сессии.
func SendToSession(ctx context.Context, sessionID string, messageType string, payload any) error {
	sessionsMu.RLock()
	session := sessionPool[sessionID]
	sessionsMu.RUnlock()

	if session == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}

	return session.JSON(ctx, messageType, payload)
}

// Close закрывает сетевое соединение.
func (s *Session) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("close websocket connection: %w", err)
	}
	return nil
}

// ReadLoop непрерывно читает сообщения клиента и передаёт их роутеру.
func (s *Session) ReadLoop(ctx context.Context) error {
	for {
		msg, op, err := wsutil.ReadClientData(s.conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("read client data: %w", err)
		}

		if err := s.handleOperation(ctx, op, msg); err != nil {
			if errors.Is(err, ErrNoRouteMatched) {
				continue
			}
			return err
		}
	}
}

// WriteMessage отправляет сообщение клиенту с указанным типом опкода.
func (s *Session) WriteMessage(ctx context.Context, op ws.OpCode, payload []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := wsutil.WriteServerMessage(s.conn, op, payload); err != nil {
		return fmt.Errorf("write server message: %w", err)
	}
	return nil
}

func (s *Session) handleOperation(ctx context.Context, op ws.OpCode, payload []byte) error {
	switch op {
	case ws.OpText:
		var env Envelope

		if err := json.Unmarshal(payload, &env); err != nil {
			return s.Error(ctx, ErrorCodeInvalidEnvelope, "failed to decode envelope", "")
		}
		if s.router == nil {
			return s.Error(ctx, ErrorCodeInternal, "router is not configured", "")
		}
		if err := s.router.Route(ctx, s, env); err != nil {
			if errors.Is(err, ErrNoRouteMatched) {
				return s.Error(ctx, ErrorCodeRouteNotFound, "no handler for envelope type", "")
			}
			_ = s.Error(ctx, ErrorCodeInternal, "handler execution failed", "")
			return fmt.Errorf("route websocket envelope: %w", err)
		}
		return nil
	case ws.OpPing:
		if err := wsutil.WriteServerMessage(s.conn, ws.OpPong, nil); err != nil {
			return fmt.Errorf("write pong: %w", err)
		}
		return nil
	default:
		return nil
	}
}

// JSON сериализует payload и отправляет его клиенту как текстовое сообщение.
func (s *Session) JSON(ctx context.Context, messageType string, payload any) error {
	body := struct {
		Type    string `json:"type"`
		Payload any    `json:"payload"`
	}{messageType, payload}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal %s envelope: %w", messageType, err)
	}

	return s.WriteMessage(ctx, ws.OpText, data)
}

func (s *Session) Error(ctx context.Context, code, message, details string) error {
	payload := ErrorPayload{Code: code, Message: message}
	if details != "" {
		payload.Details = details
	}
	return s.JSON(ctx, MessageTypeError, payload)
}
