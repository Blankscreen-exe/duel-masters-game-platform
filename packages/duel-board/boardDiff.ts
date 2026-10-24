import type { MatchState } from "./types";

/** Whose side of the board a card is on, as the viewer sees it. A spectator
 * sees player one as "me". */
export type BoardSide = "me" | "opponent";

/** The zones the match state lists card by card. Decks, and any hand the
 * viewer may not look at, only arrive as counts. */
export type VisibleZone =
  "hand" | "playzone" | "manazone" | "shieldzone" | "graveyard";

export interface CardLocation {
  side: BoardSide;
  zone: VisibleZone;
}

/** A card that changed location between two match states. A null end means the
 * card came from, or went to, somewhere the viewer cannot see: a deck or a
 * hidden hand. */
export interface CardMove {
  virtualId: string;
  from: CardLocation | null;
  to: CardLocation | null;
}

const BOARD_SIDES: readonly BoardSide[] = ["me", "opponent"];

const VISIBLE_ZONES: readonly VisibleZone[] = [
  "hand",
  "playzone",
  "manazone",
  "shieldzone",
  "graveyard",
];

/** Where each card the viewer can see currently is, by virtualId. The server
 * keeps a card's id for the whole duel, so the same card can be followed from
 * one state to the next whatever zone it moves to. */
export function locateCards(state: MatchState): Map<string, CardLocation> {
  const locations = new Map<string, CardLocation>();

  for (const side of BOARD_SIDES) {
    for (const zone of VISIBLE_ZONES) {
      for (const card of state[side][zone]) {
        locations.set(card.virtualId, { side, zone });
      }
    }
  }

  return locations;
}

/** The cards that changed location from one match state to the next. Cards
 * that only changed in place, such as being tapped or reordered within their
 * zone, are not moves. */
export function diffBoard(previous: MatchState, next: MatchState): CardMove[] {
  const before = locateCards(previous);
  const after = locateCards(next);
  const moves: CardMove[] = [];

  for (const [virtualId, to] of after) {
    const from = before.get(virtualId) ?? null;

    if (!from || !isSameLocation(from, to)) {
      moves.push({ virtualId, from, to });
    }
  }

  for (const [virtualId, from] of before) {
    if (!after.has(virtualId)) {
      moves.push({ virtualId, from, to: null });
    }
  }

  return moves;
}

function isSameLocation(a: CardLocation, b: CardLocation): boolean {
  return a.side === b.side && a.zone === b.zone;
}
