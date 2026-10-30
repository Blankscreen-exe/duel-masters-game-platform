'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { setOpponentOffered } from '@/lib/admin-actions';
import type { AdminComputerOpponent } from '@/lib/computer-opponents';
import { notify } from '@/lib/notifications';

interface OpponentOfferButtonProps {
  readonly opponent: Pick<AdminComputerOpponent, 'id' | 'name' | 'offered'>;
  readonly onChanged: (changed: AdminComputerOpponent) => void;
  readonly disabled?: boolean;
  readonly title?: string;
}

/** Offers a computer opponent to players, or retires it. The server refuses a deck not ready. */
export function OpponentOfferButton({
  opponent,
  onChanged,
  disabled = false,
  title,
}: OpponentOfferButtonProps) {
  const [pending, setPending] = useState(false);

  async function change() {
    setPending(true);
    const outcome = await setOpponentOffered(opponent.id, !opponent.offered);
    setPending(false);
    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    notify(
      outcome.value.offered
        ? `Players can now play ${opponent.name}`
        : `${opponent.name} is retired`,
    );
    onChanged(outcome.value);
  }

  return (
    <Button
      variant="secondary"
      disabled={disabled || pending}
      title={title}
      onClick={() => void change()}
    >
      {opponent.offered ? 'Retire' : 'Offer to players'}
    </Button>
  );
}
