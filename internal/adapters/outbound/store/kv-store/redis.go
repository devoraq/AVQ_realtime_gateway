// Package kvstore implements Redis-backed key-value storage adapters.
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

// ErrRedisPingFailed indicates that Redis connection is not healthy.
var ErrRedisPingFailed = errors.New("redis ping failed")

// Redis implements kvstore backed by Redis.
type Redis struct {
	name   string
	client *redis.Client
	deps   *RedisDeps
}

// RedisDeps contains dependencies required to initialize Redis adapter.
type RedisDeps struct {
	Log *slog.Logger
	Cfg *config.RedisConfig
}

// NewRedis constructs redis-backed key-value store.
func NewRedis(deps *RedisDeps) *Redis {
	if deps.Cfg == nil {
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

// Name returns component identifier.
func (r *Redis) Name() string { return r.name }

// Start verifies Redis connection by issuing a ping.
func (r *Redis) Start(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		r.deps.Log.Debug(
			"redis ping failed",
			slog.String("addr", r.deps.Cfg.Address),
			slog.Int("DB", r.deps.Cfg.DB),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("redis ping: %w", err)
	}

	r.deps.Log.Debug("Connected to Redis",
		slog.String("addr", r.deps.Cfg.Address),
		slog.Int("DB", r.deps.Cfg.DB),
	)

	return nil
}

// Stop flushes Redis database (best-effort) and closes the client.
func (r *Redis) Stop(ctx context.Context) error {
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

	r.deps.Log.Debug(
		"Redis connection closed",
		slog.String("addr", r.deps.Cfg.Address),
		slog.Int("DB", r.deps.Cfg.DB),
	)

	return nil
}

// Add stores value under the provided key with optional expiration.
func (r *Redis) Add(ctx context.Context, key string, value any, expiration time.Duration) error {
	if expiration < 0 {
		return errors.New("expiration cannot be negative")
	}

	stcmd := r.client.Set(ctx, key, value, expiration)
	if err := stcmd.Err(); err != nil {
		return fmt.Errorf("redis set %q: %w", key, err)
	}
	return nil
}

// Get fetches string value for the given key.
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("%s does not exist", key)
		}
		return "", fmt.Errorf("redis get %q: %w", key, err)
	}

	return result, nil
}

// Remove deletes provided keys.
func (r *Redis) Remove(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return errors.New("no keys provided")
	}

	if _, err := r.client.Del(ctx, keys...).Result(); err != nil {
		return fmt.Errorf("redis delete keys %v: %w", keys, err)
	}
	return nil
}

// ScanKeys returns matching keys with their values.
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

// FlushAsync empties current Redis database asynchronously.
func (r *Redis) FlushAsync(ctx context.Context) error {
	if _, err := r.client.FlushDBAsync(ctx).Result(); err != nil {
		return fmt.Errorf("redis flush async: %w", err)
	}
	return nil
}
