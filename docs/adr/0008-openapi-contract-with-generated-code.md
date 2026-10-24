# 0008. OpenAPI contract with generated code

- Status: Accepted
- Date: 2026-10-24

## Context

The Next.js site (TypeScript) calls the Go server. Both sides need the same request and response
shapes, and any mismatch is a bug.

## Decision

`api/openapi.yaml` describes every HTTP endpoint. The Go handler interfaces and types and the
TypeScript client are generated from it. Live messages (lobby, chat, games) are defined once as
schemas in `api/` and generated for both sides the same way; game messages keep the engine's
existing format. Generated code is never edited by hand, and CI fails if it is out of date.

## Alternatives considered

- **Hand-written types on both sides**: every change made twice, with mismatches only caught by
  tests.

## Consequences

- The two sides cannot drift apart, and the API is documented by its contract.
- Code generators become part of the build and are pinned to exact versions.
