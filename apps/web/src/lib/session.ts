import 'server-only';
import { cache } from 'react';
import type { Account } from './api/client';
import { serverApi } from './api/server';

/**
 * The account logged in for the request being rendered, or null for a visitor. Asked once per
 * request, however many components want it.
 */
export const getCurrentAccount = cache(async (): Promise<Account | null> => {
  const api = await serverApi();
  const { data, response } = await api.GET('/auth/me');

  if (data) return data;
  if (response.status === 401) return null;

  throw new Error(`Checking the session failed with HTTP ${String(response.status)}`);
});
