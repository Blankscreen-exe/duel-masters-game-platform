# 0012. GORM for data access

- Status: Accepted
- Date: 2026-10-24

## Context

The server stores its data in SQLite or PostgreSQL, chosen by configuration (0005). It needs one
way to define its data and talk to either database.

## Decision

Use the **GORM** ORM:

- Data is defined once as Go models; the same code runs on both databases.
- SQLite through `github.com/ncruces/go-sqlite3/gormlite`, a pure-Go driver (SQLite compiled to
  Go), so the server builds without a C compiler. It was chosen over `github.com/glebarez/sqlite`,
  whose GORM driver has had no release since March 2024. PostgreSQL through
  `gorm.io/driver/postgres` (pgx).
- Schema changes are **versioned migrations** written once in Go with
  `go-gormigrate/gormigrate`, recorded in a migrations table and applied in order at startup.
  GORM's `AutoMigrate` is not used on its own, because it cannot rename, drop or transform data and
  keeps no history.
- The data-layer tests run against both databases.

This replaces the plan in 0005 to keep hand-written queries and two sets of migrations, one per
database. The rest of 0005 (the configuration switch, database-neutral backups) stands.

## Alternatives considered

- **Hand-written SQL** (with `sqlx`): full control, but every query written by hand.
- **sqlc**: generated, type-safe code from SQL, but a separate copy of the queries per database.
- **ent**: generated, type-safe entities, but more tooling than this project needs.

## Consequences

- One set of models and migrations for both databases.
- GORM logs through the server's `slog` logger (`logger.NewSlogLogger`).
- Database-specific features are avoided unless both databases support them.
