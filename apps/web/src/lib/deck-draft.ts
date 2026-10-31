import type { Card, Deck, DeckInput } from './api/client';
import { CIVILIZATIONS } from './civilizations';

/**
 * The game's deck rules, for the deck builder to follow as the player edits. The Go server checks
 * them for real; deck-draft.test.ts keeps these numbers equal to the API contract's.
 */
export const DECK_SIZE = 40;
export const MAX_COPIES = 4;
export const DECK_NAME_MAX_LENGTH = 40;
/** The longest deck list, in characters, the server reads from text. */
export const DECK_LIST_MAX_LENGTH = 10000;

export interface DraftCard {
  readonly card: Card;
  readonly count: number;
}

/** A deck being edited in the deck builder. */
export interface DeckDraft {
  readonly name: string;
  /** The cover the player chose, or null to let the site pick one. */
  readonly coverCardId: string | null;
  /** By cost, then name. */
  readonly cards: readonly DraftCard[];
}

/** Why a card cannot be added to a draft. */
export type CannotAdd = 'not_playable' | 'max_copies' | 'deck_full';

/** Why a card cannot be added, in words for the player. */
export const CANNOT_ADD_TEXT: Record<CannotAdd, string> = {
  not_playable: 'The game cannot play this card yet',
  max_copies: `The deck has ${String(MAX_COPIES)} copies already`,
  deck_full: `The deck has ${String(DECK_SIZE)} cards already`,
};

/**
 * A saved deck as a draft. Cards that have left the card database cannot be saved again, so they
 * are left out; `dropped` counts their copies, to tell the player.
 */
export function draftFromDeck(deck: Pick<Deck, 'name' | 'cards' | 'coverCardId'>): {
  draft: DeckDraft;
  dropped: number;
} {
  const cards: DraftCard[] = [];
  let dropped = 0;
  for (const { card, count } of deck.cards) {
    if (card) cards.push({ card, count });
    else dropped += count;
  }

  const coverCardId =
    deck.coverCardId !== undefined && cards.some(({ card }) => card.id === deck.coverCardId)
      ? deck.coverCardId
      : null;

  return { draft: { name: deck.name, coverCardId, cards: sortCards(cards) }, dropped };
}

export function totalCards(draft: DeckDraft): number {
  return draft.cards.reduce((total, { count }) => total + count, 0);
}

export function copiesOf(draft: DeckDraft, cardId: string): number {
  return draft.cards.find(({ card }) => card.id === cardId)?.count ?? 0;
}

/** Why the card cannot be added, or null when it can. */
export function cannotAdd(draft: DeckDraft, card: Card): CannotAdd | null {
  if (!card.playable) return 'not_playable';
  if (copiesOf(draft, card.id) >= MAX_COPIES) return 'max_copies';
  if (totalCards(draft) >= DECK_SIZE) return 'deck_full';
  return null;
}

/** The draft with one more copy of the card, or unchanged when the rules do not allow it. */
export function addCard(draft: DeckDraft, card: Card): DeckDraft {
  if (cannotAdd(draft, card) !== null) return draft;

  const listed = draft.cards.some((entry) => entry.card.id === card.id);
  const cards = listed
    ? draft.cards.map((entry) =>
        entry.card.id === card.id ? { ...entry, count: entry.count + 1 } : entry,
      )
    : sortCards([...draft.cards, { card, count: 1 }]);

  return { ...draft, cards };
}

/** The draft with one copy fewer of the card; its last copy takes the card, and its cover, out. */
export function removeCard(draft: DeckDraft, cardId: string): DeckDraft {
  const cards = draft.cards
    .map((entry) => (entry.card.id === cardId ? { ...entry, count: entry.count - 1 } : entry))
    .filter(({ count }) => count > 0);
  const stillListed = cards.some(({ card }) => card.id === cardId);

  return {
    ...draft,
    cards,
    coverCardId: draft.coverCardId === cardId && !stillListed ? null : draft.coverCardId,
  };
}

/** The draft as the API saves it. */
export function toDeckInput(draft: DeckDraft): DeckInput {
  return {
    name: draft.name,
    ...(draft.coverCardId === null ? {} : { coverCardId: draft.coverCardId }),
    cards: draft.cards.map(({ card, count }) => ({ cardId: card.id, count })),
  };
}

/** Whether two drafts would save the same deck. */
export function sameDeck(a: DeckDraft, b: DeckDraft): boolean {
  const key = (draft: DeckDraft) =>
    JSON.stringify([
      draft.name.trim(),
      draft.coverCardId,
      draft.cards.map(({ card, count }) => [card.id, count]),
    ]);
  return key(a) === key(b);
}

/** The order card types are shown in; any type not named here follows, by name. */
const TYPE_ORDER: readonly string[] = [
  'Creature',
  'Evolution Creature',
  'Spell',
  'Castle',
  'Cross Gear',
  'Evolution Cross Gear',
];

/**
 * The draft's cards grouped by type, each group counting its copies and keeping the draft's order
 * of cost, then name.
 */
export function groupByType(
  draft: DeckDraft,
): { type: string; total: number; cards: DraftCard[] }[] {
  const groups = new Map<string, { type: string; total: number; cards: DraftCard[] }>();
  for (const entry of draft.cards) {
    const { type } = entry.card;
    const group = groups.get(type) ?? { type, total: 0, cards: [] };
    group.total += entry.count;
    group.cards.push(entry);
    groups.set(type, group);
  }

  const rank = (type: string) => {
    const place = TYPE_ORDER.indexOf(type);
    return place === -1 ? TYPE_ORDER.length : place;
  };
  return [...groups.values()].sort(
    (a, b) => rank(a.type) - rank(b.type) || compareText(a.type, b.type),
  );
}

/** The highest cost shown on its own in the mana curve; dearer cards share its last bar. */
export const CURVE_TOP = 8;

/** How many cards the draft has at each cost, from 1 to CURVE_TOP, the last bar being "8+". */
export function manaCurve(draft: DeckDraft): { label: string; count: number }[] {
  const counts = Array.from({ length: CURVE_TOP }, () => 0);
  for (const { card, count } of draft.cards) {
    const bar = Math.min(Math.max(card.cost, 1), CURVE_TOP) - 1;
    counts[bar] = (counts[bar] ?? 0) + count;
  }

  return counts.map((count, index) => ({
    label: index + 1 === CURVE_TOP ? `${String(CURVE_TOP)}+` : String(index + 1),
    count,
  }));
}

/**
 * How many of the draft's cards belong to each civilization, in the game's order. A card of two
 * civilizations counts for both.
 */
export function civilizationSplit(draft: DeckDraft): { civilization: string; count: number }[] {
  return CIVILIZATIONS.map(({ name }) => ({
    civilization: name,
    count: draft.cards
      .filter(({ card }) => card.civilizations.includes(name))
      .reduce((total, { count }) => total + count, 0),
  })).filter(({ count }) => count > 0);
}

/** The name for a copy of a deck, kept within the name limit. */
export function copyName(name: string): string {
  const suffix = ' (copy)';
  return `${name.slice(0, DECK_NAME_MAX_LENGTH - suffix.length).trimEnd()}${suffix}`;
}

function sortCards(cards: DraftCard[]): DraftCard[] {
  return cards.sort(
    (a, b) =>
      a.card.cost - b.card.cost ||
      compareText(a.card.name, b.card.name) ||
      compareText(a.card.id, b.card.id),
  );
}

/** Text order by code point, the same as the Go server's, so both list decks alike. */
function compareText(a: string, b: string): number {
  if (a === b) return 0;
  return a < b ? -1 : 1;
}
