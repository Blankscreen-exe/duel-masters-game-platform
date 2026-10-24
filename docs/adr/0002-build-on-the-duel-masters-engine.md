# 0002. Build on the duel-masters engine and board

- Status: Accepted
- Date: 2026-10-24

## Context

A Duel Masters site needs a rules engine that knows every card's effects, turns, battles and
shields. The open-source [`duel-masters`](https://github.com/sindreslungaard/duel-masters) project
(MIT licensed) has a Go engine implementing about 895 cards and a React duel board, which was
improved beforehand in a fork (animations, power and status badges, phase bar with turn count, game
log with clickable card names).

## Decision

Copy the engine and the duel board into this repository and maintain them here. The fork is no
longer developed, so the copy is the only version going forward.

## Alternatives considered

- **Link the fork (git submodule).** Pointless once the fork is abandoned, and it would split every
  change across two repositories.
- **Write a new engine.** Months or years of work to reach the same card coverage.

## Consequences

- The site starts with a complete, tested engine and the improved board.
- New cards and fixes from the original project no longer arrive automatically; they would have to
  be ported by hand.
- The engine's MIT notice must be kept (`THIRD_PARTY_NOTICES.md`) when its code is copied in.
