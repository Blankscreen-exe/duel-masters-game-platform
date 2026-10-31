import type { CardSearch } from './card-search';
import { MAX_COPIES, type DeckDraft } from './deck-draft';

/** Which cards to show by whether the deck holds them. */
export type DeckCardsChoice = 'all' | 'in' | 'out';

/** The filter modal's choices. Number boxes are text, as typed; empty does not narrow. */
export interface CardFilterChoices {
  readonly text: string;
  readonly civilizations: readonly string[];
  readonly civilizationMatch: NonNullable<CardSearch['civilizationMatch']>;
  readonly civilizationCount: NonNullable<CardSearch['civilizationCount']> | '';
  readonly types: readonly string[];
  readonly races: readonly string[];
  readonly keywords: readonly string[];
  readonly sets: readonly string[];
  readonly rarities: readonly string[];
  readonly minCost: string;
  readonly maxCost: string;
  readonly minPower: string;
  readonly maxPower: string;
  readonly formatId: string;
  readonly deckCards: DeckCardsChoice;
  readonly hideFullCards: boolean;
  readonly sort: NonNullable<CardSearch['sort']>;
  readonly order: NonNullable<CardSearch['order']>;
}

export const NO_CARD_FILTERS: CardFilterChoices = {
  text: '',
  civilizations: [],
  civilizationMatch: 'any',
  civilizationCount: '',
  types: [],
  races: [],
  keywords: [],
  sets: [],
  rarities: [],
  minCost: '',
  maxCost: '',
  minPower: '',
  maxPower: '',
  formatId: '',
  deckCards: 'all',
  hideFullCards: false,
  sort: 'name',
  order: 'asc',
};

/**
 * No card has this id, so asking for only it finds nothing: how "only the deck's cards" asks for an
 * empty deck's, as an empty list would not narrow the search at all.
 */
const NO_CARD_ID = 'none';

/** How many things narrow the search, for the button to show. Sorting narrows nothing. */
export function countCardFilters(choices: CardFilterChoices): number {
  const narrowing = [
    choices.text.trim() !== '',
    choices.civilizations.length > 0,
    choices.civilizationCount !== '',
    choices.types.length > 0,
    choices.races.length > 0,
    choices.keywords.length > 0,
    choices.sets.length > 0,
    choices.rarities.length > 0,
    choices.minCost !== '' || choices.maxCost !== '',
    choices.minPower !== '' || choices.maxPower !== '',
    choices.formatId !== '',
    choices.deckCards !== 'all',
    choices.hideFullCards,
  ];
  return narrowing.filter(Boolean).length;
}

/** What is wrong with the choices, in words, or null when they can be searched for. */
export function cardFiltersProblem(choices: CardFilterChoices): string | null {
  return (
    rangeProblem('Cost', choices.minCost, choices.maxCost) ??
    rangeProblem('Power', choices.minPower, choices.maxPower)
  );
}

function rangeProblem(label: string, from: string, to: string): string | null {
  const bounds = [from, to].filter((bound) => bound !== '');
  if (bounds.some((bound) => !/^\d+$/.test(bound))) {
    return `${label} must be a whole number, 0 or more.`;
  }
  if (bounds.length === 2 && Number(from) > Number(to)) {
    return `${label}'s "from" must not be above its "to".`;
  }
  return null;
}

/** The cards the deck-based choices name, worked out from the deck as it stands. */
export function deckCardLists(
  choices: Pick<CardFilterChoices, 'deckCards' | 'hideFullCards'>,
  draft: DeckDraft,
): Pick<CardSearch, 'cardId' | 'exceptCardId'> {
  const inDeck = draft.cards.map(({ card }) => card.id);
  const full = draft.cards.filter(({ count }) => count >= MAX_COPIES).map(({ card }) => card.id);
  const except = choices.deckCards === 'out' ? inDeck : choices.hideFullCards ? full : [];

  return {
    ...(choices.deckCards === 'in' ? { cardId: inDeck.length > 0 ? inDeck : [NO_CARD_ID] } : {}),
    ...(except.length > 0 ? { exceptCardId: except } : {}),
  };
}

/** The choices as the API takes them, leaving out whatever does not narrow or sort. */
export function cardFiltersToSearch(choices: CardFilterChoices): CardSearch {
  const text = choices.text.trim();

  return {
    ...(text === '' ? {} : { text }),
    ...list('civilization', choices.civilizations),
    ...(choices.civilizations.length > 0 && choices.civilizationMatch !== 'any'
      ? { civilizationMatch: choices.civilizationMatch }
      : {}),
    ...(choices.civilizationCount === '' ? {} : { civilizationCount: choices.civilizationCount }),
    ...list('type', choices.types),
    ...list('race', choices.races),
    ...list('keyword', choices.keywords),
    ...list('set', choices.sets),
    ...list('rarity', choices.rarities),
    ...bound('minCost', choices.minCost),
    ...bound('maxCost', choices.maxCost),
    ...bound('minPower', choices.minPower),
    ...bound('maxPower', choices.maxPower),
    ...(choices.formatId === '' ? {} : { format: choices.formatId }),
    ...(choices.sort === 'name' ? {} : { sort: choices.sort }),
    ...(choices.order === 'asc' ? {} : { order: choices.order }),
  };
}

function list<K extends string>(name: K, values: readonly string[]) {
  return (values.length > 0 ? { [name]: [...values] } : {}) as Partial<Record<K, string[]>>;
}

function bound<K extends string>(name: K, value: string) {
  return (value === '' ? {} : { [name]: Number(value) }) as Partial<Record<K, number>>;
}

/** The game's names for abilities the card database's code names would not spell out. */
const KEYWORD_NAMES: Readonly<Record<string, string>> = {
  crew_break: 'Crew breaker',
  double_break: 'Double breaker',
  triple_break: 'Triple breaker',
  gravity_zero: 'G-Zero',
  over_drive: 'Overdrive',
  power_attack: 'Power attacker',
  speed_attack: 'Speed attacker',
};

/** An ability as players know it, from its code name in the card database. */
export function keywordLabel(keyword: string): string {
  const named = KEYWORD_NAMES[keyword];
  if (named !== undefined) return named;

  const words = keyword.replace(/_kw$/, '').replace(/_evo$/, '_evolution').split('_').join(' ');
  return words.charAt(0).toUpperCase() + words.slice(1);
}
