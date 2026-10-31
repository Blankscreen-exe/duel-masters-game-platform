import type { Card, CardPage } from './api/client';

/** The cards of one search, as far down its pages as have been asked for. */
export interface CardPages {
  /** Which search these cards answer. */
  readonly key: string;
  readonly cards: readonly Card[];
  readonly total: number;
  /** The last page held. */
  readonly page: number;
}

/**
 * What is held once a page arrives: a first page, or a page of another search, starts afresh; the
 * next page of the same search joins the end; any other page changes nothing. A card that a change
 * in the catalogue moved between pages is listed once.
 */
export function withCardPage(held: CardPages | null, key: string, arrived: CardPage): CardPages {
  if (held?.key !== key || arrived.page === 1) {
    return { key, cards: arrived.cards, total: arrived.total, page: arrived.page };
  }
  if (arrived.page !== held.page + 1) return held;

  const known = new Set(held.cards.map(({ id }) => id));
  return {
    key,
    cards: [...held.cards, ...arrived.cards.filter(({ id }) => !known.has(id))],
    total: arrived.total,
    page: arrived.page,
  };
}

/** Whether the search has cards beyond those held. */
export function hasMoreCards({ cards, total }: CardPages): boolean {
  return cards.length < total;
}
