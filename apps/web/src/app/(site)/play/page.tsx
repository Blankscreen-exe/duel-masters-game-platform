import type { Metadata } from 'next';
import { LobbyView } from '@/components/lobby/lobby-view';
import { serverApi } from '@/lib/api/server';
import { getPlayableDecks } from '@/lib/playable-decks';
import { requireAccount } from '@/lib/session';

export const metadata: Metadata = { title: 'Play' };

export default async function PlayPage() {
  const account = await requireAccount('/play');
  const api = await serverApi();
  const [lobby, quickMatch, decks, formats, opponents, chat] = await Promise.all([
    api.GET('/lobby'),
    api.GET('/quick-match'),
    getPlayableDecks(),
    api.GET('/formats'),
    api.GET('/computer-opponents'),
    api.GET('/chat'),
  ]);

  if (!lobby.data || !quickMatch.data || !formats.data || !opponents.data || !chat.data) {
    throw new Error('The lobby could not be loaded');
  }

  return (
    <LobbyView
      account={account}
      lobby={lobby.data}
      quickMatch={quickMatch.data}
      decks={decks}
      formats={formats.data}
      opponents={opponents.data}
      chat={chat.data}
    />
  );
}
