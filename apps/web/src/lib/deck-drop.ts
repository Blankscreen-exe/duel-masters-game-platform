import type { Card } from './api/client';
import { CANNOT_ADD_TEXT, cannotAdd, type DeckDraft } from './deck-draft';

/** The two places a card can be dragged between in the deck editor. */
export type DeckZone = 'trunk' | 'deck';

/** What letting go of a dragged card does, and the words to show while it is held there. */
export interface DeckDrop {
  readonly action: 'add' | 'remove' | 'none';
  /** Null over nowhere that means anything. */
  readonly text: string | null;
}

const NOTHING: DeckDrop = { action: 'none', text: null };

/**
 * What dropping a card does: from the trunk to the deck adds a copy, if the rules allow; from the
 * deck to the trunk takes one out. Anywhere else the card simply goes back.
 */
export function deckDrop(
  draft: DeckDraft,
  card: Card,
  from: DeckZone,
  over: DeckZone | null,
): DeckDrop {
  if (over === null || over === from) return NOTHING;
  if (over === 'trunk') return { action: 'remove', text: 'Remove from the deck' };

  const reason = cannotAdd(draft, card);
  return reason === null
    ? { action: 'add', text: 'Add to the deck' }
    : { action: 'none', text: CANNOT_ADD_TEXT[reason] };
}
