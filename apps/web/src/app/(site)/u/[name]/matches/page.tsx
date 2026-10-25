import type { Metadata } from 'next';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import { MatchList } from '@/components/players/match-list';
import { Pagination } from '@/components/ui/pagination';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import { matchesHref, profileHref } from '@/lib/players';
import { pageCount, wholeNumberParam } from '@/lib/search-params';
import { requireAccount } from '@/lib/session';

export async function generateMetadata({
  params,
}: PageProps<'/u/[name]/matches'>): Promise<Metadata> {
  return { title: `${(await params).name}'s matches` };
}

/** Every game a player finished, counted or not, the latest first, a page at a time. */
export default async function MatchHistoryPage({
  params,
  searchParams,
}: PageProps<'/u/[name]/matches'>) {
  const name = (await params).name;
  const page = wholeNumberParam((await searchParams)['page'], 1) ?? 1;
  await requireAccount(matchesHref(name, page));

  const api = await serverApi();
  const [profile, history] = await Promise.all([
    api.GET('/players/{username}', { params: { path: { username: name } } }),
    api.GET('/players/{username}/matches', {
      params: { path: { username: name }, query: { page } },
    }),
  ]);

  if (profile.response.status === 404) notFound();
  if (!profile.data || !history.data) throw new Error('The match history could not be loaded');

  const player = profile.data;
  const matches = history.data;

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <p className="text-sm text-slate-400">
          <Link href={profileHref(player.username)} className="hover:text-accent">
            {player.username}
          </Link>
        </p>
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          Match history
        </h1>
        <p className="text-sm text-slate-400">
          {matches.total === 1 ? '1 match' : `${matches.total.toLocaleString('en')} matches`}
        </p>
      </header>

      <Panel>
        <MatchList
          matches={matches.matches}
          playerId={player.id}
          empty={
            matches.total === 0
              ? `${player.username} has not finished a game yet.`
              : 'There are no matches on this page.'
          }
        />
      </Panel>

      <Pagination
        page={matches.page}
        pages={pageCount(matches.total, matches.pageSize)}
        href={(other) => matchesHref(player.username, other)}
      />
    </div>
  );
}
