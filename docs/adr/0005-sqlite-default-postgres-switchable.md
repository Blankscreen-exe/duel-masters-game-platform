# 0005. SQLite by default, switchable to PostgreSQL

- Status: Accepted, amended by [0012](0012-gorm-for-data-access.md) (data access and migrations)
- Date: 2026-10-24

## Context

A small installation should need nothing extra to run, while a larger one may want a database
server.

## Decision

The database is chosen by configuration: **SQLite** (a file in the data folder) by default, or
**PostgreSQL** (a connection address). Features declare the data access they need as interfaces;
`store/` implements them once per database. Migrations exist for both and run automatically at
startup. The same data-layer test suite runs against both. Backups use a database-neutral format,
so data moves between the two by exporting and importing.

## Alternatives considered

- **SQLite only**: simplest, but moving to a database server later would mean rewriting the data
  layer.
- **PostgreSQL only**: every installation needs a database server.

## Consequences

- Queries stay within what both databases support.
- Two sets of migrations to maintain, both covered by tests.
