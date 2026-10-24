import type { BoardSide, CardMove } from "./boardDiff";
import type {
  BoardMeasurements,
  MeasuredCard,
  ScreenBox,
} from "./cardPositions";
import { startsNewTurn } from "./turnPhases";
import type { MatchState } from "./types";

/** A copy of a card drawn above the board, travelling from where the card was
 * to where it is now. The board's zones clip their cards, so the card itself
 * cannot be seen crossing from one zone to another. */
export interface Flight {
  virtualId: string;
  imageId: string;
  from: ScreenBox;
  to: ScreenBox;
  /** Whether the card can be seen at the end of the flight, in which case it
   * stays hidden until the copy lands on it. */
  landsOnCard: boolean;
  /** Fade the copy in at the start, or out at the end, when that end of the
   * journey has no place on screen to fly from or to. */
  fade: "in" | "out" | null;
  /** Whether the card is greyed out where the copy starts and where it ends. */
  muted: { from: boolean; to: boolean };
  /** Seconds the copy waits where it starts before it flies. */
  delay: number;
}

/** Everything known about one update of the board. */
export interface BoardChange {
  previous: MatchState;
  next: MatchState;
  moves: CardMove[];
  before: BoardMeasurements;
  after: BoardMeasurements;
}

/** How small a fading copy is when it is invisible, relative to its size. */
export const FADED_SCALE = 0.85;

/** Changes smaller than this, in pixels or degrees, are rounding noise. */
const MOVE_TOLERANCE = 0.5;

/** Names a place a card can come from or go to without its own element being
 * there to measure: a zone's row, a deck or graveyard pile, or the counter of
 * a hand the viewer cannot see. Zones use their names from the match state. */
export function anchorId(side: BoardSide, place: string): string {
  return `${side}.${place}`;
}

/** Works out which cards to animate after the board changed. A card that only
 * slid along its row travels too, so the row closes up smoothly. */
export function planFlights(change: BoardChange): Flight[] {
  const { before, after } = change;
  const moves = new Map(change.moves.map((move) => [move.virtualId, move]));
  const virtualIds = new Set([...before.cards.keys(), ...after.cards.keys()]);
  const flights: Flight[] = [];

  for (const virtualId of virtualIds) {
    const measuredBefore = before.cards.get(virtualId);
    const measuredAfter = after.cards.get(virtualId);
    const start = measuredBefore?.visible ? measuredBefore : null;
    const end = measuredAfter?.visible ? measuredAfter : null;

    if (start && end) {
      if (hasMoved(start, end)) {
        flights.push(
          flight(virtualId, end.imageId, start, end, true, start, end),
        );
      }
      continue;
    }

    // A card that stayed in its zone and only came into or out of view, by
    // scrolling or by being covered on a pile, did not travel anywhere.
    const move = moves.get(virtualId);
    // A card scrolled out of view at one end is still laid out there, which
    // tells how that zone turns its cards.
    const startShape = measuredBefore ?? measuredAfter;
    const endShape = measuredAfter ?? measuredBefore;

    if (!move || !startShape || !endShape || (!start && !end)) {
      continue;
    }

    const imageId = endShape.imageId;
    const origin =
      start ?? anchoredBox(findAnchor(change, move, "from"), startShape);
    const destination =
      end ?? anchoredBox(findAnchor(change, move, "to"), endShape);

    if (origin && destination) {
      flights.push(
        flight(
          virtualId,
          imageId,
          origin,
          destination,
          !!end,
          startShape,
          endShape,
        ),
      );
    } else if (end) {
      flights.push({
        ...flight(virtualId, imageId, end, end, true, end, end),
        fade: "in",
      });
    } else if (start) {
      flights.push({
        ...flight(virtualId, imageId, start, start, false, start, start),
        fade: "out",
      });
    }
  }

  return flights;
}

/** Holds back the cards drawn into a visible hand as a turn starts until the
 * given time, so the draw is seen while it is announced rather than before. A
 * card drawn by an effect later in the turn is not held back. */
