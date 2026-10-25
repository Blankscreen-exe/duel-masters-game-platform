/** A page address's query, as Next.js gives it to a page. */
export type SearchParams = Readonly<Record<string, string | string[] | undefined>>;

/** The first value of a query parameter given once or more. */
export function firstParam(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

/** Every value of a query parameter. */
export function allParams(value: string | string[] | undefined): string[] {
  if (value === undefined) return [];
  return Array.isArray(value) ? value : [value];
}

/**
 * A query parameter's whole number, if it is one of at least `minimum`, such as a page. Anything
 * else is left out, so a hand-edited address falls back to the default rather than an error.
 */
export function wholeNumberParam(
  value: string | string[] | undefined,
  minimum: number,
): number | undefined {
  const text = firstParam(value);
  if (text === undefined || !/^\d{1,6}$/.test(text)) return undefined;

  const number = Number(text);
  return number >= minimum ? number : undefined;
}

/** How many pages a list has; at least one, so an empty list still has a page to show. */
export function pageCount(total: number, pageSize: number): number {
  return Math.max(1, Math.ceil(total / pageSize));
}

/** A page's address with its query, leaving out empty values and the first page. */
export function hrefWithQuery(
  path: string,
  query: Readonly<Record<string, string | number | undefined>>,
): string {
  const params = new URLSearchParams();
  for (const [name, value] of Object.entries(query)) {
    if (value === undefined || value === '') continue;
    if (name === 'page' && value === 1) continue;
    params.set(name, String(value));
  }

  const text = params.toString();
  return text === '' ? path : `${path}?${text}`;
}
