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
	"runtime"
	"time"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
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

// sessionCleanupInterval is how often expired sessions are deleted.
const sessionCleanupInterval = time.Hour

// App is the assembled server.
type App struct {
	logger *slog.Logger
	db     *gorm.DB
	auth   *auth.Service
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

	app, err := assemble(ctx, cfg, logger, db)
	if err != nil {
		return nil, errors.Join(err, store.Close(db))
	}

	return app, nil
}

// assemble builds the features on an open, migrated database.
func assemble(ctx context.Context, cfg config.Config, logger *slog.Logger, db *gorm.DB) (*App, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	siteSettings, err := settings.NewService(ctx, db, logger.With("component", "settings"), auth.Settings()...)
	if err != nil {
		return nil, err
	}

	accounts := users.NewService(db, logger.With("component", "users"))
	if exist, err := accounts.AnyExist(ctx); err != nil {
		return nil, err
	} else if !exist {
		logger.Warn("there are no accounts yet: the first account to sign up becomes the admin, so sign up now")
	}

	authService := auth.NewService(db, accounts, siteSettings, auth.NewPasswordHasher(runtime.NumCPU()), time.Now, logger.With("component", "auth"))

	source, err := cards.ReadSource(cfg.CardDatabaseDir)
	if err != nil {
		return nil, fmt.Errorf("the card database at %s (CARD_DATABASE_DIR): %w", cfg.CardDatabaseDir, err)
	}
	if _, err := cards.Sync(ctx, db, source, games.PlayableCards(), logger.With("component", "cards")); err != nil {
		return nil, err
	}
	catalogue := cards.NewCatalogue(db, cfg.CardDatabaseDir)

	httpLogger := logger.With("component", "http")

	return &App{
		logger: logger,
		db:     db,
		auth:   authService,
		server: &http.Server{
			Handler: httpapi.NewHandler(httpapi.Dependencies{
				DB:            sqlDB,
				Auth:          authService,
				Catalogue:     catalogue,
				Logger:        httpLogger,
				SecureCookies: cfg.SecureCookies,
			}),
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
// shutdown timeout, stops the background work and closes the database. It
// returns nil after a clean shutdown.
func (a *App) Serve(ctx context.Context, listener net.Listener) error {
	// Background work runs while Serve does, and has stopped before the
	// database closes, whichever way Serve ends.
	backgroundCtx, stopBackground := context.WithCancel(ctx)
	background := make(chan struct{})
	go func() {
		defer close(background)
		a.cleanUpSessions(backgroundCtx)
	}()
	closeDown := func() error {
		stopBackground()
		<-background
		return store.Close(a.db)
	}

	served := make(chan error, 1)
	go func() { served <- a.server.Serve(listener) }()

	select {
	case err := <-served:
		return errors.Join(fmt.Errorf("serving HTTP: %w", err), closeDown())
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

	return errors.Join(err, closeDown())
}

// cleanUpSessions deletes expired sessions straight away, then once every
// interval, until ctx ends.
func (a *App) cleanUpSessions(ctx context.Context) {
	ticker := time.NewTicker(sessionCleanupInterval)
	defer ticker.Stop()

	for {
		deleted, err := a.auth.DeleteExpiredSessions(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			a.logger.Warn("deleting expired sessions failed", "error", err)
		case deleted > 0:
			a.logger.Info("deleted expired sessions", "count", deleted)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
