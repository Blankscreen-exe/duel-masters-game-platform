// Package app assembles the server from its configuration (the database, the
// features and the HTTP server) and runs it until it is told to stop.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
)

// HTTP server limits. A connection taken over for a live game (a websocket)
// keeps these deadlines, so the code that takes it over clears them.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 2 * time.Minute
	maxHeaderBytes    = 1 << 20
)

// shutdownTimeout is how long requests in progress may take to finish once
// the server is told to stop.
const shutdownTimeout = 15 * time.Second

// App is the assembled server.
type App struct {
	logger *slog.Logger
	db     *gorm.DB
	server *http.Server
}

// New opens and migrates the database and builds the HTTP server. The caller
// runs it with Serve, which also closes it.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := store.Open(ctx, cfg.Database, logger.With("component", "database"))
	if err != nil {
		return nil, err
	}

	if err := store.Migrate(db); err != nil {
		return nil, errors.Join(err, store.Close(db))
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Join(err, store.Close(db))
	}

	httpLogger := logger.With("component", "http")

	return &App{
		logger: logger,
		db:     db,
		server: &http.Server{
			Handler:           httpapi.NewHandler(httpapi.Dependencies{DB: sqlDB, Logger: httpLogger}),
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			MaxHeaderBytes:    maxHeaderBytes,
			ErrorLog:          slog.NewLogLogger(httpLogger.Handler(), slog.LevelWarn),
		},
	}, nil
}

// Serve answers HTTP requests on listener until ctx is cancelled. It then stops
// accepting connections, lets the requests in progress finish within the
// shutdown timeout and closes the database. It returns nil after a clean
// shutdown.
func (a *App) Serve(ctx context.Context, listener net.Listener) error {
	served := make(chan error, 1)
	go func() { served <- a.server.Serve(listener) }()

	select {
	case err := <-served:
		return errors.Join(fmt.Errorf("serving HTTP: %w", err), store.Close(a.db))
	case <-ctx.Done():
	}

	a.logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := a.server.Shutdown(shutdownCtx)
	if err != nil {
		err = fmt.Errorf("stopping the HTTP server: %w", err)
	}
	if served := <-served; !errors.Is(served, http.ErrServerClosed) {
		err = errors.Join(err, fmt.Errorf("serving HTTP: %w", served))
	}

	return errors.Join(err, store.Close(a.db))
}
