package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

type pingerFunc func(ctx context.Context) error

func (f pingerFunc) PingContext(ctx context.Context) error { return f(ctx) }

func newTestHandler(t *testing.T, db Pinger) (http.Handler, *bytes.Buffer) {
	t.Helper()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return NewHandler(Dependencies{DB: db, Logger: logger}), &logs
}

func serve(handler http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, body))
	return recorder
}

func decode[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()

	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	var body T
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body), response.Body.String())
	return body
}

func TestHealthReportsAWorkingDatabase(t *testing.T) {
	for _, driver := range []config.DatabaseDriver{config.SQLite, config.Postgres} {
		t.Run(string(driver), func(t *testing.T) {
			sqlDB, err := storetest.Open(t, driver).DB()
			require.NoError(t, err)
			handler, _ := newTestHandler(t, sqlDB)

			response := serve(handler, http.MethodGet, "/api/health", nil)

			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, api.Health{Status: api.HealthStatusOk, Database: api.HealthStatusOk}, decode[api.Health](t, response))
		})
	}
}

func TestHealthReportsADatabaseThatDoesNotAnswer(t *testing.T) {
	handler, _ := newTestHandler(t, pingerFunc(func(context.Context) error { return errors.New("connection refused") }))

	response := serve(handler, http.MethodGet, "/api/health", nil)

	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.Equal(t, api.Health{Status: api.HealthStatusUnavailable, Database: api.HealthStatusUnavailable}, decode[api.Health](t, response))
}

func TestHealthDoesNotWaitForAHungDatabase(t *testing.T) {
	handler, _ := newTestHandler(t, pingerFunc(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}))

	response := serve(handler, http.MethodGet, "/api/health", nil)

	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
}

func TestUnknownEndpointsAnswerWithAJSONError(t *testing.T) {
	handler, _ := newTestHandler(t, pingerFunc(func(context.Context) error { return nil }))

	response := serve(handler, http.MethodGet, "/api/no-such-thing", nil)

	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Equal(t, api.Error{Code: "not_found", Message: "No such endpoint."}, decode[api.Error](t, response))
}

func TestAWrongMethodListsTheAllowedOnes(t *testing.T) {
	handler, _ := newTestHandler(t, pingerFunc(func(context.Context) error { return nil }))

	response := serve(handler, http.MethodPost, "/api/health", nil)

	assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
	assert.Equal(t, []string{http.MethodGet}, response.Header().Values("Allow"))
	assert.Equal(t, "method_not_allowed", decode[api.Error](t, response).Code)
}

func TestRequestsAreLogged(t *testing.T) {
	handler, logs := newTestHandler(t, pingerFunc(func(context.Context) error { return nil }))

	serve(handler, http.MethodGet, "/api/health", nil)
	serve(handler, http.MethodGet, "/api/missing", nil)
	serve(handler, http.MethodPost, "/api/health", nil)

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	require.Len(t, lines, 3)

	var health, missing, wrongMethod map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &health))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &missing))
	require.NoError(t, json.Unmarshal([]byte(lines[2]), &wrongMethod))

	assert.Equal(t, "DEBUG", health["level"], "successful health checks are logged quietly")
	assert.EqualValues(t, http.StatusOK, health["status"])
	assert.Equal(t, "INFO", missing["level"])
	assert.Equal(t, "/api/missing", missing["path"])
	assert.EqualValues(t, http.StatusNotFound, missing["status"])
	assert.Equal(t, "INFO", wrongMethod["level"], "a failed health check request is not hidden")
	assert.EqualValues(t, http.StatusMethodNotAllowed, wrongMethod["status"])
}

func TestAPanickingHandlerAnswersWithAnError(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := logRequests(logger, recoverPanics(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("broken")
	})))

	response := serve(handler, http.MethodGet, "/api/anything", nil)

	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Equal(t, "internal_error", decode[api.Error](t, response).Code)
	assert.Contains(t, logs.String(), "a request handler panicked")
	assert.Contains(t, logs.String(), "broken")
	assert.Contains(t, logs.String(), `"status":500`, "the request log shows the error")
}

func TestAPanicAfterAnswerStartedAbortsTheConnection(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := logRequests(logger, recoverPanics(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		panic("broken half way")
	})))

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		serve(handler, http.MethodGet, "/api/anything", nil)
	})
}

func TestAbortingAHandlerIsPassedOn(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := recoverPanics(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		serve(handler, http.MethodGet, "/api/anything", nil)
	})
}

func TestRequestBodiesAreLimited(t *testing.T) {
	var readErr error
	handler := limitBodies(8, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))

	serve(handler, http.MethodPost, "/api/anything", strings.NewReader("12345678"))
	assert.NoError(t, readErr, "a body at the limit is read in full")

	serve(handler, http.MethodPost, "/api/anything", strings.NewReader("123456789"))
	var tooLarge *http.MaxBytesError
	assert.ErrorAs(t, readErr, &tooLarge)
}
