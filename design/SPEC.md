# Duel Masters Game Platform — Specification

Status: design in progress, not yet built. Last updated 2026-10-24.

A self-hosted, two-player, browser-based Duel Masters site with player accounts, a deck builder,
a lobby with quick match, ratings and match history, and an admin panel. Games are played with the
Duel Masters rules engine and duel board from the `duel-masters` project (MIT licensed), copied
into this repository and maintained here.

Items marked **(default)** are proposals accepted as starting values; they can be changed in the
admin panel or revisited later.

## Engineering principles

These apply to every part of the project and outrank convenience.

- **No "just for now" code.** A choice that may change later (a storage backend, where card
  resources come from) is still built properly, so changing it later is cheap. Nothing is
  written in a way that has to be torn out.
- **DRY.** One implementation of each rule or behaviour, shared by everything that needs it. Deck
  legality, for example, is checked by one piece of Go code used by the deck builder, the lobby,
  quick match and game creation.
- **Modular, swappable parts.** Each replaceable part sits behind a small interface and is chosen
  by configuration: the database (SQLite or PostgreSQL), the card resources source, the clock and
  randomness used by matchmaking. Switching one touches one place.
- **Contract tests.** Every interface has one shared test suite that runs against each of its
  implementations, so a new implementation is proven to behave the same.
- **Tested at every level**, as in Triple Triad: unit tests, data-layer tests against real
  databases, engine tests, and end-to-end tests against the production build.

---

## 1. Hosting and deployment

- Runs locally (`localhost`) and on a VPS, exposed to the internet through a Cloudflare tunnel or a
  reverse proxy, like Triple Triad. Cloudflare passes WebSockets through.
- Two processes, started together by Docker Compose: the **Go server** and the **Next.js** site.
- Players reach **one address**. Next.js serves the pages and forwards `/api/*` and the live game
  connection (`/ws/*`) to the Go server, so there is one origin and one login cookie.
- All environment-specific values (ports, data folder, card resources folder, database, secrets)
  come from environment variables, validated at startup.
- **One Go server per database.** Live games are held in the server's memory, so a second server
  on the same database would not see them. Migrations therefore run from a single server at
  startup.

## 2. Tech stack and architecture

| Part        | Choice                                                                                          |
| ----------- | ----------------------------------------------------------------------------------------------- |
| Backend     | **Go**, one server: accounts, sessions, decks, formats, lobby, matchmaking, chat, ratings, admin and the rules engine |
| Database    | **SQLite** by default, switchable to **PostgreSQL** in config; owned by the Go server, accessed through the **GORM** ORM |
| Frontend    | **Next.js** (reused from Triple Triad): pages only; every read and write goes to the Go API     |
| Game board  | `duel-interface` React package (copied from the `duel-masters` project)                         |
| Game resources | The `duelmasters_db` repository, **linked** into this one (git submodule): all card pictures, card data and icons, served by the Go server |

```text
Browser ──HTTP──▶ Next.js (pages) ──/api, /ws──▶ Go server ──▶ SQLite
                                                   ├─ rules engine (live games)
                                                   └─ card resources (linked duelmasters_db)
```

- The Go server is the only thing that reads or writes data. Next.js holds no secrets and no
  database.
- **Database switch**: one setting picks SQLite (a file in the data folder, nothing to install) or
  PostgreSQL (a connection address). The data layer is written once for both with the GORM ORM:
  one set of models, one set of versioned migrations that run automatically at startup, and the
  same test suite run against both (ADR 0012). Backups (section 9) work the same on either, so data can be moved from
  SQLite to PostgreSQL by exporting and importing.
- The engine now lives in the same program as the site backend, so games are created, joined and
  reported **inside the Go server**: no shared secret, signed duel tickets or result webhook
  between two servers. A player's login session authorises their game connection **(default)**.

## 3. Accounts and roles

- Players sign up and log in with a **name and password**, with the same rules as Triple Triad:
  - names are 3 to 20 letters, numbers, underscores or hyphens, unique regardless of case;
  - passwords are 8 to 128 characters, stored as Argon2id hashes (OWASP settings);
  - a login answers a wrong name and a wrong password alike, in the same time, and an account
    refuses logins for 15 minutes after 5 wrong passwords;
  - a login lasts 30 days in an HttpOnly, SameSite=Lax cookie (HTTPS-only in production); the
    database stores only a hash of each session token;
  - changing the password needs the current one and ends the account's other sessions.
- Admins can close sign-ups with the `accounts.signups_open` setting; existing players can still
  log in.
- Roles: **player** and **admin**. **The first account to sign up becomes the admin**, as in Triple
  Triad; the server warns in its log at startup while no account exists, so the owner signs up
  first.
