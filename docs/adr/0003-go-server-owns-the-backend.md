# 0003. The Go server does all backend work

- Status: Accepted
- Date: 2026-10-24

## Context

The rules engine is written in Go. The site also needs accounts, sessions, decks, formats, a lobby,
matchmaking, chat, ratings, an admin panel and a database.

## Decision

One Go server does all of it, the engine included. The frontend only renders pages and calls the Go
API.

## Alternatives considered

- **Split backend**: Next.js (TypeScript, reusing Triple Triad's server code) for accounts, decks
  and the rest, with Go only playing games. Reuses more existing code, but means two backends in
  two languages talking through a shared secret, signed duel tickets and a result webhook.

## Consequences

- One backend, one language and one source of truth for data. Games are created, joined and
  reported inside the same program, so no secret or tickets are passed between servers, and a
  player's login session can authorise their game connection.
- Deck legality, formats and ratings can be checked right next to the engine.
- Triple Triad's TypeScript backend code (accounts, admin, backups) is not reused; those parts are
  written in Go.
