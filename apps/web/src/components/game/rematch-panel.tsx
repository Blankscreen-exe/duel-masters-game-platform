'use client';

import { useRouter } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';
import { CivilizationMarks } from '@/components/cards/civilization-marks';
import { useLiveMessage } from '@/components/live/live-provider';
import { DeckPicker } from '@/components/lobby/deck-picker';
import { Button } from '@/components/ui/button';
import { agreeToRematch, chooseRematchDeck, declineRematch, getRematch } from '@/lib/lobby-actions';
import { notify } from '@/lib/notifications';
import {
  agreeLabel,
  endedText,
  rematchStep,
  secondsLeft,
  tossCallerText,
  type Rematch,
} from '@/lib/rematch';
import { minutesAndSeconds } from '@/lib/time-text';
import type { PlayableDecks } from '@/lib/lobby';

interface RematchPanelProps {
  /** The game that ended. */
  readonly gameId: string;
  readonly viewerName: string;
  readonly decks: PlayableDecks;
}

/**
 * The rematch on a player's result screen: both decks, the player's to switch, and the way to
 * offer or accept it, or to go back to the lobby, which declines it. Leaving the page declines
 * it too, so the other player is not kept waiting.
 */
export function RematchPanel({ gameId, viewerName, decks }: RematchPanelProps) {
  const router = useRouter();
  // Undefined while loading; null when there is none to decide.
  const [rematch, setRematch] = useState<Rematch | null | undefined>(undefined);
  const [pending, setPending] = useState(false);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    let current = true;
    void getRematch(gameId).then((outcome) => {
      if (!current) return;
      // A live update may have arrived first, and is newer.
      if (outcome.ok) setRematch((shown) => (shown === undefined ? outcome.value : shown));
      else setRematch(null);
    });
    return () => {
      current = false;
    };
  }, [gameId]);

  useLiveMessage('rematch', (message) => {
    if (message.rematch.gameId === gameId) setRematch(message.rematch);
  });
  useLiveMessage('rematch_ended', (message) => {
    if (message.gameId !== gameId) return;
    setRematch(null);
    const text = endedText(message.reason, message.by, viewerName);
    if (text) notify(text);
    router.push('/play');
  });

  const open = rematch !== null && rematch !== undefined;
  useEffect(() => {
    if (!open) return;
    const tick = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);
    return () => {
      window.clearInterval(tick);
    };
  }, [open]);

  useDeclineOnLeaving(gameId, open);

  if (rematch === undefined) {
    return <p className="mt-6 text-sm text-slate-400">Checking for a rematch…</p>;
  }
  if (rematch === null) return null;

  const step = rematchStep(rematch);

  async function chooseDeck(deckId: string) {
    setPending(true);
    const outcome = await chooseRematchDeck(gameId, deckId);
    setPending(false);
    if (outcome.ok) setRematch(outcome.value);
    else notify(outcome.message);
  }

  async function agree() {
    setPending(true);
    const outcome = await agreeToRematch(gameId);
    setPending(false);
    if (!outcome.ok) {
      notify(outcome.message);
    } else if ('you' in outcome.value) {
      setRematch(outcome.value);
    } else {
      router.push(`/games/${encodeURIComponent(outcome.value.gameId)}`);
    }
  }

  return (
    <section
      aria-label="Rematch"
      className="mt-6 space-y-4 border-t border-slate-700/60 pt-5 text-left"
    >
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="font-display text-lg font-bold tracking-[0.15em] text-white uppercase">
          Rematch
        </h2>
        <span className="text-xs text-slate-400 tabular-nums" aria-live="off">
          Open for {minutesAndSeconds(secondsLeft(rematch.expiresAt, now))}
        </span>
      </div>

      {step === 'accept' && !rematch.opponent.computer && (
        <p role="status" className="text-sm font-semibold text-accent">
          {rematch.opponent.username} offers a rematch.
        </p>
      )}

      <DeckPicker
        label="Your deck"
        decks={decks}
        formatId={rematch.formatId}
        value={rematch.you.deck.id ?? ''}
        onChange={(deckId) => void chooseDeck(deckId)}
      />

      <div className="text-sm">
        <p className="text-[11px] font-semibold tracking-wider text-slate-400 uppercase">
          {rematch.opponent.username}’s deck
        </p>
        <p className="mt-1 flex items-center gap-2 text-slate-100">
          {rematch.opponent.deck.name}
          <CivilizationMarks civilizations={rematch.opponent.deck.civilizations} />
          {rematch.opponent.ready && (
            <span className="text-xs font-semibold tracking-wider text-civ-nature uppercase">
              Agreed
            </span>
          )}
        </p>
      </div>

      <p className="text-xs text-slate-400">{tossCallerText(rematch)}</p>

      <Button
        className="w-full"
        disabled={pending || step === 'waiting'}
        onClick={() => void agree()}
      >
        {agreeLabel(rematch)}
      </Button>
    </section>
  );
}

/**
 * Declines an open rematch when the player leaves its page: closing or reloading the tab, or
 * moving to another page. The decline on unmounting waits a moment, so React drawing the panel
 * twice in development, which unmounts and remounts it at once, does not decline it.
 */
function useDeclineOnLeaving(gameId: string, open: boolean) {
  const pendingDecline = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (!open) return;
    window.clearTimeout(pendingDecline.current);

    const decline = () => {
      void declineRematch(gameId);
    };
    window.addEventListener('pagehide', decline);

    return () => {
      window.removeEventListener('pagehide', decline);
      pendingDecline.current = window.setTimeout(decline, 0);
    };
  }, [gameId, open]);
}
