'use client';

import Link from 'next/link';
import { SelectField } from '@/components/ui/select-field';
import { decksForFormat, type PlayableDecks } from '@/lib/lobby';

interface DeckPickerProps {
  readonly decks: PlayableDecks;
  readonly formatId: string;
  readonly value: string;
  readonly onChange: (deckId: string) => void;
  readonly label?: string;
}

/**
 * A deck to play in a format: a starter deck or one of the player's own. The ones that can be
 * played are offered; the player's others are shown as not legal.
 */
export function DeckPicker({ decks, formatId, value, onChange, label = 'Deck' }: DeckPickerProps) {
  const { starters, legal, notLegal } = decksForFormat(decks, formatId);

  if (starters.length === 0 && legal.length === 0) {
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
    <SelectField
      label={label}
      value={value}
      onChange={(event) => {
        onChange(event.target.value);
      }}
    >
      {starters.length > 0 && (
        <optgroup label="Starter decks">
          {starters.map((deck) => (
            <option key={deck.id} value={deck.id}>
              {deck.name}
            </option>
          ))}
        </optgroup>
      )}
      {legal.length + notLegal.length > 0 && (
        <optgroup label="Your decks">
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
        </optgroup>
      )}
    </SelectField>
  );
}
