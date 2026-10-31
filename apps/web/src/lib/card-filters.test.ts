import { describe, expect, it } from 'vitest';
import type { Card } from './api/client';
import {
  cardFiltersProblem,
  cardFiltersToSearch,
  countCardFilters,
  deckCardLists,
  keywordLabel,
  NO_CARD_FILTERS,
} from './card-filters';
import type { DeckDraft } from './deck-draft';

function draftOf(...cards: [string, number][]): DeckDraft {
  return {
    name: 'Deck',
    coverCardId: null,
    cards: cards.map(([id, count]) => ({ card: { id } as unknown as Card, count })),
  };
}

describe('countCardFilters', () => {
  it('counts nothing when nothing narrows the search', () => {
    expect(countCardFilters(NO_CARD_FILTERS)).toBe(0);
    expect(countCardFilters({ ...NO_CARD_FILTERS, text: '   ' })).toBe(0);
  });

  it('does not count sorting, or how civilizations match, as narrowing', () => {
    expect(
      countCardFilters({
        ...NO_CARD_FILTERS,
        sort: 'power',
        order: 'desc',
        civilizationMatch: 'only',
      }),
    ).toBe(0);
  });

  it('counts each kind of filter once, however many values it has', () => {
    expect(
      countCardFilters({
        ...NO_CARD_FILTERS,
        text: 'draw',
        civilizations: ['Fire', 'Water'],
        civilizationCount: 'single',
        races: ['Human', 'Fish'],
        minCost: '2',
        maxCost: '5',
        maxPower: '3000',
        formatId: 'open',
        deckCards: 'in',
        hideFullCards: true,
      }),
    ).toBe(9);
  });
});

describe('cardFiltersProblem', () => {
  it('accepts empty and sensible ranges', () => {
    expect(cardFiltersProblem(NO_CARD_FILTERS)).toBeNull();
    expect(cardFiltersProblem({ ...NO_CARD_FILTERS, minCost: '2', maxCost: '2' })).toBeNull();
    expect(cardFiltersProblem({ ...NO_CARD_FILTERS, maxPower: '0' })).toBeNull();
  });

  it('refuses a range the wrong way round', () => {
    expect(cardFiltersProblem({ ...NO_CARD_FILTERS, minCost: '5', maxCost: '2' })).toBe(
      'Cost\'s "from" must not be above its "to".',
    );
    expect(cardFiltersProblem({ ...NO_CARD_FILTERS, minPower: '10', maxPower: '9' })).toBe(
      'Power\'s "from" must not be above its "to".',
    );
  });

  it('refuses anything but a whole number', () => {
    for (const bad of ['-1', '1.5', 'lots']) {
      expect(cardFiltersProblem({ ...NO_CARD_FILTERS, minPower: bad })).toBe(
        'Power must be a whole number, 0 or more.',
      );
    }
  });
});

describe('deckCardLists', () => {
  const draft = draftOf(['a', 4], ['b', 1]);

  it('names nothing when asked for no deck-based choice', () => {
    expect(deckCardLists(NO_CARD_FILTERS, draft)).toEqual({});
  });

  it('names the deck’s cards, to show only them', () => {
    expect(deckCardLists({ deckCards: 'in', hideFullCards: false }, draft)).toEqual({
      cardId: ['a', 'b'],
    });
  });

  it('finds nothing in an empty deck, rather than everything', () => {
    expect(deckCardLists({ deckCards: 'in', hideFullCards: false }, draftOf())).toEqual({
      cardId: ['none'],
    });
  });

  it('names the deck’s cards, to leave them out', () => {
    expect(deckCardLists({ deckCards: 'out', hideFullCards: false }, draft)).toEqual({
      exceptCardId: ['a', 'b'],
    });
    expect(deckCardLists({ deckCards: 'out', hideFullCards: false }, draftOf())).toEqual({});
  });

  it('leaves out only cards with every copy the game allows', () => {
    expect(deckCardLists({ deckCards: 'all', hideFullCards: true }, draft)).toEqual({
      exceptCardId: ['a'],
    });
  });

  it('can show the deck’s cards without its full ones', () => {
    expect(deckCardLists({ deckCards: 'in', hideFullCards: true }, draft)).toEqual({
      cardId: ['a', 'b'],
      exceptCardId: ['a'],
    });
  });
});

describe('cardFiltersToSearch', () => {
  it('asks for nothing when nothing is chosen', () => {
    expect(cardFiltersToSearch(NO_CARD_FILTERS)).toEqual({});
  });

  it('turns the choices into what the API takes', () => {
    expect(
      cardFiltersToSearch({
        ...NO_CARD_FILTERS,
        text: ' draw a card ',
        civilizations: ['Fire'],
        civilizationMatch: 'exactly',
        civilizationCount: 'multi',
        types: ['Spell'],
        races: ['Human'],
        keywords: ['blocker'],
        sets: ['DM-01'],
        rarities: ['sr'],
        minCost: '0',
        maxCost: '5',
        minPower: '1000',
        maxPower: '6000',
        formatId: 'classic',
        sort: 'power',
        order: 'desc',
      }),
    ).toEqual({
      text: 'draw a card',
      civilization: ['Fire'],
      civilizationMatch: 'exactly',
      civilizationCount: 'multi',
      type: ['Spell'],
      race: ['Human'],
      keyword: ['blocker'],
      set: ['DM-01'],
      rarity: ['sr'],
      minCost: 0,
      maxCost: 5,
      minPower: 1000,
      maxPower: 6000,
      format: 'classic',
      sort: 'power',
      order: 'desc',
    });
  });

  it('leaves out how civilizations match when none are chosen', () => {
    expect(cardFiltersToSearch({ ...NO_CARD_FILTERS, civilizationMatch: 'only' })).toEqual({});
  });
});

describe('keywordLabel', () => {
  it('uses the game’s own name where it differs', () => {
    expect(keywordLabel('double_break')).toBe('Double breaker');
    expect(keywordLabel('speed_attack')).toBe('Speed attacker');
    expect(keywordLabel('gravity_zero')).toBe('G-Zero');
  });

  it('spells out the rest', () => {
    expect(keywordLabel('blocker')).toBe('Blocker');
    expect(keywordLabel('shield_trigger')).toBe('Shield trigger');
    expect(keywordLabel('survivor_kw')).toBe('Survivor');
    expect(keywordLabel('vortex_evo')).toBe('Vortex evolution');
  });
});
