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
	"slices"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/admin"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/mail"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/matchmaking"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/realtime"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/rematch"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/secrets"
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

// offlineAfter is how long a player with no live connection still counts as
// online, so reloading a page does not count as leaving the site.
const offlineAfter = 10 * time.Second

// App is the assembled server.
type App struct {
	logger     *slog.Logger
	db         *gorm.DB
	auth       *auth.Service
	hub        *realtime.Hub
	games      *games.Service
	lobby      *lobby.Service
	quickMatch *matchmaking.Service
	rematches  *rematch.Service
	server     *http.Server
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

	siteSettings, err := settings.NewService(ctx, db, logger.With("component", "settings"), slices.Concat(auth.Settings(), chat.Settings(), decks.Settings(), games.Settings(), lobby.Settings(), matchmaking.Settings(), opponents.Settings(), ratings.Settings(), rematch.Settings())...)
	if err != nil {
		return nil, err
	}

	accounts := users.NewService(db, logger.With("component", "users"))
	if exist, err := accounts.AnyExist(ctx); err != nil {
		return nil, err
	} else if !exist {
		logger.Warn("there are no accounts yet: the first account to sign up becomes the admin, so sign up now")
	}

	var secretBox *secrets.Box
	if cfg.SecretKey != nil {
		if secretBox, err = secrets.NewBox(cfg.SecretKey); err != nil {
			return nil, err
		}
	} else {
		logger.Warn("APP_SECRET_KEY is not set, so no mail server can be set up and new players cannot confirm their email address")
	}
	mailService := mail.NewService(db, secretBox, logger.With("component", "mail"))

	authService := auth.NewService(db, accounts, siteSettings, auth.NewPasswordHasher(runtime.NumCPU()), mailService, time.Now, logger.With("component", "auth"))

	source, err := cards.ReadSource(cfg.CardDatabaseDir)
	if err != nil {
		return nil, fmt.Errorf("the card database at %s (CARD_DATABASE_DIR): %w", cfg.CardDatabaseDir, err)
	}
	if _, err := cards.Sync(ctx, db, source, games.PlayableCards(), logger.With("component", "cards")); err != nil {
		return nil, err
	}
	if err := store.RunSeeds(db, slices.Concat(decks.Seeds(logger.With("component", "decks")), opponents.Seeds(logger.With("component", "opponents")))...); err != nil {
		return nil, err
	}
	catalogue := cards.NewCatalogue(db, cfg.CardDatabaseDir)
	deckService := decks.NewService(db, catalogue, siteSettings, logger.With("component", "decks"))
	formatService := formats.NewService(db)

	httpLogger := logger.With("component", "http")
	hub := realtime.NewHub(realtime.Options{OfflineAfter: offlineAfter, Logger: logger.With("component", "realtime")})
	ratingService := ratings.NewService(db, siteSettings)
	live := httpapi.NewLiveUpdates(hub, ratingService, logger.With("component", "live"))

	gameService := games.NewService(db, catalogue, siteSettings, ratingService, logger.With("component", "games"))
	gameService.Listen(live)
	lobbyService := lobby.NewService(lobby.Dependencies{
		Decks:     deckService,
		Formats:   formatService,
		Games:     gameService,
		Directory: live,
		Settings:  siteSettings,
		Listener:  live,
		Logger:    logger.With("component", "lobby"),
		Now:       time.Now,
	})
	quickMatch := matchmaking.NewService(matchmaking.Dependencies{
		Lobby:     lobbyService,
		Games:     gameService,
		Ratings:   ratingService,
		Directory: live,
		Settings:  siteSettings,
		Listener:  live,
		Logger:    logger.With("component", "matchmaking"),
		Now:       time.Now,
	})
	builtCards, _ := cards.Build(source, games.PlayableCards())
	opponentService := opponents.NewService(db, catalogue, formatService)
	challenges := opponents.NewChallenges(opponents.ChallengeDependencies{
		Opponents: opponentService,
		Formats:   formatService,
		Lobby:     lobbyService,
		Games:     gameService,
		Library:   computer.NewLibrary(builtCards),
		Settings:  siteSettings,
		Logger:    logger.With("component", "opponents"),
	})
	rematches := rematch.NewService(rematch.Dependencies{
		Lobby:      lobbyService,
		Games:      gameService,
		Challenges: challenges,
		Directory:  live,
		Settings:   siteSettings,
		Listener:   live,
		Logger:     logger.With("component", "rematch"),
		Now:        time.Now,
	})
	hub.Watch(presence{lobby: lobbyService, quickMatch: quickMatch, rematches: rematches})
	adminService := admin.NewService(admin.Dependencies{
		DB:          db,
		Auth:        authService,
		Mail:        mailService,
		Connections: hub,
		Games:       gameService,
		Settings:    siteSettings,
		Listener:    live,
		Logger:      logger.With("component", "admin"),
		Now:         time.Now,
	})

	return &App{
		logger:     logger,
		db:         db,
		auth:       authService,
		hub:        hub,
		games:      gameService,
		lobby:      lobbyService,
		quickMatch: quickMatch,
		rematches:  rematches,
		server: &http.Server{
			Handler: httpapi.NewHandler(httpapi.Dependencies{
				Mail:          mailService,
				DB:            sqlDB,
				Auth:          authService,
				Users:         accounts,
				Ratings:       ratingService,
				Catalogue:     catalogue,
				Formats:       formatService,
				FormatManager: formats.NewManager(db, formatService, catalogue),
				Decks:         deckService,
				Chat:          chat.NewService(db, live, siteSettings, time.Now),
				Games:         gameService,
				Lobby:         lobbyService,
				QuickMatch:    quickMatch,
				Rematches:     rematches,
				Opponents:     opponentService,
				Challenges:    challenges,
				Settings:      siteSettings,
				Admin:         adminService,
				AdminLog:      adminlog.NewService(db),
				Hub:           hub,
				Upgrader:      realtime.NewUpgrader(cfg.SiteOrigins),
				Logger:        httpLogger,
				SecureCookies: cfg.SecureCookies,
				Now:           time.Now,
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
	var background sync.WaitGroup
	background.Go(func() { a.cleanUpSessions(backgroundCtx) })
	background.Go(func() { a.games.Run(backgroundCtx) })
	background.Go(func() { a.lobby.Run(backgroundCtx) })
	background.Go(func() { a.quickMatch.Run(backgroundCtx) })
	background.Go(func() { a.rematches.Run(backgroundCtx) })

	// Games and live connections are taken over from the HTTP server, which
	// does not wait for them, so they are closed here: running games are
	// stopped and, once the background work stops, saved, before the
	// database closes.
	closeDown := func(ctx context.Context) error {
		err := errors.Join(a.games.Close(ctx), a.hub.Close(ctx))
		stopBackground()
		background.Wait()
		return errors.Join(err, store.Close(a.db))
	}

	served := make(chan error, 1)
	go func() { served <- a.server.Serve(listener) }()

	select {
	case err := <-served:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return errors.Join(fmt.Errorf("serving HTTP: %w", err), closeDown(shutdownCtx))
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

	return errors.Join(err, closeDown(shutdownCtx))
}

// cleanUpSessions deletes expired sessions and accounts that ran out of time to
// confirm their email address straight away, then once every interval, until
// ctx ends.
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

		deleted, err = a.auth.DeleteUnconfirmedAccounts(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			a.logger.Warn("deleting unconfirmed accounts failed", "error", err)
		case deleted > 0:
			a.logger.Info("deleted accounts that never confirmed their email address", "count", deleted)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// presence tells the lobby, quick match and rematches when players come and
// go, so a player who leaves the site leaves their open game, the queue and
// any rematch.
type presence struct {
	lobby      *lobby.Service
	quickMatch *matchmaking.Service
	rematches  *rematch.Service
}

func (p presence) PresenceChanged(viewer realtime.Viewer) {
	p.lobby.PresenceChanged(viewer.UserID)
	p.quickMatch.PresenceChanged(viewer.UserID)
	p.rematches.PresenceChanged(viewer.UserID)
}
