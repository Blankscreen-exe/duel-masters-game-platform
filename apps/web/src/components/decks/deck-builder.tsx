'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { TabList } from '@/components/ui/tabs';
import type { Card, CardFilters, Deck, Format } from '@/lib/api/client';
import { saveDeck } from '@/lib/deck-actions';
import {
  addCard,
  cannotAdd,
  copiesOf,
  DECK_NAME_MAX_LENGTH,
  DECK_SIZE,
  draftFromDeck,
  removeCard,
  sameDeck,
  toDeckInput,
  totalCards,
  type DeckDraft,
} from '@/lib/deck-draft';
import { notify } from '@/lib/notifications';
import { CardBrowser } from './card-browser';
import { DeckLegality } from './deck-legality';
import { DeckList } from './deck-list';
import { DeckStats } from './deck-stats';
import { ExportDeckDialog } from './export-deck-dialog';

interface DeckBuilderProps {
  readonly deck: Deck;
  readonly formats: readonly Format[];
  readonly filters: CardFilters;
}

type View = 'cards' | 'deck' | 'stats';

const PANELS = 'deck-builder';

/**
 * Builds a deck: the playable cards on one side, the deck on the other, with its statistics and
 * whether it can be played in each format. On phones each is a tab.
 */
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
  const [view, setView] = useState<View>('cards');

  const total = totalCards(draft);
  const unsaved = dropped > 0 || !sameDeck(draft, saved.draft);
  const nameMissing = draft.name.trim() === '';

  // Leaving the site, or reloading, with unsaved changes asks first.
  useEffect(() => {
    if (!unsaved) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
    };
    window.addEventListener('beforeunload', warn);
    return () => {
      window.removeEventListener('beforeunload', warn);
    };
  }, [unsaved]);

  function add(card: Card) {
    setDraft((current) => addCard(current, card));
  }

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

  const deckPanel = (
    <Panel
      title={
        <span>
          Deck{' '}
          <span className={total === DECK_SIZE ? 'text-emerald-300' : 'text-amber-300'}>
            {total}/{DECK_SIZE}
          </span>
        </span>
      }
      accent="water"
    >
      <div className="space-y-3 border-b border-slate-700/60 p-3">
        <input
          aria-label="Deck name"
          value={draft.name}
          maxLength={DECK_NAME_MAX_LENGTH}
          onChange={(event) => {
            const name = event.target.value;
            setDraft((current) => ({ ...current, name }));
          }}
          className="block w-full bg-field px-3 py-2 font-display text-lg font-bold tracking-wide text-white ring-1 ring-slate-600/60 ring-inset hover:ring-slate-400/70 focus:ring-accent focus:outline-none"
        />
        {dropped > 0 && (
          <p className="text-xs text-amber-300">
            {dropped} {dropped === 1 ? 'card is' : 'cards are'} no longer in the card database and
            will be removed when you save.
          </p>
        )}
        <div className="flex gap-2">
          <Button
            className="flex-1"
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
      <DeckList
        draft={draft}
        onAdd={add}
        onRemove={(cardId) => {
          setDraft((current) => removeCard(current, cardId));
        }}
        onCover={(cardId) => {
          setDraft((current) => ({ ...current, coverCardId: cardId }));
        }}
      />
    </Panel>
  );

  const statsPanels = (
    <>
      <Panel title="Formats">
        <DeckLegality legality={saved.deck.legality} formats={formats} unsaved={unsaved} />
      </Panel>
      <Panel title="Stats">
        <DeckStats draft={draft} />
      </Panel>
    </>
  );

  return (
    <div className="space-y-4">
      <Link
        href="/decks"
        className="font-display text-sm font-semibold tracking-wider text-slate-400 uppercase hover:text-white"
      >
        ← My decks
      </Link>

      <TabList
        label="Deck builder"
        panelIdPrefix={PANELS}
        value={view}
        onChange={setView}
        className="sticky top-16 z-20 lg:hidden"
        tabs={[
          { id: 'cards', label: 'Cards' },
          { id: 'deck', label: `Deck ${String(total)}/${String(DECK_SIZE)}` },
          { id: 'stats', label: 'Stats' },
        ]}
      />

      <div className="grid gap-6 lg:grid-cols-[1fr_24rem]">
        <div
          id={`${PANELS}-cards`}
          role="tabpanel"
          className={view === 'cards' ? '' : 'max-lg:hidden'}
        >
          <Panel title="Cards" className="p-4">
            <CardBrowser
              filters={filters}
              copiesOf={(cardId) => copiesOf(draft, cardId)}
              cannotAdd={(card) => cannotAdd(draft, card)}
              onAdd={add}
            />
          </Panel>
        </div>

        <div className="space-y-6">
          <div
            id={`${PANELS}-deck`}
            role="tabpanel"
            className={view === 'deck' ? '' : 'max-lg:hidden'}
          >
            {deckPanel}
          </div>
          <div
            id={`${PANELS}-stats`}
            role="tabpanel"
            className={`space-y-6 ${view === 'stats' ? '' : 'max-lg:hidden'}`}
          >
            {statsPanels}
          </div>
        </div>
      </div>

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
