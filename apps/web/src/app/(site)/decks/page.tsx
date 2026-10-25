import type { Metadata } from 'next';
import { DeckTile, StarterTile } from '@/components/decks/deck-tile';
import { ImportDeckButton } from '@/components/decks/import-deck-button';
import { NewDeckButton } from '@/components/decks/new-deck-button';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import { requireAccount } from '@/lib/session';

export const metadata: Metadata = { title: 'My decks' };

export default async function DecksPage() {
  await requireAccount('/decks');
  const api = await serverApi();
  const [decks, starters, formats] = await Promise.all([
    api.GET('/decks'),
    api.GET('/starter-decks'),
    api.GET('/formats'),
  ]);

  if (!decks.data || !starters.data || !formats.data) {
    throw new Error('The decks could not be loaded');
  }

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
            Start a new deck and pick 40 cards, import a deck list you already have, or play a
            starter deck.
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

      {starters.data.length > 0 && (
        <section aria-labelledby="starter-decks" className="space-y-4">
          <div>
            <h2
              id="starter-decks"
              className="font-display text-2xl font-bold tracking-widest text-white uppercase"
            >
              Starter decks
            </h2>
            <p className="mt-1 text-sm text-slate-400">
              Ready-made decks anyone can play. Copy one into your decks to change it.
            </p>
          </div>
          <ul className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {starters.data.map((deck, index) => (
              <li key={deck.id}>
                <StarterTile deck={deck} formats={formats.data} isDefault={index === 0} />
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
