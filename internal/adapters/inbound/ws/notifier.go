package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const sessionPrefix = "session:"

// sessionLookup определяет минимальный интерфейс хранилища, необходимый для доставки сообщений.
type sessionLookup interface {
	Get(ctx context.Context, key string) (string, error)
}

// Notifier отправляет payload во все активные сессии пользователя.
type Notifier struct {
	store sessionLookup
}

// NewNotifier создаёт нотификатор, использующий переданное хранилище сессий.
func NewNotifier(store sessionLookup) *Notifier {
	return &Notifier{store: store}
}

// Notify рассылает сообщение по всем WebSocket-сессиям пользователя.
func (n *Notifier) Notify(ctx context.Context, userID string, messageType string, payload any) error {
	if n == nil || n.store == nil {
		return errors.New("notifier is not initialized")
	}
	if userID == "" {
		return errors.New("user id is empty")
	}

	sessions, err := n.fetchSessions(ctx, userID)
	if err != nil {
		return err
	}

	var lastErr error
	for _, sessionID := range sessions {
		if sessionID == "" {
			continue
		}
		if err := SendToSession(ctx, sessionID, messageType, payload); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (n *Notifier) fetchSessions(ctx context.Context, userID string) ([]string, error) {
	key := sessionPrefix + userID
	value, err := n.store.Get(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "key not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("get sessions for %s: %w", userID, err)
	}

	var sessions []string
	if err := json.Unmarshal([]byte(value), &sessions); err != nil {
		return nil, fmt.Errorf("decode sessions for %s: %w", userID, err)
	}
	return sessions, nil
}
