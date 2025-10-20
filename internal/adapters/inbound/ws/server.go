package websocket

import (
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type Session struct {
	conn net.Conn
	// UserID uuid.UUID
}

func NewSession(conn net.Conn) *Session {
	return &Session{
		conn: conn,
		// UserID: user,
	}
}

func (s *Session) ReadLoop() {
	for {
		_, op, err := wsutil.ReadClientData(s.conn)
		if err != nil {
			return
		}

		s.handleOperation(op)
	}
}

func (s *Session) handleOperation(op ws.OpCode) {
	switch op {
	case ws.OpText:
		wsutil.WriteServerMessage(s.conn, ws.OpText, []byte("pong"))
	case ws.OpPing:
		wsutil.WriteServerMessage(s.conn, ws.OpPong, nil)
	}
}