- Banned players cannot log in or play; a ban ends their open sessions at once.
- No email **(default)**: a forgotten password is reset by an admin.

## 4. Cards

### 4.1 Card catalogue

- **Game resources come from the card database repository**
  (https://github.com/Blankscreen-exe/duelmasters_db), linked into this repository (see 4.3): the
  pictures of **all 2,647 cards** (WebP, about 145 MB), the card data (rarity, set, collector
  number, abilities, keywords, races, rules text) and the civilization, rarity, set and keyword
  icons. Nothing is fetched from outside at runtime.
- The Go server reads them through one **card resources** component (package `cards`), pointed
  at a folder by configuration (`CARD_DATABASE_DIR`). No other code knows where pictures or card
  data come from.
- **The catalogue is copied into the server's database** at startup, when a fingerprint of the
  card database, the engine's cards and the copy format has changed, in one transaction (ADR
  0013). Cards that leave the card database stay in the catalogue, hidden, so nothing referring
  to them breaks. Pictures and set icons are served from the files.
- **A reprinted card is one card** with every printing listed in release order (18 names, such
  as Solar Ray in DM-01, DM-10 and DM-32). Its details and picture are its first printing's, the
  same picture the board shows in games; deck copy limits count the card, not the printing.
- The playable cards are the ones the rules engine implements (895). The card list shows every
  card (2,625), with the ones the engine cannot play yet marked and kept out of decks.
- Cards are known by their first printing's id in the card database (`0079`). Engine cards are
  linked by name: exactly, then ignoring case, accents, spaces and punctuation, then through an
  alias for the engine's misspelt "Hypersprint Warior Uzesol". A test fails if any playable card
  is not linked or has no picture.
- Card pictures are 491 × 682 WebP, the same shape the board expects. The board's picture address,
  hard-coded to the original project's server today, becomes a setting.

### 4.2 Ownership

- **Every playable card is free to use.** There is no collection, trunk, shop or card rewards.

### 4.3 Linked card database

- `duelmasters_db` is a **git submodule** of this repository: this repository records which
  version of it to use, and the pictures are never copied into this repository's history.
- Cloning: `git clone --recurse-submodules`; an existing clone runs
  `git submodule update --init`. The Docker build does the same.
- Updating the card database (new pictures, fixed data) is one commit that moves the pointer to a
  newer version, tested like any other change.
- Size: the pictures are stored in plain git inside `duelmasters_db`. If that ever becomes a
  problem, Git LFS is applied there; this repository does not change.

### 4.4 Rights

The card pictures and texts are official Wizards of the Coast / Takara Tomy material and the
repository is public. The README and site footer state that the project is a non-commercial fan
project not affiliated with or endorsed by them, and credit them as the owners.

## 5. Decks and formats

- **Deck size: exactly 40 cards. Copy limit: 4 of the same card** **(default)**, the official
  rules.
- A player can have any number of decks **(default: up to 50)**, each with a name and a cover card.
- **Formats** are managed by admins. A format has a name, the sets it allows, and a list of banned
  and limited cards (limited = at most 1 or 2 copies). Starting formats **(default)**:
  - **Open**: every playable card.
  - **DM-01 to DM-12**: the first twelve sets.
- A deck is checked against a format when it is used. The deck builder shows, for each format,
  whether the deck is legal and what breaks the rules.
- **Starter decks**: admins build ready-made decks; every new account receives copies of them.
- Decks can be exported and imported as text (`4 Bolshack Dragon` per line).

## 6. Lobby

- **Games**: a player creates a game with these options; others see it in the lobby and join it.

| Option             | Values                                         |
| ------------------ | ---------------------------------------------- |
| Name               | Free text, up to 50 characters (random if empty) |
| Deck               | One of the player's decks, legal in the format |
| Format             | One of the admin's formats                     |
| Visibility         | **Public** (listed) or **Private** (invite link only) |
| Spectators         | Allowed or not                                 |

- **Quick match**: the player picks a format and deck and waits; they are paired with the next
  waiting player in that format, closest rating first **(default: widening the rating range the
  longer they wait)**.
- **Spectators**: logged-in players can watch public games that allow them. Visitors cannot watch
  games.
- **Who goes first is never a game option**: every game starts with a coin toss (7.1).
- **Online players** list, with each player's rank and what they are doing.
- **Lobby chat**: one global chat. **Visitors who are not logged in can read it; posting needs an
  account.** Admins can delete messages and mute players.

## 7. Playing

- The game screen shows the duel board (everything we built: animations, power and status
  badges, phase bar with turn count, game log with clickable card names) between two name bars
  with each player's rank.
