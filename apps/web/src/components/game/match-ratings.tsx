'use client';

import { useEffect, useState } from 'react';
import { useLiveMessage } from '@/components/live/live-provider';
import { RatingChange } from '@/components/players/rating-change';
import { getMatch } from '@/lib/lobby-actions';
import type { Match } from '@/lib/players';

/**
 * How long to wait before each look for a game in the match history: it is saved a moment after
 * it ends. Its players are told the moment it is (match_recorded); spectators look again.
 */
const LOOK_AGAIN_AFTER_MS = [500, 1000, 2000, 4000] as const;

/**
 * The game once it is in the match history, with what it did to both players' ratings; undefined
 * until then, and null if it never shows.
 */
function useRecordedMatch(gameId: string): Match | null | undefined {
  const [match, setMatch] = useState<Match | null | undefined>(undefined);

  useEffect(() => {
    let current = true;
    let timer: number | undefined;

    const look = async (attempt: number) => {
      const outcome = await getMatch(gameId);
      if (!current) return;
      if (outcome.ok && outcome.value) {
        const found = outcome.value;
        setMatch((shown) => shown ?? found);
        return;
      }

      const delay = LOOK_AGAIN_AFTER_MS[attempt];
      if (delay === undefined) {
        setMatch((shown) => shown ?? null);
        return;
      }
      timer = window.setTimeout(() => void look(attempt + 1), delay);
    };
    void look(0);

    return () => {
      current = false;
      window.clearTimeout(timer);
    };
  }, [gameId]);

  useLiveMessage('match_recorded', (message) => {
    if (message.match.id === gameId) setMatch(message.match);
  });

  return match;
}

/** What a game that counted did to both players' ratings, on its result screen. */
export function MatchRatings({ gameId }: { readonly gameId: string }) {
  const match = useRecordedMatch(gameId);

  if (match === undefined) {
    return (
      <p role="status" className="mt-4 text-sm text-slate-400">
        Working out the new ratings…
      </p>
    );
  }
  if (!match?.counted) return null;

  return (
    <ul aria-label="Rating changes" className="mt-4 space-y-1 text-sm">
      {[match.host, match.guest].map((player) => (
        <li key={player.id} className="flex items-center justify-center gap-3">
          <span className="font-semibold text-slate-200">{player.username}</span>
          <RatingChange player={player} />
        </li>
      ))}
    </ul>
  );
}
