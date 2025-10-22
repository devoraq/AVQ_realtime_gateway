// Package logger предоставляет адаптер для форматированного логирования на
// базе slog с цветовой подсветкой и человекочитаемым выводом.
package logger

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"

	"github.com/fatih/color"
)

// PrettyHandlerOptions задает настройки для PrettyHandler и проксирует
// стандартные аргументы slog.HandlerOptions.
type PrettyHandlerOptions struct {
	Opts slog.HandlerOptions
}

// PrettyHandler реализует интерфейс slog.Handler, добавляя цветовую
// подсветку уровней и форматирование JSON-полей лога.
type PrettyHandler struct {
	l *log.Logger
}

// Enabled всегда возвращает true, чтобы делегировать контроль уровней
// верхнему slog.Logger.
func (h *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// Handle формирует финальную строку лога с подсветкой уровня и красиво
// форматированными атрибутами записи.
func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.GreenString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	fields := make(map[string]interface{}, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()

		return true
	})

	var b []byte
	var err error
	if len(fields) > 0 {
		b, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
	}

	timeStr := r.Time.Format("[15:05:05.000]")
	msg := color.WhiteString(r.Message)

	h.l.Println(timeStr, level, msg, color.WhiteString(string(b)))

	return nil
}

// WithAttrs возвращает тот же обработчик, так как PrettyHandler не хранит
// состояние дополнительных атрибутов.
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup возвращает исходный обработчик, поскольку группировка не влияет
// на форматирование PrettyHandler.
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h
}

// NewPrettyHandler создает обработчик, который выводит структурированные логи
// slog в читабельном формате с использованием стандартного log.Logger.
func NewPrettyHandler(
	out io.Writer,
	opts PrettyHandlerOptions,
) *PrettyHandler {
	h := &PrettyHandler{
		l: log.New(out, "", 0),
	}

	return h
}
