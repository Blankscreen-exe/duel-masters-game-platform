import { describe, expect, it } from 'vitest';
import type { Card, CardPage } from './api/client';
import { hasMoreCards, withCardPage } from './card-pages';

function pageOf(page: number, ids: string[], total = 4): CardPage {
  return {
    cards: ids.map((id) => ({ id }) as unknown as Card),
    total,
    page,
    pageSize: 2,
  };
}

const ids = (cards: readonly Card[]) => cards.map(({ id }) => id);

describe('withCardPage', () => {
  it('starts from the first page', () => {
    const held = withCardPage(null, 'fire', pageOf(1, ['a', 'b']));

    expect(ids(held.cards)).toEqual(['a', 'b']);
    expect(held).toMatchObject({ key: 'fire', total: 4, page: 1 });
  });

  it('adds the next page to the end', () => {
    const first = withCardPage(null, 'fire', pageOf(1, ['a', 'b']));
    const held = withCardPage(first, 'fire', pageOf(2, ['c', 'd']));

    expect(ids(held.cards)).toEqual(['a', 'b', 'c', 'd']);
    expect(held.page).toBe(2);
  });

  it('starts afresh for another search', () => {
    const first = withCardPage(null, 'fire', pageOf(1, ['a', 'b']));

    expect(ids(withCardPage(first, 'water', pageOf(1, ['x'])).cards)).toEqual(['x']);
  });

  it('starts afresh when the first page arrives again', () => {
    const first = withCardPage(null, 'fire', pageOf(1, ['a', 'b']));
    const second = withCardPage(first, 'fire', pageOf(2, ['c', 'd']));

    expect(ids(withCardPage(second, 'fire', pageOf(1, ['a', 'b'])).cards)).toEqual(['a', 'b']);
  });

  it('ignores a page out of turn', () => {
    const first = withCardPage(null, 'fire', pageOf(1, ['a', 'b']));

    expect(withCardPage(first, 'fire', pageOf(3, ['e']))).toBe(first);
  });

  it('lists a card once when it moved between pages', () => {
    const first = withCardPage(null, 'fire', pageOf(1, ['a', 'b']));

    expect(ids(withCardPage(first, 'fire', pageOf(2, ['b', 'c'])).cards)).toEqual(['a', 'b', 'c']);
  });
});

describe('hasMoreCards', () => {
  it('is true until every card is held', () => {
    const first = withCardPage(null, 'fire', pageOf(1, ['a', 'b']));
    const second = withCardPage(first, 'fire', pageOf(2, ['c', 'd']));

    expect(hasMoreCards(first)).toBe(true);
    expect(hasMoreCards(second)).toBe(false);
  });

  it('is false for a search with no cards', () => {
    expect(hasMoreCards(withCardPage(null, 'none', pageOf(1, [], 0)))).toBe(false);
  });
});
