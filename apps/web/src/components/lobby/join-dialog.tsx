'use client';

import { useRouter } from 'next/navigation';
import { useState, type ReactNode } from 'react';
import { RankBadge } from '@/components/players/rank-badge';
import { Button } from '@/components/ui/button';
import { Modal, ModalClose } from '@/components/ui/modal';
import type { DeckSummary } from '@/lib/api/client';
import { joinOpenGame, type OpenGame } from '@/lib/lobby-actions';
import { notify } from '@/lib/notifications';
import { DeckPicker, firstLegalDeck } from './deck-picker';

interface JoinFormProps {
  readonly game: OpenGame;
  readonly decks: readonly DeckSummary[];
  /** The way back without joining, beside the join button. */
  readonly cancel: ReactNode;
}

/** Picks one of the player's decks legal in an open game's format, and joins it. */
export function JoinForm({ game, decks, cancel }: JoinFormProps) {
  const router = useRouter();
  const [deckId, setDeckId] = useState(() => firstLegalDeck(decks, game.formatId));
  const [pending, setPending] = useState(false);

  async function join() {
    setPending(true);
    const outcome = await joinOpenGame(game.id, deckId);
    if (!outcome.ok) {
      setPending(false);
      notify(outcome.message);
      return;
    }
    router.push(`/games/${encodeURIComponent(outcome.value.gameId)}`);
  }

  return (
    <>
      <p className="mb-4 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-slate-300">
        <span>
          Against <strong className="text-white">{game.host.username}</strong>
        </span>
        {game.host.rank && <RankBadge rank={game.host.rank} />}
        <span>in {game.formatName}.</span>
      </p>
      <DeckPicker decks={decks} formatId={game.formatId} value={deckId} onChange={setDeckId} />
      <div className="mt-6 flex justify-end gap-2">
        {cancel}
        <Button disabled={pending || deckId === ''} onClick={() => void join()}>
          {pending ? 'Joining…' : 'Join game'}
        </Button>
      </div>
    </>
  );
}

/** Joins an open game from the lobby list. */
export function JoinDialog({
  game,
  decks,
  onClose,
}: {
  readonly game: OpenGame;
  readonly decks: readonly DeckSummary[];
  readonly onClose: () => void;
}) {
  return (
    <Modal title={`Join ${game.name}`} onClose={onClose}>
      <JoinForm
        game={game}
        decks={decks}
        cancel={<ModalClose variant="secondary">Cancel</ModalClose>}
      />
    </Modal>
  );
}
