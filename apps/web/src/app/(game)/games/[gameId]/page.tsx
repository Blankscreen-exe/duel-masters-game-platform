import type { Metadata } from 'next';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import { cache } from 'react';
import { GameScreen } from '@/components/game/game-screen';
import { buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import type { components } from '@/lib/api/schema';
import { serverApi } from '@/lib/api/server';
import { requireAccount } from '@/lib/session';

type LiveGame = components['schemas']['LiveGame'];

/** The game, asked once per request for both the page and its title; null when not watchable. */
const getGame = cache(async (gameId: string): Promise<LiveGame | null> => {
  await requireAccount(`/games/${encodeURIComponent(gameId)}`);
  const api = await serverApi();
  const { data, response } = await api.GET('/games/{gameId}', { params: { path: { gameId } } });

  if (data) return data;
  if (response.status === 404) notFound();
  if (response.status === 403) return null;

  throw new Error(`Loading the game failed with HTTP ${String(response.status)}`);
});

export async function generateMetadata({
  params,
}: PageProps<'/games/[gameId]'>): Promise<Metadata> {
  const game = await getGame((await params).gameId);
  return { title: game ? `${game.host.username} vs ${game.guest.username}` : 'Game' };
}

/** A running game: played by its two players, watched by everyone else it allows. */
export default async function GamePage({ params }: PageProps<'/games/[gameId]'>) {
  const { gameId } = await params;
  const account = await requireAccount(`/games/${encodeURIComponent(gameId)}`);
  const game = await getGame(gameId);

  if (!game) return <NotWatchable />;

  return <GameScreen gameId={game.id} viewerId={account.id} isPlayer={game.role === 'player'} />;
}

function NotWatchable() {
  return (
    <div className="flex flex-1 items-center justify-center p-4">
      <Panel accent="darkness" className="max-w-md p-6 text-center">
        <h1 className="font-display text-3xl font-bold tracking-widest text-white uppercase">
          Private game
        </h1>
        <p className="mt-3 text-slate-400">
          The players of this game chose not to have anyone watch.
        </p>
        <Link href="/play" className={buttonClasses('secondary', 'mt-6')}>
          To the lobby
        </Link>
      </Panel>
    </div>
  );
}
