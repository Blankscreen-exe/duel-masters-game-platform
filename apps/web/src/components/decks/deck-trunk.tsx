'use client';

import { useEffect, useRef, useState } from 'react';
import { CardImage } from '@/components/cards/card-image';
import { CivilizationMarks } from '@/components/cards/civilization-marks';
import { Button } from '@/components/ui/button';
import type { Card, CardFilters, Format } from '@/lib/api/client';
import {
  cardFiltersToSearch,
  countCardFilters,
  deckCardLists,
  NO_CARD_FILTERS,
  type CardFilterChoices,
} from '@/lib/card-filters';
import { SEARCH_TEXT_MAX_LENGTH } from '@/lib/card-search';
import {
  CANNOT_ADD_TEXT,
  cannotAdd,
  copiesOf,
  type CannotAdd,
  type DeckDraft,
} from '@/lib/deck-draft';
import { useCardPages } from '@/lib/use-card-pages';
import { useDebouncedValue } from '@/lib/use-debounced-value';
import { CardFilterDialog } from './card-filter-dialog';
import { useCardDrag } from './deck-drag';

/** Cards asked for at a time, more than fill the trunk on any screen. */
const PAGE_SIZE = 30;
/** How long typing must pause before searching. */
const TYPING_PAUSE_MS = 300;
/** How far below the list's end more cards start loading, so scrolling seldom meets its end. */
const LOAD_AHEAD = '300px';

interface DeckTrunkProps {
  readonly draft: DeckDraft;
  readonly filters: CardFilters;
  readonly formats: readonly Format[];
  readonly onAdd: (card: Card) => void;
  readonly onEnlarge: (card: Card) => void;
}

/** The filter modal's choices in force, with the deck's cards as they stood when they were made. */
interface AppliedFilters {
  readonly choices: CardFilterChoices;
  readonly deckCards: ReturnType<typeof deckCardLists>;
}

const NOTHING_APPLIED: AppliedFilters = { choices: NO_CARD_FILTERS, deckCards: {} };

/**
 * The trunk: every playable card, searchable, as a scrolling list of small cards with their names.
 * Picking a card enlarges it; dragging it to the deck, or its button, adds a copy.
 */
export function DeckTrunk({ draft, filters, formats, onAdd, onEnlarge }: DeckTrunkProps) {
  const [text, setText] = useState('');
  const [applied, setApplied] = useState(NOTHING_APPLIED);
  const [choosing, setChoosing] = useState(false);

  const q = useDebouncedValue(text.trim(), TYPING_PAUSE_MS);
  const { pages, searching, error, more, loadMore } = useCardPages(
    {
      playable: true,
      ...(q === '' ? {} : { q }),
      ...cardFiltersToSearch(applied.choices),
      ...applied.deckCards,
    },
    PAGE_SIZE,
  );
  const filtering = countCardFilters(applied.choices);

  return (
    <div className="flex h-full flex-col">
      <div className="space-y-2 border-b border-slate-700/60 p-3">
        <div className="flex gap-2">
          <input
            type="search"
            aria-label="Search cards by name or text"
            placeholder="Search cards…"
            value={text}
            maxLength={SEARCH_TEXT_MAX_LENGTH}
            onChange={(event) => {
              setText(event.target.value);
            }}
            className="block min-w-0 flex-1 bg-field px-3 py-2 text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset placeholder:text-slate-500 hover:ring-slate-400/70 focus:ring-accent focus:outline-none"
          />
          <Button
            variant="secondary"
            className="shrink-0 px-3! py-1.5! text-sm"
            onClick={() => {
              setChoosing(true);
            }}
          >
            Filters
            {filtering > 0 && (
              <span className="grid size-5 place-items-center bg-accent text-xs text-slate-900">
                {filtering}
              </span>
            )}
          </Button>
        </div>
        <div className="flex items-center justify-between gap-2">
          <p aria-live="polite" className="text-xs tracking-wider text-slate-500 uppercase">
            {statusText(pages?.total, searching, error)}
          </p>
          {applied !== NOTHING_APPLIED && (
            <button
              type="button"
              onClick={() => {
                setApplied(NOTHING_APPLIED);
              }}
              className="text-xs font-semibold text-slate-400 underline hover:text-white"
            >
              Clear filters
            </button>
          )}
        </div>
      </div>

      <div className="min-h-0 flex-1 lg:overflow-y-auto">
        <ul className={`divide-y divide-slate-800 ${searching ? 'opacity-60' : ''}`}>
          {pages?.cards.map((card) => (
            <TrunkRow
              key={card.id}
              card={card}
              copies={copiesOf(draft, card.id)}
              reason={cannotAdd(draft, card)}
              onAdd={onAdd}
              onEnlarge={onEnlarge}
            />
          ))}
        </ul>
        {more && <LoadMore onReach={error === null ? loadMore : null} onPress={loadMore} />}
      </div>

      {choosing && (
        <CardFilterDialog
          choices={applied.choices}
          filters={filters}
          formats={formats}
          onApply={(choices) => {
            setApplied({ choices, deckCards: deckCardLists(choices, draft) });
            setChoosing(false);
          }}
          onClose={() => {
            setChoosing(false);
          }}
        />
      )}
    </div>
  );
}

