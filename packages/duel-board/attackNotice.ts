import {
  cardHasFlag,
  TAPPED_FLAG,
  type ActionMessage,
  type CardState,
  type MatchState,
  type PromptAttack,
} from "./types";

/** One end of an attack as it is shown. */
export interface AttackEnd {
  name: string;
  /** Absent for a player or shields, which have no power. */
  power?: number;
}

/** What an attack will do, as far as the board can tell. */
export interface AttackOutcome {
  /** How it went for the attacker, which colours the headline. */
  tone: "win" | "lose" | "draw" | "neutral";
  headline: string;
  /** The creatures the battle destroys. */
  destroyed: string[];
  /** A creature of the opponent's that may still block, by name. */
  blockers: string[];
}

/** An attack as it is shown: who attacks, with what power, and what. */
export interface AttackNotice {
  /** Whose creature attacks, which colours the attack. */
  side: "mine" | "theirs";
  attacker: Required<AttackEnd>;
  target: AttackEnd;
  outcome?: AttackOutcome;
}

function hasStatus(
  statuses: MatchState["me"]["statuses"],
  virtualId: string,
  id: string,
): boolean {
  return (statuses[virtualId] ?? []).some((status) => status.id === id);
}

/** The opponent's creatures that may block this attacker. Some effects let a
 * creature get past blockers the board cannot see, so this is what may block. */
function possibleBlockers(state: MatchState, attackerId: string): string[] {
  if (hasStatus(state.me.statuses, attackerId, "cant_be_blocked")) {
    return [];
  }

  return state.opponent.playzone
    .filter(
      (card) =>
        !cardHasFlag(card.flags, TAPPED_FLAG) &&
        hasStatus(state.opponent.statuses, card.virtualId, "blocker"),
    )
    .map((card) => card.name);
}

function attackerOf(
  state: MatchState,
  virtualId: string,
): { card: CardState; power: number } | null {
  const card = state.me.playzone.find((c) => c.virtualId === virtualId);
  const power = state.me.powers[virtualId];

  return card && power ? { card, power: power.attacking } : null;
}

/** The result of a battle, worked out the way the rules engine does it: the
 * higher power destroys the lower and a tie destroys both, then a winner that
 * is destroyed after winning goes too, and so does a winner beaten by slayer. */
export function battleOutcome(
  state: MatchState,
  attackerId: string,
  targetId: string,
): AttackOutcome | null {
  const attacker = attackerOf(state, attackerId);
  const target = state.opponent.playzone.find((c) => c.virtualId === targetId);
  const targetPower = state.opponent.powers[targetId]?.current;

  if (!attacker || !target || targetPower === undefined) {
    return null;
  }

  const attackerSlayer = hasStatus(state.me.statuses, attackerId, "slayer");
  const targetSlayer = hasStatus(state.opponent.statuses, targetId, "slayer");
  const attackerSuicide = hasStatus(
    state.me.statuses,
    attackerId,
    "destroy_after_battle",
  );
  const targetSuicide = hasStatus(
    state.opponent.statuses,
    targetId,
    "destroy_after_battle",
  );

  const attackerWins = attacker.power >= targetPower;
  const targetWins = attacker.power <= targetPower;

  const attackerDestroyed =
    targetWins || (attackerWins && (targetSlayer || attackerSuicide));
  const targetDestroyed =
    attackerWins || (targetWins && (attackerSlayer || targetSuicide));

  const headline =
    attacker.power === targetPower
      ? "It's a tie"
      : attacker.power > targetPower
        ? "You win the battle"
        : "You lose the battle";

  return {
    tone:
      attacker.power === targetPower
        ? "draw"
        : attacker.power > targetPower
          ? "win"
          : "lose",
    headline,
    destroyed: [
      ...(attackerDestroyed ? [attacker.card.name] : []),
      ...(targetDestroyed ? [target.name] : []),
    ],
    blockers: possibleBlockers(state, attackerId),
  };
}

function creatureAttack(
  state: MatchState,
  attackerId: string,
  targetId: string,
): AttackNotice | null {
  const attacker = attackerOf(state, attackerId);
  const target = state.opponent.playzone.find((c) => c.virtualId === targetId);
  const outcome = battleOutcome(state, attackerId, targetId);

  if (!attacker || !target || !outcome) {
    return null;
  }

  return {
    side: "mine",
    attacker: { name: attacker.card.name, power: attacker.power },
    target: {
      name: target.name,
      power: state.opponent.powers[targetId].current,
    },
    outcome,
  };
}

/** "3" or "1 and 3". */
function joinNumbers(numbers: number[]): string {
  return numbers.length < 2
    ? numbers.join("")
    : `${numbers.slice(0, -1).join(", ")} and ${numbers[numbers.length - 1]}`;
}

function shieldAttack(
  state: MatchState,
  attackerId: string,
  shieldIds: string[],
): AttackNotice | null {
  const attacker = attackerOf(state, attackerId);

  if (!attacker) {
    return null;
  }

  const numbers = shieldIds
    .map((id) => state.opponent.shieldMap[id])
    .filter((number) => number !== undefined)
    .sort((a, b) => a - b);
  const shields = `${shieldIds.length === 1 ? "Shield" : "Shields"} ${joinNumbers(numbers)}`;

  return {
    side: "mine",
    attacker: { name: attacker.card.name, power: attacker.power },
    target: { name: shields },
    outcome: {
      tone: "neutral",
      headline: `Breaks ${shields.toLowerCase()}`,
      destroyed: [],
      blockers: possibleBlockers(state, attackerId),
    },
  };
}

/** The attack to confirm once the player has answered a question asking what
 * to attack, or null if the answer goes straight to the server: it is not such
 * a question, its selection is not complete and the server should say so, or
 * the attack can no longer be called off. */
export function attackToConfirm(
  state: MatchState,
  question: ActionMessage,
  selected: string[],
): AttackNotice | null {
  const attackerId = question.source?.virtualId;

  if (!attackerId || !question.cancellable) {
    return null;
  }

  if (question.kind === "attack_target" && selected.length === 1) {
    return creatureAttack(state, attackerId, selected[0]);
  }

  if (
    question.kind === "break_shields" &&
    selected.length > 0 &&
    selected.length === question.maxSelections
  ) {
    return shieldAttack(state, attackerId, selected);
  }

  return null;
}

/** The attack to confirm when a creature attacks a player with no shields left,
 * which nothing else asks about, or null while they have shields to choose. */
export function finishingAttack(
  state: MatchState,
  attackerId: string,
): AttackNotice | null {
  const attacker = attackerOf(state, attackerId);

  if (!attacker || state.opponent.shieldzone.length > 0) {
    return null;
  }

  return {
    side: "mine",
    attacker: { name: attacker.card.name, power: attacker.power },
    target: { name: "Opponent" },
    outcome: {
      tone: "win",
      headline: "You win the game",
      destroyed: [],
      blockers: possibleBlockers(state, attackerId),
    },
  };
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
