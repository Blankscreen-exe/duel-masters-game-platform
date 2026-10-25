'use client';

import Link from 'next/link';
import { useId } from 'react';
import type { DeckSummary } from '@/lib/api/client';
import { decksForFormat } from '@/lib/lobby';

interface DeckPickerProps {
  readonly decks: readonly DeckSummary[];
  readonly formatId: string;
  readonly value: string;
  readonly onChange: (deckId: string) => void;
  readonly label?: string;
}

/** One of the player's decks for a format: the legal ones to choose, the others shown as not legal. */
export function DeckPicker({ decks, formatId, value, onChange, label = 'Deck' }: DeckPickerProps) {
  const id = useId();
  const { legal, notLegal } = decksForFormat(decks, formatId);

  if (legal.length === 0) {
    return (
      <p className="text-sm text-slate-400">
        None of your decks can be played in this format yet.{' '}
        <Link href="/decks" className="font-semibold text-accent hover:underline">
          Build one
        </Link>
        .
      </p>
    );
  }

  return (
    <div>
      <label
        htmlFor={id}
        className="block text-[11px] font-semibold tracking-wider text-slate-400 uppercase"
      >
        {label}
      </label>
      <select
        id={id}
        value={value}
        onChange={(event) => {
          onChange(event.target.value);
        }}
        className="mt-1 block w-full bg-field px-3 py-2 text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset hover:ring-slate-400/70 focus:ring-accent focus:outline-none"
      >
        {legal.map((deck) => (
          <option key={deck.id} value={deck.id}>
            {deck.name}
          </option>
        ))}
        {notLegal.map((deck) => (
          <option key={deck.id} value={deck.id} disabled>
            {deck.name} (not legal here)
          </option>
        ))}
      </select>
    </div>
  );
}

/** The first deck a player can play in the format, to choose by default. */
export function firstLegalDeck(decks: readonly DeckSummary[], formatId: string): string {
  return decksForFormat(decks, formatId).legal[0]?.id ?? '';
}
