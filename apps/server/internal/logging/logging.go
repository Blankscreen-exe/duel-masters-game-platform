// Package logging builds the server's logger and routes the rules engine's
// logrus output into it, so the whole server writes one log format.
package logging

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"slices"

	"github.com/sirupsen/logrus"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
)

// New returns a logger writing to w at the configured level and format.
func New(cfg config.Log, w io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{Level: cfg.Level}

	if cfg.Format == config.LogJSON {
		return slog.New(slog.NewJSONHandler(w, options))
	}

	return slog.New(slog.NewTextHandler(w, options))
}

// RouteLogrus makes from, a logrus logger such as the one the rules engine
// uses, log through to instead of writing its own output, and sets its level to
// match, so entries to would drop are not even formatted. Call it once at
// startup: it replaces from's output, level and hooks.
func RouteLogrus(from *logrus.Logger, to *slog.Logger, level slog.Level) {
	from.SetOutput(io.Discard)
	from.SetLevel(logrusLevel(level))
	from.ReplaceHooks(logrus.LevelHooks{})
	from.AddHook(&slogHook{logger: to})
}

// slogHook passes every logrus entry on to a slog logger.
type slogHook struct {
	logger *slog.Logger
}

func (h *slogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *slogHook) Fire(entry *logrus.Entry) error {
	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background()
	}

	attrs := make([]slog.Attr, 0, len(entry.Data))
	for _, key := range slices.Sorted(maps.Keys(entry.Data)) {
		attrs = append(attrs, slog.Any(key, entry.Data[key]))
	}

	h.logger.LogAttrs(ctx, slogLevel(entry.Level), entry.Message, attrs...)

	return nil
}

func slogLevel(level logrus.Level) slog.Level {
	switch level {
	case logrus.TraceLevel, logrus.DebugLevel:
		return slog.LevelDebug
	case logrus.InfoLevel:
		return slog.LevelInfo
	case logrus.WarnLevel:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

func logrusLevel(level slog.Level) logrus.Level {
	switch {
	case level <= slog.LevelDebug:
		return logrus.DebugLevel
	case level <= slog.LevelInfo:
		return logrus.InfoLevel
	case level <= slog.LevelWarn:
		return logrus.WarnLevel
	default:
		return logrus.ErrorLevel
	}
}
