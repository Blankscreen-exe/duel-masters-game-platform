'use client';

import { useEffect, useState, type ReactNode } from 'react';
import { Panel } from '@/components/ui/panel';
import { TabList } from '@/components/ui/tabs';
import type { Card, CardFilters } from '@/lib/api/client';
import {
  addCard,
  cannotAdd,
  copiesOf,
  DECK_SIZE,
  removeCard,
  totalCards,
  type DeckDraft,
} from '@/lib/deck-draft';
import { CardBrowser } from './card-browser';
import { DeckList } from './deck-list';
import { DeckStats } from './deck-stats';

interface DeckWorkbenchProps {
  readonly draft: DeckDraft;
  readonly onChange: (change: (current: DeckDraft) => DeckDraft) => void;
  readonly filters: CardFilters;
  /** Above the deck's cards, such as its name and the button that saves it. */
  readonly deckHeader?: ReactNode;
  /** Above the statistics, such as whether the deck can be played. */
  readonly sidePanels?: ReactNode;
  /** Whether the deck has a cover card to choose. */
  readonly hasCover?: boolean;
}

type View = 'cards' | 'deck' | 'stats';

const PANELS = 'deck-builder';

/**
 * Where a deck is built, whoever it is for: the playable cards on one side, the deck on the other,
 * with its statistics. On phones each is a tab. What holds the deck saves it.
 */
export function DeckWorkbench({
  draft,
  onChange,
  filters,
  deckHeader,
  sidePanels,
  hasCover = false,
}: DeckWorkbenchProps) {
  const [view, setView] = useState<View>('cards');
  const total = totalCards(draft);

  function add(card: Card) {
    onChange((current) => addCard(current, card));
  }

  return (
    <>
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
              {deckHeader}
              <DeckList
                draft={draft}
                onAdd={add}
                onRemove={(cardId) => {
                  onChange((current) => removeCard(current, cardId));
                }}
                {...(hasCover
                  ? {
                      onCover: (cardId: string | null) => {
                        onChange((current) => ({ ...current, coverCardId: cardId }));
                      },
                    }
                  : {})}
              />
            </Panel>
          </div>
          <div
            id={`${PANELS}-stats`}
            role="tabpanel"
            className={`space-y-6 ${view === 'stats' ? '' : 'max-lg:hidden'}`}
          >
            {sidePanels}
            <Panel title="Stats">
              <DeckStats draft={draft} />
            </Panel>
          </div>
        </div>
      </div>
    </>
  );
}

/** Leaving the site, or reloading, with unsaved changes asks first. */
export function useWarnOnLeaving(unsaved: boolean) {
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
}
