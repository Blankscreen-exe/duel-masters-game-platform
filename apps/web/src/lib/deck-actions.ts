import { api, type Deck, type DeckInput, type DeckListReading } from './api/client';
import { attempt, attemptEmpty, type Outcome } from './api/outcome';
import { copyName } from './deck-draft';

export function createDeck(body: DeckInput): Promise<Outcome<Deck>> {
  return attempt(() => api.POST('/decks', { body }));
}

export function saveDeck(deckId: string, body: DeckInput): Promise<Outcome<Deck>> {
  return attempt(() => api.PUT('/decks/{deckId}', { params: { path: { deckId } }, body }));
}

export function deleteDeck(deckId: string): Promise<Outcome<null>> {
  return attemptEmpty(() => api.DELETE('/decks/{deckId}', { params: { path: { deckId } } }));
}

/** Makes a new deck with the same cards and cover, named as a copy. */
export async function copyDeck(deckId: string): Promise<Outcome<Deck>> {
  const original = await attempt(() =>
    api.GET('/decks/{deckId}', { params: { path: { deckId } } }),
  );
  if (!original.ok) return original;

  const deck = original.value;
  // Cards that have left the card database cannot be saved again.
  const cards = deck.cards
    .filter(({ card }) => card !== undefined)
    .map(({ cardId, count }) => ({ cardId, count }));
  const coverKept = cards.some(({ cardId }) => cardId === deck.coverCardId);

  return createDeck({
    name: copyName(deck.name),
    cards,
    ...(coverKept && deck.coverCardId !== undefined ? { coverCardId: deck.coverCardId } : {}),
  });
}

/** Copies a starter deck into the player's own decks, to change. */
export function copyStarterDeck(deckId: string): Promise<Outcome<Deck>> {
  return attempt(() => api.POST('/starter-decks/{deckId}/copy', { params: { path: { deckId } } }));
}

/** The deck written as text, one "4 Bolshack Dragon" per line. */
export function deckText(deckId: string): Promise<Outcome<string>> {
  return attempt(() =>
    api.GET('/decks/{deckId}/text', { params: { path: { deckId } }, parseAs: 'text' }),
  );
}

/** Finds the cards of a deck list written as text, without saving anything. */
export function readDeckList(text: string): Promise<Outcome<DeckListReading>> {
  return attempt(() => api.POST('/decks/text', { body: { text } }));
}
