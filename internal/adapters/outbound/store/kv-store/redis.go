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

var ErrRedisPingFailed = errors.New("redis ping failed")

type Redis struct {
	name   string
	client *redis.Client
	deps   *RedisDeps
}

type RedisDeps struct {
	Log *slog.Logger
	Cfg *config.RedisConfig
}

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

func (r *Redis) Name() string { return r.name }

func (r *Redis) Start(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.deps.Cfg.Timeout)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		r.deps.Log.Error(
			ErrRedisPingFailed.Error(),
			slog.String("addr", r.deps.Cfg.Address),
			slog.Int("DB", r.deps.Cfg.DB),
			slog.String("error", err.Error()),
		)
		return err
	}

	r.deps.Log.Info("Connected to Redis",
		slog.String("addr", r.deps.Cfg.Address),
		slog.Int("DB", r.deps.Cfg.DB),
	)

	return nil
}

func (r *Redis) Stop(ctx context.Context) error {
	if err := r.FlushAsync(ctx); err != nil {
		r.deps.Log.Error(
			"failed to flush db redis",
			slog.String("addr", r.deps.Cfg.Address),
			slog.Int("DB", r.deps.Cfg.DB),
			slog.String("error", err.Error()),
		)
		return err
	}
	if err := r.client.Close(); err != nil {
		r.deps.Log.Error(
			"failed to close redis connection",
			slog.String("addr", r.deps.Cfg.Address),
			slog.Int("DB", r.deps.Cfg.DB),
			slog.String("error", err.Error()),
		)
		return err
	}

	r.deps.Log.Info(
		"Redis connection closed",
		slog.String("addr", r.deps.Cfg.Address),
		slog.Int("DB", r.deps.Cfg.DB),
	)

	return nil
}

func (r *Redis) Add(ctx context.Context, key string, value any, expiration time.Duration) error {
	if expiration < 0 {
		return fmt.Errorf("expiration cannot be negative")
	}

	stcmd := r.client.Set(ctx, key, value, expiration)
	if stcmd.Err() != nil {
		return stcmd.Err()
	}
	return nil
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("%s does not exist", key)
		}
		return "", err
	}

	return result, nil
}

func (r *Redis) Remove(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return fmt.Errorf("no keys provided")
	}

	_, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) ScanKeys(ctx context.Context, match string, step int64) (map[string]string, error) {
	iter := r.client.Scan(ctx, 0, match, step).Iterator()
	result := make(map[string]string)

	for iter.Next(ctx) {
		key := iter.Val()
		val, err := r.client.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		result[key] = val
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Redis) FlushAsync(ctx context.Context) error {
	if _, err := r.client.FlushDBAsync(ctx).Result(); err != nil {
		return err
	}
	return nil
}
