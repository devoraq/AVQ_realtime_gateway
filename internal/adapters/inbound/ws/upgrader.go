package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gobwas/ws"
)

// SessionStore описывает операции с хранилищем активных WebSocket-сессий.
type SessionStore interface {
	Add(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Remove(ctx context.Context, keys ...string) error
}

// Gateway обслуживает HTTP-upgrade в WebSocket и управляет регистрацией сессий.
type Gateway struct {
	store  SessionStore
	router Router
}

// NewGateway создаёт экземпляр шлюза с переданным хранилищем и маршрутизатором.
func NewGateway(store SessionStore, router Router) *Gateway {
	if store == nil {
		panic("session store cannot be nil")
	}
	if router == nil {
		panic("router cannot be nil")
	}

	return &Gateway{
		store:  store,
		router: router,
	}
}

// HandleWS обрабатывает upgrade, регистрирует сессию и запускает ReadLoop.
func (g *Gateway) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusBadRequest)
		return
	}

	session, err := NewSession(conn, g.router)
	if err != nil {
		_ = conn.Close()
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	slog.Debug(
		"New user connected",
		slog.String("user_id", session.UserID.String()),
		slog.String("session_id", session.ID.String()),
	)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer g.sessionRemove(ctx, session)

	if err := g.appendSession(ctx, session.UserID.String(), session.ID.String()); err != nil {
		_ = session.Close()
		http.Error(w, "failed to register session", http.StatusInternalServerError)
		return
	}
	registerSession(session)

	if err := session.ReadLoop(ctx); err != nil {
		slog.Error("websocket session read loop failed",
			slog.String("session_id", session.ID.String()),
			slog.String("error", err.Error()),
		)
	}
}

func (g *Gateway) sessionRemove(ctx context.Context, session *Session) {
	if session == nil {
		return
	}
	unregisterSession(session.ID.String())

	if err := g.removeSession(ctx, session.UserID.String(), session.ID.String()); err != nil {
		slog.Warn("failed to remove websocket session",
			slog.String("session_id", session.ID.String()),
			slog.String("error", err.Error()),
		)
	}
	if err := session.Close(); err != nil {
		slog.Warn("failed to close websocket session",
			slog.String("session_id", session.ID.String()),
			slog.String("error", err.Error()),
		)
	}
}

func (g *Gateway) appendSession(ctx context.Context, userID, sessionID string) error {
	key := "session:" + userID
	sessions, err := g.readSessions(ctx, key)
	if err != nil {
		return fmt.Errorf("read sessions %s: %w", key, err)
	}

	for _, existing := range sessions {
		if existing == sessionID {
			return nil
		}
	}
	sessions = append(sessions, sessionID)
	return g.writeSessions(ctx, key, sessions)
}

func (g *Gateway) removeSession(ctx context.Context, userID, sessionID string) error {
	key := "session:" + userID
	sessions, err := g.readSessions(ctx, key)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("read sessions %s: %w", key, err)
	}

	filtered := sessions[:0]
	for _, existing := range sessions {
		if existing != sessionID {
			filtered = append(filtered, existing)
		}
	}
	if len(filtered) == 0 {
		if err := g.store.Remove(ctx, key); err != nil {
			return fmt.Errorf("remove key %s: %w", key, err)
		}
		return nil
	}
	return g.writeSessions(ctx, key, filtered)
}

func (g *Gateway) readSessions(ctx context.Context, key string) ([]string, error) {
	value, err := g.store.Get(ctx, key)
	if err != nil {
		if isNotFound(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("get key %s: %w", key, err)
	}

	var sessions []string
	if err := json.Unmarshal([]byte(value), &sessions); err != nil {
		return nil, fmt.Errorf("decode sessions for %s: %w", key, err)
	}
	return sessions, nil
}

func (g *Gateway) writeSessions(ctx context.Context, key string, sessions []string) error {
	payload, err := json.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("encode sessions for %s: %w", key, err)
	}
	if err := g.store.Add(ctx, key, string(payload), 0); err != nil {
		return fmt.Errorf("store sessions %s: %w", key, err)
	}
	return nil
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "key not found")
}
