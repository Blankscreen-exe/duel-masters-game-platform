'use client';

import Link from 'next/link';
import { useState } from 'react';
import { PlayerName } from '@/components/players/player-name';
import { buttonClasses, Button } from '@/components/ui/button';
import { describeActivity, type PlayableDecks } from '@/lib/lobby';
import type { OnlinePlayer, OpenGame, RunningGame } from '@/lib/lobby-actions';
import { JoinDialog } from './join-dialog';

const ROW =
  'flex items-center justify-between gap-3 border-b border-slate-800/80 px-4 py-3 last:border-0 hover:bg-white/[0.03]';

/** A tag saying whether a game is waiting for an opponent or under way, in the status colours. */
const STATUS_TAG =
  'shrink-0 px-2 py-0.5 font-display text-xs font-semibold tracking-wider uppercase';

interface GamesListProps {
  readonly openGames: readonly OpenGame[];
  readonly runningGames: readonly RunningGame[];
  /** Leaves the running games out. */
  readonly openOnly: boolean;
  readonly decks: PlayableDecks;
  readonly userId: string;
  readonly canJoin: boolean;
}

/**
 * The listed games in one list: the open ones first, each joinable but the player's own, then the
 * running ones, with a way in to watch those that allow it.
 */
export function GamesList({
  openGames,
  runningGames,
  openOnly,
  decks,
  userId,
  canJoin,
}: GamesListProps) {
  const [joining, setJoining] = useState<OpenGame | null>(null);
  const running = openOnly ? [] : runningGames;

  if (openGames.length === 0 && running.length === 0) {
    return (
      <p className="px-4 py-6 text-center text-sm text-slate-400">
        {openOnly ? 'No open games right now.' : 'No games right now.'} Open one, or try quick
        match.
      </p>
    );
  }

  return (
    <>
      <ul>
        {openGames.map((game) => (
          <li key={game.id} className={ROW}>
            <div className="flex min-w-0 items-center gap-3">
              <span className={`${STATUS_TAG} bg-emerald-500/15 text-emerald-300`}>Open</span>
              <div className="min-w-0">
                <p className="truncate font-semibold text-slate-100">{game.name}</p>
                <p className="flex flex-wrap items-center gap-x-1.5 text-xs text-slate-400">
                  <PlayerName username={game.host.username} rank={game.host.rank} />
                  <span>
                    · {game.formatName}
                    {game.spectators ? '' : ' · no spectators'}
                  </span>
                </p>
              </div>
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
        {running.map((game) => (
          <li key={game.id} className={ROW}>
            <div className="flex min-w-0 items-center gap-3">
              <span className={`${STATUS_TAG} bg-red-500/15 text-red-300`}>Playing</span>
              <div className="min-w-0">
                <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
                  <PlayerName username={game.host.username} rank={game.host.rank} />
                  <span className="text-slate-500">vs</span>
                  <PlayerName username={game.guest.username} rank={game.guest.rank} />
                </p>
                <p className="text-xs text-slate-400">
                  {game.name} · {game.formatName}
                </p>
              </div>
            </div>
            {game.spectators && (
              <Link
                href={`/games/${encodeURIComponent(game.id)}`}
                className={buttonClasses('quiet')}
              >
                Watch
              </Link>
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

/** The players online, and what each is doing. */
export function OnlinePlayers({ players }: { readonly players: readonly OnlinePlayer[] }) {
  return (
    <ul>
      {players.map((player) => (
        <li key={player.id} className={`${ROW} py-2`}>
          <span className="min-w-0 text-sm">
            <PlayerName username={player.username} rank={player.rank} />
          </span>
          <span className="shrink-0 text-xs text-slate-400">
            {describeActivity(player.activity)}
          </span>
        </li>
      ))}
    </ul>
  );
}
