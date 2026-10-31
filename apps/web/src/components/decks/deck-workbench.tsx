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
import type { DeckEditing } from './deck-card-buttons';
import { DeckCards } from './deck-cards';
import { DeckDrag, DropArea } from './deck-drag';
import { DeckStats } from './deck-stats';
import { DeckTrunk } from './deck-trunk';
import { EnlargedCard } from './enlarged-card';

interface DeckWorkbenchProps {
  readonly draft: DeckDraft;
  readonly onChange: (change: (current: DeckDraft) => DeckDraft) => void;
  readonly filters: CardFilters;
  /** In the deck's title bar, which stays in view, such as its name and the button that saves it. */
  readonly deckActions?: ReactNode;
  /** Above the deck's cards, such as a warning about the deck. */
  readonly deckNotice?: ReactNode;
  /** Beside the statistics, such as whether the deck can be played. */
  readonly sidePanels?: ReactNode;
  /** Whether the deck has a cover card to choose. */
  readonly hasCover?: boolean;
}

type View = 'trunk' | 'deck' | 'stats';

const PANELS = 'deck-builder';

/**
 * Where a deck is built, whoever it is for: the deck as cards in the main area with its statistics
 * below, and the trunk of playable cards in the side bar, staying in view while the deck scrolls.
 * Cards are dragged between the two, or moved with their buttons, and any card of either can be
 * enlarged. On phones each is a tab, with the buttons. What holds the deck saves it.
 */
export function DeckWorkbench({
  draft,
  onChange,
  filters,
  deckActions,
  deckNotice,
  sidePanels,
  hasCover = false,
}: DeckWorkbenchProps) {
  const [view, setView] = useState<View>('trunk');
  const [enlarged, setEnlarged] = useState<Card | null>(null);
  const total = totalCards(draft);

  const editing: DeckEditing = {
    draft,
    onAdd: (card) => {
      onChange((current) => addCard(current, card));
    },
    onRemove: (cardId) => {
      onChange((current) => removeCard(current, cardId));
    },
    ...(hasCover
      ? {
          onCover: (cardId: string | null) => {
            onChange((current) => ({ ...current, coverCardId: cardId }));
          },
        }
      : {}),
  };

  return (
    <DeckDrag draft={draft} onAdd={editing.onAdd} onRemove={editing.onRemove}>
      <TabList
        label="Deck builder"
        panelIdPrefix={PANELS}
        value={view}
        onChange={setView}
        className="sticky top-[59px] z-20 lg:hidden"
        tabs={[
          { id: 'trunk', label: 'Trunk' },
          { id: 'deck', label: `Deck ${String(total)}/${String(DECK_SIZE)}` },
          { id: 'stats', label: 'Stats' },
        ]}
      />

      <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
        {/* All of this takes a dropped card, so a short deck is as easy to drop on as a long one. */}
        <DropArea zone="deck" draft={draft} className="min-w-0 space-y-6 max-lg:contents">
          <div
            id={`${PANELS}-deck`}
            role="tabpanel"
            className={view === 'deck' ? '' : 'max-lg:hidden'}
          >
            <Panel
              title={
                <span className="whitespace-nowrap">
                  Deck{' '}
                  <span className={total === DECK_SIZE ? 'text-emerald-300' : 'text-amber-300'}>
                    {total}/{DECK_SIZE}
                  </span>
                </span>
              }
              actions={deckActions}
              // Right under the site's header, so no cards show between the two.
              headerClassName="flex-wrap bg-panel lg:sticky lg:top-[59px] lg:z-10"
              accent="water"
            >
              {deckNotice}
              <DeckCards editing={editing} onEnlarge={setEnlarged} />
            </Panel>
          </div>
          <div
            id={`${PANELS}-stats`}
            role="tabpanel"
            className={`grid items-start gap-6 md:grid-cols-2 ${view === 'stats' ? '' : 'max-lg:hidden'}`}
          >
            <Panel title="Stats">
              <DeckStats draft={draft} />
            </Panel>
            {sidePanels}
          </div>
        </DropArea>

        <DropArea
          zone="trunk"
          draft={draft}
          id={`${PANELS}-trunk`}
          role="tabpanel"
          className={`min-w-0 lg:sticky lg:top-20 lg:h-[calc(100dvh-6rem)] ${view === 'trunk' ? '' : 'max-lg:hidden'}`}
        >
          <Panel title="Trunk" className="flex h-full flex-col">
            <div className="min-h-0 flex-1">
              <DeckTrunk
                filters={filters}
                copiesOf={(cardId) => copiesOf(draft, cardId)}
                cannotAdd={(card) => cannotAdd(draft, card)}
                onAdd={editing.onAdd}
                onEnlarge={setEnlarged}
              />
            </div>
          </Panel>
        </DropArea>
      </div>

      {enlarged && (
        <EnlargedCard
          card={enlarged}
          editing={editing}
          onClose={() => {
            setEnlarged(null);
          }}
        />
      )}
    </DeckDrag>
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
