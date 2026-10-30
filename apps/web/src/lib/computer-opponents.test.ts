import { describe, expect, it } from 'vitest';
import type { Card } from './api/client';
import {
  formOf,
  newOpponentForm,
  opponentDraft,
  rulesBroken,
  rulesText,
  sameForm,
  toOpponentInput,
  withCardType,
  type AdminComputerOpponent,
  type OpponentForm,
} from './computer-opponents';
import type { DeckDraft } from './deck-draft';

function card(id: string, type: string, civilizations: string[]): Card {
  return {
    id,
    name: `Card ${id}`,
    type,
    civilizations,
    cost: 1,
    playable: true,
  } as unknown as Card;
}

function draftOf(...cards: { card: Card; count: number }[]): DeckDraft {
  return { name: '', coverCardId: null, cards };
}

const form: OpponentForm = {
  name: 'Grim',
  description: 'Hits hard.',
  style: 'rush',
  strength: 'normal',
  formatId: 'open',
  maxCivilizations: 1,
  cardTypes: ['Creature'],
};

const saved: AdminComputerOpponent = {
  ...form,
  id: 'o1',
  formatName: 'Open',
  offered: true,
  cards: [{ cardId: 'a', count: 4, card: card('a', 'Creature', ['Fire']) }],
  cardCount: 4,
  civilizations: ['Fire'],
  problems: [],
};

describe('newOpponentForm', () => {
  it('starts with the first choices and the card types the game plays', () => {
    const options = { styles: [{ name: 'balanced', label: 'Balanced' }], cardTypes: ['Castle'] };

    expect(newOpponentForm(options, 'open')).toEqual({
      name: '',
      description: '',
      style: 'balanced',
      strength: 'normal',
      formatId: 'open',
      maxCivilizations: 0,
      cardTypes: ['Creature', 'Evolution Creature', 'Spell'],
    });
  });
});

describe('formOf and opponentDraft', () => {
  it('splits an opponent into what its form and the deck builder edit', () => {
    expect(formOf(saved)).toEqual(form);
    expect(opponentDraft(saved).cards.map(({ count }) => count)).toEqual([4]);
    expect(opponentDraft(null).cards).toEqual([]);
  });

  it('leaves out cards gone from the card database', () => {
    expect(opponentDraft({ cards: [{ cardId: 'gone', count: 4 }] }).cards).toEqual([]);
  });
});

describe('toOpponentInput', () => {
  it('is what the API saves', () => {
    const draft = draftOf({ card: card('a', 'Creature', ['Fire']), count: 3 });

    expect(toOpponentInput(form, draft)).toEqual({ ...form, cards: [{ cardId: 'a', count: 3 }] });
  });
});

describe('sameForm', () => {
  it('ignores spaces around the words and the order of the card types', () => {
    const types = { ...form, cardTypes: ['Spell', 'Creature'] };

    expect(sameForm({ ...form, name: ' Grim ' }, form)).toBe(true);
    expect(sameForm(types, { ...types, cardTypes: ['Creature', 'Spell'] })).toBe(true);
  });

  it('sees a change', () => {
    expect(sameForm({ ...form, style: 'wall' }, form)).toBe(false);
    expect(sameForm({ ...form, maxCivilizations: 2 }, form)).toBe(false);
  });
});

describe('withCardType', () => {
  it('allows and bars a card type', () => {
    expect(withCardType(form, 'Spell', true).cardTypes).toEqual(['Creature', 'Spell']);
    expect(withCardType(form, 'Creature', false).cardTypes).toEqual([]);
    expect(withCardType(form, 'Creature', true).cardTypes).toEqual(['Creature']);
  });
});

describe('rulesBroken', () => {
  const fire = card('a', 'Creature', ['Fire']);

  it('finds nothing wrong with a whole deck keeping to the rules', () => {
    expect(rulesBroken(form, draftOf({ card: fire, count: 40 }))).toEqual([]);
  });

  it('counts the cards', () => {
    expect(rulesBroken(form, draftOf({ card: fire, count: 4 }))).toEqual([
      'The deck has 4 cards, not 40.',
    ]);
  });

  it('counts the civilizations, unless any number will do', () => {
    const draft = draftOf(
      { card: fire, count: 20 },
      { card: card('b', 'Creature', ['Water', 'Darkness']), count: 20 },
    );

    expect(rulesBroken(form, draft)).toEqual([
      'The deck has 3 civilizations, and its rules allow 1.',
    ]);
    expect(rulesBroken({ ...form, maxCivilizations: 0 }, draft)).toEqual([]);
  });

  it('names cards of a type the rules do not allow', () => {
    const draft = draftOf(
      { card: fire, count: 36 },
      { card: card('s', 'Spell', ['Fire']), count: 4 },
    );

    expect(rulesBroken(form, draft)).toEqual(['Card s is a Spell, which its rules do not allow.']);
  });
});

describe('rulesText', () => {
  it('says what the rules ask for', () => {
    expect(rulesText(saved)).toBe('Civilizations: one');
    expect(rulesText({ ...saved, maxCivilizations: 0 })).toBe('Civilizations: any number');
    expect(rulesText({ ...saved, maxCivilizations: 5 })).toBe('Civilizations: 5');
  });
});
