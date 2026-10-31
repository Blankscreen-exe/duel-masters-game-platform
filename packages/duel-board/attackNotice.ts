import type { MatchState, PromptAttack } from "./types";

/** What an attack is aimed at: the player, or one of their creatures. */
export type AttackTargetKind = "player" | "creature";

/** One end of an attack as it is shown. */
export interface AttackEnd {
  name: string;
  /** Absent for a player, or a creature not chosen yet. */
  power?: number;
}

/** An attack as it is shown: who attacks, with what power, and what. */
export interface AttackNotice {
  /** Whose creature attacks, which colours the attack. */
  side: "mine" | "theirs";
  attacker: Required<AttackEnd>;
  target: AttackEnd;
}

/** The attack a player is about to make with one of their creatures, or null
 * if the board no longer has it. Which creature or shields it hits is only
 * chosen once the attack goes ahead. */
export function plannedAttack(
  state: MatchState,
  virtualId: string,
  target: AttackTargetKind,
): AttackNotice | null {
  const creature = state.me.playzone.find(
    (card) => card.virtualId === virtualId,
  );
  const power = state.me.powers[virtualId];

  if (!creature || !power) {
    return null;
  }

  return {
    side: "mine",
    attacker: { name: creature.name, power: power.attacking },
    target: { name: plannedTargetName(state, target) },
  };
}

function plannedTargetName(
  state: MatchState,
  target: AttackTargetKind,
): string {
  if (target === "creature") {
    return "Opponent's creature";
  }

  return state.opponent.shieldzone.length > 0
    ? "Opponent's shields"
    : "Opponent";
}

/** The opponent's attack that a block question asks about. */
export function blockedAttack({
  attacker,
  target,
}: PromptAttack): AttackNotice {
  return { side: "theirs", attacker, target: target ?? { name: "You" } };
}

/** The attack in a sentence, for screen readers. */
export function describeAttack({ attacker, target }: AttackNotice): string {
  const power = target.power === undefined ? "" : ` with ${target.power} power`;

  return `${attacker.name} with ${attacker.power} power attacks ${target.name}${power}`;
}
