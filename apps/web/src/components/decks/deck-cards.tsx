'use client';

import { CardImage } from '@/components/cards/card-image';
import type { Card } from '@/lib/api/client';
import { groupByCost } from '@/lib/deck-draft';
import { DeckCardButtons, type DeckEditing } from './deck-card-buttons';
import { useCardDrag } from './deck-drag';

interface DeckCardsProps {
  readonly editing: DeckEditing;
  readonly onEnlarge: (card: Card) => void;
}

/**
 * The deck laid out as cards, by cost: four copies of a card are four cards. Picking one enlarges
 * it and dragging it to the trunk removes it; its buttons, which do the same as dragging, show when
 * pointed at, and always on a touch screen, which cannot point.
 */
export function DeckCards({ editing, onEnlarge }: DeckCardsProps) {
  const groups = groupByCost(editing.draft);

  if (groups.length === 0) {
    return (
      <p className="px-4 py-16 text-center text-sm text-slate-400">
        The deck is empty. Add cards from the trunk
        <span className="max-lg:hidden">: drag them here, or use their + button</span>.
      </p>
    );
  }

  return (
    <div className="divide-y divide-slate-800">
      {groups.map(({ cost, cards }) => (
        <section key={cost} aria-label={`Cost ${String(cost)}`} className="p-3">
          <h3 className="mb-2 font-display text-xs font-bold tracking-widest text-slate-500 uppercase">
            Cost {cost}
          </h3>
          <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-5 xl:grid-cols-6">
            {cards.flatMap(({ card, count }) =>
              Array.from({ length: count }, (_, copy) => (
                // Copies of a card are alike, so which one goes does not matter.
                <DeckCopy
                  key={`${card.id}-${String(copy)}`}
                  card={card}
                  copy={copy}
                  editing={editing}
                  onEnlarge={onEnlarge}
                />
              )),
            )}
          </ul>
        </section>
      ))}
    </div>
  );
}

interface DeckCopyProps extends DeckCardsProps {
  readonly card: Card;
  /** Which copy of the card this is, from 0. */
  readonly copy: number;
}

/** One copy of a card in the deck. */
function DeckCopy({ card, copy, editing, onEnlarge }: DeckCopyProps) {
  const { ref, grip, dragging } = useCardDrag(card, 'deck', copy);

  return (
    <li ref={ref} className={`group relative ${dragging ? 'opacity-30' : ''}`}>
      <button
        type="button"
        aria-label={`Enlarge ${card.name}`}
        title={card.name}
        onClick={() => {
          onEnlarge(card);
        }}
        {...grip}
        className="block w-full cursor-zoom-in transition select-none [-webkit-touch-callout:none] hover:-translate-y-0.5"
      >
        <CardImage
          src={card.imageUrl}
          name={card.name}
          className="w-full transition group-hover:shadow-[0_0_16px_rgb(56_189_248/0.4)]"
        />
      </button>
      {editing.draft.coverCardId === card.id && copy === 0 && (
        <span
          title="The cover card"
          className="absolute top-1 left-1 grid size-6 place-items-center bg-night/90 text-sm text-amber-300"
        >
          ★<span className="sr-only">The cover card</span>
        </span>
      )}
      <div className="absolute inset-x-0 bottom-0 flex justify-center gap-0.5 bg-night/80 py-1 opacity-0 transition group-focus-within:opacity-100 group-hover:opacity-100 pointer-coarse:opacity-100">
        <DeckCardButtons card={card} editing={editing} />
      </div>
    </li>
  );
}
