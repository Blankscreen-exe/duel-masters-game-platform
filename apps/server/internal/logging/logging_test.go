package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
)

func logLines(t *testing.T, output *bytes.Buffer) []map[string]any {
	t.Helper()

	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry), "line %q", line)
		lines = append(lines, entry)
	}

	return lines
}

func TestNewWritesTheConfiguredFormat(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		var output bytes.Buffer
		New(config.Log{Level: slog.LevelInfo, Format: config.LogJSON}, &output).Info("started", "port", 8080)

		lines := logLines(t, &output)
		require.Len(t, lines, 1)
		assert.Equal(t, "started", lines[0]["msg"])
		assert.EqualValues(t, 8080, lines[0]["port"])
	})

	t.Run("text", func(t *testing.T) {
		var output bytes.Buffer
		New(config.Log{Level: slog.LevelInfo, Format: config.LogText}, &output).Info("started", "port", 8080)

		assert.Contains(t, output.String(), "msg=started port=8080")
	})

	t.Run("lines below the level are dropped", func(t *testing.T) {
		var output bytes.Buffer
		New(config.Log{Level: slog.LevelWarn, Format: config.LogJSON}, &output).Info("quiet")

		assert.Empty(t, output.String())
	})
}

func TestRouteLogrusSendsEngineLogsToTheServerLogger(t *testing.T) {
	var output bytes.Buffer
	var engineOutput bytes.Buffer
	engine := logrus.New()
	engine.SetOutput(&engineOutput)

	RouteLogrus(engine, New(config.Log{Level: slog.LevelDebug, Format: config.LogJSON}, &output), slog.LevelDebug)

	engine.WithFields(logrus.Fields{"match_id": "m1", "turns": 7}).Warn("player disconnected")
	engine.Debug("detail")
	engine.Error("failed")

	assert.Empty(t, engineOutput.String(), "logrus no longer writes its own output")

	lines := logLines(t, &output)
	require.Len(t, lines, 3)

	assert.Equal(t, "player disconnected", lines[0]["msg"])
	assert.Equal(t, "WARN", lines[0]["level"])
	assert.Equal(t, "m1", lines[0]["match_id"])
	assert.EqualValues(t, 7, lines[0]["turns"])

	assert.Equal(t, "DEBUG", lines[1]["level"])
	assert.Equal(t, "ERROR", lines[2]["level"])
}

func TestRouteLogrusMatchesTheServerLevel(t *testing.T) {
	var output bytes.Buffer
	engine := logrus.New()

	RouteLogrus(engine, New(config.Log{Level: slog.LevelWarn, Format: config.LogJSON}, &output), slog.LevelWarn)

	assert.Equal(t, logrus.WarnLevel, engine.GetLevel())

	engine.Info("dropped before formatting")
	engine.Warn("kept")

	lines := logLines(t, &output)
	require.Len(t, lines, 1)
	assert.Equal(t, "kept", lines[0]["msg"])
}

func TestLevelMappings(t *testing.T) {
	for level, want := range map[logrus.Level]slog.Level{
		logrus.TraceLevel: slog.LevelDebug,
		logrus.DebugLevel: slog.LevelDebug,
		logrus.InfoLevel:  slog.LevelInfo,
		logrus.WarnLevel:  slog.LevelWarn,
		logrus.ErrorLevel: slog.LevelError,
		logrus.FatalLevel: slog.LevelError,
		logrus.PanicLevel: slog.LevelError,
	} {
		assert.Equal(t, want, slogLevel(level), "logrus %s", level)
	}

	for level, want := range map[slog.Level]logrus.Level{
		slog.LevelDebug - 4: logrus.DebugLevel,
		slog.LevelDebug:     logrus.DebugLevel,
		slog.LevelInfo:      logrus.InfoLevel,
		slog.LevelWarn:      logrus.WarnLevel,
		slog.LevelError:     logrus.ErrorLevel,
		slog.LevelError + 4: logrus.ErrorLevel,
	} {
		assert.Equal(t, want, logrusLevel(level), "slog %s", level)
	}
}
