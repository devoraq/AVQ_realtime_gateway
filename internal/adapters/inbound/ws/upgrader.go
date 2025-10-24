package websocket

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gobwas/ws"
)

// SessionStore описывает абстракцию для хранения активных WebSocket-сессий.
// Типичная реализация использует Redis или in-memory map.
type SessionStore interface {
	Add(ctx context.Context, key string, value any, expiration time.Duration) error
	Remove(ctx context.Context, keys ...string) error
	ScanKeys(ctx context.Context, match string, step int64) (map[string]string, error)
}

// Gateway отвечает за апгрейд HTTP-соединений в WebSocket и регистрацию
// сессий в хранилище.
//
// Пример:
//
//	store := redis.NewSessionStore(...)
//	router := websocket.NewHandlerChain()
//	gateway := websocket.NewGateway(store, router)
//	http.HandleFunc("/ws", gateway.HandleWS)
//	_ = http.ListenAndServe(":8080", nil)
type Gateway struct {
	store  SessionStore
	router Router
}

// NewGateway создает новый экземпляр шлюза, валидируя переданные зависимости.
// Паникует, если store или router не заданы.
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

// HandleWS обрабатывает HTTP-запрос на апгрейд в WebSocket, создает сессию,
// регистрирует ее в хранилище и запускает цикл чтения сообщений. Функция
// предназначена для использования как http.HandlerFunc.
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

	if err := g.store.Add(r.Context(), session.ID.String(), session.UserID.String(), 0); err != nil {
		_ = session.Close()
		http.Error(w, "failed to register session", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer g.sessionRemove(session)

	if err := session.ReadLoop(ctx); err != nil {
		log.Printf("session %s read loop error: %v", session.ID, err)
	}
}

func (g *Gateway) sessionRemove(session *Session) {
	if err := g.store.Remove(context.Background(), session.ID.String()); err != nil {
		log.Printf("failed to remove session %s: %v", session.ID, err)
	}
	if err := session.Close(); err != nil {
		log.Printf("failed to close session %s: %v", session.ID, err)
	}
}
