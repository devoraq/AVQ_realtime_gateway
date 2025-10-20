package kvstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DENFNC/devPractice/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

type ScanResult map[string]string

type Redis struct {
	client *redis.Client
}

func NewRedis(cfg *config.RedisConfig) *Redis {
	if cfg == nil {
		panic("redis config cannot be nil")
	}

	redisConfig := &redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	client := redis.NewClient(redisConfig)

	return &Redis{
		client: client,
	}
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

func (r *Redis) ScanKeys(ctx context.Context, match string, step int64) (ScanResult, error) {
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
