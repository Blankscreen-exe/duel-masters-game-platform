import {
  cardHasFlag,
  PLAYABLE_FLAG,
  type MatchState,
  type TurnStep,
} from "./types";

export type TurnPhase = "draw" | "mana" | "summon" | "attack" | "end";

export interface TurnPhaseInfo {
  phase: TurnPhase;
  label: string;
  description: string;
}

/** The phases shown to players, in turn order. The server's steps that pass
 * without waiting for anyone are folded into the phase around them. */
export const TURN_PHASES: readonly TurnPhaseInfo[] = [
  {
    phase: "draw",
    label: "Draw",
    description: "Cards untap and the player draws a card.",
  },
  {
    phase: "mana",
    label: "Mana",
    description:
      "The player may put one card from their hand into their mana zone.",
  },
  {
    phase: "summon",
    label: "Summon",
    description:
      "The player summons creatures and casts spells. Mana can no longer be added this turn.",
  },
  {
    phase: "attack",
    label: "Battle",
    description:
      "The player attacks and uses tap abilities. Nothing more can be summoned, cast or put into mana this turn.",
  },
  {
    phase: "end",
    label: "End",
    description: "End of turn effects resolve and the turn passes.",
  },
];

const PHASE_OF_STEP: Record<TurnStep, TurnPhase> = {
  begin_turn: "draw",
  untap: "draw",
  start_of_turn: "draw",
  draw: "draw",
  charge: "mana",
  main: "summon",
  attack: "attack",
  end: "end",
  end_of_turn: "end",
};

/** The phase a server step belongs to, or null before the first turn and for
 * a step this client does not know. */
export function phaseOfStep(step: MatchState["step"]): TurnPhase | null {
  return (step && PHASE_OF_STEP[step]) || null;
}

/** The phase to show. The mana, summon and battle phases each last until the
 * player moves the turn on, so this is the server's step as it stands. */
export function displayPhase(state: MatchState): TurnPhase | null {
  return phaseOfStep(state.step);
}

/** A step the player moves the turn on to, as the server names it. */
export type PhaseStep = "main" | "attack";

/** The phase button beside End turn: the phase it moves the turn on to. */
export interface PhaseAdvance {
  step: PhaseStep;
  label: string;
  /** For the narrow controls of a phone. */
  shortLabel: string;
}

/** What the phase button offers right now, or null when there is no phase to
 * move on to: before or after the phases a player acts in, in the battle
 * phase, which only End turn leaves, and when it is not the viewer's turn. The
 * turn never goes back, so each phase can only be left forwards. */
export function phaseAdvance(state: MatchState): PhaseAdvance | null {
  if (state.spectator || !state.myTurn) {
    return null;
  }

  switch (displayPhase(state)) {
    case "mana":
      return {
        step: "main",
        label: "Go to summon phase",
        shortLabel: "Summon phase",
      };
    case "summon":
      return { step: "attack", label: "Battle phase", shortLabel: "Battle phase" };
    default:
      return null;
  }
}

/** Whether a new turn began between two states the board showed. Several
 * states can arrive before the board is drawn again, so the start of a turn is
 * often never seen on its own: a new turn began if the turn changed hands, or
 * if its phase went back to an earlier one, as every turn starts over from the
 * draw. The second also tells a spectator, who is never told whose turn it is,
 * and covers a player taking an extra turn. */
export function startsNewTurn(previous: MatchState, next: MatchState): boolean {
  const before = displayPhase(previous);
  const after = displayPhase(next);

  if (!before || !after) {
    return false;
  }

  if (!next.spectator && previous.myTurn !== next.myTurn) {
    return true;
  }

  return phaseIndex(after) < phaseIndex(before);
}

function phaseIndex(phase: TurnPhase): number {
  return TURN_PHASES.findIndex((info) => info.phase === phase);
}

/** The number of the turn under way, or null before the first turn has begun,
 * as from a server too old to count turns. */
export function turnNumber(state: MatchState): number | null {
  return state.turn > 0 ? state.turn : null;
}

/** A short line telling the viewer what they can do right now, or null when
 * there is nothing to say. */
export function turnHint(state: MatchState): string | null {
  if (state.spectator) {
    return null;
  }

  if (!state.myTurn) {
    return "Opponent's turn";
  }

  switch (displayPhase(state)) {
    case "mana":
      return state.hasAddedManaThisRound || !state.canChargeManaThisRound
        ? "Go to the summon phase when you're ready"
        : "Add a card to your mana zone, or go on to the summon phase";
    case "summon":
      return "Summon creatures and cast spells, then go to the battle phase";
    case "attack":
      return "Attack and use tap abilities, then end your turn";
    default:
      return null;
  }
}

