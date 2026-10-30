'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { CivilizationMarks } from '@/components/cards/civilization-marks';
import { Button } from '@/components/ui/button';
import { SelectField } from '@/components/ui/select-field';
import { deckForFormat, defaultDeck, opponentFacts, type PlayableDecks } from '@/lib/lobby';
import { playComputerOpponent, type ComputerOpponent } from '@/lib/lobby-actions';
import { notify } from '@/lib/notifications';
import { DeckPicker } from './deck-picker';

interface VsComputerPanelProps {
  readonly opponents: readonly ComputerOpponent[];
  readonly decks: PlayableDecks;
  readonly disabled: boolean;
}

/**
 * A game against the computer: pick a personality and a deck legal in its format. The game is
 * private and does not change ratings. On a wider screen the pickers and the button sit in one row,
 * with what the personality is like under them.
 */
export function VsComputerPanel({ opponents, decks, disabled }: VsComputerPanelProps) {
  const router = useRouter();
  const [opponentId, setOpponentId] = useState(opponents[0]?.id ?? '');
  const opponent = opponents.find((each) => each.id === opponentId);
  const formatId = opponent?.formatId ?? '';
  const [deckId, setDeckId] = useState(() => defaultDeck(decks, formatId));
  const [pending, setPending] = useState(false);

  if (!opponent) {
    return (
      <p className="p-4 text-sm text-slate-400">
        There are no computer opponents to play right now.
      </p>
    );
  }

  async function play() {
    setPending(true);
    const outcome = await playComputerOpponent(opponentId, deckId);
    if (!outcome.ok) {
      setPending(false);
      notify(outcome.message);
      return;
    }
    // Nothing else tells this player: the game is theirs alone. The panel leaves with the page.
    router.push(`/games/${encodeURIComponent(outcome.value.gameId)}`);
  }

  return (
    <div className="space-y-3 p-4">
      <div className="grid items-end gap-4 md:grid-cols-[1fr_1fr_auto]">
        <SelectField
          label="Opponent"
          value={opponentId}
          onChange={(event) => {
            const chosen = event.target.value;
            const chosenFormat = opponents.find((each) => each.id === chosen)?.formatId ?? '';
            setOpponentId(chosen);
            setDeckId(deckForFormat(decks, chosenFormat, deckId));
          }}
        >
          {opponents.map((each) => (
            <option key={each.id} value={each.id}>
              {each.name}
            </option>
          ))}
        </SelectField>

        <DeckPicker decks={decks} formatId={formatId} value={deckId} onChange={setDeckId} />

        <Button
          className="w-full md:w-auto"
          disabled={disabled || pending || deckId === ''}
          onClick={() => void play()}
        >
          {pending ? 'Starting…' : `Play ${opponent.name}`}
        </Button>
      </div>

      <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-slate-300">
        <CivilizationMarks civilizations={opponent.civilizations} />
        <span>{opponent.description}</span>
        <span className="text-xs text-slate-500">
          {opponentFacts(opponent)} · Practice: does not change your rating
        </span>
      </p>
    </div>
  );
}
