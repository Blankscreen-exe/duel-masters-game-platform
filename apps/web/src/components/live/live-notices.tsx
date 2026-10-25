'use client';

import { useRouter } from 'next/navigation';
import { notify } from '@/lib/notifications';
import { useLiveMessage } from './live-provider';

/**
 * What every page does with the player's own live notices: going to their game as soon as it
 * starts, wherever on the site they are, and saying when their open game closed unjoined.
 */
export function LiveNotices() {
  const router = useRouter();

  useLiveMessage('game_started', (message) => {
    notify('Your game is starting');
    router.push(`/games/${encodeURIComponent(message.gameId)}`);
  });

  useLiveMessage('open_game_expired', (message) => {
    notify(`Nobody joined “${message.openGame.name}” in time, so it closed.`);
    router.refresh();
  });

  return null;
}
