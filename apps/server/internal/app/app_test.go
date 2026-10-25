package app

import (
	"bytes"
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

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()

	cardDatabase := cardstest.Write(t, []map[string]any{cardstest.Printing("0001", "Test Card", "DM-01", "1/1")}, []string{"0001"}, nil)
	cfg, err := config.Load([]string{"DATA_DIR=" + filepath.Join(t.TempDir(), "data"), "CARD_DATABASE_DIR=" + cardDatabase})
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

func TestNewWarnsWhileThereAreNoAccounts(t *testing.T) {
	cfg := testConfig(t)
	const warning = "there are no accounts yet"

	var logs bytes.Buffer
	app, err := New(context.Background(), cfg, slog.New(slog.NewTextHandler(&logs, nil)))
	require.NoError(t, err)
	assert.Contains(t, logs.String(), warning)

	_, _, err = app.auth.SignUp(context.Background(), "Shobu", "correct horse battery staple")
	require.NoError(t, err)
	require.NoError(t, store.Close(app.db))

	logs.Reset()
	app, err = New(context.Background(), cfg, slog.New(slog.NewTextHandler(&logs, nil)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close(app.db)) })
	assert.NotContains(t, logs.String(), warning)
}

func TestServeDeletesExpiredSessions(t *testing.T) {
	app, err := New(context.Background(), testConfig(t), quietLogger())
	require.NoError(t, err)

	_, live, err := app.auth.SignUp(context.Background(), "Shobu", "correct horse battery staple")
	require.NoError(t, err)

	var userID string
	require.NoError(t, app.db.Raw("SELECT id FROM users").Scan(&userID).Error)
	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, app.db.Exec("INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)", "expired-session", userID, past, past).Error)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- app.Serve(ctx, listener) }()

	countSessions := func() int64 {
		var count int64
		require.NoError(t, app.db.Table("sessions").Count(&count).Error)
		return count
	}
	deadline := time.Now().Add(5 * time.Second)
	for countSessions() != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, int64(1), countSessions(), "the expired session is deleted when the server starts; the live one stays")

	_, err = app.auth.Authenticate(context.Background(), live.Token)
	assert.NoError(t, err)

	cancel()
	assert.NoError(t, <-served)
}

func TestNewNeedsTheCardDatabase(t *testing.T) {
	cfg := testConfig(t)
	cfg.CardDatabaseDir = filepath.Join(t.TempDir(), "missing")

	_, err := New(context.Background(), cfg, quietLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CARD_DATABASE_DIR", "the error says which setting to fix")
}
