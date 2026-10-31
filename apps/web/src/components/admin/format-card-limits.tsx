'use client';

import { useState } from 'react';
import { CardImage } from '@/components/cards/card-image';
import { Button } from '@/components/ui/button';
import { FIELD_CONTROL, FIELD_HINT, FIELD_LABEL } from '@/components/ui/field';
import type { Card } from '@/lib/api/client';
import { SEARCH_TEXT_MAX_LENGTH } from '@/lib/card-search';
import {
  changedLimit,
  LIMIT_CHOICES,
  withLimit,
  withoutLimit,
  type FormatForm,
  type FormatLimit,
} from '@/lib/formats';
import { useCardPages } from '@/lib/use-card-pages';
import { useDebouncedValue } from '@/lib/use-debounced-value';

/** Cards shown for a search; a narrower search finds what is not among them. */
const RESULTS = 12;
const TYPING_PAUSE_MS = 300;

interface FormatCardLimitsProps {
  readonly form: FormatForm;
  readonly onChange: (change: (current: FormatForm) => FormatForm) => void;
}

/** The cards a format bans or limits, and a search to find more. */
export function FormatCardLimits({ form, onChange }: FormatCardLimitsProps) {
  const [text, setText] = useState('');
  const q = useDebouncedValue(text.trim(), TYPING_PAUSE_MS);

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section aria-label="Banned and limited cards" className="min-w-0">
        <h3 className={FIELD_LABEL}>
          Banned and limited cards{form.limits.length > 0 && ` · ${String(form.limits.length)}`}
        </h3>
        {form.limits.length === 0 ? (
          <p className="mt-2 text-sm text-slate-400">
            None: decks follow only the game&apos;s own limit of 4 copies.
          </p>
        ) : (
          <ul className="mt-2 divide-y divide-slate-800">
            {form.limits.map((limit) => (
              <LimitRow key={limit.cardId} limit={limit} onChange={onChange} />
            ))}
          </ul>
        )}
      </section>

      <section aria-label="Find a card to ban or limit" className="min-w-0">
        <label htmlFor="format-card-search" className={FIELD_LABEL}>
          Find a card to ban or limit
        </label>
        <input
          id="format-card-search"
          type="search"
          value={text}
          maxLength={SEARCH_TEXT_MAX_LENGTH}
          placeholder="Name or text…"
          onChange={(event) => {
            setText(event.target.value);
          }}
          className={FIELD_CONTROL}
        />
        {q === '' ? (
          <p className={FIELD_HINT}>Type part of a card&apos;s name or text.</p>
        ) : (
          <SearchResults q={q} form={form} onChange={onChange} />
        )}
      </section>
    </div>
  );
}

function LimitRow({
  limit,
  onChange,
}: {
  readonly limit: FormatLimit;
  readonly onChange: FormatCardLimitsProps['onChange'];
}) {
  const name = limit.card?.name ?? `A card no longer in the card database (${limit.cardId})`;

  return (
    <li className="flex items-center gap-3 py-1.5">
      <CardImage src={limit.card?.imageUrl} name={name} className="w-9 shrink-0" />
      <span className="min-w-0 flex-1 text-sm font-semibold text-slate-100">{name}</span>
      <select
        aria-label={`Limit on ${name}`}
        value={limit.maxCopies}
        onChange={(event) => {
          const maxCopies = Number(event.target.value);
          onChange((current) => changedLimit(current, limit.cardId, maxCopies));
        }}
        className="bg-field px-2 py-1 text-xs text-slate-200 ring-1 ring-slate-600/60 ring-inset focus:ring-accent focus:outline-none"
      >
        {LIMIT_CHOICES.map(({ maxCopies, label }) => (
          <option key={maxCopies} value={maxCopies}>
            {label}
          </option>
        ))}
      </select>
      <button
        type="button"
        aria-label={`Remove the limit on ${name}`}
        title="Remove the limit"
        onClick={() => {
          onChange((current) => withoutLimit(current, limit.cardId));
        }}
        className="grid size-7 shrink-0 place-items-center bg-slate-800 text-slate-300 ring-1 ring-slate-600/60 ring-inset transition hover:bg-slate-700 hover:text-white"
      >
        ×
      </button>
    </li>
  );
}

function SearchResults({
  q,
  form,
  onChange,
}: {
  readonly q: string;
  readonly form: FormatForm;
  readonly onChange: FormatCardLimitsProps['onChange'];
}) {
  const { pages, searching, error } = useCardPages({ q }, RESULTS);
  const cards = pages?.cards ?? [];

  function limit(card: Card, maxCopies: number) {
    onChange((current) => withLimit(current, card, maxCopies));
  }

  return (
    <div className="mt-2">
      <p aria-live="polite" className="text-xs tracking-wider text-slate-500 uppercase">
        {error ??
          (searching
            ? 'Searching…'
            : `${String(pages?.total ?? 0)} ${pages?.total === 1 ? 'card' : 'cards'}${(pages?.total ?? 0) > RESULTS ? `, the first ${String(RESULTS)} shown` : ''}`)}
      </p>
      <ul className={`mt-1 divide-y divide-slate-800 ${searching ? 'opacity-60' : ''}`}>
        {cards.map((card) => (
          <li key={card.id} className="flex flex-wrap items-center gap-3 py-1.5">
            <CardImage src={card.imageUrl} name={card.name} className="w-9 shrink-0" />
            <span className="min-w-0 flex-1 basis-32 text-sm font-semibold text-slate-100">
              {card.name}
            </span>
            {form.limits.some(({ cardId }) => cardId === card.id) ? (
              <span className="text-xs text-accent">In the list</span>
            ) : (
              <span className="flex gap-1">
                {LIMIT_CHOICES.map(({ maxCopies, label }) => (
                  <Button
                    key={maxCopies}
                    variant="secondary"
                    aria-label={`${label}: ${card.name}`}
                    className="px-2! py-1! text-xs!"
                    onClick={() => {
                      limit(card, maxCopies);
                    }}
                  >
                    {maxCopies === 0 ? 'Ban' : `Max ${String(maxCopies)}`}
                  </Button>
                ))}
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