- **Leave** and **Surrender** buttons; surrendering asks for confirmation and loses the game.
- At the end, a **result screen**: winner, how it ended, turns, length, each player's rating
  change, and buttons for **Rematch** and **Back to lobby** (7.2).
- **Disconnects**: the engine already lets a player reconnect to a running game and tells the
  opponent. A player who does not come back within **2 minutes (default)** loses.
- **Engine changes needed**: surrender, rematch (with deck changes), the disconnect time limit, the
  configurable card picture address, and reporting results straight into the database. The coin
  toss rules already exist in the engine; the animation is new work in the duel board.

### 7.1 Coin toss

Every game, rematches included, starts with a coin toss that both players (and spectators) see.

1. One player **calls heads or tails**: picked at random for a new game, the loser of the previous
   game in a rematch (after a drawn game, picked at random). The other player automatically gets
   the other side. While one player calls, the other sees who is calling.
2. The coin is **flipped with an animation** both players watch; the server decides the result
   before the animation starts, and both screens land on the same side.
3. The **winner chooses to play first or second**. The other player sees who is choosing.
4. The game log records the call, the result and the choice.

### 7.2 Rematch

- Either player can offer a rematch from the result screen; the other accepts or declines.
- Before accepting, **each player can switch to another of their decks** that is legal in the
  game's format. The offer shows each player's chosen deck by name and civilizations only.
- When both have accepted, a new game starts in the same format and with the same options,
  beginning with a coin toss (7.1). Declining, leaving or closing the result screen ends the
  rematch offer and sends both players to the lobby.

## 8. Ratings and history

- Each player has an **Elo-style rating starting at 1000** and a named tier, as in Triple Triad
  **(default)**:

| Tier     | Rating         |
| -------- | -------------- |
| Bronze   | below 1100     |
| Silver   | 1100–1299      |
| Gold     | 1300–1499      |
| Platinum | 1500–1699      |
| Diamond  | 1700 and above |

- Each tier except Diamond is split into III, II and I.
- Wins raise the rating and losses lower it; players also have win/loss streaks.
- Games shorter than **60 seconds (default)** do not count, so instant leaves cannot farm rating.
- **Match history** keeps every finished game: players, decks, format, winner, how it ended,
  turns, length and rating changes.

## 9. Admin panel

- **Dashboard**: players online, games running, server status.
- **Users**: search, ban/unban, reset password, make admin.
- **Games**: watch or end a running game; recent results.
- **Formats**: create and edit formats, their sets and their banned/limited cards.
- **Starter decks**: build them and choose which new accounts receive.
- **Chat**: delete messages, mute players.
- **Settings**: every server setting, kept in a key-value table and edited here. Each setting has a
  key, a value, a type (text, number, yes/no), a default, a short description and allowed limits;
  the panel shows them grouped, checks values against their type and limits, and can reset one to
  its default. Changes apply without a restart. Starting settings: sign-ups open or closed, new
  games allowed or paused, rating speed, minimum game length for ratings, quick match rating range,
  disconnect time limit, maximum decks per player.
- **Backup**: export and import all data in one zip, as in Triple Triad, but with each table saved
  in a database-neutral format (not a SQLite file), so a backup from SQLite restores into
  PostgreSQL and the other way round.

## 10. Pages

Every page has hover feedback and works on phones (see 11.2).

### 10.1 Shared elements

- **Header (desktop)**: logo, menu (Play, Decks, Cards, Rankings, Rules), the player's name and
  rank, user menu (Profile, Settings, Admin for admins, Log out). Visitors see Log in and Sign up.
- **Bottom tab bar (phone)**: Play, Decks, Cards, Rankings, Profile.
- **Footer**: rules, credits, the MIT license notice of the engine, and "not affiliated with
  Wizards of the Coast".
- **Card preview**: hover (desktop) or tap (phone) any card for the full picture and its text.
- **Toasts** ("Deck saved", "Shobu joined your game"), **confirm dialogs** for risky actions,
  **loading placeholders**, and friendly **empty states**.

### 10.2 Public pages

| Page        | Path      | Contents                                                                         |
| ----------- | --------- | -------------------------------------------------------------------------------- |
| Home        | `/`       | Banner with Play now, what the site is, showcase cards, live numbers (players online, games running), the lobby chat (read-only for visitors) |
| Log in      | `/login`  | Name and password                                                                |
| Sign up     | `/signup` | Name, password, repeat password                                                  |
| Rules       | `/rules`  | How to play: turn steps, zones, and a keyword list with the board's badge icons  |
| Card list   | `/cards`  | Every playable card; filters (civilization, cost, type, race, set, keyword, text); card details |

