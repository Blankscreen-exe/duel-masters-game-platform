// Command server runs the Duel Masters Game Platform server. It is configured
// with environment variables (see internal/config) and stops cleanly on
// Ctrl+C or SIGTERM.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/app"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.Environ())
	if err != nil {
		return err
	}

	logger := logging.New(cfg.Log, os.Stdout)
	logging.RouteLogrus(logrus.StandardLogger(), logger.With("component", "engine"), cfg.Log.Level)

	server, err := app.New(ctx, cfg, logger)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", cfg.HTTPAddr, err)
	}

	logger.Info("listening", "address", listener.Addr().String(), "environment", cfg.Environment, "database", cfg.Database.Driver)

	return server.Serve(ctx, listener)
}
