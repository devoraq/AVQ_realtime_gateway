package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

// Session keeps metadata about WebSocket connection.
type Session struct {
	ID     uuid.UUID
	UserID uuid.UUID

	conn   net.Conn
	router Router
}

// NewSession constructs a session with random identifiers for tracing.
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

// Close terminates the underlying network connection.
func (s *Session) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("close websocket connection: %w", err)
	}
	return nil
}

// ReadLoop continuously reads client messages and dispatches them to router.
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

func (s *Session) handleOperation(ctx context.Context, op ws.OpCode, payload []byte) error {
	switch op {
	case ws.OpText:
		var env Envelope

		if err := json.Unmarshal(payload, &env); err != nil {
			return fmt.Errorf("decode websocket envelope: %w", err)
		}
		if s.router == nil {
			return ErrNoRouteMatched
		}
		if err := s.router.Route(ctx, s, env); err != nil {
			if errors.Is(err, ErrNoRouteMatched) {
				return ErrNoRouteMatched
			}
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
