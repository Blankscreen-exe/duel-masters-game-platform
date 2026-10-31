# UI fix list

Look-and-feel changes found while playing. We work through them one at a time, after the engine
fixes in progress. Paths are relative to `packages/duel-board/` unless they say otherwise.

---

### [x] 1. The duel's pop-ups keep the old demo look
- **Status:** Fixed, with the rest of the board. The site and the board share one set of looks, the
  `bz-*` styles in `apps/web/src/app/globals.css`.
- **Where:** `Popup.tsx`, `Action.tsx` and the other pop-ups the board opens during a game
  (questions, card previews, waiting, warnings, forfeit).
- **Problem:** they still use the design of the original demo board instead of the site's Battle
  Zone theme.
- **Fix:** restyle them on the site's theme, sharing the site's modal look rather than copying it,
  with the same hover states and phone layout.

### [ ] 2. A player can't look at the field while a question is open
- **Where:** `Action.tsx`, `Duel.tsx`
- **Problem:** a question (choosing cards, yes/no, picking a blocker) covers the board, so a player
  can't check the field before deciding.
- **Decided:** a "View field" button shrinks the question to a bar along the bottom of the board,
  showing its text and a "Back to choice" button. With the question shrunk:
  - the whole board shows, and any card can be tapped or hovered for its details, with the
    graveyard and mana open to look through;
  - moves stay locked: nothing on the board can be played, attacked with or charged until the
    question is answered;
  - it works the same on phones and desktops.
