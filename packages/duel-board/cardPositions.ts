/** Marks an element that scrolls or clips the cards inside it. A card scrolled
 * out of such an element is still laid out, just not visible. */
export const CLIPS_CARDS_ATTRIBUTE = "data-clips-cards";

/** Spread onto an element to mark it with CLIPS_CARDS_ATTRIBUTE. */
export const CLIPS_CARDS_PROPS = { [CLIPS_CARDS_ATTRIBUTE]: "" };

/** Where something is drawn on screen, in viewport pixels. */
export interface ScreenBox {
  /** Centre point. Unlike the corners, it stays put when the box is rotated. */
  x: number;
  y: number;
  /** Size before any rotation, so a tapped card still reports its upright
   * width and height. */
  width: number;
  height: number;
  /** Clockwise rotation in degrees. */
  rotation: number;
  /** False when the centre is scrolled or clipped out of view. */
  visible: boolean;
}

export interface MeasuredCard extends ScreenBox {
  imageId: string;
  /** Whether the card is shown greyed out, as used up mana is. */
  muted: boolean;
}

export interface TrackedCard {
  element: HTMLElement;
  imageId: string;
  rotation: number;
  muted: boolean;
}

/** The latest on-screen measurements of the board, keyed by card virtualId and
 * by anchor id. Anchors are places a card can come from or go to without a
 * card element of its own there, such as a deck pile or a zone's row. */
export interface BoardMeasurements {
  cards: Map<string, MeasuredCard>;
  anchors: Map<string, ScreenBox>;
}

/** Keeps the elements of the cards and anchors on the board so their positions
 * can be read just before and just after the board changes. */
export class CardPositionRegistry {
  private readonly cards = new Map<string, TrackedCard>();
  private readonly anchors = new Map<string, HTMLElement>();

  trackCard(virtualId: string, card: TrackedCard): void {
    this.cards.set(virtualId, card);
  }

  /** Stops tracking a card, unless another element has already taken over the
   * virtualId, as happens when a card mounts in its new zone first. */
  untrackCard(virtualId: string, element: HTMLElement): void {
    if (this.cards.get(virtualId)?.element === element) {
      this.cards.delete(virtualId);
    }
  }

  trackAnchor(anchorId: string, element: HTMLElement): void {
    this.anchors.set(anchorId, element);
  }

  untrackAnchor(anchorId: string, element: HTMLElement): void {
    if (this.anchors.get(anchorId) === element) {
      this.anchors.delete(anchorId);
    }
  }

  /** Reads the current layout. Elements that are not rendered, such as those
   * of a duel hidden with display: none, are left out. */
  measure(): BoardMeasurements {
    const cards = new Map<string, MeasuredCard>();
    const anchors = new Map<string, ScreenBox>();

    for (const [virtualId, card] of this.cards) {
      const measured = measureTrackedCard(card);

      if (measured) {
        cards.set(virtualId, measured);
      }
    }

    for (const [anchorId, element] of this.anchors) {
      const box = measureElement(element, 0);

      if (box) {
        anchors.set(anchorId, box);
      }
    }

    return { cards, anchors };
  }

  /** Reads where one card is drawn right now, or null when it is not on the
   * board or not rendered. */
  measureCard(virtualId: string): MeasuredCard | null {
    const card = this.cards.get(virtualId);
    return card ? measureTrackedCard(card) : null;
  }
}

function measureTrackedCard(card: TrackedCard): MeasuredCard | null {
  const box = measureElement(card.element, card.rotation);
  return box ? { ...box, imageId: card.imageId, muted: card.muted } : null;
}

function measureElement(
  element: HTMLElement,
  rotation: number,
): ScreenBox | null {
  // The offset size is the layout size, unaffected by rotation, while the
  // bounding rectangle grows to enclose a rotated element but keeps its centre.
  const width = element.offsetWidth;
  const height = element.offsetHeight;

  if (width === 0 || height === 0) {
    return null;
  }

  const rect = element.getBoundingClientRect();
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;
  const clip = element.parentElement?.closest(`[${CLIPS_CARDS_ATTRIBUTE}]`);
  const clipRect = clip?.getBoundingClientRect();
  const visible =
    !clipRect ||
    (x >= clipRect.left &&
      x <= clipRect.right &&
      y >= clipRect.top &&
      y <= clipRect.bottom);

  return { x, y, width, height, rotation, visible };
}
