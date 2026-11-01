// Package config reads the server's settings from environment variables and
// checks them once at startup, so a misconfigured server refuses to start with
// every problem listed, instead of failing later.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/secrets"
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

// defaultLocalPlayerAddr is LocalPlayerAddr in development, and what the
// repository's .mcp.json points Claude Code at.
const defaultLocalPlayerAddr = "127.0.0.1:8081"

// Config is the server's configuration.
type Config struct {
	Environment Environment `env:"APP_ENV" envDefault:"development"`
	// HTTPAddr is the address the HTTP server listens on.
	HTTPAddr string `env:"HTTP_ADDR" envDefault:"127.0.0.1:8080"`
	// DataDir holds the server's files, including the SQLite database. An
	// empty value counts as unset.
	DataDir string `env:"DATA_DIR" envDefault:"./data"`
	// CardDatabaseDir is the card database's folder (resources/duelmasters_db
	// in the repository). The default finds it from apps/server, where the
	// server runs in development.
	CardDatabaseDir string `env:"CARD_DATABASE_DIR" envDefault:"../../resources/duelmasters_db"`
	// SecureCookiesSetting is SECURE_COOKIES as given, nil when unset.
	SecureCookiesSetting *bool `env:"SECURE_COOKIES"`
	// SecureCookies makes browsers send the session cookie over HTTPS only. It
	// is on in production and off in development unless SECURE_COOKIES says
	// otherwise; turn it off only when players reach the server over plain
	// HTTP, such as on a home network.
	SecureCookies bool `env:"-"`
	// SiteOrigins are the addresses players open the site from, such as
	// https://duelmasters.example, comma-separated. Live connections from any
	// other page are refused. Development defaults to the Next.js site on
	// port 3000; production must set them.
	SiteOrigins []string `env:"SITE_ORIGINS" envSeparator:","`
	// SecretKeyText is APP_SECRET_KEY as given: 32 random bytes in base64, as
	// `openssl rand -base64 32` makes. It is never logged, and is kept out of
	// the database so a stolen database gives no secrets away. Required in
	// production; in development, leaving it out leaves mail unavailable.
	SecretKeyText string `env:"APP_SECRET_KEY"`
	// SecretKey is SecretKeyText parsed; nil when it is not set.
	SecretKey []byte `env:"-"`
	// LocalPlayerAddr is the address Claude Code reaches the tools it plays with
	// on (see docs/claude-opponent.md), which must be on this computer alone.
	// Development defaults to 127.0.0.1:8081; otherwise, unset, it is not served.
	LocalPlayerAddr string `env:"LOCAL_PLAYER_ADDR"`
	Database        Database
	Log             Log
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

	if len(c.SiteOrigins) == 0 && c.Environment == Development {
		c.SiteOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}
	for i, origin := range c.SiteOrigins {
		c.SiteOrigins[i] = strings.TrimRight(strings.TrimSpace(origin), "/")
	}

	if c.LocalPlayerAddr == "" && c.Environment == Development {
		c.LocalPlayerAddr = defaultLocalPlayerAddr
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

	if c.LocalPlayerAddr != "" && !isLoopback(c.LocalPlayerAddr) {
		problems = append(problems, fmt.Errorf("LOCAL_PLAYER_ADDR must be on this computer alone, such as %s, got %q", defaultLocalPlayerAddr, c.LocalPlayerAddr))
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

	if len(c.SiteOrigins) == 0 {
		problems = append(problems, errors.New("SITE_ORIGINS is required in production: the addresses players open the site from, such as https://duelmasters.example"))
	}
	for _, origin := range c.SiteOrigins {
		if !isOrigin(origin) {
			problems = append(problems, fmt.Errorf("SITE_ORIGINS must be addresses such as https://duelmasters.example, without a path, got %q", origin))
		}
	}

	switch {
	case c.SecretKeyText != "":
		key, err := secrets.ParseKey(c.SecretKeyText)
		if err != nil {
			problems = append(problems, fmt.Errorf("APP_SECRET_KEY %w; make one with: openssl rand -base64 32", err))
		}
		c.SecretKey = key
	case c.Environment == Production:
		problems = append(problems, errors.New("APP_SECRET_KEY is required in production; make one with: openssl rand -base64 32"))
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

// isLoopback reports whether an address is host:port on this computer alone.
func isLoopback(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// isOrigin reports whether text is an origin: a scheme of http or https and a
// host, with nothing after them.
func isOrigin(text string) bool {
	parsed, err := url.Parse(text)
	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" &&
		parsed.Path == "" && parsed.RawQuery == "" && parsed.Fragment == "" && parsed.User == nil
}
