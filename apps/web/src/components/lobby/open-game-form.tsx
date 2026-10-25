'use client';

import { useState, type SubmitEvent } from 'react';
import { Button } from '@/components/ui/button';
import { TextField } from '@/components/ui/text-field';
import type { DeckSummary, Format } from '@/lib/api/client';
import { openGame } from '@/lib/lobby-actions';
import { notify } from '@/lib/notifications';
import { DeckPicker, firstLegalDeck } from './deck-picker';
import { FormatPicker } from './format-picker';

/** The longest game name, as the server allows. */
const NAME_MAX_LENGTH = 50;

interface OpenGameFormProps {
  readonly decks: readonly DeckSummary[];
  readonly formats: readonly Format[];
  readonly disabled: boolean;
}

/** Opens a game for an opponent to join: listed in the lobby, or by invite link only. */
export function OpenGameForm({ decks, formats, disabled }: OpenGameFormProps) {
  const [formatId, setFormatId] = useState(formats[0]?.id ?? '');
  const [deckId, setDeckId] = useState(() => firstLegalDeck(decks, formats[0]?.id ?? ''));
  const [name, setName] = useState('');
  const [listed, setListed] = useState(true);
  const [spectators, setSpectators] = useState(true);
  const [pending, setPending] = useState(false);

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    const outcome = await openGame({ name, formatId, deckId, listed, spectators });
    setPending(false);
    if (!outcome.ok) notify(outcome.message);
  }

  return (
    <form onSubmit={(event) => void submit(event)} className="space-y-4 p-4">
      <TextField
        label="Name"
        value={name}
        maxLength={NAME_MAX_LENGTH}
        placeholder="Leave empty for a random name"
        onChange={(event) => {
          setName(event.target.value);
        }}
      />

      <FormatPicker
        formats={formats}
        value={formatId}
        onChange={(chosen) => {
          setFormatId(chosen);
          setDeckId(firstLegalDeck(decks, chosen));
        }}
      />

      <DeckPicker decks={decks} formatId={formatId} value={deckId} onChange={setDeckId} />

      <fieldset className="space-y-2 text-sm text-slate-300">
        <legend className="sr-only">Who can see the game</legend>
        <label className="flex cursor-pointer items-center gap-2 hover:text-white">
          <input
            type="checkbox"
            checked={listed}
            onChange={(event) => {
              setListed(event.target.checked);
            }}
            className="size-4 accent-sky-400"
          />
          List it in the lobby (otherwise only your invite link leads to it)
        </label>
        <label className="flex cursor-pointer items-center gap-2 hover:text-white">
          <input
            type="checkbox"
            checked={spectators}
            onChange={(event) => {
              setSpectators(event.target.checked);
            }}
            className="size-4 accent-sky-400"
          />
          Let others watch
        </label>
      </fieldset>

      <Button type="submit" className="w-full" disabled={disabled || pending || deckId === ''}>
        {pending ? 'Opening…' : 'Open game'}
      </Button>
    </form>
  );
}
