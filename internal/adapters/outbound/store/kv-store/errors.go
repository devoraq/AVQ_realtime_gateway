package kvstore

import "errors"

var (
	// ErrPingFailed сигнализирует о недоступности Redis во время health-check.
	ErrPingFailed = errors.New("kvstore: redis ping failed")
	// ErrNegativeTTL возвращается, если передан отрицательный TTL.
	ErrNegativeTTL = errors.New("kvstore: expiration cannot be negative")
	// ErrKeyNotFound сообщает, что ключ отсутствует в хранилище.
	ErrKeyNotFound = errors.New("kvstore: key not found")
	// ErrNoKeysProvided используется, когда для удаления не переданы ключи.
	ErrNoKeysProvided = errors.New("kvstore: no keys provided")
)