### 10.3 Player pages

| Page          | Path                  | Contents                                                              |
| ------------- | --------------------- | --------------------------------------------------------------------- |
| Lobby         | `/play`               | Open games (join/watch), create game, quick match, online players, lobby chat. On phones the games list becomes stacked cards |
| Game          | `/play/[gameId]`      | Board, name bars, Leave/Surrender, result screen. Spectators see the same without the buttons |
| My decks      | `/decks`              | Deck tiles: cover card, civilizations, count, legal formats; new, copy, delete, import |
| Deck builder  | `/decks/[deckId]`     | Card browser with filters; deck list grouped by cost with + and −; mana curve, civilization split, count and rule warnings; rename, cover card, export. On phones: Cards, Deck and Stats tabs |
| Rankings      | `/rankings`           | Leaderboard with tier badges, your position highlighted, search, pages |
| Profile       | `/u/[name]`           | Rank, rating, wins and losses, streak, favourite civilizations, recent matches |
| Match history | `/u/[name]/matches`   | All matches: opponent, result, decks, format, turns, length, rating change |
| Settings      | `/settings`           | Change password; game options (animations, opponent cards facing you); blocked users |

### 10.4 Admin pages

`/admin` (dashboard), `/admin/users`, `/admin/games`, `/admin/formats`, `/admin/starter-decks`,
`/admin/chat`, `/admin/settings`, `/admin/backup`, as described in section 9.

## 11. UI theme: Battle Zone

Chosen 2026-10-24. Reference mockup: `design/themes/theme-battle-zone.html` (the other files in
`design/themes/` are rejected alternatives).

### 11.1 Look

| Token          | Value                                                                                 | Used for                  |
| -------------- | ------------------------------------------------------------------------------------- | ------------------------- |
| Page           | `#07090e`, radial glow `#1a2233` at the top                                           | Body                      |
| Panel          | `#0f141d` at 95%, 1 px ring `slate-500` at 40%, corners cut at 45° (14 px)             | Panels, dialogs           |
| Header         | Gradient `#1b2230` → `#0d1119`, a five-colour civilization stripe along the bottom    | Top navigation            |
| Civilizations  | Light `#fde047`, Water `#3b82f6`, Darkness `#52525b`, Fire `#ef4444`, Nature `#22c55e` | Stripe, panel tops, dots |
| Primary button | Silver gradient `slate-100` → `slate-500`, dark text, cut corners (8 px)              | Main actions              |
| Accent         | Sky `#38bdf8` (active menu, selected options), blue `#2563eb` (your turn, phases)     | Highlights                |
| Status         | Waiting: emerald; in game and danger: red                                             | Tags, Surrender           |
| Fonts          | **Rajdhani** (headings, buttons, labels, uppercase with wide spacing), **Inter** (body) |                         |

- Fonts are bundled with the app, not loaded from Google at runtime.

### 11.2 Interaction and phones

- **Hover**: buttons brighten and glow; table rows and list items light up; cards lift and grow
  slightly; links and menu items show the sky underline. Every hover effect has a matching
  keyboard focus style.
- **Motion** respects the player's "reduce motion" setting.
- **Phones**: bottom tab bar instead of the top menu; tables become stacked cards; tap targets at
  least 44 px; forms one column; side panels move below the main content.

## 12. Data model (outline)

| Table                              | Contents                                                          |
| ---------------------------------- | ----------------------------------------------------------------- |
| users                              | name, password hash, role, rating, streaks, banned and muted flags, settings |
| sessions                           | user, token hash, expiry                                          |
| card_sets                          | sets in release order, with their icons                           |
| cards / card_printings             | the card catalogue copied from the card database: one row per card (engine id, details, search text, name order, whether still in the card database) and one per printing (set, collector number, rarity, picture) |
| card_civilizations / card_races / card_keywords / card_categories / card_abilities | each card's lists, for filtering and display |
| card_catalogue_state               | the fingerprint of the last copy                                  |
| decks / deck_cards                 | player's decks, cover card, the 40 cards                          |
| formats / format_sets / format_cards | formats, their allowed sets, banned and limited cards           |
| starter_decks / starter_deck_cards | admin-built starter decks                                         |
| games                              | players, decks, format, options, state, winner, how it ended, turns, length, rating changes |
| chat_messages                      | lobby chat, with moderation state                                 |
| blocks                             | who blocked whom                                                  |
| settings                           | key-value server settings (key, value, updated by, updated at), managed in the admin panel (section 9). Types, defaults, descriptions and limits are defined in the Go code, so an unknown key or invalid value is refused |

