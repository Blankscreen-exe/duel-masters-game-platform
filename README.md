# Duel Masters Game Platform

A self-hosted, two-player, browser-based Duel Masters site: player accounts, a deck builder, a
lobby with quick match, ratings and match history, and an admin panel. Games are played with the
rules engine and duel board from the [`duel-masters`](https://github.com/sindreslungaard/duel-masters)
project, maintained here.

> **Status:** design agreed, not yet built. The full specification lives in
> [`design/SPEC.md`](design/SPEC.md) and the reasons behind each decision in
> [`docs/adr/`](docs/adr/).

## Getting the code

The card pictures and data come from a linked repository
([`duelmasters_db`](https://github.com/Blankscreen-exe/duelmasters_db)), so clone with its
submodule:

```bash
git clone --recurse-submodules https://github.com/Blankscreen-exe/duel-masters-game-platform.git
```

In an existing clone, fetch it with `git submodule update --init`.

## Theme mockups

Open [`design/themes/index.html`](design/themes/index.html) in a browser to see the theme mockups.
Battle Zone is the chosen theme.

## License

The code is [MIT](LICENSE) licensed. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for
material from others.

This is a non-commercial fan project. It is not affiliated with, endorsed, sponsored or approved by
Wizards of the Coast or Takara Tomy. Duel Masters card pictures and text are © Wizards of the Coast
and Takara Tomy.
