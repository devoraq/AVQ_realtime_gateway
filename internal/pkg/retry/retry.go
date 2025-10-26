// Package retry предоставляет универсальный механизм для повторного выполнения операций
// с использованием настраиваемой экспоненциальной задержки.
package retry

import (
	"context"
	"crypto/rand"
	"log/slog"
	"math/big"
	"time"
)

// Config определяет параметры для механизма повторных попыток.
type Config struct {
	Attempts int
	Initial  time.Duration
	Max      time.Duration
	Factor   float64
	Jitter   bool
}

// defaultConfig возвращает настройки по умолчанию, если они не предоставлены.
func defaultConfig() Config {
	return Config{
		Attempts: 3,
		Initial:  1 * time.Second,
		Max:      30 * time.Second,
		Factor:   2.0,
		Jitter:   true,
	}
}

// Do выполняет переданную функцию fn с логикой повторных попыток.
// Функция будет повторяться до успешного выполнения или пока не исчерпаются все попытки.
// Поддерживает отмену через context, экспоненциальный рост задержки и случайный разброс (jitter).
func Do(ctx context.Context, log *slog.Logger, cfg Config, fn func(ctx context.Context) error) error {
	if cfg.Attempts <= 0 {
		cfg = defaultConfig()
	}

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

		if i == cfg.Attempts-1 {
			break
		}

		backoff := float64(cfg.Initial) * pow(cfg.Factor, i)

		if cfg.Jitter {
			jitter, randErr := rand.Int(rand.Reader, big.NewInt(int64(backoff/2)))
			if randErr == nil {
				backoff += float64(jitter.Int64())
			}
		}

		if backoff > float64(cfg.Max) {
			backoff = float64(cfg.Max)
		}

		delay := time.Duration(backoff)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err // Возврат последней ошибки.
}

// pow - простая реализация возведения в степень для float, чтобы не импортировать math.
func pow(base float64, exp int) float64 {
	result := 1.0
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}
