![banner](docs/images/banner.png)

A self-hosted, two-player, browser-based Duel Masters site: player accounts, a deck builder, a
lobby with quick match, computer opponents, ratings and match history, and an admin panel. Games
are played with the rules engine and duel board from the [`duel-masters`](https://github.com/sindreslungaard/duel-masters)
project, maintained here.

![The duel board in the middle of a game](docs/images/game-board.png)

> **Status:** in active development, not yet ready to run in public. Accounts, decks, the lobby,
> live games, computer opponents, ratings and most of the admin panel work. The full specification
> lives in [`design/SPEC.md`](design/SPEC.md) and the reasons behind each decision in
> [`docs/adr/`](docs/adr/).

## Screenshots

| | | |
| :-: | :-: | :-: |
| ![Calling the coin toss](docs/images/coin-toss.png) | ![Confirming an attack: both powers and the result](docs/images/confirm-attack.png) | ![Choosing whether to block](docs/images/block-question.png) |
| Calling the coin toss | Confirming an attack: both powers and the result | Choosing whether to block |
| ![The board on a phone](docs/images/phone.png) | ![Home, with the lobby chat](docs/images/home.png) | ![Finding a game](docs/images/play.png) |
| The board on a phone | Home, with the lobby chat | Finding a game |
| ![Building a deck](docs/images/deck-builder.png) | ![The card database](docs/images/cards.png) | ![The admin panel](docs/images/admin.png) |
| Building a deck | The card database | The admin panel |

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
pnpm dev:web       # the website on http://localhost:3000, in a second terminal
pnpm test          # tests the website and packages, then runs the server tests
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
| `CARD_DATABASE_DIR` | `../../resources/duelmasters_db` | The card database's folder; the default finds the linked one from `apps/server`. The Docker image includes it. |
| `POSTGRES_URL` |                            | PostgreSQL address, required with `DB_DRIVER=postgres`.      |
| `LOG_LEVEL`    | `info`                     | `debug`, `info`, `warn` or `error`                           |
| `LOG_FORMAT`   | `text` (`json` in production) | `text` or `json`                                           |
| `SECURE_COOKIES` | `false` (`true` in production) | Send the login cookie over HTTPS only. Turn off only for plain HTTP, such as a home network. |
| `SITE_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` in development; required in production | The addresses players open the site from, comma-separated. Live connections from other pages are refused. `*.` stands for any one subdomain, for tunnels whose address changes: `https://*.trycloudflare.com`. |
| `APP_SECRET_KEY` | required in production | Encrypts the mail server's settings in the database. Make one with `openssl rand -base64 32`. Without it in development, no mail server can be set up. |

The database tests run on SQLite and, when `TEST_POSTGRES_URL` points at a PostgreSQL server
(for example `postgres://test:test@127.0.0.1:5432/platform_test?sslmode=disable`), on PostgreSQL
too, each test in its own schema.

The website passes every `/api` request on to the server, so the browser sees one site. It has one
setting:

| Variable        | Default                 | Description                 |
| --------------- | ----------------------- | --------------------------- |
| `GO_SERVER_URL` | `http://127.0.0.1:8080` | Where the Go server listens |

In `apps/web`, `pnpm lint`, `pnpm typecheck`, `pnpm format` and `pnpm build` check, format and
build the website.

After changing [`api/openapi.yaml`](api/openapi.yaml), regenerate the server code and the
website's API types with `pnpm api:generate`.

Changes to the rules engine follow
[`apps/server/internal/engine/CONTRIBUTING.md`](apps/server/internal/engine/CONTRIBUTING.md).
Known engine bugs and the order we fix them in are in
[`docs/engine-fixes.md`](docs/engine-fixes.md).

The first account to sign up becomes the admin, so sign up straight after the first start; the
server warns in its log until an account exists. That account needs no email code. Set up the mail
server under Admin panel, Email: until then, other new players wait, and you can let them in from
the Players page. A player who is sent a code has 24 hours to confirm their address (an admin
setting) before the account is deleted.

Running the site on your own computer, you can also play against Claude, using decks it builds
itself: see [`docs/claude-opponent.md`](docs/claude-opponent.md).

## Running with Docker

Docker Compose runs the whole site on one server: the Go server, the website, and
[Caddy](https://caddyserver.com) in front of them, which serves it over HTTPS with a certificate it
gets and renews by itself. On a server with Docker installed:

```bash
git clone --recurse-submodules https://github.com/Blankscreen-exe/duel-masters-game-platform.git
cd duel-masters-game-platform
cp .env.example .env    # set DOMAIN, and APP_SECRET_KEY from: openssl rand -base64 32
docker compose up -d --build                       # with SQLite on a data volume
docker compose --profile postgres up -d --build    # or with PostgreSQL (see .env.example)
```

Point the domain's DNS at the server and open ports 80 and 443 first, so the certificate can be
issued. Then sign up straight away to become the admin. Only Caddy is reachable from outside: it
sends `/api` (with the live connections) to the Go server and everything else to the website. Its
settings are in [`deploy/Caddyfile`](deploy/Caddyfile).

- **Trying it on your own computer:** keep `DOMAIN=localhost` and open <https://localhost>. Caddy
  uses its own certificate there, so the browser warns once.
- **Behind a tunnel or another proxy that adds HTTPS** (such as a Cloudflare tunnel): set
  `DOMAIN=:80` for plain HTTP, point the tunnel at `http://localhost:80` and set `SITE_ORIGINS` to
  the tunnel's address.

To update, run `git pull && git submodule update --init && docker compose up -d --build`. The data
lives in Docker volumes (`data` for the server's files and SQLite database, `postgres-data`, and
`caddy-data` for the certificates), which survive updates and `docker compose down`, but not
`docker compose down -v`.

## Theme mockups

Open [`design/themes/index.html`](design/themes/index.html) in a browser to see the theme mockups.
Battle Zone is the chosen theme.

## License

The code is [MIT](LICENSE) licensed. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for
material from others.

This is a non-commercial fan project. It is not affiliated with, endorsed, sponsored or approved by
Wizards of the Coast or Takara Tomy. Duel Masters card pictures and text are © Wizards of the Coast
and Takara Tomy.
