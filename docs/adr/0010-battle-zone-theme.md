# 0010. Battle Zone theme

- Status: Accepted
- Date: 2026-10-24

## Context

Triple Triad's theme is modelled on Final Fantasy VIII's menus and does not suit Duel Masters.
Four directions were mocked up as standalone pages (`design/themes/`): Battle Zone, Shobu Anime,
Kaijudo Arcane and Five Civilizations.

## Decision

Use **Battle Zone**: dark and modern, angled panel corners, silver primary buttons, the five
civilization colours as accents, Rajdhani and Inter fonts. Every page has hover feedback with a
matching keyboard focus style and works on phones (bottom tab bar, stacked lists, large tap
targets). Details in `design/SPEC.md`, section 11.

## Alternatives considered

The owner compared all four mockups and chose Battle Zone:

- **Shobu Anime**: orange halftone background, thick black outlines, slanted comic titles.
- **Kaijudo Arcane**: deep violet, gold trim, glowing edges.
- **Five Civilizations**: light panels, each section marked with a civilization colour.

## Consequences

- The board's dark look fits the theme without changes.
- The rejected mockups stay in `design/themes/` for reference.
