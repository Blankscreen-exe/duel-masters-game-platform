'use client';

import type { Card } from '@/lib/api/client';
import { CANNOT_ADD_TEXT, cannotAdd, copiesOf, type DeckDraft } from '@/lib/deck-draft';

/** The deck being edited, and the ways of changing it. */
export interface DeckEditing {
  readonly draft: DeckDraft;
  readonly onAdd: (card: Card) => void;
  readonly onRemove: (cardId: string) => void;
  /** Lets a card be chosen as the deck's cover, for a deck that has one. */
  readonly onCover?: (cardId: string | null) => void;
}

interface DeckCardButtonsProps {
  readonly card: Card;
  readonly editing: DeckEditing;
  /** Shows how many copies the deck has, between the buttons. */
  readonly showCount?: boolean;
}

const STEP =
  'grid size-7 shrink-0 place-items-center bg-slate-800 font-display text-base font-bold text-slate-200 ring-1 ring-slate-600/60 ring-inset transition hover:bg-slate-700 hover:text-white disabled:cursor-not-allowed disabled:opacity-40';

/** One fewer, one more, and the cover choice for a card of the deck being edited. */
export function DeckCardButtons({ card, editing, showCount = false }: DeckCardButtonsProps) {
  const { draft, onAdd, onRemove, onCover } = editing;
  const copies = copiesOf(draft, card.id);
  const reason = cannotAdd(draft, card);
  const isCover = draft.coverCardId === card.id;

  return (
    <>
      <button
        type="button"
        aria-label={`One fewer ${card.name}`}
        title="One fewer"
        className={STEP}
        disabled={copies === 0}
        onClick={() => {
          onRemove(card.id);
        }}
      >
        −
      </button>
      {showCount && (
        <span className="w-6 text-center font-display font-bold text-accent">{copies}</span>
      )}
      <button
        type="button"
        aria-label={`One more ${card.name}`}
        title={reason === null ? 'One more' : CANNOT_ADD_TEXT[reason]}
        className={STEP}
        disabled={reason !== null}
        onClick={() => {
          onAdd(card);
        }}
      >
        +
      </button>
      {onCover && copies > 0 && (
        <button
          type="button"
          aria-pressed={isCover}
          aria-label={
            isCover
              ? `${card.name} is the cover; use the site's pick`
              : `Use ${card.name} as the cover`
          }
          title={isCover ? 'The cover card' : 'Use as the cover'}
          className={`${STEP} ${isCover ? 'text-amber-300 hover:text-amber-200' : 'hover:text-amber-200'}`}
          onClick={() => {
            onCover(isCover ? null : card.id);
          }}
        >
          ★
        </button>
      )}
    </>
  );
}
