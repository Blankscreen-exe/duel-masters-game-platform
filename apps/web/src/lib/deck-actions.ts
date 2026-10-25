import {
  api,
  errorMessage,
  UNREACHABLE,
  type ApiError,
  type Deck,
  type DeckInput,
  type DeckListReading,
} from './api/client';
import { copyName } from './deck-draft';

/** How a request from the browser went: its answer, or what to tell the player. */
export type Outcome<T> =
  { readonly ok: true; readonly value: T } | { readonly ok: false; readonly message: string };

interface Answer<T> {
  readonly data?: T;
  readonly error?: ApiError;
  readonly response: Response;
}

/** Sends a request, turning a refusal or a lost connection into a message for the player. */
async function attempt<T>(send: () => Promise<Answer<T>>): Promise<Outcome<T>> {
  try {
    const { data, error, response } = await send();
    if (response.ok && data !== undefined) return { ok: true, value: data };
    return { ok: false, message: errorMessage(error) };
  } catch {
    return { ok: false, message: UNREACHABLE };
  }
}

/** Sends a request answered with no content. */
async function attemptEmpty(send: () => Promise<Answer<unknown>>): Promise<Outcome<null>> {
  try {
    const { error, response } = await send();
    if (response.ok) return { ok: true, value: null };
    return { ok: false, message: errorMessage(error) };
  } catch {
    return { ok: false, message: UNREACHABLE };
  }
}

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
