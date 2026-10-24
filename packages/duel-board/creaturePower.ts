import type { BoardSide } from "./boardDiff";
import type { MeasuredCard } from "./cardPositions";
import { CARD_ASPECT_RATIO } from "./cardImage";
import type { CreaturePower, MatchState } from "./types";

/** How a creature's power compares with the power printed on it. */
export type PowerTone = "raised" | "lowered" | "printed";

export function powerTone({ printed, current }: CreaturePower): PowerTone {
  if (current > printed) {
    return "raised";
  }

  return current < printed ? "lowered" : "printed";
}

/** How much more power the creature has while it attacks. */
export function attackBonus({ current, attacking }: CreaturePower): number {
  return Math.max(0, attacking - current);
}

/** What a creature's power badge says. */
export interface PowerReadout {
  /** The power the creature has right now. */
  shown: number;
  /** How the shown power came about, which colours the badge. */
  tone: PowerTone | "attacking";
}

/** What a creature's power badge says: its power now, which while it attacks
 * is the power it attacks with, bonuses such as power attacker's included. */
export function powerReadout(
  power: CreaturePower,
  attacking: boolean,
): PowerReadout {
  if (attacking) {
    return { shown: power.attacking, tone: "attacking" };
  }

  return { shown: power.current, tone: powerTone(power) };
}

/** A change of power, written the way it is announced: "+2000" or "−3000". */
export function formatPowerDelta(delta: number): string {
  return delta >= 0 ? `+${delta}` : `−${Math.abs(delta)}`;
}

/** A sentence about a creature's power, for screen readers and tooltips. */
export function describePower(power: CreaturePower): string {
  const parts = [`Power ${power.current}`];

  if (power.current !== power.printed) {
    parts.push(`printed ${power.printed}`);
  }

  if (attackBonus(power) > 0) {
    parts.push(`${power.attacking} when attacking`);
  }

  return parts.join(", ");
}

/** The power of a creature on either side of the board, if it has one. */
export function findCreaturePower(
  state: MatchState,
  virtualId: string,
): CreaturePower | null {
  return (
    state.me.powers?.[virtualId] ?? state.opponent.powers?.[virtualId] ?? null
  );
}

export interface PowerChange {
  virtualId: string;
  side: BoardSide;
  delta: number;
}

const BOARD_SIDES: readonly BoardSide[] = ["me", "opponent"];

/** The creatures whose power changed between two states while they stayed in
 * the battle zone. A creature that just arrived has nothing to compare with.
 * Only the power outside attacks counts, since attacking power comes and goes
 * with each turn. */
export function powerChanges(
  previous: MatchState,
  next: MatchState,
): PowerChange[] {
  const changes: PowerChange[] = [];

  for (const side of BOARD_SIDES) {
    const before = previous[side].powers ?? {};

    for (const [virtualId, power] of Object.entries(next[side].powers ?? {})) {
      const earlier = before[virtualId];

      if (earlier && earlier.current !== power.current) {
        changes.push({
          virtualId,
          side,
          delta: power.current - earlier.current,
        });
      }
    }
  }

  return changes;
}

/** Where a power change is announced: just above the card as it is drawn.
 * A tapped card lies on its side, so its top edge is closer to its centre. */
export function powerPopupPoint(card: MeasuredCard): { x: number; y: number } {
  const sideways = Math.abs(card.rotation) % 180 === 90;
  const halfHeight = sideways ? card.width / 2 : card.height / 2;

  return { x: card.x, y: card.y - halfHeight };
}

/** How far up from the bottom of its box a card's lower edge is drawn, as a
 * share of the box's height. A tapped card lies on its side, which leaves a
 * gap above and below it. */
export function lowerEdgeShare(rotated: boolean): number {
  return rotated ? (1 - CARD_ASPECT_RATIO) / 2 : 0;
}
