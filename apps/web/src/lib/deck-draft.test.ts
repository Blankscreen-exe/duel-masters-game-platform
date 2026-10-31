import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { parse } from 'yaml';
import type { Card, Deck } from './api/client';
import {
  addCard,
  cannotAdd,
  civilizationSplit,
  copiesOf,
  copyName,
  DECK_LIST_MAX_LENGTH,
  DECK_NAME_MAX_LENGTH,
  DECK_SIZE,
  draftFromDeck,
  groupByType,
  manaCurve,
  MAX_COPIES,
  removeCard,
  sameDeck,
  toDeckInput,
  totalCards,
  type DeckDraft,
} from './deck-draft';

function card(id: string, name: string, cost: number, overrides: Partial<Card> = {}): Card {
  return {
    id,
    name,
    type: 'Creature',
    cost,
    powerText: '1000',
    civilizations: ['Fire'],
    races: [],
    rulesText: '',
    abilities: [],
    keywords: [],
    categories: [],
    playable: true,
    printings: [],
    ...overrides,
  };
}

const bolshack = card('0079', 'Bolshack Dragon', 6);
const hulcus = card('0033', 'Aqua Hulcus', 3, { civilizations: ['Water'] });
const zaschack = card('9026', 'Uberdragon Zaschack', 9, { civilizations: ['Fire', 'Darkness'] });
const future = card('1500', 'Future Card', 4, { playable: false });

const empty: DeckDraft = { name: 'Deck', coverCardId: null, cards: [] };

function withCopies(draft: DeckDraft, added: Card, copies: number): DeckDraft {
  let next = draft;
  for (let i = 0; i < copies; i++) next = addCard(next, added);
  return next;
}

/** A draft of ten different cards, four copies each: a full deck. */
function fullDraft(): DeckDraft {
  let draft = empty;
  for (let i = 1; i <= 10; i++) {
    draft = withCopies(draft, card(`f${String(i)}`, `Filler ${String(i)}`, i), 4);
  }
  return draft;
}

describe('the deck rules', () => {
  const contract = parse(
    readFileSync(fileURLToPath(new URL('../../../../api/openapi.yaml', import.meta.url)), 'utf8'),
  ) as {
    components: {
      schemas: {
        DeckInput: { properties: { name: { maxLength: number }; cards: { maxItems: number } } };
        DeckEntry: { properties: { count: { maximum: number } } };
        DeckListText: { properties: { text: { maxLength: number } } };
      };
    };
  };
  const { DeckInput, DeckEntry, DeckListText } = contract.components.schemas;

  it('match the API contract', () => {
    expect(DECK_NAME_MAX_LENGTH).toBe(DeckInput.properties.name.maxLength);
    expect(DECK_SIZE).toBe(DeckInput.properties.cards.maxItems);
    expect(MAX_COPIES).toBe(DeckEntry.properties.count.maximum);
    expect(DECK_LIST_MAX_LENGTH).toBe(DeckListText.properties.text.maxLength);
  });
});

describe('adding and removing cards', () => {
  it('keeps the cards by cost, then name, counting copies', () => {
    let draft = addCard(empty, bolshack);
    draft = addCard(draft, hulcus);
    draft = addCard(draft, bolshack);

    expect(draft.cards.map(({ card: { name }, count }) => [name, count])).toEqual([
      ['Aqua Hulcus', 1],
      ['Bolshack Dragon', 2],
    ]);
    expect(totalCards(draft)).toBe(3);
    expect(copiesOf(draft, bolshack.id)).toBe(2);
    expect(copiesOf(draft, zaschack.id)).toBe(0);
  });

  it('stops at four copies of a card', () => {
    const draft = withCopies(empty, bolshack, 5);

    expect(copiesOf(draft, bolshack.id)).toBe(MAX_COPIES);
    expect(cannotAdd(draft, bolshack)).toBe('max_copies');
  });

  it('stops at forty cards', () => {
    const draft = fullDraft();

    expect(totalCards(draft)).toBe(DECK_SIZE);
    expect(cannotAdd(draft, hulcus)).toBe('deck_full');
    expect(addCard(draft, hulcus)).toBe(draft);
  });

  it('never adds a card the game cannot play', () => {
    expect(cannotAdd(empty, future)).toBe('not_playable');
    expect(addCard(empty, future)).toBe(empty);
    expect(cannotAdd(empty, bolshack)).toBeNull();
  });

  it('takes out a card with its last copy, and its cover with it', () => {
    let draft: DeckDraft = { ...withCopies(empty, bolshack, 2), coverCardId: bolshack.id };

    draft = removeCard(draft, bolshack.id);
    expect(copiesOf(draft, bolshack.id)).toBe(1);
    expect(draft.coverCardId).toBe(bolshack.id);

    draft = removeCard(draft, bolshack.id);
    expect(draft.cards).toEqual([]);
    expect(draft.coverCardId).toBeNull();

    expect(removeCard(draft, bolshack.id)).toEqual(draft);
  });
});