export function holdTurnDraws(
  flights: Flight[],
  change: BoardChange,
  seconds: number,
): Flight[] {
  if (!startsNewTurn(change.previous, change.next)) {
    return flights;
  }

  const drawn = new Set(
    change.moves
      .filter((move) => move.from === null && move.to?.zone === "hand")
      .map((move) => move.virtualId),
  );

  return flights.map((flight) =>
    drawn.has(flight.virtualId) ? { ...flight, delay: seconds } : flight,
  );
}

const SHORTEST_FLIGHT_SECONDS = 0.3;
const LONGEST_FLIGHT_SECONDS = 0.6;
const FLIGHT_PIXELS_PER_SECOND = 2500;
const FADE_SECONDS = 0.25;

/** How long a flight takes. Longer journeys take a little longer, so the eye
 * can follow a card across the board while a short slide stays snappy. */
export function flightDuration(flight: Flight): number {
  if (flight.fade) {
    return FADE_SECONDS;
  }

  const distance = Math.hypot(
    flight.to.x - flight.from.x,
    flight.to.y - flight.from.y,
  );

  return Math.min(
    LONGEST_FLIGHT_SECONDS,
    SHORTEST_FLIGHT_SECONDS + distance / FLIGHT_PIXELS_PER_SECOND,
  );
}

/** The signed number of degrees to turn from one rotation to another the short
 * way round, between -180 and 180. */
export function rotationDelta(from: number, to: number): number {
  return ((((to - from) % 360) + 540) % 360) - 180;
}

function flight(
  virtualId: string,
  imageId: string,
  from: ScreenBox,
  to: ScreenBox,
  landsOnCard: boolean,
  startCard: MeasuredCard,
  endCard: MeasuredCard,
): Flight {
  return {
    virtualId,
    imageId,
    from: screenBox(from),
    to: screenBox(to),
    landsOnCard,
    fade: null,
    muted: { from: startCard.muted, to: endCard.muted },
    delay: 0,
  };
}

/** Copies just the box out of a measured card. */
function screenBox({
  x,
  y,
  width,
  height,
  rotation,
  visible,
}: ScreenBox): ScreenBox {
  return { x, y, width, height, rotation, visible };
}

function hasMoved(a: MeasuredCard, b: MeasuredCard): boolean {
  return (
    a.muted !== b.muted ||
    Math.abs(a.x - b.x) > MOVE_TOLERANCE ||
    Math.abs(a.y - b.y) > MOVE_TOLERANCE ||
    Math.abs(a.height - b.height) > MOVE_TOLERANCE ||
    Math.abs(rotationDelta(a.rotation, b.rotation)) > MOVE_TOLERANCE
  );
}

/** A card-shaped box centred on an anchor, as tall as the anchor and turned
 * like the card at the other end of the journey. */
function anchoredBox(
  anchor: ScreenBox | null,
  card: MeasuredCard,
): ScreenBox | null {
  if (!anchor) {
    return null;
  }

  return {
    x: anchor.x,
    y: anchor.y,
    width: (anchor.height * card.width) / card.height,
    height: anchor.height,
    rotation: card.rotation,
    visible: true,
  };
}

/** The anchor for one end of a card's journey, looked up on the screen as it
 * was at that end. */
function findAnchor(
  change: BoardChange,
  move: CardMove,
  end: "from" | "to",
): ScreenBox | null {
  const anchors = end === "from" ? change.before.anchors : change.after.anchors;
  const location = move[end];
  const id = location
    ? anchorId(location.side, location.zone)
    : hiddenAnchorId(change, move, end);
  const anchor = id ? anchors.get(id) : undefined;

  return anchor?.visible ? anchor : null;
}

/** Where a card that came from, or went, out of sight was: its owner's deck,
 * unless the owner's hand is hidden from the viewer and shrank or grew by the
 * card. */
function hiddenAnchorId(
  change: BoardChange,
  move: CardMove,
  end: "from" | "to",
): string | null {
  const side = (move.from ?? move.to)?.side;

  if (!side) {
    return null;
  }

  const handHidden = side === "opponent" || change.next.spectator;
  const handGrowth =
    change.next[side].handCount - change.previous[side].handCount;
  const viaHand = end === "from" ? handGrowth < 0 : handGrowth > 0;

  return anchorId(side, handHidden && viaHand ? "hand" : "deck");
}