## 13. Repository layout

A monorepo: deployable apps in `apps/`, shared code in `packages/`. The Go server follows the
standard Go layout and is organised **by feature**, so each feature can be changed or replaced
inside one folder.

```text
duel-masters-game-platform/
├─ apps/
│  ├─ server/                         Go backend, one Go module
│  │  ├─ cmd/server/main.go           wiring only: config, store, features, HTTP server
│  │  ├─ internal/                    importable only by this server (enforced by Go)
│  │  │  ├─ config/                   environment settings, checked at startup
│  │  │  ├─ httpapi/                  routes, handlers, middleware (sessions, limits, logging)
│  │  │  ├─ realtime/                 live connections: lobby updates, chat, games
│  │  │  ├─ auth/                     passwords, sessions, roles, bans
│  │  │  ├─ users/                    accounts, profiles, blocks
│  │  │  ├─ cards/                    card catalogue and the card resources interface
│  │  │  ├─ decks/                    decks, import/export, the single legality check
│  │  │  ├─ formats/                  formats, allowed sets, banned and limited cards
│  │  │  ├─ lobby/                    open games, invites, spectators
│  │  │  ├─ matchmaking/              quick match queue
│  │  │  ├─ games/                    the only bridge to the engine: start, surrender, results
│  │  │  ├─ chat/                     lobby chat and moderation
│  │  │  ├─ ratings/                  Elo, tiers, streaks, history
│  │  │  ├─ settings/                 key-value settings: definitions, types, defaults, limits
│  │  │  ├─ backup/                   database-neutral export and import
│  │  │  ├─ store/                    database connection (GORM: SQLite or PostgreSQL), migrations
│  │  │  └─ engine/                   the rules engine in its original layout (game/, server/,
│  │  │                                internal/, tests/), with its rule book CONTRIBUTING.md
│  │  └─ Dockerfile
│  └─ web/                            Next.js site (from Triple Triad)
│     ├─ src/app/                     pages, one folder per route
│     ├─ src/components/              ui/, layout/, cards/, decks/, lobby/, admin/
│     ├─ src/lib/api/                 the only code that talks to the Go server (generated client)
│     ├─ public/                      fonts, logo, static images
│     └─ Dockerfile
├─ packages/
│  ├─ duel-board/                     the duel board (copied duel-interface) with its tests
│  └─ config/                         shared TypeScript, ESLint and Prettier settings
├─ api/openapi.yaml                   the API contract between web and server (13.1)
├─ resources/duelmasters_db/          the linked card database (git submodule, 4.3)
├─ e2e/                               Playwright tests of the whole site, both apps running
├─ design/                            this specification and the theme mockups
├─ docs/                              DEPLOYMENT.md, ARCHITECTURE.md, adr/ (decision records)
├─ .github/workflows/                 CI for server, web and e2e
├─ docker-compose.yml                 runs server and web together
├─ package.json, pnpm-workspace.yaml  one entry point for dev, test and lint across both apps
└─ README.md, LICENSE, THIRD_PARTY_NOTICES.md
```

- **Feature folders** hold their own logic, models and data access (through GORM); `store/` opens
  the database and runs the migrations. Features never import each other's internals, only their
  exported interfaces.
- **The engine is self-contained** in `internal/engine/`, in the same layout as in `duel-masters`
  (ADR 0011); `games/` is the only package that uses it. Every engine change follows
  `internal/engine/CONTRIBUTING.md`.
- **Decision records**: each significant decision gets a short note in `docs/adr/` (what, why,
  alternatives), starting with the ones already made in this specification.

### 13.1 API contract

- `api/openapi.yaml` describes every HTTP endpoint, request and response. The Go handler
  interfaces and request/response types, and the TypeScript client in `apps/web/src/lib/api/`, are
  **generated** from it, so the two sides cannot drift apart. Generated code is never edited by
  hand; CI fails if it is out of date.
- Live messages (lobby updates, chat, games) are defined once in `api/` as schemas and generated
  for both sides in the same way. The game messages keep the engine's existing message format.

### 13.2 Licensing

- This project's own code is **MIT** licensed (`LICENSE`).
- `THIRD_PARTY_NOTICES.md` keeps the MIT notice of the `duel-masters` engine and board, credits
  for other bundled open-source work, and the card art and text ownership notice (4.4).

## 14. Out of scope for now

- Card collections, packs, shops or rewards.
- Turn timers (the engine has none).
- Tournaments, friends lists, private messages.
- Visitors watching games (spectating needs an account).
- Email (verification, password reset by email).

## 15. Open details (to settle before or during build)

None at the moment.
