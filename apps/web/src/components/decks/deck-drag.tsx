'use client';

import {
  DndContext,
  DragOverlay,
  MeasuringStrategy,
  MouseSensor,
  pointerWithin,
  TouchSensor,
  useDndContext,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type Active,
  type Announcements,
  type DragEndEvent,
  type Modifier,
  type Over,
} from '@dnd-kit/core';
import { getEventCoordinates } from '@dnd-kit/utilities';
import type { ReactNode } from 'react';
import { CardImage } from '@/components/cards/card-image';
import type { Card } from '@/lib/api/client';
import type { DeckDraft } from '@/lib/deck-draft';
import { deckDrop, type DeckZone } from '@/lib/deck-drop';
import { useMediaQuery } from '@/lib/use-media-query';

/** How far the mouse must move before a press is a drag rather than a click, in pixels. */
const DRAG_DISTANCE = 6;
/** How long a finger must rest on a card before it can be dragged, leaving swipes to scroll. */
const TOUCH_HOLD_MS = 250;
const TOUCH_TOLERANCE = 5;
/**
 * The trunk and the deck are side by side from Tailwind's `lg` width. Below it they are separate
 * tabs, with nowhere to drag to, so cards there are moved with their buttons.
 */
const SIDE_BY_SIDE = '(min-width: 64rem)';

interface DraggedCard {
  readonly card: Card;
  readonly from: DeckZone;
}

interface DeckDragProps {
  readonly draft: DeckDraft;
  readonly onAdd: (card: Card) => void;
  readonly onRemove: (cardId: string) => void;
  readonly children: ReactNode;
}

/**
 * Lets cards be dragged between the trunk and the deck. Dragging only ever adds or removes a copy,
 * as the cards' own buttons do, and both places stay in view, so the page never scrolls for it.
 */
export function DeckDrag({ draft, onAdd, onRemove, children }: DeckDragProps) {
  const sideBySide = useMediaQuery(SIDE_BY_SIDE);
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: DRAG_DISTANCE } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: TOUCH_HOLD_MS, tolerance: TOUCH_TOLERANCE },
    }),
  );

  function drop({ active, over }: DragEndEvent) {
    const dragged = draggedCard(active);
    if (!dragged) return;

    const { action } = deckDrop(draft, dragged.card, dragged.from, zoneOf(over));
    if (action === 'add') onAdd(dragged.card);
    if (action === 'remove') onRemove(dragged.card.id);
  }

  return (
    <DndContext
      id="deck-editor"
      sensors={sideBySide ? sensors : []}
      collisionDetection={pointerWithin}
      autoScroll={false}
      measuring={{ droppable: { strategy: MeasuringStrategy.Always } }}
      accessibility={{ announcements: ANNOUNCEMENTS }}
      onDragEnd={drop}
    >
      {children}
      <DragOverlay dropAnimation={null} modifiers={[toPointer]}>
        <DragGhost draft={draft} />
      </DragOverlay>
    </DndContext>
  );
}

interface DropAreaProps {
  readonly zone: DeckZone;
  readonly draft: DeckDraft;
  readonly id?: string;
  readonly role?: string;
  readonly className?: string;
  readonly children: ReactNode;
}

/** A place cards can be dragged to, lit up while holding a card there would do something. */
export function DropArea({ zone, draft, className = '', children, ...props }: DropAreaProps) {
  const { setNodeRef, isOver, active } = useDroppable({ id: zone });
  const dragged = draggedCard(active);
  const wanted =
    isOver && dragged ? deckDrop(draft, dragged.card, dragged.from, zone).action : 'none';
  const refused = isOver && dragged && dragged.from !== zone && wanted === 'none';

  return (
    <div
      ref={setNodeRef}
      {...props}
      className={`${className} ${wanted === 'none' ? '' : 'outline-2 outline-offset-2 outline-accent'} ${refused ? 'outline-2 outline-offset-2 outline-red-400' : ''}`}
    >
      {children}
    </div>
  );
}

/**
 * Makes one card, in one place, draggable. The `grip` goes on what is pressed to drag it, and `ref`
 * on the whole of it. Each copy of a card has its own `copy`, as each is a thing to drag.
 */
export function useCardDrag(card: Card, from: DeckZone, copy = 0) {
  const data: DraggedCard = { card, from };
  const { setNodeRef, listeners, isDragging } = useDraggable({
    id: `${from}:${card.id}:${String(copy)}`,
    data,
  });

  return { ref: setNodeRef, grip: listeners, dragging: isDragging };
}

/** The card following the pointer, saying what letting go would do. */
function DragGhost({ draft }: { readonly draft: DeckDraft }) {
  const { active, over } = useDndContext();
  const dragged = draggedCard(active);
  if (!dragged) return null;

  const { action, text } = deckDrop(draft, dragged.card, dragged.from, zoneOf(over));

  return (
    <div
      data-drag-ghost
      className="absolute w-28 -translate-x-1/2 -translate-y-1/2 cursor-grabbing drop-shadow-[0_8px_16px_rgb(0_0_0/0.7)]"
    >
      <CardImage src={dragged.card.imageUrl} name={dragged.card.name} className="w-full" />
      {text !== null && (
        <p
          className={`absolute inset-x-[-1.5rem] top-full mt-1 px-2 py-1 text-center text-xs font-semibold ${
            action === 'none' ? 'bg-red-950 text-red-200' : 'bg-accent text-slate-900'
          }`}
        >
          {text}
        </p>
      )}
    </div>
  );
}

/** Keeps the corner the ghost centres itself on at the pointer, wherever the card was grabbed. */
const toPointer: Modifier = ({ transform, activatorEvent, activeNodeRect }) => {
  const pressed = activatorEvent ? getEventCoordinates(activatorEvent) : null;
  if (!pressed || !activeNodeRect) return transform;

  return {
    ...transform,
    x: transform.x + pressed.x - activeNodeRect.left,
    y: transform.y + pressed.y - activeNodeRect.top,
  };
};

const ANNOUNCEMENTS: Announcements = {
  onDragStart: ({ active }) => `Picked up ${nameOf(active)}.`,
  onDragOver: ({ active, over }) => `${nameOf(active)} is over ${zoneOf(over) ?? 'nothing'}.`,
  onDragEnd: ({ active, over }) => `Dropped ${nameOf(active)} on ${zoneOf(over) ?? 'nothing'}.`,
  onDragCancel: ({ active }) => `Put ${nameOf(active)} back.`,
};

function nameOf(active: Active): string {
  return draggedCard(active)?.card.name ?? 'the card';
}

function draggedCard(active: Active | null): DraggedCard | undefined {
  return active?.data.current as DraggedCard | undefined;
}

function zoneOf(over: Over | null): DeckZone | null {
  return over === null ? null : (over.id as DeckZone);
}
