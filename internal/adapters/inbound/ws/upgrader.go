package websocket

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gobwas/ws"
)

type SessionStore interface {
	Add(ctx context.Context, key string, value any, expiration time.Duration) error
	Remove(ctx context.Context, keys ...string) error
	ScanKeys(ctx context.Context, match string, step int64) (map[string]string, error)
}

type Gateway struct {
	store  SessionStore
	router Router
}

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

	go func() {
		sessionCtx, cancel := context.WithCancel(r.Context())
		defer cancel()

		defer func() {
			if err := g.store.Remove(context.Background(), session.ID.String()); err != nil {
				log.Printf("failed to remove session %s: %v", session.ID, err)
			}
			if err := session.Close(); err != nil {
				log.Printf("failed to close session %s: %v", session.ID, err)
			}
		}()

		if err := session.ReadLoop(sessionCtx); err != nil {
			log.Printf("session %s read loop error: %v", session.ID, err)
		}
	}()
}
