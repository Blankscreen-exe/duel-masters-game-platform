# Architecture decision records

Each significant decision is recorded here: what was decided, why, the alternatives considered and
what follows from it. Records are never rewritten; a changed decision gets a new record that
supersedes the old one.

| No.                                                     | Decision                                              | Status   |
| ------------------------------------------------------- | ----------------------------------------------------- | -------- |
| [0001](0001-record-architecture-decisions.md)           | Record architecture decisions                         | Accepted |
| [0002](0002-build-on-the-duel-masters-engine.md)        | Build on the duel-masters engine and board            | Accepted |
| [0003](0003-go-server-owns-the-backend.md)              | The Go server does all backend work                   | Accepted |
| [0004](0004-nextjs-frontend-from-triple-triad.md)       | Next.js frontend, reused from Triple Triad            | Accepted |
| [0005](0005-sqlite-default-postgres-switchable.md)      | SQLite by default, switchable to PostgreSQL           | Accepted |
| [0006](0006-link-the-card-database.md)                  | Link the card database as a git submodule             | Accepted |
| [0007](0007-monorepo-organised-by-feature.md)           | Monorepo, Go server organised by feature              | Accepted |
| [0008](0008-openapi-contract-with-generated-code.md)    | OpenAPI contract with generated code                  | Accepted |
| [0009](0009-mit-license.md)                             | MIT license                                           | Accepted |
| [0010](0010-battle-zone-theme.md)                       | Battle Zone theme                                     | Accepted |

## Template

```markdown
# NNNN. Title

- Status: Proposed | Accepted | Superseded by NNNN
- Date: YYYY-MM-DD

## Context

What problem or question led to the decision.

## Decision

What was decided.

## Alternatives considered

The other options and why they were not chosen.

## Consequences

What follows from the decision, good and bad.
```
