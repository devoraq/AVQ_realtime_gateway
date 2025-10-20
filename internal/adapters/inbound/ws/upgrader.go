package websocket

import (
	"net/http"

	"github.com/gobwas/ws"
)

type Gateway struct {
}

func NewGateway() *Gateway {
	return &Gateway{}
}

func (g *Gateway) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusBadRequest)
		return
	}

	session := NewSession(conn)

	go session.ReadLoop()
}
