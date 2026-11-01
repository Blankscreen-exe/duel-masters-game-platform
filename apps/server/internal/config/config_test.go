package config

import (
	"encoding/base64"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/secrets"
)

// siteOrigins is the setting production needs, for tests about other settings.
const siteOrigins = "SITE_ORIGINS=https://duelmasters.example"

func TestLoadUsesDefaults(t *testing.T) {
	cfg, err := Load(nil)
	require.NoError(t, err)

	assert.Equal(t, Development, cfg.Environment)
	assert.Equal(t, "127.0.0.1:8080", cfg.HTTPAddr)
	assert.Equal(t, "./data", cfg.DataDir)
	assert.Equal(t, SQLite, cfg.Database.Driver)
	assert.Equal(t, filepath.Join("./data", "platform.db"), cfg.Database.SQLitePath)
	assert.Equal(t, slog.LevelInfo, cfg.Log.Level)
	assert.Equal(t, LogText, cfg.Log.Format)
	assert.False(t, cfg.SecureCookies, "development serves plain HTTP")
	assert.Equal(t, "../../resources/duelmasters_db", cfg.CardDatabaseDir)
	assert.Equal(t, []string{"http://localhost:3000", "http://127.0.0.1:3000"}, cfg.SiteOrigins, "the Next.js site in development")
}

// secretKey is a well-formed APP_SECRET_KEY, which production requires.
var secretKey = "APP_SECRET_KEY=" + base64.StdEncoding.EncodeToString(make([]byte, secrets.KeySize))

func TestSecretKey(t *testing.T) {
	t.Run("is read", func(t *testing.T) {
		cfg, err := Load([]string{secretKey})
		require.NoError(t, err)
		assert.Equal(t, make([]byte, secrets.KeySize), cfg.SecretKey)
	})

	t.Run("is optional in development", func(t *testing.T) {
		cfg, err := Load(nil)
		require.NoError(t, err)
		assert.Nil(t, cfg.SecretKey)
	})

	t.Run("is required in production", func(t *testing.T) {
		_, err := Load([]string{"APP_ENV=production", siteOrigins})
		assert.ErrorContains(t, err, "APP_SECRET_KEY is required")
	})

	t.Run("must be well formed, and the error keeps it secret", func(t *testing.T) {
		_, err := Load([]string{"APP_SECRET_KEY=short-and-leaky"})
		assert.ErrorContains(t, err, "APP_SECRET_KEY")
		assert.NotContains(t, err.Error(), "short-and-leaky")

		_, err = Load([]string{"APP_SECRET_KEY=" + base64.StdEncoding.EncodeToString([]byte("too short"))})
		assert.ErrorContains(t, err, "32 bytes")
	})
}

func TestSecureCookies(t *testing.T) {
	for _, test := range []struct {
		name    string
		environ []string
		want    bool
	}{
		{"on in production", []string{"APP_ENV=production", siteOrigins, secretKey}, true},
		{"off in development", nil, false},
		{"turned off in production", []string{"APP_ENV=production", siteOrigins, secretKey, "SECURE_COOKIES=false"}, false},
		{"turned on in development", []string{"SECURE_COOKIES=true"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Load(test.environ)
			require.NoError(t, err)
			assert.Equal(t, test.want, cfg.SecureCookies)
		})
	}

	_, err := Load([]string{"SECURE_COOKIES=maybe"})
	assert.Error(t, err)
}

func TestLoadReadsEveryVariable(t *testing.T) {
	cfg, err := Load([]string{
		"APP_ENV=production",
		secretKey,
		"HTTP_ADDR=0.0.0.0:9000",
		"DATA_DIR=/var/lib/platform",
		"DB_DRIVER=postgres",
		"POSTGRES_URL=postgres://user:secret@db:5432/platform",
		"LOG_LEVEL=debug",
		"CARD_DATABASE_DIR=/resources/duelmasters_db",
		"LOG_FORMAT=text",
		"SITE_ORIGINS=https://duelmasters.example/, http://lan-host:3000",
	})
	require.NoError(t, err)

	assert.Equal(t, Production, cfg.Environment)
	assert.Equal(t, "0.0.0.0:9000", cfg.HTTPAddr)
	assert.Equal(t, "/var/lib/platform", cfg.DataDir)
	assert.Equal(t, Postgres, cfg.Database.Driver)
	assert.Equal(t, "postgres://user:secret@db:5432/platform", cfg.Database.PostgresURL)
	assert.Empty(t, cfg.Database.SQLitePath, "no SQLite file is chosen for PostgreSQL")
	assert.Equal(t, slog.LevelDebug, cfg.Log.Level)
	assert.Equal(t, "/resources/duelmasters_db", cfg.CardDatabaseDir)
	assert.Equal(t, LogText, cfg.Log.Format, "an explicit format wins over the environment's")
	assert.Equal(t, []string{"https://duelmasters.example", "http://lan-host:3000"}, cfg.SiteOrigins, "trimmed")
}

