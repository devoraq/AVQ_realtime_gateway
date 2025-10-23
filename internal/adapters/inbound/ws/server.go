// Package websocket предоставляет инфраструктурный слой входящих адаптеров
// для работы с WebSocket-соединениями, включая управление сессиями и
// маршрутизацию сообщений. Функции пакета используются HTTP-шлюзом для
// апгрейда соединений и дальнейшей обработки входящих событий.
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

// Session инкапсулирует данные активного WebSocket-соединения и отвечает за
// цикл чтения входящих сообщений.
//
// Пример инициализации:
//
//	conn, _, _, _ := ws.UpgradeHTTP(req, resp)
//	router := websocket.NewHandlerChain()
//	session, err := websocket.NewSession(conn, router)
//	if err != nil {
//		// обработка ошибки
//	}
//	go session.ReadLoop(context.Background())
type Session struct {
	ID     uuid.UUID
	UserID uuid.UUID

	conn   net.Conn
	router Router
}

// NewSession создает новую веб-сессию, генерируя идентификаторы пользователя и
// сессии, и связывает ее с переданным роутером. Возвращает ошибку, если
// генерация идентификаторов не удалась.
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

// Close закрывает сетевое соединение, связанное с сессией. Обычно вызывается
// инфраструктурой после завершения ReadLoop.
func (s *Session) Close() error {
	return s.conn.Close()
}

// ReadLoop запускает бесконечный цикл чтения входящих сообщений и делегирует
// их обработку роутеру. Рекомендуется запускать цикл в отдельной горутине;
// функция завершится при закрытии соединения или возникновении критической
// ошибки чтения.
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

// handleOperation выполняет обработку одного сообщения в зависимости от его
// типа: текстовые кадры преобразуются в Envelope и отправляются в роутер,
// ping-кадры получают ответ pong, а остальные типы игнорируются.
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