interface TrunkRowProps {
  readonly card: Card;
  /** How many the deck has. */
  readonly copies: number;
  /** Why it cannot be added, or null when it can. */
  readonly reason: CannotAdd | null;
  readonly onAdd: (card: Card) => void;
  readonly onEnlarge: (card: Card) => void;
}

/** One card in the trunk, to enlarge, to drag to the deck, or to add with its button. */
function TrunkRow({ card, copies, reason, onAdd, onEnlarge }: TrunkRowProps) {
  const { ref, grip, dragging } = useCardDrag(card, 'trunk');

  return (
    <li
      ref={ref}
      className={`flex items-center gap-1 pr-2 hover:bg-white/[0.04] ${dragging ? 'opacity-30' : ''}`}
    >
      <button
        type="button"
        aria-label={`Enlarge ${card.name}${copies > 0 ? `, ${String(copies)} in the deck` : ''}`}
        onClick={() => {
          onEnlarge(card);
        }}
        {...grip}
        className="group flex min-w-0 flex-1 cursor-zoom-in items-center gap-2.5 p-1.5 text-left select-none [-webkit-touch-callout:none]"
      >
        <CardImage
          src={card.imageUrl}
          name={card.name}
          className="w-11 shrink-0 transition group-hover:shadow-[0_0_10px_rgb(56_189_248/0.5)]"
        />
        <span className="min-w-0 flex-1">
          <span className="line-clamp-2 text-sm font-semibold text-slate-100 group-hover:text-white">
            {card.name}
          </span>
          <span className="mt-0.5 flex items-center gap-2 text-xs text-slate-400">
            Cost {card.cost}
            <CivilizationMarks civilizations={card.civilizations} />
          </span>
        </span>
      </button>
      {copies > 0 && (
        <span
          title={`${String(copies)} in the deck`}
          className="grid size-6 shrink-0 place-items-center bg-accent font-display text-sm font-bold text-slate-900"
        >
          {copies}
        </span>
      )}
      <Button
        variant="secondary"
        aria-label={`Add ${card.name}`}
        title={reason === null ? `Add ${card.name}` : CANNOT_ADD_TEXT[reason]}
        disabled={reason !== null}
        className="shrink-0 px-2.5! py-1!"
        onClick={() => {
          onAdd(card);
        }}
      >
        +
      </Button>
    </li>
  );
}

interface LoadMoreProps {
  /** Called when the end of the list comes into view; null to wait to be pressed. */
  readonly onReach: (() => void) | null;
  readonly onPress: () => void;
}

/** The end of a list with more to come: loads as it comes into view, or when pressed. */
function LoadMore({ onReach, onPress }: LoadMoreProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const element = ref.current;
    if (!element || !onReach) return;
    // Against the screen, which also counts the list's own scrolling when it has one.
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) onReach();
      },
      { rootMargin: LOAD_AHEAD },
    );
    observer.observe(element);
    return () => {
      observer.disconnect();
    };
  }, [onReach]);

  return (
    <div ref={ref} className="p-3 text-center">
      <Button variant="secondary" onClick={onPress}>
        {onReach ? 'More cards' : 'Try again'}
      </Button>
    </div>
  );
}

function statusText(total: number | undefined, searching: boolean, error: string | null): string {
  if (error !== null) return error;
  if (total === undefined || searching) return 'Searching…';
  return `${String(total)} playable ${total === 1 ? 'card' : 'cards'}`;
}
