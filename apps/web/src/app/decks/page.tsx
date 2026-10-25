import type { Metadata } from 'next';
import { DeckTile } from '@/components/decks/deck-tile';
import { ImportDeckButton } from '@/components/decks/import-deck-button';
import { NewDeckButton } from '@/components/decks/new-deck-button';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import { requireAccount } from '@/lib/session';

export const metadata: Metadata = { title: 'My decks' };

export default async function DecksPage() {
  await requireAccount('/decks');
  const api = await serverApi();
  const [decks, formats] = await Promise.all([api.GET('/decks'), api.GET('/formats')]);

  if (!decks.data || !formats.data) throw new Error('The decks could not be loaded');

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          My decks
        </h1>
        <div className="flex gap-2">
          <ImportDeckButton />
          <NewDeckButton />
        </div>
      </header>

      {decks.data.length === 0 ? (
        <Panel accent="water" className="p-8 text-center">
          <p className="font-display text-xl font-semibold text-slate-100">No decks yet</p>
          <p className="mt-2 text-slate-400">
            Start a new deck and pick 40 cards, or import a deck list you already have.
          </p>
        </Panel>
      ) : (
        <ul className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {decks.data.map((deck) => (
            <li key={deck.id}>
              <DeckTile deck={deck} formats={formats.data} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
