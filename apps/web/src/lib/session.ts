import 'server-only';
import { notFound, redirect } from 'next/navigation';
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

/**
 * The logged-in account, for pages only players can see. A visitor is sent to log in, and comes
 * back to `returnTo` afterwards.
 */
export async function requireAccount(returnTo: string): Promise<Account> {
  const account = await getCurrentAccount();
  if (!account) redirect(`/login?next=${encodeURIComponent(returnTo)}`);

  return account;
}

/**
 * The logged-in admin, for the admin panel's pages. A visitor is sent to log in, as for any page
 * for players; a player is answered as if the page did not exist.
 */
export async function requireAdmin(returnTo: string): Promise<Account> {
  const account = await requireAccount(returnTo);
  if (account.role !== 'admin') notFound();

  return account;
}
