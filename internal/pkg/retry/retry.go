package retry

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"time"
)

// Config определяет параметры для механизма повторных попыток.
// Эта структура живёт только внутри пакета retry и не зависит от остального приложения.
type Config struct {
	Attempts int
	Initial  time.Duration
	Max      time.Duration
	Factor   float64
	Jitter   bool
}

// Do выполняет переданную функцию с логикой повторных попыток.
// Теперь он принимает локальную структуру Config.
func Do(ctx context.Context, log *slog.Logger, cfg Config, fn func(ctx context.Context) error) error {
	var err error
	for i := 0; i < cfg.Attempts; i++ {
		if err = fn(ctx); err == nil {
			return nil
		}

		log.Warn("Operation failed, retrying...",
			slog.String("error", err.Error()),
			slog.Int("attempt", i+1),
			slog.Int("total_attempts", cfg.Attempts),
		)

		delay := float64(cfg.Initial) * math.Pow(cfg.Factor, float64(i))
		if cfg.Jitter {
			delay += rand.Float64() * delay * 0.5
		}

		if time.Duration(delay) > cfg.Max {
			delay = float64(cfg.Max)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(delay)):
		}
	}
	return err
}
