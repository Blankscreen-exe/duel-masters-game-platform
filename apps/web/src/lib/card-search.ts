import type { paths } from './api/schema';

/** The card search as the API takes it. */
export type CardSearch = NonNullable<paths['/cards']['get']['parameters']['query']>;

type SearchParams = Readonly<Record<string, string | string[] | undefined>>;

/** The longest search text the API accepts. */
export const SEARCH_TEXT_MAX_LENGTH = 100;
export const CARDS_PER_PAGE = 60;

const LIST_FILTERS = ['civilization', 'type', 'race', 'keyword', 'set'] as const;

/**
 * The card search in a page address (/cards?q=…&civilization=Fire&page=2). Anything that is not a
 * valid value is left out, so a hand-edited address shows a wider search rather than an error.
 */
export function parseCardSearch(params: SearchParams): CardSearch {
  const search: CardSearch = { pageSize: CARDS_PER_PAGE };

  const q = first(params['q'])?.trim().slice(0, SEARCH_TEXT_MAX_LENGTH);
  if (q) search.q = q;

  for (const filter of LIST_FILTERS) {
    const chosen = [...new Set(all(params[filter]).filter((value) => value !== ''))];
    if (chosen.length > 0) search[filter] = chosen;
  }

  const minCost = wholeNumber(first(params['minCost']), 0);
  const maxCost = wholeNumber(first(params['maxCost']), 0);
  if (minCost !== undefined) search.minCost = minCost;
  if (maxCost !== undefined) search.maxCost = maxCost;

  if (first(params['playable']) === 'true') search.playable = true;

  const page = wholeNumber(first(params['page']), 1);
  if (page !== undefined) search.page = page;

  return search;
}

/** The page address of a card search, with only the parts that narrow it. */
export function cardSearchHref(search: CardSearch, page = search.page ?? 1): string {
  const query = new URLSearchParams();

  if (search.q) query.set('q', search.q);
  for (const filter of LIST_FILTERS) {
    for (const value of search[filter] ?? []) query.append(filter, value);
  }
  if (search.minCost !== undefined) query.set('minCost', String(search.minCost));
  if (search.maxCost !== undefined) query.set('maxCost', String(search.maxCost));
  if (search.playable) query.set('playable', 'true');
  if (page > 1) query.set('page', String(page));

  const text = query.toString();
  return text === '' ? '/cards' : `/cards?${text}`;
}

/** Whether the search narrows the catalogue at all. */
export function isFiltered(search: CardSearch): boolean {
  return cardSearchHref({ ...search, page: 1 }) !== '/cards';
}

/** How many pages a search has; at least one, so an empty search still has a page to show. */
export function pageCount(total: number, pageSize: number): number {
  return Math.max(1, Math.ceil(total / pageSize));
}

function first(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

function all(value: string | string[] | undefined): string[] {
  if (value === undefined) return [];
  return Array.isArray(value) ? value : [value];
}

function wholeNumber(value: string | undefined, minimum: number): number | undefined {
  if (value === undefined || !/^\d{1,6}$/.test(value)) return undefined;

  const number = Number(value);
  return number >= minimum ? number : undefined;
}
