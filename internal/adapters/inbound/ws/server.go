package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

type Session struct {
	ID     uuid.UUID
	UserID uuid.UUID
	conn   net.Conn
	router Router
}

func NewSession(conn net.Conn, router Router) (*Session, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	userID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:     id,
		UserID: userID,
		conn:   conn,
		router: router,
	}, nil
}

func (s *Session) Close() error {
	return s.conn.Close()
}

func (s *Session) ReadLoop(ctx context.Context) error {
	for {
		msg, op, err := wsutil.ReadClientData(s.conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
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
			return err
		}

		if s.router == nil {
			return ErrNoRouteMatched
		}

		return s.router.Route(ctx, s, env)
	case ws.OpPing:
		return wsutil.WriteServerMessage(s.conn, ws.OpPong, nil)
	default:
		return nil
	}
}
