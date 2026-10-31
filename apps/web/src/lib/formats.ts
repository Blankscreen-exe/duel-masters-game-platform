import type { Card } from './api/client';
import type { components } from './api/schema';

export type AdminFormat = components['schemas']['AdminFormat'];
export type FormatInput = components['schemas']['FormatInput'];

/** The API's limits on a format's words; formats.test.ts keeps these equal to the contract's. */
export const FORMAT_NAME_MAX_LENGTH = 40;
export const FORMAT_DESCRIPTION_MAX_LENGTH = 200;

/** The most copies a limit can allow: 0 is banned; anything above 2 is left to the game's rules. */
export const LIMIT_CHOICES = [
  { maxCopies: 0, label: 'Banned' },
  { maxCopies: 1, label: 'Limited to 1' },
  { maxCopies: 2, label: 'Limited to 2' },
] as const;

/** A card a format bans or limits, in the editor. A card gone from the catalogue has no details. */
export interface FormatLimit {
  readonly cardId: string;
  readonly maxCopies: number;
  readonly card?: Card;
}

/** A format being edited. */
export interface FormatForm {
  readonly name: string;
  readonly description: string;
  readonly allSets: boolean;
  /** The sets chosen, kept even while every set is allowed, in case that is changed back. */
  readonly sets: readonly string[];
  readonly limits: readonly FormatLimit[];
}

export const NEW_FORMAT: FormatForm = {
  name: '',
  description: '',
  allSets: true,
  sets: [],
  limits: [],
};

export function formOfFormat(format: AdminFormat): FormatForm {
  return {
    name: format.name,
    description: format.description,
    allSets: format.allSets,
    sets: format.sets,
    limits: format.limits.map(({ cardId, maxCopies, card }) => ({
      cardId,
      maxCopies,
      ...(card ? { card } : {}),
    })),
  };
}

/** The form as the API saves it. */
export function toFormatInput(form: FormatForm): FormatInput {
  return {
    name: form.name,
    description: form.description,
    allSets: form.allSets,
    sets: form.allSets ? [] : [...form.sets],
    limits: form.limits.map(({ cardId, maxCopies }) => ({ cardId, maxCopies })),
  };
}

/** Whether two forms would save the same format. */
export function sameFormatForm(a: FormatForm, b: FormatForm): boolean {
  const key = (form: FormatForm) =>
    JSON.stringify([
      form.name.trim(),
      form.description.trim(),
      form.allSets,
      form.allSets ? [] : [...form.sets].sort(),
      form.limits.map(({ cardId, maxCopies }) => [cardId, maxCopies]).sort(),
    ]);
  return key(a) === key(b);
}

/** What stops the form being saved, in words, or null when nothing does. */
export function formatFormProblem(form: FormatForm): string | null {
  if (form.name.trim() === '') return 'The format needs a name.';
  if (form.description.trim() === '') return 'The format needs a description.';
  if (!form.allSets && form.sets.length === 0) return 'Choose at least one set.';
  return null;
}

/** The form with a card banned or limited, or with its limit changed; kept in order of name. */
export function withLimit(form: FormatForm, card: Card, maxCopies: number): FormatForm {
  const others = form.limits.filter(({ cardId }) => cardId !== card.id);
  const limits = [...others, { cardId: card.id, maxCopies, card }].sort((a, b) =>
    (a.card?.name ?? '').localeCompare(b.card?.name ?? ''),
  );
  return { ...form, limits };
}

/** The form with a card's limit changed, for a limit already on it, whatever its details. */
export function changedLimit(form: FormatForm, cardId: string, maxCopies: number): FormatForm {
  return {
    ...form,
    limits: form.limits.map((limit) => (limit.cardId === cardId ? { ...limit, maxCopies } : limit)),
  };
}

export function withoutLimit(form: FormatForm, cardId: string): FormatForm {
  return { ...form, limits: form.limits.filter((limit) => limit.cardId !== cardId) };
}

/** What a format allows, in a few words. */
export function formatRulesText(format: AdminFormat): string {
  const sets = format.allSets
    ? 'Every set'
    : format.sets.length === 1
      ? `Set ${format.sets[0] ?? ''}`
      : `${String(format.sets.length)} sets`;
  const banned = format.limits.filter(({ maxCopies }) => maxCopies === 0).length;
  const limited = format.limits.length - banned;

  return [
    sets,
    ...(banned > 0 ? [`${String(banned)} banned`] : []),
    ...(limited > 0 ? [`${String(limited)} limited`] : []),
  ].join(' · ');
}

/** The ids in a new order, with one moved a place up or down, or null if it cannot move that way. */
export function movedFormat(
  ids: readonly string[],
  id: string,
  direction: 'up' | 'down',
): string[] | null {
  const from = ids.indexOf(id);
  const to = direction === 'up' ? from - 1 : from + 1;
  if (from === -1 || to < 0 || to >= ids.length) return null;

  const moved = [...ids];
  moved.splice(from, 1);
  moved.splice(to, 0, id);
  return moved;
}

/**
 * The formats a computer opponent's editor offers: those players can choose, and its own even if
 * retired, marked so, so that the choice it has can be seen and kept.
 */
export function formatsForOpponent(all: readonly AdminFormat[], own: string): AdminFormat[] {
  return all
    .filter(({ id, retired }) => !retired || id === own)
    .map((format) => (format.retired ? { ...format, name: `${format.name} (retired)` } : format));
}
