'use client';

import { useEffect, useState } from 'react';
import { CardImage } from '@/components/cards/card-image';
import { Button } from '@/components/ui/button';
import {
  api,
  errorMessage,
  UNREACHABLE,
  type Card,
  type CardFilters,
  type CardPage,
} from '@/lib/api/client';
import { SEARCH_TEXT_MAX_LENGTH, pageCount, type CardSearch } from '@/lib/card-search';
import { CIVILIZATIONS } from '@/lib/civilizations';
import type { CannotAdd } from '@/lib/deck-draft';
import { useDebouncedValue } from '@/lib/use-debounced-value';

/** Cards per page in the deck builder, a few rows on any screen. */
const PAGE_SIZE = 30;
/** How long typing must pause before searching. */
const TYPING_PAUSE_MS = 300;
/** The dearest cost with its own choice; dearer cards are chosen together. */
const TOP_COST = 10;

const CANNOT_ADD: Record<CannotAdd, string> = {
  not_playable: 'The game cannot play this card yet',
  max_copies: 'The deck has 4 copies already',
  deck_full: 'The deck has 40 cards already',
};

interface CardBrowserProps {
  readonly filters: CardFilters;
  readonly copiesOf: (cardId: string) => number;
  readonly cannotAdd: (card: Card) => CannotAdd | null;
  readonly onAdd: (card: Card) => void;
}

type Results =
  | { readonly key: string; readonly page: CardPage; readonly error?: undefined }
  | { readonly key: string; readonly page?: undefined; readonly error: string };

