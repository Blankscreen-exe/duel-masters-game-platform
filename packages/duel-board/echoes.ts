import type { ScreenBox } from "./cardPositions";
import { flightDuration, type BoardChange, type Flight } from "./flights";

/** A see-through copy of a card that just acted, drawn on top of it, which
 * grows and fades so the eye is drawn to the card. */
export interface Echo {
  virtualId: string;
  imageId: string;
  box: ScreenBox;
  /** Seconds to wait first, so a card flying into place echoes once it has
   * landed rather than where it is still headed. */
  delay: number;
}

/** The echoes for the cards an update reports as having acted. A card is
 * echoed where it now is, or where it was if it has just left the screen. */
export function planEchoes(change: BoardChange, flights: Flight[]): Echo[] {
  const landing = new Map(
    flights
      .filter((flight) => flight.landsOnCard)
      .map((flight) => [flight.virtualId, flight]),
  );

  return (change.next.activatedCards ?? []).flatMap((virtualId) => {
    const now = change.after.cards.get(virtualId);
    const card = now?.visible ? now : change.before.cards.get(virtualId);

    if (!card?.visible) {
      return [];
    }

    const flight = card === now ? landing.get(virtualId) : undefined;

    return [
      {
        virtualId,
        imageId: card.imageId,
        box: card,
        delay: flight ? flight.delay + flightDuration(flight) : 0,
      },
    ];
  });
}
