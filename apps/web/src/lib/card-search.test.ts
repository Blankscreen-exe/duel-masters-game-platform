import { describe, expect, it } from 'vitest';
import {
  CARDS_PER_PAGE,
  cardSearchHref,
  isFiltered,
  pageCount,
  parseCardSearch,
  SEARCH_TEXT_MAX_LENGTH,
} from './card-search';

describe('parseCardSearch', () => {
  it('reads every filter', () => {
    expect(
      parseCardSearch({
        q: '  Bolshack ',
        civilization: ['Fire', 'Nature'],
        type: 'Creature',
        race: 'Armored Dragon',
        keyword: 'Double breaker',
        set: 'DM-01',
        minCost: '3',
        maxCost: '7',
        playable: 'true',
        page: '2',
      }),
    ).toEqual({
      pageSize: CARDS_PER_PAGE,
      q: 'Bolshack',
      civilization: ['Fire', 'Nature'],
      type: ['Creature'],
      race: ['Armored Dragon'],
      keyword: ['Double breaker'],
      set: ['DM-01'],
      minCost: 3,
      maxCost: 7,
      playable: true,
      page: 2,
    });
  });

  it('is the whole catalogue without filters', () => {
    expect(parseCardSearch({})).toEqual({ pageSize: CARDS_PER_PAGE });
  });

  it('leaves out values that are not valid', () => {
    expect(
      parseCardSearch({
        q: '   ',
        civilization: ['', 'Fire', 'Fire'],
        minCost: '-1',
        maxCost: 'seven',
        playable: 'yes',
        page: '0',
      }),
    ).toEqual({ pageSize: CARDS_PER_PAGE, civilization: ['Fire'] });
    expect(parseCardSearch({ page: '1.5' })).toEqual({ pageSize: CARDS_PER_PAGE });
    expect(parseCardSearch({ page: '99999999999' })).toEqual({ pageSize: CARDS_PER_PAGE });
  });

  it('shortens search text to what the API accepts', () => {
    expect(parseCardSearch({ q: 'a'.repeat(150) }).q).toHaveLength(SEARCH_TEXT_MAX_LENGTH);
  });
});

describe('cardSearchHref', () => {
  it('gives back the address the search came from', () => {
    const href =
      '/cards?q=Bolshack&civilization=Fire&civilization=Nature&minCost=3&playable=true&page=2';
    const search = parseCardSearch(Object.fromEntries(groupParams(href)));

    expect(cardSearchHref(search)).toBe(href);
  });

  it('moves to another page of the same search', () => {
    expect(cardSearchHref(parseCardSearch({ q: 'dragon', page: '3' }), 1)).toBe('/cards?q=dragon');
    expect(cardSearchHref(parseCardSearch({ q: 'dragon' }), 2)).toBe('/cards?q=dragon&page=2');
  });

  it('is the plain card list without filters', () => {
    expect(cardSearchHref(parseCardSearch({}))).toBe('/cards');
  });
});

describe('isFiltered', () => {
  it('ignores the page', () => {
    expect(isFiltered(parseCardSearch({ page: '4' }))).toBe(false);
    expect(isFiltered(parseCardSearch({ maxCost: '0' }))).toBe(true);
  });
});

describe('pageCount', () => {
  it('rounds up, and has a page even without cards', () => {
    expect(pageCount(0, 60)).toBe(1);
    expect(pageCount(60, 60)).toBe(1);
    expect(pageCount(61, 60)).toBe(2);
  });
});

/** Query parameters as Next.js gives them to a page: repeated names become a list. */
function groupParams(href: string): Map<string, string | string[]> {
  const grouped = new Map<string, string | string[]>();
  for (const [name, value] of new URL(href, 'https://site.invalid').searchParams) {
    const existing = grouped.get(name);
    grouped.set(name, existing === undefined ? value : [existing, value].flat());
  }
  return grouped;
}
