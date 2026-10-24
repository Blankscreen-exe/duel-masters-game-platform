# 0004. Next.js frontend, reused from Triple Triad

- Status: Accepted
- Date: 2026-10-24

## Context

With the Go server doing all backend work (0003), the frontend only renders pages. The owner's
Triple Triad project already has a tested Next.js frontend: page shell, shared UI components, login
and sign-up, lobby, rankings, profile and admin pages.

## Decision

Use Next.js, starting from Triple Triad's frontend:

- keep the page shell, shared components and page structure;
- rewire every page that read Triple Triad's own backend to call the Go API through one client
  (`apps/web/src/lib/api/`);
- replace the Triple Triad card, board, card maker and art pack components with the duel board and
  Duel Masters cards.

Next.js forwards `/api/*` and `/ws/*` to the Go server, so players reach one address with one login
cookie.

## Alternatives considered

- **React with Vite**, served by the Go server: one program and one port, but nothing reused from
  Triple Triad.

## Consequences

- A familiar stack and a head start on pages and components.
- Next.js runs as a second process beside Go; most of its server features go unused.
