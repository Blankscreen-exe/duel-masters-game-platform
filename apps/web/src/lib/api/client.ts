import createClient from 'openapi-fetch';
import { jsonChanges } from './json-changes';
import type { components, paths } from './schema';

/**
 * The API from the browser. It is on the same site, through Next.js's /api forwarding, so the
 * session cookie goes along with every request.
 */
export const api = createClient<paths>({ baseUrl: '/api' });
api.use(jsonChanges);

type Schemas = components['schemas'];
export type ApiError = Schemas['Error'];
export type Account = Schemas['Account'];
export type Card = Schemas['Card'];
export type CardPage = Schemas['CardPage'];
export type CardFilters = Schemas['CardFilters'];
export type Format = Schemas['Format'];
export type Deck = Schemas['Deck'];
export type DeckCard = Schemas['DeckCard'];
export type DeckInput = Schemas['DeckInput'];
export type DeckSummary = Schemas['DeckSummary'];
export type DeckLegality = Schemas['DeckLegality'];
export type DeckListReading = Schemas['DeckListReading'];

/** What went wrong, for the player: the server's own message when it sent one. */
export function errorMessage(error: ApiError | undefined): string {
  return error?.message ?? 'Something went wrong. Please try again.';
}

/** For a request that never reached the server. */
export const UNREACHABLE = 'The server cannot be reached. Please try again in a moment.';
