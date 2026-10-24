import type {
  BoardMeasurements,
  MeasuredCard,
  ScreenBox,
} from "../cardPositions";
import type { CardState, MatchState, PlayerState, ShieldState } from "../types";

export function card(
  virtualId: string,
  flags = 0,
  type: CardState["type"] = "creature",
): CardState {
  return {
    virtualId,
    uid: `image-${virtualId}`,
    name: `Card ${virtualId}`,
    civilization: "fire",
    flags,
    type,
  };
}

export function shield(virtualId: string): ShieldState {
  return { virtualId, uid: "", flags: 0 };
}

function player(zones: Partial<PlayerState>): PlayerState {
  return {
    username: "Player",
    color: "#fff",
    deck: 30,
    handCount: zones.hand?.length ?? 0,
    hand: [],
    shieldzone: [],
    shieldMap: {},
    powers: {},
    statuses: {},
    manazone: [],
    graveyard: [],
    playzone: [],
    ...zones,
  };
}

/** A match state as the server sends it to a player, on their turn in the
 * charge step unless overridden. The opponent's hand is only ever a count, so
 * give it with handCount rather than hand. */
export function matchState(
  me: Partial<PlayerState> = {},
  opponent: Partial<PlayerState> = {},
  overrides: Partial<Omit<MatchState, "me" | "opponent">> = {},
): MatchState {
  return {
    myTurn: true,
    hasAddedManaThisRound: false,
    hasAttackedThisRound: false,
    canChargeManaThisRound: true,
    me: player(me),
    opponent: player({ ...opponent, hand: [] }),
    spectator: false,
    step: "charge",
    brokenShields: [],
    activatedCards: [],
    turnManaCharged: false,
    turn: 1,
    ...overrides,
  };
}

/** A card 60 by 84 pixels, upright and visible, centred at x, y. */
export function box(
  x: number,
  y: number,
  overrides: Partial<ScreenBox> = {},
): ScreenBox {
  return {
    x,
    y,
    width: 60,
    height: 84,
    rotation: 0,
    visible: true,
    ...overrides,
  };
}

export function measuredCard(
  virtualId: string,
  x: number,
  y: number,
  { muted = false, ...overrides }: Partial<Omit<MeasuredCard, "imageId">> = {},
): [string, MeasuredCard] {
  return [
    virtualId,
    { ...box(x, y, overrides), imageId: `image-${virtualId}`, muted },
  ];
}

export function measurements(
  cards: [string, MeasuredCard][] = [],
  anchors: [string, ScreenBox][] = [],
): BoardMeasurements {
  return { cards: new Map(cards), anchors: new Map(anchors) };
}
