'use client';

import { useRouter } from 'next/navigation';
import { useLiveMessage } from '@/components/live/live-provider';

/**
 * Draws the page afresh when the player's account changes, so the header, and any profile or
 * ranking on the page, shows it: once one of their games is saved, with its rating change, and
 * when an admin changes the account, such as muting them. In the site's header only: a game's own
 * page is not drawn again, as the game it shows has ended.
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

  return null;
}
