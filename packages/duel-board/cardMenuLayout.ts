import { CARD_ASPECT_RATIO } from "./cardImage";
import type { ScreenBox } from "./cardPositions";

export interface Viewport {
  width: number;
  height: number;
}

/** How much bigger the lifted card is than the card in its row. */
const LIFT_SCALE = 1.5;

/** A small card on a phone, grown by LIFT_SCALE alone, would leave no room
 * for its buttons, so the lifted card is never shorter than this. */
const MIN_LIFTED_HEIGHT = 260;

/** The share of the screen's height the lifted card may take at most. */
const MAX_LIFTED_SHARE = 0.85;

/** Space kept between the card and the edges of the screen. */
const SCREEN_MARGIN = 12;

/** Where the lifted card of the menu over the card is drawn: centred on the
 * card it came from, as far as the screen allows, or in the middle of the
 * screen when that is unknown. */
export function liftedBox(
  origin: ScreenBox | null,
  viewport: Viewport,
): ScreenBox {
  const height = Math.min(
    viewport.height * MAX_LIFTED_SHARE,
    Math.max(MIN_LIFTED_HEIGHT, (origin?.height ?? 0) * LIFT_SCALE),
  );
  const width = height * CARD_ASPECT_RATIO;

  return {
    x: clamp(
      origin?.x ?? viewport.width / 2,
      SCREEN_MARGIN + width / 2,
      viewport.width - SCREEN_MARGIN - width / 2,
    ),
    y: clamp(
      origin?.y ?? viewport.height / 2,
      SCREEN_MARGIN + height / 2,
      viewport.height - SCREEN_MARGIN - height / 2,
    ),
    width,
    height,
    rotation: 0,
    visible: true,
  };
}

/** What a phone's action sheet holds, which decides how tall it is. */
export interface SheetContents {
  actions: number;
  /** Actions that are disabled, each with a line saying why. */
  reasons: number;
}

export interface SheetLayout {
  /** Where the card is shown, upright and as large as fits. */
  card: ScreenBox;
  /** The edge the sheet slides in from: the bottom of an upright phone, or
   * the right of one on its side, where there is no height to spare. */
  edge: "bottom" | "right";
  /** The sheet's height when at the bottom, or its width when on the right. */
  size: number;
}

const SHEET_PADDING = 16;
const SHEET_TITLE = 28;
const SHEET_GRABBER = 20;
const SHEET_ACTION = 56;
const SHEET_REASON = 18;
const MAX_SHEET_HEIGHT_SHARE = 0.6;
const MIN_SIDE_SHEET_WIDTH = 240;
const MAX_SIDE_SHEET_WIDTH = 340;
const SIDE_SHEET_SHARE = 0.42;
/** A card larger than this is no easier to read, just emptier. */
const MAX_SHEET_CARD_HEIGHT = 560;

/** Lays out a phone's card menu: the card shown large where the sheet leaves
 * room, and the sheet along the edge a thumb reaches. */
export function sheetLayout(
  viewport: Viewport,
  contents: SheetContents,
): SheetLayout {
  if (viewport.height >= viewport.width) {
    const contentHeight =
      SHEET_PADDING * 2 +
      SHEET_GRABBER +
      SHEET_TITLE +
      contents.actions * SHEET_ACTION +
      contents.reasons * SHEET_REASON;
    const size = Math.min(
      contentHeight,
      viewport.height * MAX_SHEET_HEIGHT_SHARE,
    );
    const areaHeight = viewport.height - size;

    return {
      card: cardIn(
        viewport.width,
        areaHeight,
        viewport.width / 2,
        areaHeight / 2,
      ),
      edge: "bottom",
      size,
    };
  }

  const size = clamp(
    viewport.width * SIDE_SHEET_SHARE,
    MIN_SIDE_SHEET_WIDTH,
    MAX_SIDE_SHEET_WIDTH,
  );
  const areaWidth = viewport.width - size;

  return {
    card: cardIn(
      areaWidth,
      viewport.height,
      areaWidth / 2,
      viewport.height / 2,
    ),
    edge: "right",
    size,
  };
}

/** The largest upright card that fits in an area, centred on a point. */
function cardIn(
  areaWidth: number,
  areaHeight: number,
  x: number,
  y: number,
): ScreenBox {
  const height = Math.max(
    0,
    Math.min(
      areaHeight - SCREEN_MARGIN * 2,
      (areaWidth - SCREEN_MARGIN * 2) / CARD_ASPECT_RATIO,
      MAX_SHEET_CARD_HEIGHT,
    ),
  );

  return {
    x,
    y,
    width: height * CARD_ASPECT_RATIO,
    height,
    rotation: 0,
    visible: true,
  };
}

/** The transform that makes a card drawn at `shown` look like the card at
 * `origin`, so it can animate from one to the other. */
export function restingTransform(origin: ScreenBox, shown: ScreenBox) {
  return {
    x: origin.x - shown.x,
    y: origin.y - shown.y,
    scale: origin.height / shown.height,
    rotate: origin.rotation,
    opacity: 1,
  };
}

function clamp(value: number, min: number, max: number): number {
  return max < min ? (min + max) / 2 : Math.min(max, Math.max(min, value));
}
