package app

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()

	cfg, err := config.Load([]string{"DATA_DIR=" + filepath.Join(t.TempDir(), "data")})
	require.NoError(t, err)

	return cfg
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServeAnswersUntilCancelledThenShutsDownCleanly(t *testing.T) {
	cfg := testConfig(t)

	app, err := New(context.Background(), cfg, quietLogger())
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- app.Serve(ctx, listener) }()

	response, err := http.Get("http://" + listener.Addr().String() + "/api/health")
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, http.StatusOK, response.StatusCode)

	assert.FileExists(t, cfg.Database.SQLitePath, "the database was created and migrated")

	cancel()
	select {
	case err := <-served:
		assert.NoError(t, err, "a requested shutdown is clean")
	case <-time.After(shutdownTimeout + 5*time.Second):
		t.Fatal("Serve did not return after cancellation")
	}

	sqlDB, err := app.db.DB()
	require.NoError(t, err)
	assert.Error(t, sqlDB.Ping(), "the database is closed")

	_, err = http.Get("http://" + listener.Addr().String() + "/api/health")
	assert.Error(t, err, "the server no longer accepts connections")
}

func TestServeReportsAListenerThatFails(t *testing.T) {
	app, err := New(context.Background(), testConfig(t), quietLogger())
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, listener.Close())

	err = app.Serve(context.Background(), listener)
	assert.Error(t, err)

	sqlDB, dbErr := app.db.DB()
	require.NoError(t, dbErr)
	assert.Error(t, sqlDB.Ping(), "the database is closed even when serving fails")
}

func TestNewRefusesADatabaseItCannotReach(t *testing.T) {
	cfg := testConfig(t)
	cfg.Database = config.Database{Driver: config.Postgres, PostgresURL: "postgres://nobody:wrong@127.0.0.1:1/none?connect_timeout=2"}

	_, err := New(context.Background(), cfg, quietLogger())
	assert.Error(t, err)
}
