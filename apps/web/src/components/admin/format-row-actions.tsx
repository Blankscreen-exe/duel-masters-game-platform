'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { reorderFormats } from '@/lib/admin-actions';
import { movedFormat, type AdminFormat } from '@/lib/formats';
import { notify } from '@/lib/notifications';
import { FormatRetireButton } from './format-retire-button';

interface FormatRowActionsProps {
  readonly format: Pick<AdminFormat, 'id' | 'name' | 'retired' | 'opponents'>;
  /** Every format's id, in order. */
  readonly order: readonly string[];
}

/** The list's way to move a format up or down and to retire it, which draws the list again after. */
export function FormatRowActions({ format, order }: FormatRowActionsProps) {
  const router = useRouter();
  const [moving, setMoving] = useState(false);

  async function move(direction: 'up' | 'down') {
    const moved = movedFormat(order, format.id, direction);
    if (!moved) return;

    setMoving(true);
    const outcome = await reorderFormats(moved);
    setMoving(false);
    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    router.refresh();
  }

  return (
    <>
      <Button
        variant="quiet"
        aria-label={`Move ${format.name} up`}
        title="Move up"
        disabled={moving || movedFormat(order, format.id, 'up') === null}
        onClick={() => void move('up')}
      >
        ↑
      </Button>
      <Button
        variant="quiet"
        aria-label={`Move ${format.name} down`}
        title="Move down"
        disabled={moving || movedFormat(order, format.id, 'down') === null}
        onClick={() => void move('down')}
      >
        ↓
      </Button>
      <FormatRetireButton
        format={format}
        onChanged={() => {
          router.refresh();
        }}
      />
    </>
  );
}
