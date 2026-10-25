'use client';

import { Duel, type CardPictures, type DuelFinishedMessage } from '@duel-masters/duel-board';
import { useRouter } from 'next/navigation';
import { socketUrl } from '@/lib/live/messages';
import { GameResultPanel } from './game-result-panel';
import type { GameViewer } from './game-viewer';

/** Card pictures by the engine's card ids, and the site's card back. */
const PICTURES: CardPictures = {
  card: (imageId) => `/api/cards/images/${encodeURIComponent(imageId)}`,
  back: '/card-back.svg',
};

interface GameBoardProps {
  readonly gameId: string;
  readonly viewer: GameViewer;
}

/**
 * The duel board for a game, connected to its websocket. Drawn in the browser only: the board
 * reads the screen's size and the player's saved settings as it draws.
 */
export default function GameBoard({ gameId, viewer }: GameBoardProps) {
  const router = useRouter();

  return (
    <Duel
      socketUrl={socketUrl(window.location, `/api/games/${encodeURIComponent(gameId)}/socket`)}
      pictures={PICTURES}
      onLeaveDuel={() => {
        router.push('/play');
      }}
      renderDuelFinished={(message: DuelFinishedMessage) => (
        <GameResultPanel message={message} viewer={viewer} />
      )}
    />
  );
}
