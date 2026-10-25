'use client';

import { useRouter } from 'next/navigation';
import { useLiveMessage } from '@/components/live/live-provider';

/**
 * Draws the page afresh once one of the player's games is saved, so the rank in the header, and
 * any profile or ranking on the page, shows what the game did. In the site's header only: a game's
 * own page is not drawn again, as the game it shows has ended.
 */
export function RankRefresh({ accountId }: { readonly accountId: string }) {
  const router = useRouter();

  useLiveMessage('match_recorded', (message) => {
    if (message.match.host.id === accountId || message.match.guest.id === accountId) {
      router.refresh();
    }
  });

  return null;
}
