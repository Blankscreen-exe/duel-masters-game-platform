# Duel Masters Game Platform

A self-hosted, two-player, browser-based Duel Masters site: player accounts, a deck builder, a
lobby with quick match, ratings and match history, and an admin panel. Games are played with the
rules engine and duel board from the [`duel-masters`](https://github.com/sindreslungaard/duel-masters)
project, maintained here.

> **Status:** early build. The rules engine (`apps/server/internal/engine`), the duel board
> (`packages/duel-board`) and the server foundation (configuration, database, settings, health
> check, Docker) are in place; accounts, decks, the lobby and the website come next. The full
> specification lives in [`design/SPEC.md`](design/SPEC.md) and the reasons behind each decision
> in [`docs/adr/`](docs/adr/).

## Getting the code

The card pictures and data come from a linked repository
([`duelmasters_db`](https://github.com/Blankscreen-exe/duelmasters_db)), so clone with its
submodule:

```bash
git clone --recurse-submodules https://github.com/Blankscreen-exe/duel-masters-game-platform.git
```

In an existing clone, fetch it with `git submodule update --init`.

## Development

**Requirements:** Go (the version in [`apps/server/go.mod`](apps/server/go.mod) or newer),
Node.js 24+ and pnpm 11+ (`corepack enable` sets up the pinned pnpm version).

```bash
pnpm install
pnpm dev:server    # the server on http://127.0.0.1:8080 (check it at /api/health)
pnpm test          # type-checks and tests the packages, then runs the server tests
pnpm test:race     # server tests with the race detector (needs a C compiler)
pnpm vet           # go vet on the server
```

The server is configured with environment variables, checked at startup:

| Variable       | Default                    | Description                                                  |
| -------------- | -------------------------- | ------------------------------------------------------------ |
| `APP_ENV`      | `development`              | `development` or `production`                                |
| `HTTP_ADDR`    | `127.0.0.1:8080`           | Address to listen on. Use `0.0.0.0:8080` inside a container. |
| `DATA_DIR`     | `./data`                   | Where the server keeps its files, including the SQLite file. |
| `DB_DRIVER`    | `sqlite`                   | `sqlite` or `postgres`                                       |
| `SQLITE_PATH`  | `<DATA_DIR>/platform.db`   | The SQLite database file.                                    |
| `POSTGRES_URL` |                            | PostgreSQL address, required with `DB_DRIVER=postgres`.      |
| `LOG_LEVEL`    | `info`                     | `debug`, `info`, `warn` or `error`                           |
| `LOG_FORMAT`   | `text` (`json` in production) | `text` or `json`                                           |
| `SECURE_COOKIES` | `false` (`true` in production) | Send the login cookie over HTTPS only. Turn off only for plain HTTP, such as a home network. |

The database tests run on SQLite and, when `TEST_POSTGRES_URL` points at a PostgreSQL server
(for example `postgres://test:test@127.0.0.1:5432/platform_test?sslmode=disable`), on PostgreSQL
too, each test in its own schema.

After changing [`api/openapi.yaml`](api/openapi.yaml), regenerate the server code with
`go generate ./...` in `apps/server`.

Changes to the rules engine follow
[`apps/server/internal/engine/CONTRIBUTING.md`](apps/server/internal/engine/CONTRIBUTING.md).
Known engine bugs and the order we fix them in are in
[`docs/engine-fixes.md`](docs/engine-fixes.md).

The first account to sign up becomes the admin, so sign up straight after the first start; the
server warns in its log until an account exists.

## Running with Docker

```bash
cp .env.example .env                               # optional: adjust the settings
docker compose up -d --build                       # server with SQLite on a data volume
docker compose --profile postgres up -d --build    # server with PostgreSQL (see .env.example)
```

The server is published on `127.0.0.1:8080`; put a tunnel or reverse proxy in front of it for
internet access.

## Theme mockups

Open [`design/themes/index.html`](design/themes/index.html) in a browser to see the theme mockups.
Battle Zone is the chosen theme.

## License

The code is [MIT](LICENSE) licensed. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for
material from others.

This is a non-commercial fan project. It is not affiliated with, endorsed, sponsored or approved by
Wizards of the Coast or Takara Tomy. Duel Masters card pictures and text are © Wizards of the Coast
and Takara Tomy.
