import type { components } from './api/schema';

type CardAbility = components['schemas']['CardAbility'];

/**
 * An ability line split into its rule and its reminder text, so the reminder can be shown in a
 * quieter style. The card database's line already contains the reminder, usually at its end; a
 * reminder found elsewhere, or not found, leaves the line whole.
 */
export function splitReminder(ability: CardAbility): { rule: string; reminder?: string } {
  const { text, reminder } = ability;

  if (!reminder || !text.endsWith(reminder)) return { rule: text };

  const rule = text.slice(0, -reminder.length).trimEnd();
  return rule === '' ? { rule: text } : { rule, reminder };
}
