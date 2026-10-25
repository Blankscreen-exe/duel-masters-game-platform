'use client';

import Link from 'next/link';
import { useState } from 'react';
import { buttonClasses, Button } from '@/components/ui/button';
import type { DeckSummary } from '@/lib/api/client';
import { describeActivity } from '@/lib/lobby';
import type { OnlinePlayer, OpenGame, RunningGame } from '@/lib/lobby-actions';
import { JoinDialog } from './join-dialog';

const ROW =
  'flex items-center justify-between gap-3 border-b border-slate-800/80 px-4 py-3 last:border-0 hover:bg-white/[0.03]';

/** The listed open games, each joinable but the player's own. */
export function OpenGamesList({
  games,
  decks,
  userId,
  canJoin,
}: {
  readonly games: readonly OpenGame[];
  readonly decks: readonly DeckSummary[];
  readonly userId: string;
  readonly canJoin: boolean;
}) {
  const [joining, setJoining] = useState<OpenGame | null>(null);

  if (games.length === 0) {
    return (
      <p className="px-4 py-6 text-center text-sm text-slate-400">
        No open games right now. Open one, or try quick match.
      </p>
    );
  }

  return (
    <>
      <ul>
        {games.map((game) => (
          <li key={game.id} className={ROW}>
            <div className="min-w-0">
              <p className="truncate font-semibold text-slate-100">{game.name}</p>
              <p className="text-xs text-slate-400">
                {game.host.username} · {game.formatName}
                {game.spectators ? '' : ' · no spectators'}
              </p>
            </div>
            {game.host.id === userId ? (
              <span className="text-xs tracking-wider text-slate-500 uppercase">Yours</span>
            ) : (
              <Button
                variant="secondary"
                disabled={!canJoin}
                onClick={() => {
                  setJoining(game);
                }}
              >
                Join
              </Button>
            )}
          </li>
        ))}
      </ul>
      {joining && (
        <JoinDialog
          game={joining}
          decks={decks}
          onClose={() => {
            setJoining(null);
          }}
        />
      )}
    </>
  );
}

/** The listed running games, with a way in to watch those that allow it. */
export function RunningGamesList({ games }: { readonly games: readonly RunningGame[] }) {
  if (games.length === 0) {
    return <p className="px-4 py-6 text-center text-sm text-slate-400">No games running.</p>;
  }

  return (
    <ul>
      {games.map((game) => (
        <li key={game.id} className={ROW}>
          <div className="min-w-0">
            <p className="truncate font-semibold text-slate-100">
              {game.host.username} <span className="text-slate-500">vs</span> {game.guest.username}
            </p>
            <p className="text-xs text-slate-400">
              {game.name} · {game.formatName}
            </p>
          </div>
          {game.spectators && (
            <Link href={`/games/${encodeURIComponent(game.id)}`} className={buttonClasses('quiet')}>
              Watch
            </Link>
          )}
        </li>
      ))}
    </ul>
  );
}

/** The players online, and what each is doing. */
export function OnlinePlayers({ players }: { readonly players: readonly OnlinePlayer[] }) {
  return (
    <ul>
      {players.map((player) => (
        <li key={player.id} className={`${ROW} py-2`}>
          <span className="truncate text-sm font-semibold text-slate-100">{player.username}</span>
          <span className="shrink-0 text-xs text-slate-400">
            {describeActivity(player.activity)}
          </span>
        </li>
      ))}
    </ul>
  );
}
