'use client';

import { useRouter } from 'next/navigation';
import { useLiveMessage } from '@/components/live/live-provider';

/**
 * Draws the page afresh when the player's account changes, so the header, and any profile or
 * ranking on the page, shows it: once one of their games is saved, with its rating change, and
 * when an admin changes the account, such as muting them, or the notifications, which the header
 * counts and the lobby pops up. In the site's header only: a game's own page is not drawn again, so
 * nothing interrupts a game.
 */
export function AccountRefresh({ accountId }: { readonly accountId: string }) {
  const router = useRouter();

  useLiveMessage('match_recorded', (message) => {
    if (message.match.host.id === accountId || message.match.guest.id === accountId) {
      router.refresh();
    }
  });
  useLiveMessage('account_changed', () => {
    router.refresh();
  });
  useLiveMessage('announcements_changed', () => {
    router.refresh();
  });

  return null;
}
