'use client';

import dynamic from 'next/dynamic';
import type { GameViewer } from './game-viewer';

/** The board, loaded in the browser only (see GameBoard). */
const GameBoard = dynamic(() => import('./game-board'), {
  ssr: false,
  loading: () => (
    <div role="status" className="flex flex-1 items-center justify-center text-slate-400">
      Loading the game…
    </div>
  ),
});

export function GameScreen(props: { readonly gameId: string; readonly viewer: GameViewer }) {
  return <GameBoard {...props} />;
}
