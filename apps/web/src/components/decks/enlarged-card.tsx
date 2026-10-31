'use client';

import { CardFacts } from '@/components/cards/card-facts';
import { CardImage } from '@/components/cards/card-image';
import { Modal, ModalClose } from '@/components/ui/modal';
import type { Card } from '@/lib/api/client';
import { DeckCardButtons, type DeckEditing } from './deck-card-buttons';

interface EnlargedCardProps {
  readonly card: Card;
  readonly editing: DeckEditing;
  readonly onClose: () => void;
}

/** A card shown large, to read it, with its copies in the deck to change while looking at it. */
export function EnlargedCard({ card, editing, onClose }: EnlargedCardProps) {
  return (
    <Modal
      title={card.name}
      onClose={onClose}
      className="w-[min(100%-2rem,52rem)]"
      // First in the dialog, which takes the focus: anything lower scrolls a phone past the picture.
      actions={<ModalClose variant="secondary">Close</ModalClose>}
    >
      <div className="grid gap-5 md:grid-cols-[minmax(0,24rem)_minmax(0,1fr)]">
        <CardImage
          src={card.imageUrl}
          name={card.name}
          priority
          className="mx-auto w-full max-w-sm"
        />
        <div className="flex flex-col">
          <CardFacts card={card} columns="grid-cols-2" />
          <div className="mt-5 flex flex-wrap items-center gap-2 border-t border-slate-700/60 pt-4 md:mt-auto">
            <span className="mr-1 text-sm font-semibold text-slate-300">In the deck</span>
            <DeckCardButtons card={card} editing={editing} showCount />
          </div>
        </div>
      </div>
    </Modal>
  );
}
