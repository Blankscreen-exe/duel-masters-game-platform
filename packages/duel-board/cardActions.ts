import type { DragZone } from "./BoardZone";
import { FACE_DOWN_IMAGE_ID } from "./cardImage";
import { displayPhase } from "./turnPhases";
import {
  cardHasFlag,
  PLAYABLE_FLAG,
  SUMMONING_SICK_FLAG,
  TAP_ABILITY_FLAG,
  type CardState,
  type MatchState,
  type ShieldState,
} from "./types";

/** A card on the board together with the zone it sits in. */
export interface BoardCard {
  card: CardState | ShieldState;
  zone: DragZone;
}

export type CardActionId =
  "play" | "charge" | "attackPlayer" | "attackCreature" | "tapAbility" | "view";

export interface CardAction {
  id: CardActionId;
  label: string;
  /** Why the action cannot be taken right now, or null when it can. */
  disabledReason: string | null;
}

/** Where to look for a card, paired with the zone each list is shown in. */
const BOARD_ZONES: readonly [
  DragZone,
  (state: MatchState) => (CardState | ShieldState)[],
][] = [
  ["hand", (state) => state.me.hand],
  ["myPlayzone", (state) => state.me.playzone],
  ["myManazone", (state) => state.me.manazone],
  ["myShieldzone", (state) => state.me.shieldzone],
  ["opponentPlayzone", (state) => state.opponent.playzone],
  ["opponentManazone", (state) => state.opponent.manazone],
  ["opponentShieldzone", (state) => state.opponent.shieldzone],
];

/** Finds a card on the board by its virtualId. */
export function findBoardCard(
  state: MatchState,
  virtualId: string,
): BoardCard | null {
  for (const [zone, cardsIn] of BOARD_ZONES) {
    const card = cardsIn(state).find((c) => c.virtualId === virtualId);

    if (card) {
      return { card, zone };
    }
  }

  return null;
}

/** Whether the card's face can be seen, and so enlarged. */
export function isFaceUp(card: CardState | ShieldState): boolean {
  return !!card.uid && card.uid !== FACE_DOWN_IMAGE_ID;
}

/** What the viewer can do with a card. Playing and attacking are only offered
 * on the viewer's own turn, for cards in their hand and battle zone; any card
 * whose face can be seen can be viewed. The server has the final say on every
 * action, so these only stop clicks it would certainly refuse. */
export function cardActions(
  { card, zone }: BoardCard,
  state: MatchState,
): CardAction[] {
  const actions: CardAction[] = [];
  const canAct = state.myTurn && !state.spectator;

  if (canAct && zone === "hand") {
    actions.push(playAction(card, state), chargeAction(state));
  }

  if (canAct && zone === "myPlayzone") {
    // A creature summoned this turn has to wait for the next one, and the
    // rest wait for the battle phase.
    const sick = cardHasFlag(card.flags, SUMMONING_SICK_FLAG);
    const inBattle = displayPhase(state) === "attack";
    const attackReason = sick
      ? "Summoning sickness: it can't attack until your next turn"
      : inBattle
        ? null
        : "Go to the battle phase to attack";

    actions.push(
      {
        id: "attackPlayer",
        label: "Attack player",
        disabledReason: attackReason,
      },
      {
        id: "attackCreature",
        label: "Attack creature",
        disabledReason: attackReason,
      },
    );

    if (cardHasFlag(card.flags, TAP_ABILITY_FLAG)) {
      actions.push({
        id: "tapAbility",
        label: "Use tap ability",
        disabledReason: sick
          ? "Summoning sickness: it can't use its tap ability until your next turn"
          : inBattle
            ? null
            : "Go to the battle phase to use its tap ability",
      });
    }
  }

  if (isFaceUp(card)) {
    actions.push({ id: "view", label: "View", disabledReason: null });
  }

  return actions;
}

function playAction(
  card: CardState | ShieldState,
  state: MatchState,
): CardAction {
  const type = "type" in card ? card.type : "";
  const label =
    type === "creature" ? "Summon" : type === "spell" ? "Cast" : "Play";

  // Cards are only played in the summon phase, which ends for good once the
  // turn goes on to the battle phase.
  const phase = displayPhase(state);
  const what = type === "spell" ? "cast spells" : "summon creatures";

  if (phase === "mana") {
    return {
      id: "play",
      label,
      disabledReason: `Go to the summon phase to ${what}`,
    };
  }

  if (state.hasAttackedThisRound || phase === "attack") {
    return {
      id: "play",
      label,
      disabledReason: `You can't ${what} in the battle phase`,
    };
  }

  return {
    id: "play",
    label,
    disabledReason: cardHasFlag(card.flags, PLAYABLE_FLAG)
      ? null
      : "You don't have enough mana to play this card",
  };
}

function chargeAction(state: MatchState): CardAction {
  let disabledReason: string | null = null;

  if (state.hasAddedManaThisRound) {
    disabledReason = "You've already added mana to your manazone this turn";
  } else if (displayPhase(state) !== "mana" || !state.canChargeManaThisRound) {
    disabledReason = "Mana can only be added in the mana phase";
  }

  return { id: "charge", label: "Add to mana", disabledReason };
}
