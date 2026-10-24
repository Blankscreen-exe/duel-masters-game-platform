# 0006. Link the card database as a git submodule

- Status: Accepted
- Date: 2026-10-24

## Context

The site needs card pictures, card data and icons, all served locally. They already live in the
public [`duelmasters_db`](https://github.com/Blankscreen-exe/duelmasters_db) repository: pictures of
all 2,647 cards (about 145 MB of WebP files), card data and icons.

## Decision

Link `duelmasters_db` into this repository as a git submodule at `resources/duelmasters_db`. The
Go server reads it through one **card resources** component whose folder is set by configuration.

## Alternatives considered

- **Copy the pictures into this repository.** Duplicates 145 MB, and moving to a link later would
  either leave the copy in this repository's history for good or require rewriting that history.
- **Git LFS in this repository.** Not needed: the pictures are not stored here at all. If size ever
  matters, LFS is applied in `duelmasters_db` and this repository does not change.

## Consequences

- One source of truth for card resources; updating them is one commit moving the submodule pointer.
- Clones need `--recurse-submodules` (or `git submodule update --init`), and so does the Docker
  build.
- Only the card resources component knows where resources come from, so replacing the source
  later touches one place.
