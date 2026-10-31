import 'server-only';
import { notFound, redirect } from 'next/navigation';
import { cache } from 'react';
import type { Account } from './api/client';
import { serverApi } from './api/server';
import { verifyEmailHref } from './verification';

/**
 * The account logged in for the request being rendered, or null for a visitor, whether or not its
 * email address is confirmed. Asked once per request, however many components want it. Most pages
 * want getCurrentAccount.
 */
export const getSessionAccount = cache(async (): Promise<Account | null> => {
  const api = await serverApi();
  const { data, response } = await api.GET('/auth/me');

  if (data) return data;
  if (response.status === 401) return null;

  throw new Error(`Checking the session failed with HTTP ${String(response.status)}`);
});

/**
 * The account logged in for the request being rendered, or null for a visitor. Until a player has
 * confirmed their email address they count as a visitor, as they do on the server.
 */
export async function getCurrentAccount(): Promise<Account | null> {
  const account = await getSessionAccount();
  return account?.emailVerified ? account : null;
}

/**
 * The logged-in account, for pages only players can see. A visitor is sent to log in, and comes
 * back to `returnTo` afterwards; a player who has not confirmed their email address goes to do so.
 */
export async function requireAccount(returnTo: string): Promise<Account> {
  const account = await getSessionAccount();
  if (!account) redirect(`/login?next=${encodeURIComponent(returnTo)}`);
  if (!account.emailVerified) redirect(verifyEmailHref(returnTo));

  return account;
}

/** For the log-in and sign-up pages: there is nothing to do there once logged in. */
export async function redirectIfLoggedIn(next: string): Promise<void> {
  const account = await getSessionAccount();
  if (!account) return;

  redirect(account.emailVerified ? next : verifyEmailHref(next));
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
