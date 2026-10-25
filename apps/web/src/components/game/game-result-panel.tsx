import type { DuelFinishedMessage } from '@duel-masters/duel-board';
import Link from 'next/link';
import { buttonClasses } from '@/components/ui/button';
import { describeResult } from '@/lib/game-result';
import type { GameViewer } from './game-viewer';
import { RematchPanel } from './rematch-panel';

const HEADLINE_COLOUR = {
  won: 'text-civ-light',
  lost: 'text-civ-fire',
  none: 'text-white',
} as const;

/** The site's result screen, shown on the board once a game has ended. */
export function GameResultPanel({
  message,
  viewer,
}: {
  readonly message: DuelFinishedMessage;
  readonly viewer: GameViewer;
}) {
  const result = describeResult(message, viewer.id, viewer.isPlayer);

  return (
    <div className="p-6 text-center text-white">
      <p
        className={`font-display text-4xl font-bold tracking-widest uppercase ${HEADLINE_COLOUR[result.outcome]}`}
      >
        {result.headline}
      </p>
      <ul className="mt-4 space-y-1 text-sm text-slate-300">
        {result.details.map((detail) => (
          <li key={detail}>{detail}</li>
        ))}
      </ul>
      {viewer.isPlayer && (
        <RematchPanel gameId={message.duelId} viewerName={viewer.username} decks={viewer.decks} />
      )}
      {/* Leaving the result screen declines a rematch still being decided. */}
      <Link href="/play" className={buttonClasses('secondary', 'mt-4 w-full')}>
        Back to the lobby
      </Link>
    </div>
  );
}
