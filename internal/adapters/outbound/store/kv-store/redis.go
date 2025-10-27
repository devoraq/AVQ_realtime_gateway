// Package kvstore реализует обёртку над Redis как key-value хранилищем.
package kvstore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/redis/go-redis/v9"
)

// Redis инкапсулирует клиента Redis и зависимости, необходимые адаптеру.
type Redis struct {
	name   string
	client *redis.Client
	deps   *RedisDeps
}

// RedisDeps описывает зависимости, которые требуются для создания адаптера.
type RedisDeps struct {
	Log *slog.Logger
	Cfg *config.RedisConfig
}

// NewRedis конструирует новый адаптер, валидируя переданные зависимости.
func NewRedis(deps *RedisDeps) *Redis {
	if deps == nil || deps.Cfg == nil {
		panic("redis config cannot be nil")
	}
	if deps.Log == nil {
		panic("logger cannot be nil")
	}

	opts := &redis.Options{
		Addr:     deps.Cfg.Address,
		Password: deps.Cfg.Password,
		DB:       deps.Cfg.DB,
	}

	client := redis.NewClient(opts)

	return &Redis{
		name:   "redis",
		client: client,
		deps:   deps,
	}
}

// Name возвращает идентификатор компонента.
func (r *Redis) Name() string { return r.name }

// Start выполняет health-check и удостоверяется, что Redis доступен.
func (r *Redis) Start(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		r.deps.Log.Debug(
			"redis ping failed",
			slog.String("addr", r.deps.Cfg.Address),
			slog.Int("DB", r.deps.Cfg.DB),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%w: %w", ErrPingFailed, err)
	}

	r.deps.Log.Debug("Connected to Redis",
		slog.String("addr", r.deps.Cfg.Address),
		slog.Int("DB", r.deps.Cfg.DB),
	)

	return nil
}

// Stop закрывает соединение и очищает БД в best-effort режиме.
func (r *Redis) Stop(ctx context.Context) error {
	defer r.deps.Log.Debug(
		"Redis connection closed",
		slog.String("addr", r.deps.Cfg.Address),
		slog.Int("DB", r.deps.Cfg.DB),
	)
	if err := r.FlushAsync(ctx); err != nil {
		r.deps.Log.Error(
			"failed to flush db redis",
			slog.String("addr", r.deps.Cfg.Address),
			slog.Int("DB", r.deps.Cfg.DB),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("redis flush async: %w", err)
	}
	if err := r.client.Close(); err != nil {
		r.deps.Log.Error(
			"failed to close redis connection",
			slog.String("addr", r.deps.Cfg.Address),
			slog.Int("DB", r.deps.Cfg.DB),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("redis close: %w", err)
	}

	return nil
}

// Add записывает значение по ключу с заданным TTL.
func (r *Redis) Add(ctx context.Context, key string, value any, expiration time.Duration) error {
	if expiration < 0 {
		return ErrNegativeTTL
	}

	stcmd := r.client.Set(ctx, key, value, expiration)
	if err := stcmd.Err(); err != nil {
		return fmt.Errorf("redis set %q: %w", key, err)
	}
	return nil
}

// Get возвращает значение ключа.
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
		}
		return "", fmt.Errorf("redis get %q: %w", key, err)
	}

	return result, nil
}

// Remove удаляет набор ключей.
func (r *Redis) Remove(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return ErrNoKeysProvided
	}

	if _, err := r.client.Del(ctx, keys...).Result(); err != nil {
		return fmt.Errorf("redis delete keys %v: %w", keys, err)
	}
	return nil
}

// ScanKeys ищет ключи по шаблону и возвращает их значения.
func (r *Redis) ScanKeys(ctx context.Context, match string, step int64) (map[string]string, error) {
	iter := r.client.Scan(ctx, 0, match, step).Iterator()
	result := make(map[string]string)

	for iter.Next(ctx) {
		key := iter.Val()
		val, err := r.client.Get(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redis get %q during scan: %w", key, err)
		}

		result[key] = val
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("redis scan iterator: %w", err)
	}

	return result, nil
}

// FlushAsync очищает текущую базу Redis асинхронно.
func (r *Redis) FlushAsync(ctx context.Context) error {
	if _, err := r.client.FlushDBAsync(ctx).Result(); err != nil {
		return fmt.Errorf("redis flush async: %w", err)
	}
	return nil
}
