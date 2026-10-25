import type { DuelFinishedMessage } from '@duel-masters/duel-board';
import Link from 'next/link';
import { buttonClasses } from '@/components/ui/button';
import { describeResult } from '@/lib/game-result';

const HEADLINE_COLOUR = {
  won: 'text-civ-light',
  lost: 'text-civ-fire',
  none: 'text-white',
} as const;

/** The site's result screen, shown on the board once a game has ended. */
export function GameResultPanel({
  message,
  viewerId,
  isPlayer,
}: {
  readonly message: DuelFinishedMessage;
  readonly viewerId: string;
  readonly isPlayer: boolean;
}) {
  const result = describeResult(message, viewerId, isPlayer);

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
      <Link href="/play" className={buttonClasses('primary', 'mt-6')}>
        Back to the lobby
      </Link>
    </div>
  );
}
