'use client';

import Link from 'next/link';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import type { CardFilters, Deck, Format } from '@/lib/api/client';
import { saveDeck } from '@/lib/deck-actions';
import {
  DECK_NAME_MAX_LENGTH,
  draftFromDeck,
  sameDeck,
  toDeckInput,
  type DeckDraft,
} from '@/lib/deck-draft';
import { notify } from '@/lib/notifications';
import { DeckLegality } from './deck-legality';
import { DeckWorkbench, useWarnOnLeaving } from './deck-workbench';
import { ExportDeckDialog } from './export-deck-dialog';

interface DeckBuilderProps {
  readonly deck: Deck;
  readonly formats: readonly Format[];
  readonly filters: CardFilters;
}

/** Builds a player's deck, with its name, whether it can be played in each format, and export. */
export function DeckBuilder({ deck, formats, filters }: DeckBuilderProps) {
  const [initial] = useState(() => draftFromDeck(deck));
  const [saved, setSaved] = useState<{ deck: Deck; draft: DeckDraft }>({
    deck,
    draft: initial.draft,
  });
  const [draft, setDraft] = useState(initial.draft);
  // Copies of cards gone from the card database, which saving removes.
  const [dropped, setDropped] = useState(initial.dropped);
  const [saving, setSaving] = useState(false);
  const [exporting, setExporting] = useState(false);

  const unsaved = dropped > 0 || !sameDeck(draft, saved.draft);
  const nameMissing = draft.name.trim() === '';
  useWarnOnLeaving(unsaved);

  async function save() {
    setSaving(true);
    const outcome = await saveDeck(saved.deck.id, toDeckInput(draft));
    setSaving(false);

    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    const next = draftFromDeck(outcome.value).draft;
    setSaved({ deck: outcome.value, draft: next });
    setDraft(next);
    setDropped(0);
    notify('Deck saved');
  }

  return (
    <div className="space-y-4">
      <Link
        href="/decks"
        className="font-display text-sm font-semibold tracking-wider text-slate-400 uppercase hover:text-white"
      >
        ← My decks
      </Link>

      <DeckWorkbench
        draft={draft}
        onChange={setDraft}
        filters={filters}
        formats={formats}
        hasCover
        deckActions={
          <div className="flex min-w-60 flex-1 flex-wrap items-center gap-2">
            <input
              aria-label="Deck name"
              value={draft.name}
              maxLength={DECK_NAME_MAX_LENGTH}
              onChange={(event) => {
                const name = event.target.value;
                setDraft((current) => ({ ...current, name }));
              }}
              className="block min-w-40 flex-1 bg-field px-3 py-1.5 font-display text-base font-bold tracking-wide text-white ring-1 ring-slate-600/60 ring-inset hover:ring-slate-400/70 focus:ring-accent focus:outline-none"
            />
            <div className="flex gap-2">
              <Button
                disabled={!unsaved || saving || nameMissing}
                title={nameMissing ? 'The deck needs a name' : undefined}
                onClick={() => void save()}
              >
                {saving ? 'Saving…' : unsaved ? 'Save' : 'Saved'}
              </Button>
              <Button
                variant="secondary"
                disabled={unsaved}
                title={unsaved ? 'Save first to export your changes' : 'The deck as text'}
                onClick={() => {
                  setExporting(true);
                }}
              >
                Export
              </Button>
            </div>
          </div>
        }
        deckNotice={
          dropped > 0 && (
            <p className="border-b border-slate-700/60 px-4 py-2 text-xs text-amber-300">
              {dropped} {dropped === 1 ? 'card is' : 'cards are'} no longer in the card database and
              will be removed when you save.
            </p>
          )
        }
        sidePanels={
          <Panel title="Formats">
            <DeckLegality legality={saved.deck.legality} formats={formats} unsaved={unsaved} />
          </Panel>
        }
      />

      {exporting && (
        <ExportDeckDialog
          deckId={saved.deck.id}
          deckName={saved.deck.name}
          onClose={() => {
            setExporting(false);
          }}
        />
      )}
    </div>
  );
}
