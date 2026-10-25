import type { Metadata } from 'next';
import Link from 'next/link';
import { redirect } from 'next/navigation';
import { JoinForm } from '@/components/lobby/join-dialog';
import { buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import { getPlayableDecks } from '@/lib/playable-decks';
import { requireAccount } from '@/lib/session';

export const metadata: Metadata = { title: 'Join a game' };

/** Where an invite link leads: the open game, listed or not, with a deck to join it with. */
export default async function JoinPage({ params }: PageProps<'/play/join/[openGameId]'>) {
  const { openGameId } = await params;
  const account = await requireAccount(`/play/join/${encodeURIComponent(openGameId)}`);
  const api = await serverApi();
  const [game, decks] = await Promise.all([
    api.GET('/lobby/games/{openGameId}', { params: { path: { openGameId } } }),
    getPlayableDecks(),
  ]);

  if (game.response.status === 404) return <NoLongerOpen />;
  if (!game.data) throw new Error('The open game could not be loaded');
  // The host's own invite link leads back to the lobby, where their game waits.
  if (game.data.host.id === account.id) redirect('/play');

  return (
    <Panel title={`Join ${game.data.name}`} accent="water" className="mx-auto max-w-md">
      <div className="p-4">
        <JoinForm
          game={game.data}
          decks={decks}
          cancel={
            <Link href="/play" className={buttonClasses('secondary')}>
              To the lobby
            </Link>
          }
        />
      </div>
    </Panel>
  );
}

function NoLongerOpen() {
  return (
    <Panel accent="darkness" className="mx-auto max-w-md p-6 text-center">
      <h1 className="font-display text-3xl font-bold tracking-widest text-white uppercase">
        Game closed
      </h1>
      <p className="mt-3 text-slate-400">
        This game is no longer open: it has started, or its host closed it.
      </p>
      <Link href="/play" className={buttonClasses('secondary', 'mt-6')}>
        To the lobby
      </Link>
    </Panel>
  );
}
