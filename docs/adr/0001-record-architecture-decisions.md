# 0001. Record architecture decisions

- Status: Accepted
- Date: 2026-10-24

## Context

The project makes many decisions whose reasons are easy to forget: why a part is built a certain
way, and what was turned down. Without a record, later changes risk undoing a decision without
knowing why it was made.

## Decision

Every significant decision is recorded as a short, numbered record in `docs/adr/`, using the
template in its README. Records are not rewritten; a changed decision gets a new record that
supersedes the old one.

## Alternatives considered

- **Only the specification.** `design/SPEC.md` says what the product is, not why each choice was
  made or what was rejected.
- **Commit messages and issues.** Scattered and hard to find later.

## Consequences

- A decision and its reasons can be found in one place.
- Writing a record is a small extra step for each significant decision.
