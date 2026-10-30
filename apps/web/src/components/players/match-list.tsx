import Link from 'next/link';
import { CivilizationMarks } from '@/components/cards/civilization-marks';
import {
  MATCH_ENDING,
  MATCH_RESULT,
  matchSide,
  profileHref,
  type Match,
  type MatchPlayer,
} from '@/lib/players';
import { minutesAndSeconds } from '@/lib/time-text';
import { RatingChange } from './rating-change';
import { dayText } from '@/lib/dates';

const RESULT_CLASSES = {
  won: 'text-civ-light ring-civ-light/50',
  lost: 'text-civ-fire ring-civ-fire/50',
  none: 'text-slate-400 ring-slate-500/50',
} as const;

interface MatchListProps {
  readonly matches: readonly Match[];
  /** The player whose matches these are: each is shown from their side. */
  readonly playerId: string;
  /** Said when there are none. */
  readonly empty: string;
}

/**
 * A player's matches, the latest first: the result, the opponent, both decks, how it went and what
 * it did to the player's rating. A row on a desktop; a card on a phone.
 */
export function MatchList({ matches, playerId, empty }: MatchListProps) {
  if (matches.length === 0) {
    return <p className="px-4 py-6 text-center text-sm text-slate-400">{empty}</p>;
  }

  return (
    <ul>
      {matches.map((match) => {
        const { player, opponent, result } = matchSide(match, playerId);
        return (
          <li
            key={match.id}
            className="grid gap-x-4 gap-y-2 border-b border-slate-800/80 px-4 py-3 last:border-0 hover:bg-white/[0.03] sm:grid-cols-[5.5rem_1fr_auto] sm:items-center"
          >
            <div className="flex items-center gap-3 sm:block">
              <span
                className={`inline-block px-2 py-0.5 font-display text-sm font-bold tracking-wider uppercase ring-1 ring-inset ${RESULT_CLASSES[result]}`}
              >
                {MATCH_RESULT[result]}
              </span>
              <time dateTime={match.endedAt} className="text-xs text-slate-500 sm:mt-1 sm:block">
                {dayText(match.endedAt)}
              </time>
            </div>

            <div className="min-w-0 space-y-1">
              <p className="truncate text-sm text-slate-300">
                vs{' '}
                {opponent.computer ? (
                  <>
                    <span className="font-semibold text-slate-100">{opponent.username}</span>
                    <span className="ml-2 px-1.5 py-0.5 text-[10px] font-semibold tracking-wider text-slate-300 uppercase ring-1 ring-slate-600/60 ring-inset">
                      Computer
                    </span>
                  </>
                ) : (
                  <Link
                    href={profileHref(opponent.username)}
                    className="font-semibold text-slate-100 hover:text-accent"
                  >
                    {opponent.username}
                  </Link>
                )}
                <span className="text-slate-500"> · {match.name}</span>
              </p>
              <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-400">
                <DeckName player={player} />
                <span className="text-slate-600">vs</span>
                <DeckName player={opponent} />
              </p>
              <p className="text-xs text-slate-500">
                {match.formatName} · {MATCH_ENDING[match.ending]} ·{' '}
                {match.turns === 1 ? '1 turn' : `${String(match.turns)} turns`} ·{' '}
                {minutesAndSeconds(match.durationSeconds)}
              </p>
            </div>

            <div className="text-sm sm:text-right">
              {opponent.computer ? (
                <span className="text-xs text-slate-500">Practice</span>
              ) : (
                <RatingChange player={player} />
              )}
            </div>
          </li>
        );
      })}
    </ul>
  );
}

function DeckName({ player }: { readonly player: MatchPlayer }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1.5">
      <span className="truncate">{player.deckName}</span>
      <CivilizationMarks civilizations={player.civilizations} />
    </span>
  );
}
