# 0007. Monorepo, Go server organised by feature

- Status: Accepted
- Date: 2026-10-24

## Context

The project has a Go server, a Next.js site, a shared duel board package, an API contract,
end-to-end tests and design documents, which change together.

## Decision

One repository (layout in `design/SPEC.md`, section 13):

- deployable apps in `apps/` (`server`, `web`), shared code in `packages/`;
- the Go server follows the standard Go layout: `cmd/server` for wiring only, everything else in
  `internal/`, grouped **by feature** (`auth`, `decks`, `lobby`, `ratings`, ...);
- each feature declares the data access it needs as an interface, implemented in `store/`;
  features use each other only through exported interfaces;
- the engine stays self-contained in `internal/engine/`, and `games/` is the only package that uses
  it.

## Alternatives considered

- **Separate repositories per app**: changes spanning server, contract and site would need
  coordinated releases across repositories.
- **Grouping by technical layer** (all handlers together, all repositories together): a single
  feature change would touch many folders, and features would be harder to replace.

## Consequences

- A feature can be changed or replaced inside one folder.
- One place to run development, tests and lint for everything.
