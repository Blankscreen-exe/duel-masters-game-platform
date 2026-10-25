import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { cache } from 'react';
import { DeckBuilder } from '@/components/decks/deck-builder';
import type { Deck } from '@/lib/api/client';
import { serverApi } from '@/lib/api/server';
import { requireAccount } from '@/lib/session';

/** The deck, asked once per request for both the page and its title. */
const getDeck = cache(async (deckId: string): Promise<Deck> => {
  await requireAccount(`/decks/${encodeURIComponent(deckId)}`);
  const api = await serverApi();
  const { data, response } = await api.GET('/decks/{deckId}', { params: { path: { deckId } } });

  if (data) return data;
  if (response.status === 404) notFound();

  throw new Error(`Loading the deck failed with HTTP ${String(response.status)}`);
});

export async function generateMetadata({
  params,
}: PageProps<'/decks/[deckId]'>): Promise<Metadata> {
  const deck = await getDeck((await params).deckId);
  return { title: deck.name };
}

export default async function DeckPage({ params }: PageProps<'/decks/[deckId]'>) {
  const deck = await getDeck((await params).deckId);
  const api = await serverApi();
  const [formats, filters] = await Promise.all([api.GET('/formats'), api.GET('/cards/filters')]);

  if (!formats.data || !filters.data) throw new Error('The deck builder could not be loaded');

  // Keyed by deck, so moving to another deck starts the builder afresh.
  return <DeckBuilder key={deck.id} deck={deck} formats={formats.data} filters={filters.data} />;
}