/** What going on to the battle phase right now would give up. */
export interface AttackForfeit {
  /** Cards in hand the player could still afford to play. */
  playableCards: number;
  canAddMana: boolean;
}

/** Nothing can be summoned, cast or put into mana in the battle phase, so a
 * player still able to is asked first. Returns null when the player has nothing
 * left to lose by it, as once in the battle phase. */
export function attackForfeit(state: MatchState): AttackForfeit | null {
  if (state.spectator || !state.myTurn || state.hasAttackedThisRound) {
    return null;
  }

  const playableCards = state.me.hand.filter((card) =>
    cardHasFlag(card.flags, PLAYABLE_FLAG),
  ).length;
  const canAddMana =
    state.canChargeManaThisRound &&
    !state.hasAddedManaThisRound &&
    state.me.hand.length > 0;

  return playableCards > 0 || canAddMana ? { playableCards, canAddMana } : null;
}

export function describeAttackForfeit({
  playableCards,
  canAddMana,
}: AttackForfeit): string {
  const stillPossible: string[] = [];

  if (playableCards > 0) {
    stillPossible.push(
      playableCards === 1
        ? "play 1 card from your hand"
        : `play ${playableCards} cards from your hand`,
    );
  }

  if (canAddMana) {
    stillPossible.push("add mana");
  }

  return `You can still ${stillPossible.join(" and ")}. Once you go to the battle phase, you can't summon, cast spells or add mana for the rest of this turn.`;
}

/** A title announcing a new turn or phase across the middle of the board. */
export interface PhaseBanner {
  kind: "turn" | "phase";
  /** The phase a phase title announces. */
  phase?: TurnPhase;
  title: string;
  /** Whose turn it is, which colours the title. */
  tone: "mine" | "theirs" | "neutral";
}

/** The titles to announce between two states: a new turn, then the phase it
 * is in, or just the phase when the turn moved on within itself. */
export function phaseBanners(
  previous: MatchState | null,
  next: MatchState,
): PhaseBanner[] {
  const phase = displayPhase(next);

  // The first state of a connection describes a turn already under way.
  if (!previous || !phase) {
    return [];
  }

  const before = displayPhase(previous);
  const tone = next.spectator ? "neutral" : next.myTurn ? "mine" : "theirs";
  const phaseBanner = (shown: TurnPhase): PhaseBanner => ({
    kind: "phase",
    phase: shown,
    title: `${TURN_PHASES.find((info) => info.phase === shown)?.label} phase`,
    tone,
  });

  if (startsNewTurn(previous, next)) {
    // Every turn is told from its start, though the board may only have been
    // shown it once it was already past the draw.
    return [
      {
        kind: "turn",
        title: next.spectator
          ? "Next turn"
          : next.myTurn
            ? "Your turn"
            : "Opponent's turn",
        tone,
      },
      phaseBanner("draw"),
      ...(phase === "draw" ? [] : [phaseBanner(phase)]),
    ];
  }

  return phase !== before ? [phaseBanner(phase)] : [];
}

/** Adds titles to the queue of those waiting to be shown, the first of which
 * is on screen. A new turn replaces everything, the title on screen included,
 * so the turn is announced the moment it starts. Within a turn only the latest
 * phase is worth announcing, so a phase still waiting is dropped for a newer
 * one, except the draw, which is how every turn is told to begin. */
export function queueBanners<T extends PhaseBanner>(
  queue: readonly T[],
  incoming: readonly T[],
): T[] {
  if (incoming.length === 0) {
    return [...queue];
  }

  if (incoming.some((banner) => banner.kind === "turn")) {
    return [...incoming];
  }

  const [showing, ...waiting] = queue;
  const newerPhase = incoming.some((next) => next.kind === "phase");
  const kept = waiting.filter(
    (banner) =>
      banner.kind !== "phase" || banner.phase === "draw" || !newerPhase,
  );

  return [...(showing ? [showing] : []), ...kept, ...incoming];
}

/** How long a title takes to sweep in from the left, and out to the right. */
export const BANNER_ENTER_SECONDS = 0.35;
export const BANNER_EXIT_SECONDS = 0.35;

/** How long a title holds in the middle. The start of a turn passes by itself,
 * so its titles are brief and the turn reaches the phases a player acts in
 * sooner. */
export function bannerHoldSeconds(banner: PhaseBanner): number {
  if (banner.kind === "turn") {
    return 0.8;
  }

  return banner.phase === "draw" ? 0.6 : 1;
}

/** How long a title is on screen, from coming in to going out. */
export function bannerSeconds(banner: PhaseBanner): number {
  return BANNER_ENTER_SECONDS + bannerHoldSeconds(banner) + BANNER_EXIT_SECONDS;
}
