# Duel Masters Game Platform

A self-hosted, two-player, browser-based Duel Masters site: player accounts, a deck builder, a
lobby with quick match, ratings and match history, and an admin panel. Games are played with the
rules engine and duel board from the [`duel-masters`](https://github.com/sindreslungaard/duel-masters)
project, maintained here.

> **Status:** early build. The rules engine (`apps/server/internal/engine`) and the duel board
> (`packages/duel-board`) are in place; the server and website around them come next. The full
> specification lives in [`design/SPEC.md`](design/SPEC.md) and the reasons behind each decision
> in [`docs/adr/`](docs/adr/).

## Getting the code

The card pictures and data come from a linked repository
([`duelmasters_db`](https://github.com/Blankscreen-exe/duelmasters_db)), so clone with its
submodule:

```bash
git clone --recurse-submodules https://github.com/Blankscreen-exe/duel-masters-game-platform.git
```

In an existing clone, fetch it with `git submodule update --init`.

## Development

**Requirements:** Go (the version in [`apps/server/go.mod`](apps/server/go.mod) or newer),
Node.js 24+ and pnpm 11+ (`corepack enable` sets up the pinned pnpm version).

```bash
pnpm install
pnpm test          # type-checks and tests the packages, then runs the server tests
pnpm test:race     # server tests with the race detector (needs a C compiler)
pnpm vet           # go vet on the server
```

Changes to the rules engine follow
[`apps/server/internal/engine/CONTRIBUTING.md`](apps/server/internal/engine/CONTRIBUTING.md).
Known engine bugs and the order we fix them in are in
[`docs/engine-fixes.md`](docs/engine-fixes.md).

## Theme mockups

Open [`design/themes/index.html`](design/themes/index.html) in a browser to see the theme mockups.
Battle Zone is the chosen theme.

## License

The code is [MIT](LICENSE) licensed. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for
material from others.

This is a non-commercial fan project. It is not affiliated with, endorsed, sponsored or approved by
Wizards of the Coast or Takara Tomy. Duel Masters card pictures and text are © Wizards of the Coast
and Takara Tomy.
