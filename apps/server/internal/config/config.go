// Package config reads the server's settings from environment variables and
// checks them once at startup, so a misconfigured server refuses to start with
// every problem listed, instead of failing later.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"

	"github.com/caarlos0/env/v11"
)

// Environment is where the server runs.
type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

// DatabaseDriver is the database the server stores its data in.
type DatabaseDriver string

const (
	SQLite   DatabaseDriver = "sqlite"
	Postgres DatabaseDriver = "postgres"
)

// LogFormat is how log lines are written.
type LogFormat string

const (
	// LogText is readable by people, for development.
	LogText LogFormat = "text"
	// LogJSON is one JSON object per line, for log collectors.
	LogJSON LogFormat = "json"
)

// sqliteFileName is the database file created in the data folder when no
// SQLite path is given.
const sqliteFileName = "platform.db"

// Config is the server's configuration.
type Config struct {
	Environment Environment `env:"APP_ENV" envDefault:"development"`
	// HTTPAddr is the address the HTTP server listens on.
	HTTPAddr string `env:"HTTP_ADDR" envDefault:"127.0.0.1:8080"`
	// DataDir holds the server's files, including the SQLite database. An
	// empty value counts as unset.
	DataDir string `env:"DATA_DIR" envDefault:"./data"`
	// SecureCookiesSetting is SECURE_COOKIES as given, nil when unset.
	SecureCookiesSetting *bool `env:"SECURE_COOKIES"`
	// SecureCookies makes browsers send the session cookie over HTTPS only. It
	// is on in production and off in development unless SECURE_COOKIES says
	// otherwise; turn it off only when players reach the server over plain
	// HTTP, such as on a home network.
	SecureCookies bool `env:"-"`
	Database      Database
	Log           Log
}

// Database selects and locates the database.
type Database struct {
	Driver DatabaseDriver `env:"DB_DRIVER" envDefault:"sqlite"`
	// SQLitePath is the SQLite database file. It defaults to platform.db in the
	// data folder.
	SQLitePath string `env:"SQLITE_PATH"`
	// PostgresURL is the PostgreSQL connection address, required with the
	// postgres driver. It contains a password, so it is never logged.
	PostgresURL string `env:"POSTGRES_URL"`
}

// Log sets how much the server logs and in which format.
type Log struct {
	// LevelName is LOG_LEVEL as given: debug, info, warn or error.
	LevelName string `env:"LOG_LEVEL" envDefault:"info"`
	// Level is LevelName parsed.
	Level slog.Level `env:"-"`
	// Format defaults to text in development and JSON in production.
	Format LogFormat `env:"LOG_FORMAT"`
}

// Load reads the configuration from environment variables given as
// "KEY=value" entries, as os.Environ returns them, fills in the defaults that
// depend on other settings and checks the result.
func Load(environ []string) (Config, error) {
	var cfg Config
	if err := env.ParseWithOptions(&cfg, env.Options{Environment: env.ToMap(environ)}); err != nil {
		return Config{}, fmt.Errorf("reading configuration: %w", err)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Database.Driver == SQLite && c.Database.SQLitePath == "" {
		c.Database.SQLitePath = filepath.Join(c.DataDir, sqliteFileName)
	}

	c.SecureCookies = c.Environment == Production
	if c.SecureCookiesSetting != nil {
		c.SecureCookies = *c.SecureCookiesSetting
	}

	if c.Log.Format == "" {
		c.Log.Format = LogText
		if c.Environment == Production {
			c.Log.Format = LogJSON
		}
	}
}

func (c *Config) validate() error {
	var problems []error

	switch c.Environment {
	case Development, Production:
	default:
		problems = append(problems, fmt.Errorf("APP_ENV must be %q or %q, got %q", Development, Production, c.Environment))
	}

	if _, _, err := net.SplitHostPort(c.HTTPAddr); err != nil {
		problems = append(problems, fmt.Errorf("HTTP_ADDR must be host:port, got %q", c.HTTPAddr))
	}

	switch c.Database.Driver {
	case SQLite:
		if c.Database.PostgresURL != "" {
			problems = append(problems, errors.New("POSTGRES_URL is set but DB_DRIVER is sqlite; set DB_DRIVER=postgres or remove POSTGRES_URL"))
		}
	case Postgres:
		if c.Database.PostgresURL == "" {
			problems = append(problems, errors.New("POSTGRES_URL is required when DB_DRIVER is postgres"))
		}
		if c.Database.SQLitePath != "" {
			problems = append(problems, errors.New("SQLITE_PATH is set but DB_DRIVER is postgres; remove SQLITE_PATH"))
		}
	default:
		problems = append(problems, fmt.Errorf("DB_DRIVER must be %q or %q, got %q", SQLite, Postgres, c.Database.Driver))
	}

	if err := c.Log.Level.UnmarshalText([]byte(c.Log.LevelName)); err != nil {
		problems = append(problems, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, got %q", c.Log.LevelName))
	}

	switch c.Log.Format {
	case LogText, LogJSON:
	default:
		problems = append(problems, fmt.Errorf("LOG_FORMAT must be %q or %q, got %q", LogText, LogJSON, c.Log.Format))
	}

	return errors.Join(problems...)
}
