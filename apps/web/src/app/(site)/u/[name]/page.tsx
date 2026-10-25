import type { Metadata } from 'next';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import { cache } from 'react';
import { CivilizationMarks } from '@/components/cards/civilization-marks';
import { MatchList } from '@/components/players/match-list';
import { RankBadge } from '@/components/players/rank-badge';
import { buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import {
  matchesHref,
  profileHref,
  streakDescription,
  streakText,
  winRate,
  type PlayerProfile,
} from '@/lib/players';
import { requireAccount } from '@/lib/session';
import { dayText } from '@/lib/dates';

/** The profile, asked once per request for both the page and its title. */
const getProfile = cache(async (name: string): Promise<PlayerProfile> => {
  await requireAccount(profileHref(name));
  const api = await serverApi();
  const { data, response } = await api.GET('/players/{username}', {
    params: { path: { username: name } },
  });

  if (data) return data;
  if (response.status === 404) notFound();

  throw new Error(`Loading the profile failed with HTTP ${String(response.status)}`);
});

export async function generateMetadata({ params }: PageProps<'/u/[name]'>): Promise<Metadata> {
  const profile = await getProfile((await params).name);
  return { title: profile.username };
}

/** A player's rank, record, favourite civilizations and latest matches. */
export default async function ProfilePage({ params }: PageProps<'/u/[name]'>) {
  const profile = await getProfile((await params).name);

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-center gap-4">
        <span
          aria-hidden
          className="cut-corners grid size-16 place-items-center bg-linear-to-br from-slate-200 to-slate-500 font-display text-3xl font-bold text-slate-900"
        >
          {profile.username.charAt(0).toUpperCase()}
        </span>
        <div className="space-y-1">
          <h1 className="font-display text-4xl font-bold tracking-widest text-white">
            {profile.username}
          </h1>
          <p className="flex flex-wrap items-center gap-3 text-sm text-slate-400">
            <RankBadge rank={profile.rank} large />
            <span>
              Joined <time dateTime={profile.joinedAt}>{dayText(profile.joinedAt)}</time>
            </span>
          </p>
        </div>
      </header>

      <dl className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        <Stat label="Rating" value={String(profile.rank.rating)} />
        <Stat
          label="Place"
          value={profile.place === undefined ? '—' : `#${String(profile.place)}`}
          hint={profile.place === undefined ? 'Not ranked yet' : undefined}
        />
        <Stat label="Won" value={String(profile.wins)} />
        <Stat label="Lost" value={String(profile.losses)} />
        <Stat label="Win rate" value={winRate(profile.wins, profile.losses)} />
        <Stat
          label="Streak"
          value={streakText(profile.streak)}
          hint={streakDescription(profile.streak)}
        />
      </dl>

      <div className="grid gap-6 lg:grid-cols-[18rem_1fr]">
        <Panel title="Favourite civilizations" accent="nature" className="self-start">
          {profile.favouriteCivilizations.length === 0 ? (
            <p className="px-4 py-6 text-center text-sm text-slate-400">No games played yet.</p>
          ) : (
            <ul className="space-y-2 p-4">
              {profile.favouriteCivilizations.map((played) => (
                <li
                  key={played.civilization}
                  className="flex items-center justify-between gap-3 text-sm"
                >
                  <span className="flex items-center gap-2 text-slate-100">
                    <CivilizationMarks civilizations={[played.civilization]} />
                    {played.civilization}
                  </span>
                  <span className="text-slate-400 tabular-nums">
                    {played.games === 1 ? '1 game' : `${String(played.games)} games`}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Panel>

        <Panel
          title="Recent matches"
          accent="water"
          actions={
            profile.recentMatches.length > 0 && (
              <Link href={matchesHref(profile.username)} className={buttonClasses('quiet')}>
                All matches
              </Link>
            )
          }
        >
          <MatchList
            matches={profile.recentMatches}
            playerId={profile.id}
            empty={`${profile.username} has not finished a game yet.`}
          />
        </Panel>
      </div>
    </div>
  );
}

function Stat({
  label,
  value,
  hint,
}: {
  readonly label: string;
  readonly value: string;
  readonly hint?: string;
}) {
  return (
    <Panel className="px-4 py-3">
      <dt className="text-[11px] font-semibold tracking-wider text-slate-400 uppercase">{label}</dt>
      <dd className="mt-1 font-display text-2xl font-bold text-white tabular-nums" title={hint}>
        {value}
      </dd>
    </Panel>
  );
}
