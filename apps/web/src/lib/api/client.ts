import createClient from 'openapi-fetch';
import type { components, paths } from './schema';

/**
 * The API from the browser. It is on the same site, through Next.js's /api forwarding, so the
 * session cookie goes along with every request.
 */
export const api = createClient<paths>({ baseUrl: '/api' });

export type ApiError = components['schemas']['Error'];
export type Account = components['schemas']['Account'];
export type Card = components['schemas']['Card'];
export type CardPage = components['schemas']['CardPage'];
export type CardFilters = components['schemas']['CardFilters'];
