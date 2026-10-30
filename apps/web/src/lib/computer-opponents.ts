import type { components } from './api/schema';
import { DECK_SIZE, draftFromDeck, totalCards, type DeckDraft } from './deck-draft';

type Schemas = components['schemas'];
export type AdminComputerOpponent = Schemas['AdminComputerOpponent'];
export type ComputerOpponentInput = Schemas['ComputerOpponentInput'];
export type ComputerOpponentOptions = Schemas['ComputerOpponentOptions'];

/** The longest description, as the server allows. */
export const OPPONENT_DESCRIPTION_MAX_LENGTH = 200;

/** The card types a new opponent's deck may have: those the game plays today. */
const STARTING_CARD_TYPES = ['Creature', 'Evolution Creature', 'Spell'];

/** How many civilizations a deck's rules can ask for, with their words; 0 is any number. */
export const CIVILIZATION_LIMITS: readonly { readonly value: number; readonly label: string }[] = [
  { value: 0, label: 'Any number' },
  { value: 1, label: 'One' },
  { value: 2, label: 'Up to two' },
  { value: 3, label: 'Up to three' },
  { value: 4, label: 'Up to four' },
];

/** What an opponent is, apart from its deck, as its form edits it. */
export type OpponentForm = Omit<ComputerOpponentInput, 'cards'>;

/** The form of an opponent not made yet, with the first of the choices picked. */
export function newOpponentForm(
  options: ComputerOpponentOptions,
  firstFormatId: string,
): OpponentForm {
  return {
    name: '',
    description: '',
    style: options.styles[0]?.name ?? '',
    strength: 'normal',
    formatId: firstFormatId,
    maxCivilizations: 0,
    cardTypes: STARTING_CARD_TYPES,
  };
}

/** An opponent as its form shows it. */
export function formOf(opponent: AdminComputerOpponent): OpponentForm {
  return {
    name: opponent.name,
    description: opponent.description,
    style: opponent.style,
    strength: opponent.strength,
    formatId: opponent.formatId,
    maxCivilizations: opponent.maxCivilizations,
    cardTypes: opponent.cardTypes,
  };
}

/** An opponent's deck as the deck builder edits it; it has no name or cover of its own. */
export function opponentDraft(opponent: Pick<AdminComputerOpponent, 'cards'> | null): DeckDraft {
  return draftFromDeck({ name: '', cards: opponent?.cards ?? [] }).draft;
}

/** The form and deck as the API saves them. */
export function toOpponentInput(form: OpponentForm, draft: DeckDraft): ComputerOpponentInput {
  return {
    ...form,
    cards: draft.cards.map(({ card, count }) => ({ cardId: card.id, count })),
  };
}

/** Whether two forms would save the same opponent. */
export function sameForm(a: OpponentForm, b: OpponentForm): boolean {
  const key = (form: OpponentForm) =>
    JSON.stringify([
      form.name.trim(),
      form.description.trim(),
      form.style,
      form.strength,
      form.formatId,
      form.maxCivilizations,
      [...form.cardTypes].sort(),
    ]);
  return key(a) === key(b);
}

/** The form with a card type allowed or not. */
export function withCardType(form: OpponentForm, cardType: string, allowed: boolean): OpponentForm {
  const others = form.cardTypes.filter((each) => each !== cardType);
  return { ...form, cardTypes: allowed ? [...others, cardType] : others };
}

/**
 * Whether the draft keeps to the form's rules, in words, as the admin builds it. Only the server
 * knows the format's rules, which it checks on saving; these are the rules this page has at hand.
 */
export function rulesBroken(form: OpponentForm, draft: DeckDraft): string[] {
  const broken: string[] = [];

  const total = totalCards(draft);
  if (total !== DECK_SIZE) {
    broken.push(`The deck has ${String(total)} cards, not ${String(DECK_SIZE)}.`);
  }

  const civilizations = new Set(draft.cards.flatMap(({ card }) => card.civilizations));
  if (form.maxCivilizations > 0 && civilizations.size > form.maxCivilizations) {
    broken.push(
      `The deck has ${String(civilizations.size)} civilizations, and its rules allow ${String(form.maxCivilizations)}.`,
    );
  }

  const barred = draft.cards.filter(({ card }) => !form.cardTypes.includes(card.type));
  for (const { card } of barred) {
    broken.push(`${card.name} is a ${card.type}, which its rules do not allow.`);
  }

  return broken;
}

/** An opponent's deck rules, in a few words. */
export function rulesText(opponent: AdminComputerOpponent): string {
  const limit = CIVILIZATION_LIMITS.find(({ value }) => value === opponent.maxCivilizations);
  const civilizations = limit ? limit.label.toLowerCase() : String(opponent.maxCivilizations);
  return `Civilizations: ${civilizations}`;
}
