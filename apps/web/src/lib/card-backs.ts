/** The card backs the site can show, by picture in `public/`. */
export const CARD_BACKS = {
  /** The back of the printed Duel Masters cards. */
  original: '/card-back-original.webp',
  /** The site's own drawing, in the Battle Zone colours. */
  battleZone: '/card-back.svg',
} as const;

/** The card back in use. Name another of `CARD_BACKS` here to switch the whole game to it. */
export const CARD_BACK = CARD_BACKS.original;
