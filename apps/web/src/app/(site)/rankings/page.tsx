import type { Metadata } from 'next';
import Link from 'next/link';
import { RankBadge } from '@/components/players/rank-badge';
import { Button, buttonClasses } from '@/components/ui/button';
import { Pagination } from '@/components/ui/pagination';
import { Panel } from '@/components/ui/panel';
import { TextField } from '@/components/ui/text-field';
import { serverApi } from '@/lib/api/server';
import {
  pageOfPlace,
  parseRankingsQuery,
  profileHref,
  rankingsHref,
  RANKINGS_SEARCH_MAX_LENGTH,
  streakDescription,
  streakText,
  type RankedPlayer,
} from '@/lib/players';
import { pageCount } from '@/lib/search-params';
import { requireAccount } from '@/lib/session';

export const metadata: Metadata = { title: 'Rankings' };

/** Everyone who has finished a game that counted, the best rated first, with a name search. */
export default async function RankingsPage({ searchParams }: PageProps<'/rankings'>) {
  const query = parseRankingsQuery(await searchParams);
  const account = await requireAccount(rankingsHref(query));
  const api = await serverApi();
  const { data: rankings, response } = await api.GET('/rankings', { params: { query } });

  if (!rankings) {
    throw new Error(`The rankings failed to load with HTTP ${String(response.status)}`);
  }

  const pages = pageCount(rankings.total, rankings.pageSize);
  const you = rankings.you;

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          Rankings
        </h1>
        <form action="/rankings" role="search" className="flex items-end gap-2">
          <TextField
            label="Find a player"
            name="search"
            type="search"
            defaultValue={query.search}
            maxLength={RANKINGS_SEARCH_MAX_LENGTH}
            className="w-56"
          />
          <Button type="submit" variant="secondary">
            Search
          </Button>
        </form>
      </header>

      <Panel accent="light" className="flex flex-wrap items-center justify-between gap-4 p-4">
        {you ? (
          <>
            <p className="flex flex-wrap items-center gap-3 text-slate-200">
              <span className="font-display text-lg font-semibold">You are #{you.place}</span>
              <RankBadge rank={you.rank} withRating />
            </p>
            <Link
              href={rankingsHref({}, pageOfPlace(you.place, rankings.pageSize))}
              className={buttonClasses('quiet')}
            >
              Show my place
            </Link>
          </>
        ) : (
          <p className="text-slate-300">
            Finish a game that counts to get your place in the rankings.{' '}
            <Link href="/play" className="font-semibold text-accent hover:underline">
              Play now
            </Link>
          </p>
        )}
      </Panel>

      <Panel>
        {rankings.players.length === 0 ? (
          <p className="px-4 py-8 text-center text-slate-400">
            {query.search
              ? `No ranked player's name contains “${query.search}”.`
              : 'Nobody is ranked yet. The first game that counts starts the rankings.'}
          </p>
        ) : (
          <table className="w-full text-left text-sm">
            <caption className="sr-only">
              Players by rating{query.search ? `, names containing “${query.search}”` : ''}
            </caption>
            <thead className="text-[11px] tracking-wider text-slate-400 uppercase">
              <tr className="border-b border-slate-700/60">
                <th scope="col" className="px-4 py-2.5 font-semibold">
                  #
                </th>
                <th scope="col" className="px-2 py-2.5 font-semibold">
                  Player
                </th>
                <th scope="col" className="px-2 py-2.5 font-semibold">
                  Rank
                </th>
                <th
                  scope="col"
                  className="hidden px-2 py-2.5 text-right font-semibold sm:table-cell"
                >
                  Won
                </th>
                <th
                  scope="col"
                  className="hidden px-2 py-2.5 text-right font-semibold sm:table-cell"
                >
                  Lost
                </th>
                <th scope="col" className="px-4 py-2.5 text-right font-semibold">
                  Streak
                </th>
              </tr>
            </thead>
            <tbody>
              {rankings.players.map((player) => (
                <RankingRow key={player.id} player={player} isYou={player.id === account.id} />
              ))}
            </tbody>
          </table>
        )}
      </Panel>

      <Pagination page={rankings.page} pages={pages} href={(page) => rankingsHref(query, page)} />
    </div>
  );
}

function RankingRow({ player, isYou }: { readonly player: RankedPlayer; readonly isYou: boolean }) {
  return (
    <tr
      aria-current={isYou ? 'true' : undefined}
      className={`border-b border-slate-800/80 last:border-0 hover:bg-white/[0.03] ${isYou ? 'bg-accent/10' : ''}`}
    >
      <td className="px-4 py-2.5 font-display text-base font-bold text-slate-300 tabular-nums">
        {player.place}
      </td>
      <td className="max-w-0 px-2 py-2.5">
        <Link
          href={profileHref(player.username)}
          className="block truncate font-semibold text-slate-100 hover:text-accent"
        >
          {player.username}
          {isYou && <span className="ml-2 text-xs font-normal text-accent">(you)</span>}
        </Link>
      </td>
      <td className="px-2 py-2.5">
        <RankBadge rank={player.rank} withRating />
      </td>
      <td className="hidden px-2 py-2.5 text-right text-slate-300 tabular-nums sm:table-cell">
        {player.wins}
      </td>
      <td className="hidden px-2 py-2.5 text-right text-slate-300 tabular-nums sm:table-cell">
        {player.losses}
      </td>
      <td
        title={streakDescription(player.streak)}
        className="px-4 py-2.5 text-right font-semibold text-slate-300 tabular-nums"
      >
        {streakText(player.streak)}
      </td>
    </tr>
  );
}
