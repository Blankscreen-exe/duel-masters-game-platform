import { describe, expect, it } from 'vitest';
import type { Card } from './api/client';
import { DECK_SIZE, MAX_COPIES, type DeckDraft } from './deck-draft';
import { deckDrop } from './deck-drop';

function card(id: string, playable = true): Card {
  return { id, name: `Card ${id}`, cost: 1, playable } as unknown as Card;
}

function draftOf(...cards: { card: Card; count: number }[]): DeckDraft {
  return { name: 'Deck', coverCardId: null, cards };
}

describe('deckDrop', () => {
  const fire = card('fire');

  it('adds a copy dragged from the trunk to the deck', () => {
    expect(deckDrop(draftOf(), fire, 'trunk', 'deck')).toEqual({
      action: 'add',
      text: 'Add to the deck',
    });
  });

  it('removes a copy dragged from the deck to the trunk', () => {
    const draft = draftOf({ card: fire, count: 2 });

    expect(deckDrop(draft, fire, 'deck', 'trunk')).toEqual({
      action: 'remove',
      text: 'Remove from the deck',
    });
  });

  it('says why a card cannot go in', () => {
    const atMost = draftOf({ card: fire, count: MAX_COPIES });
    const full = draftOf({ card: card('other'), count: DECK_SIZE });

    expect(deckDrop(atMost, fire, 'trunk', 'deck')).toEqual({
      action: 'none',
      text: `The deck has ${String(MAX_COPIES)} copies already`,
    });
    expect(deckDrop(full, fire, 'trunk', 'deck')).toEqual({
      action: 'none',
      text: `The deck has ${String(DECK_SIZE)} cards already`,
    });
    expect(deckDrop(draftOf(), card('new', false), 'trunk', 'deck').action).toBe('none');
  });

  it('does nothing where the card came from', () => {
    const nothing = { action: 'none', text: null };

    expect(deckDrop(draftOf(), fire, 'trunk', 'trunk')).toEqual(nothing);
    expect(deckDrop(draftOf({ card: fire, count: 1 }), fire, 'deck', 'deck')).toEqual(nothing);
  });

  it('does nothing away from both places', () => {
    const nothing = { action: 'none', text: null };

    expect(deckDrop(draftOf(), fire, 'trunk', null)).toEqual(nothing);
    expect(deckDrop(draftOf({ card: fire, count: 1 }), fire, 'deck', null)).toEqual(nothing);
  });
});