func TestLoadDefaultsDependOnOtherSettings(t *testing.T) {
	t.Run("the SQLite file goes in the data folder", func(t *testing.T) {
		cfg, err := Load([]string{"DATA_DIR=/srv/data"})
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("/srv/data", "platform.db"), cfg.Database.SQLitePath)
	})

	t.Run("an explicit SQLite file is kept", func(t *testing.T) {
		cfg, err := Load([]string{"SQLITE_PATH=/tmp/other.db"})
		require.NoError(t, err)
		assert.Equal(t, "/tmp/other.db", cfg.Database.SQLitePath)
	})

	t.Run("an empty data folder counts as unset", func(t *testing.T) {
		cfg, err := Load([]string{"DATA_DIR="})
		require.NoError(t, err)
		assert.Equal(t, "./data", cfg.DataDir)
	})

	t.Run("production logs JSON", func(t *testing.T) {
		cfg, err := Load([]string{"APP_ENV=production", siteOrigins, secretKey})
		require.NoError(t, err)
		assert.Equal(t, LogJSON, cfg.Log.Format)
	})
}

func TestLoadRefusesInvalidSettings(t *testing.T) {
	tests := []struct {
		name    string
		environ []string
		problem string
	}{
		{"unknown environment", []string{"APP_ENV=staging"}, "APP_ENV"},
		{"address without a port", []string{"HTTP_ADDR=localhost"}, "HTTP_ADDR"},
		{"unknown database", []string{"DB_DRIVER=mysql"}, "DB_DRIVER"},
		{"postgres without an address", []string{"DB_DRIVER=postgres"}, "POSTGRES_URL is required"},
		{"postgres address with sqlite", []string{"POSTGRES_URL=postgres://db/platform"}, "POSTGRES_URL is set"},
		{"sqlite file with postgres", []string{"DB_DRIVER=postgres", "POSTGRES_URL=postgres://db/platform", "SQLITE_PATH=/tmp/x.db"}, "SQLITE_PATH is set"},
		{"unknown log format", []string{"LOG_FORMAT=xml"}, "LOG_FORMAT"},
		{"unknown log level", []string{"LOG_LEVEL=loud"}, "LOG_LEVEL"},
		{"production without site origins", []string{"APP_ENV=production"}, "SITE_ORIGINS is required"},
		{"a site origin with a path", []string{"SITE_ORIGINS=https://example.com/play"}, "SITE_ORIGINS must be"},
		{"a site origin without a scheme", []string{"SITE_ORIGINS=example.com"}, "SITE_ORIGINS must be"},
		{"a site origin of another scheme", []string{"SITE_ORIGINS=ftp://example.com"}, "SITE_ORIGINS must be"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(test.environ)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.problem)
		})
	}
}

func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	_, err := Load([]string{"APP_ENV=staging", "DB_DRIVER=mysql", "LOG_FORMAT=xml"})
	require.Error(t, err)

	for _, problem := range []string{"APP_ENV", "DB_DRIVER", "LOG_FORMAT"} {
		assert.Contains(t, err.Error(), problem)
	}
}

func TestLoadLocalPlayerAddress(t *testing.T) {
	cfg, err := Load(nil)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:8081", cfg.LocalPlayerAddr, "on by default in development")

	cfg, err = Load([]string{"APP_ENV=production", "SITE_ORIGINS=https://duelmasters.example", secretKey})
	require.NoError(t, err)
	assert.Empty(t, cfg.LocalPlayerAddr, "not served in production unless asked")

	cfg, err = Load([]string{"LOCAL_PLAYER_ADDR=127.0.0.1:9191"})
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9191", cfg.LocalPlayerAddr)

	for _, address := range []string{"0.0.0.0:8081", "192.168.1.20:8081", ":8081", "example.com:8081", "nonsense"} {
		_, err = Load([]string{"LOCAL_PLAYER_ADDR=" + address})
		assert.ErrorContains(t, err, "LOCAL_PLAYER_ADDR", address)
	}
}
