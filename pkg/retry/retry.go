// Package retry предоставляет универсальный механизм для повторного выполнения операций
// с использованием настраиваемой экспоненциальной задержки.
package retry

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
)

// Do выполняет переданную функцию fn с логикой повторных попыток.
// Функция будет повторяться до успешного выполнения или пока не исчерпаются все попытки.
// Поддерживает отмену через context, экспоненциальный рост задержки и случайный разброс (jitter).
func Do(ctx context.Context, cfg *config.RetryConfig, fn func(ctx context.Context) error) (lastErr error) {
	sanitized := sanitize(cfg)

	delay := sanitized.Initial

	for attempt := 1; attempt <= sanitized.Attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		// если это была последняя попытка — выходим
		if attempt == sanitized.Attempts {
			break
		}

		// считаем следующую задержку
		delay = nextDelay(delay, sanitized)

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

	return lastErr
}

// nextDelay возвращает увеличенный интервал с crypto-jitter.
func nextDelay(prev time.Duration, cfg config.RetryConfig) time.Duration {
	backoff := float64(prev) * cfg.Factor
	if backoff > float64(cfg.Max) {
		backoff = float64(cfg.Max)
	}

	if cfg.Jitter {
		jitterMax := int64(backoff / 2)
		if jitterMax > 0 {
			if n, err := rand.Int(rand.Reader, big.NewInt(jitterMax)); err == nil {
				backoff += float64(n.Int64())
			}
		}
	}

	return time.Duration(backoff)
}

func sanitize(cfg *config.RetryConfig) config.RetryConfig {
	if cfg == nil {
		cfg = &config.RetryConfig{}
	}

	sanitized := *cfg

	if sanitized.Attempts <= 0 {
		sanitized.Attempts = 3
	}
	if sanitized.Initial <= 0 {
		sanitized.Initial = time.Second
	}
	if sanitized.Max <= 0 {
		sanitized.Max = 30 * time.Second
	}
	if sanitized.Factor < 1 {
		sanitized.Factor = 2.0
	}

	if sanitized.Initial > sanitized.Max {
		sanitized.Initial = sanitized.Max
	}

	return sanitized
}
