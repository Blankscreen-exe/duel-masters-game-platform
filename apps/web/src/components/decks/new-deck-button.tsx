'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { createDeck } from '@/lib/deck-actions';
import { notify } from '@/lib/notifications';

/** Starts an empty deck and opens it in the deck builder. */
export function NewDeckButton() {
  const router = useRouter();
  const [pending, setPending] = useState(false);

  async function start() {
    setPending(true);
    const outcome = await createDeck({ name: 'New deck', cards: [] });
    if (!outcome.ok) {
      setPending(false);
      notify(outcome.message);
      return;
    }
    router.push(`/decks/${encodeURIComponent(outcome.value.id)}`);
  }

  return (
    <Button disabled={pending} onClick={() => void start()}>
      {pending ? 'Creating…' : 'New deck'}
    </Button>
  );
}
