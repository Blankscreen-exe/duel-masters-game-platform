'use client';

import { CivilizationMarks } from '@/components/cards/civilization-marks';
import { cannotAdd, groupByCost, type DeckDraft } from '@/lib/deck-draft';
import type { Card } from '@/lib/api/client';

interface DeckListProps {
  readonly draft: DeckDraft;
  readonly onAdd: (card: Card) => void;
  readonly onRemove: (cardId: string) => void;
  /** Lets a card be chosen as the deck's cover, for a deck that has one. */
  readonly onCover?: (cardId: string | null) => void;
}

const STEP =
  'grid size-7 shrink-0 place-items-center bg-slate-800 font-display text-base font-bold text-slate-200 ring-1 ring-slate-600/60 ring-inset transition hover:bg-slate-700 hover:text-white disabled:cursor-not-allowed disabled:opacity-40';

/** The deck's cards grouped by cost, each with buttons for one more, one fewer and any cover. */
export function DeckList({ draft, onAdd, onRemove, onCover }: DeckListProps) {
  const groups = groupByCost(draft);

  if (groups.length === 0) {
    return (
      <p className="px-4 py-8 text-center text-sm text-slate-400">
        The deck is empty. Pick cards from the list to add them.
      </p>
    );
  }

  return (
    <div className="divide-y divide-slate-800">
      {groups.map(({ cost, cards }) => (
        <section key={cost} aria-label={`Cost ${String(cost)}`} className="px-3 py-2">
          <h3 className="mb-1 font-display text-xs font-bold tracking-widest text-slate-500 uppercase">
            Cost {cost}
          </h3>
          <ul className="space-y-1">
            {cards.map(({ card, count }) => {
              const isCover = draft.coverCardId === card.id;
              return (
                <li key={card.id} className="flex items-center gap-2 text-sm hover:bg-white/[0.03]">
                  <button
                    type="button"
                    aria-label={`One fewer ${card.name}`}
                    className={STEP}
                    onClick={() => {
                      onRemove(card.id);
                    }}
                  >
                    −
                  </button>
                  <span className="w-5 text-center font-display font-bold text-accent">
                    {count}
                  </span>
                  <button
                    type="button"
                    aria-label={`One more ${card.name}`}
                    className={STEP}
                    disabled={cannotAdd(draft, card) !== null}
                    onClick={() => {
                      onAdd(card);
                    }}
                  >
                    +
                  </button>
                  <span className="min-w-0 flex-1 truncate text-slate-100" title={card.name}>
                    {card.name}
                  </span>
                  <CivilizationMarks civilizations={card.civilizations} />
                  {onCover && (
                    <button
                      type="button"
                      aria-pressed={isCover}
                      aria-label={
                        isCover
                          ? `${card.name} is the cover; use the site's pick`
                          : `Use ${card.name} as the cover`
                      }
                      title={isCover ? 'The cover card' : 'Use as the cover'}
                      onClick={() => {
                        onCover(isCover ? null : card.id);
                      }}
                      className={`px-1 text-base leading-none transition ${isCover ? 'text-amber-300' : 'text-slate-600 hover:text-amber-200'}`}
                    >
                      ★
                    </button>
                  )}
                </li>
              );
            })}
          </ul>
        </section>
      ))}
    </div>
  );
}
