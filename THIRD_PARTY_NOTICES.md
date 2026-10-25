# Third-party notices

This project's own code is MIT licensed (see [`LICENSE`](LICENSE)). It includes or links the
following material from others.

## duel-masters rules engine and duel board

`apps/server/internal/engine/` (the rules engine, its card data and its tests) and
`packages/duel-board/` (the duel board) come from
[sindreslungaard/duel-masters](https://github.com/sindreslungaard/duel-masters), copied at
upstream commit `e5443966` together with changes made in the
[Blankscreen-exe fork](https://github.com/Blankscreen-exe/duel-masters) (duel board improvements
and a spectator lock fix), and maintained here since. They are used under this license:

```text
MIT License

Copyright (c) 2019 Sindre Slungaard

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Fonts

The website bundles two fonts under the SIL Open Font License 1.1, with their license texts beside
them in [`apps/web/src/app/fonts/`](apps/web/src/app/fonts/):

- **Rajdhani**, © 2014 Indian Type Foundry (`rajdhani-OFL.txt`);
- **Inter**, © 2016 The Inter Project Authors (`inter-OFL.txt`).

## Duel Masters card pictures and text

The card pictures, names and rules text in the linked card database
([`resources/duelmasters_db`](https://github.com/Blankscreen-exe/duelmasters_db)) are official
Duel Masters material, © Wizards of the Coast and Takara Tomy. They were collected from
[db.duelmasters.us](https://db.duelmasters.us/).

This is a non-commercial fan project. It is not affiliated with, endorsed, sponsored or approved by
Wizards of the Coast or Takara Tomy, and it claims no ownership of their material.