/** The playable cards, searchable; picking one adds a copy to the deck. */
export function CardBrowser({ filters, copiesOf, cannotAdd, onAdd }: CardBrowserProps) {
  const [text, setText] = useState('');
  const [civilizations, setCivilizations] = useState<readonly string[]>([]);
  const [type, setType] = useState('');
  const [cost, setCost] = useState('');
  const [page, setPage] = useState(1);
  const [results, setResults] = useState<Results | null>(null);

  const q = useDebouncedValue(text.trim(), TYPING_PAUSE_MS);
  const search: CardSearch = {
    playable: true,
    page,
    pageSize: PAGE_SIZE,
    ...(q === '' ? {} : { q }),
    ...(civilizations.length > 0 ? { civilization: [...civilizations] } : {}),
    ...(type === '' ? {} : { type: [type] }),
    ...costRange(cost),
  };
  const key = JSON.stringify(search);
  const current = results?.key === key ? results : null;
  const loading = current === null;

  useEffect(() => {
    const query: CardSearch = JSON.parse(key) as CardSearch;
    const controller = new AbortController();
    api
      .GET('/cards', { params: { query }, signal: controller.signal })
      .then(({ data, error }) => {
        setResults(data ? { key, page: data } : { key, error: errorMessage(error) });
      })
      .catch(() => {
        // A search replaced by a newer one is cancelled on purpose.
        if (!controller.signal.aborted) setResults({ key, error: UNREACHABLE });
      });
    return () => {
      controller.abort();
    };
  }, [key]);

  /** Any change of search starts again from its first page. */
  function narrow(change: () => void) {
    change();
    setPage(1);
  }

  const shown = results?.page;
  const pages = shown ? pageCount(shown.total, shown.pageSize) : 1;

  return (
    <div className="space-y-4">
      <div className="space-y-3">
        <input
          type="search"
          aria-label="Search cards by name or text"
          placeholder="Search cards…"
          value={text}
          maxLength={SEARCH_TEXT_MAX_LENGTH}
          onChange={(event) => {
            narrow(() => {
              setText(event.target.value);
            });
          }}
          className="block w-full bg-field px-3 py-2 text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset placeholder:text-slate-500 hover:ring-slate-400/70 focus:ring-accent focus:outline-none"
        />
        <div className="flex flex-wrap items-center gap-2">
          {CIVILIZATIONS.filter(({ name }) => filters.civilizations.includes(name)).map(
            ({ name }) => {
              const chosen = civilizations.includes(name);
              return (
                <button
                  key={name}
                  type="button"
                  aria-pressed={chosen}
                  onClick={() => {
                    narrow(() => {
                      setCivilizations(
                        chosen ? civilizations.filter((c) => c !== name) : [...civilizations, name],
                      );
                    });
                  }}
                  className={`px-2.5 py-1 text-xs font-semibold ring-1 ring-inset transition ${
                    chosen
                      ? 'bg-slate-600 text-white ring-accent'
                      : 'bg-slate-800/80 text-slate-300 ring-slate-600/60 hover:text-white'
                  }`}
                >
                  {name}
                </button>
              );
            },
          )}
          <select
            aria-label="Type"
            value={type}
            onChange={(event) => {
              narrow(() => {
                setType(event.target.value);
              });
            }}
            className="bg-field px-2 py-1 text-xs text-slate-200 ring-1 ring-slate-600/60 ring-inset focus:ring-accent focus:outline-none"
          >
            <option value="">Any type</option>
            {filters.types.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
          <select
            aria-label="Cost"
            value={cost}
            onChange={(event) => {
              narrow(() => {
                setCost(event.target.value);
              });
            }}
            className="bg-field px-2 py-1 text-xs text-slate-200 ring-1 ring-slate-600/60 ring-inset focus:ring-accent focus:outline-none"
          >
            <option value="">Any cost</option>
            {Array.from({ length: TOP_COST }, (_, index) => index + 1).map((value) => (
              <option key={value} value={String(value)}>
                {value === TOP_COST ? `Cost ${String(value)}+` : `Cost ${String(value)}`}
              </option>
            ))}
          </select>
        </div>
      </div>

      <p aria-live="polite" className="text-xs tracking-wider text-slate-500 uppercase">
        {current === null
          ? 'Searching…'
          : (current.error ?? `${String(current.page.total)} playable cards`)}
      </p>

      {shown && (
        <ul
          className={`grid grid-cols-3 gap-3 sm:grid-cols-4 lg:grid-cols-5 ${loading ? 'opacity-60' : ''}`}
        >
          {shown.cards.map((card) => {
            const copies = copiesOf(card.id);
            const reason = cannotAdd(card);
            return (
              <li key={card.id}>
                <button
                  type="button"
                  disabled={reason !== null}
                  title={reason === null ? `Add ${card.name}` : CANNOT_ADD[reason]}
                  aria-label={`Add ${card.name}${copies > 0 ? `, ${String(copies)} in the deck` : ''}`}
                  onClick={() => {
                    onAdd(card);
                  }}
                  className="group relative block w-full text-left transition enabled:hover:-translate-y-0.5 disabled:cursor-not-allowed"
                >
                  <CardImage
                    src={card.imageUrl}
                    name={card.name}
                    className={`transition group-enabled:group-hover:shadow-[0_0_16px_rgb(56_189_248/0.4)] ${reason === null ? '' : 'opacity-40 grayscale'}`}
                  />
                  {copies > 0 && (
                    <span className="absolute top-1 right-1 grid size-6 place-items-center bg-accent font-display text-sm font-bold text-slate-900 shadow">
                      {copies}
                    </span>
                  )}
                  <span className="mt-1 line-clamp-1 block text-[11px] text-slate-300">
                    {card.name}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}

      {pages > 1 && (
        <nav aria-label="Card pages" className="flex items-center justify-center gap-3">
          <Button
            variant="secondary"
            disabled={page <= 1}
            onClick={() => {
              setPage(page - 1);
            }}
          >
            Previous
          </Button>
          <span className="font-display text-sm font-semibold tracking-wider text-slate-400 uppercase">
            Page {page} of {pages}
          </span>
          <Button
            variant="secondary"
            disabled={page >= pages}
            onClick={() => {
              setPage(page + 1);
            }}
          >
            Next
          </Button>
        </nav>
      )}
    </div>
  );
}

/** The search's cost bounds for a cost choice: exactly that cost, or the top cost and dearer. */
function costRange(cost: string): Pick<CardSearch, 'minCost' | 'maxCost'> {
  if (cost === '') return {};

  const value = Number(cost);
  return value >= TOP_COST ? { minCost: value } : { minCost: value, maxCost: value };
}
