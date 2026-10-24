# 0011. The engine keeps its original layout

- Status: Accepted
- Date: 2026-10-24

## Context

The rules engine and duel board are copied in (0002). The engine could either keep the folder
structure it has in `duel-masters` or be reorganised to match the rest of the server. The original
project also contains a standalone web server around the engine, which this project replaces
(0003).

## Decision

- The engine keeps its original structure under `apps/server/internal/engine/`: `game/` (match,
  cards, fx, cnd, civ, family), `server/` (the live game protocol), `internal/` (helpers),
  `tests/` (scenario harness and tests) and `DuelMastersCards.json`. Only the Go import prefix
  changes.
- Left behind, because this project rebuilds their jobs its own way: `api/` and `cmd/` (the
  standalone HTTP server with its shared secret and duel tickets), `flags/`, `moderation/`, and
  the helpers the engine does not use (`mail.go`, `crypto.go`, `option.go`, `ratelimit.go`).
- The engine's rule book (`AGENTS.md` in the original) becomes
  `apps/server/internal/engine/CONTRIBUTING.md`, with its paths updated. Every engine change
  follows it.
- The duel board becomes `packages/duel-board` (`@duel-masters/duel-board`), unchanged apart from
  its package name.
- The move changes nothing else. Upgrades (Go version, libraries, React 19) and new behaviour are
  separate changes, so a test failure can always be traced to one cause.

## Alternatives considered

- **Reorganise the engine** (for example `engine/match`, `engine/cards`): tidier names, but every
  fix or card ported from the original project would need its paths translated.

## Consequences

- Fixes and cards from the original project can be ported almost as they are.
- The move was verified by running the same tests before and after: the identical list of 1,091
  passing engine tests (one skipped, as before), the race detector, and all 213 board tests.
- Two board libraries resolved to newer patch versions inside the same version ranges: `motion`
  14.6.0 (was 14.4.0) and `@types/react` 18.3.31 (was 18.3.27).
