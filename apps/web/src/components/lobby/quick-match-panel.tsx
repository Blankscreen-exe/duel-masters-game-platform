'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import type { Format } from '@/lib/api/client';
import type { QueueCount } from '@/lib/lobby-actions';
import { enterQuickMatch, leaveQuickMatch } from '@/lib/lobby-actions';
import { defaultDeck, waitingIn, type PlayableDecks } from '@/lib/lobby';
import { notify } from '@/lib/notifications';
import { DeckPicker } from './deck-picker';
import { FormatPicker } from './format-picker';

interface QuickMatchPanelProps {
  readonly decks: PlayableDecks;
  readonly formats: readonly Format[];
  readonly queue: readonly QueueCount[];
  /** The format this player waits in, or null when not waiting. */
  readonly waitingFormatName: string | null;
  readonly disabled: boolean;
}

/**
 * Quick match: pick a format and a deck, and play the next player waiting in that format. On a
 * wider screen the pickers and the button sit in one row.
 */
export function QuickMatchPanel({
  decks,
  formats,
  queue,
  waitingFormatName,
  disabled,
}: QuickMatchPanelProps) {
  const [formatId, setFormatId] = useState(formats[0]?.id ?? '');
  const [deckId, setDeckId] = useState(() => defaultDeck(decks, formats[0]?.id ?? ''));
  const [pending, setPending] = useState(false);

  async function enter() {
    setPending(true);
    const outcome = await enterQuickMatch(formatId, deckId);
    setPending(false);
    if (!outcome.ok) notify(outcome.message);
  }

  async function leave() {
    setPending(true);
    const outcome = await leaveQuickMatch();
    setPending(false);
    if (!outcome.ok) notify(outcome.message);
  }

  if (waitingFormatName !== null) {
    return (
      <div className="flex flex-wrap items-center justify-center gap-4 p-4">
        <span
          aria-hidden
          className="block size-8 shrink-0 animate-spin rounded-full border-2 border-accent border-t-transparent"
        />
        <p
          aria-live="polite"
          className="font-display text-lg font-semibold tracking-wide text-white"
        >
          Looking for an opponent in {waitingFormatName}…
        </p>
        <Button variant="secondary" disabled={pending} onClick={() => void leave()}>
          Stop looking
        </Button>
      </div>
    );
  }

  return (
    <div className="grid items-end gap-4 p-4 md:grid-cols-[1fr_1fr_auto]">
      <FormatPicker
        formats={formats}
        value={formatId}
        onChange={(chosen) => {
          setFormatId(chosen);
          setDeckId(defaultDeck(decks, chosen));
        }}
        note={(format) => {
          const waiting = waitingIn(queue, format.id);
          return waiting > 0 ? ` (${String(waiting)} waiting)` : '';
        }}
      />
      <DeckPicker decks={decks} formatId={formatId} value={deckId} onChange={setDeckId} />
      <Button
        className="w-full md:w-auto"
        disabled={disabled || pending || deckId === ''}
        onClick={() => void enter()}
      >
        {pending ? 'Joining…' : 'Find a match'}
      </Button>
    </div>
  );
}
