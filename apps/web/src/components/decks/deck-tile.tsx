'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { CardImage } from '@/components/cards/card-image';
import { CivilizationMarks } from '@/components/cards/civilization-marks';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import type { DeckSummary, Format } from '@/lib/api/client';
import { copyDeck, copyStarterDeck, deleteDeck } from '@/lib/deck-actions';
import { DECK_SIZE } from '@/lib/deck-draft';
import { notify } from '@/lib/notifications';

interface DeckTileProps {
  readonly deck: DeckSummary;
  readonly formats: readonly Format[];
}

const TILE =
  'group cut-corners flex flex-col bg-panel/95 ring-1 ring-slate-500/40 transition ring-inset hover:-translate-y-0.5 hover:ring-accent/60';

/** A deck on the My decks page: its cover, size and formats, leading to the deck builder. */
export function DeckTile({ deck, formats }: DeckTileProps) {
  const router = useRouter();
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [copying, setCopying] = useState(false);

  async function copy() {
    setCopying(true);
    const outcome = await copyDeck(deck.id);
    setCopying(false);

    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    notify(`Copied as “${outcome.value.name}”`);
    router.refresh();
  }

  async function remove() {
    const outcome = await deleteDeck(deck.id);
    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    setConfirmingDelete(false);
    notify(`Deleted “${deck.name}”`);
    router.refresh();
  }

  return (
    <article className={TILE}>
      <Link href={`/decks/${encodeURIComponent(deck.id)}`} className="flex flex-1 gap-4 p-4">
        <DeckFace deck={deck} formats={formats} />
      </Link>

      <div className="flex justify-end gap-1 border-t border-slate-700/50 px-2 py-1">
        <Button variant="quiet" disabled={copying} onClick={() => void copy()}>
          {copying ? 'Copying…' : 'Copy'}
        </Button>
        <Button
          variant="quiet"
          className="hover:text-red-300"
          onClick={() => {
            setConfirmingDelete(true);
          }}
        >
          Delete
        </Button>
      </div>

      {confirmingDelete && (
        <ConfirmDialog
          title="Delete deck"
          confirmLabel="Delete deck"
          onConfirm={remove}
          onClose={() => {
            setConfirmingDelete(false);
          }}
        >
          Delete “{deck.name}”? This cannot be undone.
        </ConfirmDialog>
      )}
    </article>
  );
}

interface StarterTileProps extends DeckTileProps {
  /** Whether it is every player's default deck, the first of the starter decks. */
  readonly isDefault: boolean;
}

/**
 * A starter deck on the My decks page: ready to play as it is, or to copy into the player's own
 * decks to change.
 */
export function StarterTile({ deck, formats, isDefault }: StarterTileProps) {
  const router = useRouter();
  const [copying, setCopying] = useState(false);

  async function copy() {
    setCopying(true);
    const outcome = await copyStarterDeck(deck.id);
    setCopying(false);

    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    notify(`“${outcome.value.name}” is now one of your decks`);
    router.refresh();
  }

  return (
    <article className={TILE}>
      <div className="flex flex-1 gap-4 p-4">
        <DeckFace deck={deck} formats={formats} />
      </div>
      <div className="flex items-center justify-between gap-1 border-t border-slate-700/50 px-2 py-1">
        <span className="px-2 text-xs text-slate-400">
          {isDefault ? 'Everyone’s default deck' : 'Ready to play'}
        </span>
        <Button variant="quiet" disabled={copying} onClick={() => void copy()}>
          {copying ? 'Copying…' : 'Copy to my decks'}
        </Button>
      </div>
    </article>
  );
}

/** A deck's cover, name, size, civilizations and the formats it can be played in. */
function DeckFace({ deck, formats }: DeckTileProps) {
  const legal = formats.filter((format) => deck.legalFormatIds.includes(format.id));

  return (
    <>
      <CardImage
        src={deck.cover?.imageUrl}
        name={deck.cover ? `${deck.name} cover` : 'Empty deck'}
        className="w-20 shrink-0 transition group-hover:shadow-[0_0_16px_rgb(56_189_248/0.35)]"
      />
      <div className="min-w-0 flex-1 space-y-2">
        <h2 className="truncate font-display text-xl font-bold tracking-wide text-white">
          {deck.name}
        </h2>
        <p className="flex items-center gap-2 text-sm text-slate-400">
          <span className={deck.cardCount === DECK_SIZE ? 'text-slate-200' : 'text-amber-300'}>
            {deck.cardCount}/{DECK_SIZE} cards
          </span>
          <CivilizationMarks civilizations={deck.civilizations} />
        </p>
        <ul aria-label="Legal in" className="flex flex-wrap gap-1">
          {legal.length > 0 ? (
            legal.map((format) => (
              <li
                key={format.id}
                className="px-1.5 py-0.5 text-[10px] font-bold tracking-wider text-emerald-300 uppercase ring-1 ring-emerald-400/40 ring-inset"
              >
                {format.name}
              </li>
            ))
          ) : (
            <li className="text-xs text-slate-500">Not legal in any format yet</li>
          )}
        </ul>
      </div>
    </>
  );
}