describe('saved decks and drafts', () => {
  const saved: Deck = {
    id: 'd1',
    name: 'Fire Rush',
    coverCardId: '0079',
    cards: [
      { cardId: '0079', count: 4, card: bolshack },
      { cardId: 'gone', count: 2 },
      { cardId: '0033', count: 1, card: hulcus },
    ],
    cardCount: 7,
    civilizations: ['Water', 'Fire'],
    legality: [],
    createdAt: '2026-10-25T00:00:00Z',
    updatedAt: '2026-10-25T00:00:00Z',
  };

  it('leaves out cards that have left the card database, counting them', () => {
    const { draft, dropped } = draftFromDeck(saved);

    expect(dropped).toBe(2);
    expect(draft).toEqual({
      name: 'Fire Rush',
      coverCardId: '0079',
      cards: [
        { card: hulcus, count: 1 },
        { card: bolshack, count: 4 },
      ],
    });
  });

  it('forgets a cover that is no longer in the deck', () => {
    const { draft } = draftFromDeck({ ...saved, coverCardId: 'gone' });
    expect(draft.coverCardId).toBeNull();

    const { draft: unset } = draftFromDeck({ ...saved, coverCardId: undefined });
    expect(unset.coverCardId).toBeNull();
  });

  it('saves the draft as the API takes it', () => {
    const { draft } = draftFromDeck(saved);

    expect(toDeckInput(draft)).toEqual({
      name: 'Fire Rush',
      coverCardId: '0079',
      cards: [
        { cardId: '0033', count: 1 },
        { cardId: '0079', count: 4 },
      ],
    });
    expect(toDeckInput({ ...draft, coverCardId: null })).not.toHaveProperty('coverCardId');
  });

  it('tells a changed draft from an unchanged one', () => {
    const { draft } = draftFromDeck(saved);

    expect(sameDeck(draft, { ...draft, name: '  Fire Rush ' })).toBe(true);
    expect(sameDeck(draft, { ...draft, name: 'Water Rush' })).toBe(false);
    expect(sameDeck(draft, { ...draft, coverCardId: null })).toBe(false);
    expect(sameDeck(draft, addCard(draft, hulcus))).toBe(false);
    expect(sameDeck(draft, removeCard(addCard(draft, hulcus), hulcus.id))).toBe(true);
  });
});

describe('deck statistics', () => {
  const draft = withCopies(withCopies(withCopies(empty, hulcus, 3), bolshack, 4), zaschack, 1);

  describe('groupByType', () => {
    const typed = (type: string, cost: number) => card(type, `A ${type}`, cost, { type });
    const shown = (groups: ReturnType<typeof groupByType>) =>
      groups.map(({ type, total, cards }) => [
        type,
        total,
        cards.map(({ card: { name } }) => name),
      ]);

    it('counts copies and keeps each type in order of cost, then name', () => {
      expect(shown(groupByType(draft))).toEqual([
        ['Creature', 8, ['Aqua Hulcus', 'Bolshack Dragon', 'Uberdragon Zaschack']],
      ]);
    });

    it('shows creatures, then evolution creatures, then spells, whatever their costs', () => {
      let mixed = withCopies(empty, typed('Spell', 1), 2);
      mixed = withCopies(mixed, typed('Evolution Creature', 2), 1);
      mixed = withCopies(mixed, typed('Creature', 9), 3);

      expect(groupByType(mixed).map(({ type }) => type)).toEqual([
        'Creature',
        'Evolution Creature',
        'Spell',
      ]);
    });

    it('puts a type it does not know last, by name', () => {
      let mixed = withCopies(empty, typed('Zeta', 1), 1);
      mixed = withCopies(mixed, typed('Alpha', 1), 1);
      mixed = withCopies(mixed, typed('Castle', 9), 1);

      expect(groupByType(mixed).map(({ type }) => type)).toEqual(['Castle', 'Alpha', 'Zeta']);
    });

    it('has no groups for an empty deck', () => {
      expect(groupByType(empty)).toEqual([]);
    });
  });

  it('counts the mana curve, with the dearest cards together in the last bar', () => {
    expect(manaCurve(draft)).toEqual([
      { label: '1', count: 0 },
      { label: '2', count: 0 },
      { label: '3', count: 3 },
      { label: '4', count: 0 },
      { label: '5', count: 0 },
      { label: '6', count: 4 },
      { label: '7', count: 0 },
      { label: '8+', count: 1 },
    ]);
  });

  it('splits the cards by civilization in the game order, counting a two-civilization card for both', () => {
    expect(civilizationSplit(draft)).toEqual([
      { civilization: 'Water', count: 3 },
      { civilization: 'Darkness', count: 1 },
      { civilization: 'Fire', count: 5 },
    ]);
    expect(civilizationSplit(empty)).toEqual([]);
  });
});

describe('copyName', () => {
  it('adds "(copy)" within the name limit', () => {
    expect(copyName('Fire Rush')).toBe('Fire Rush (copy)');

    const long = copyName('x'.repeat(DECK_NAME_MAX_LENGTH));
    expect(long).toHaveLength(DECK_NAME_MAX_LENGTH);
    expect(long.endsWith(' (copy)')).toBe(true);
  });
});
