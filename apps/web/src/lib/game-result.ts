import type { DuelFinishedMessage } from '@duel-masters/duel-board';
import { minutesAndSeconds } from './time-text';

/** What the result screen says about a finished game. */
export interface GameResult {
  /** Such as "You won" or "Shobu won". */
  readonly headline: string;
  /** How it ended, and whether it counted. */
  readonly details: readonly string[];
  /** Whether the viewer won, lost, or neither (a spectator, or no winner). */
  readonly outcome: 'won' | 'lost' | 'none';
}

/** How a game ended, in words, for the viewer: a player of the game, or someone watching. */
export function describeResult(
  message: DuelFinishedMessage,
  viewerId: string,
  isPlayer: boolean,
): GameResult {
  const winner = message.winner;
  const viewerWon = winner?.uid === viewerId;

  let headline: string;
  let outcome: GameResult['outcome'] = 'none';
  if (!winner) {
    headline = 'The game was stopped';
  } else if (isPlayer) {
    headline = viewerWon ? 'You won' : 'You lost';
    outcome = viewerWon ? 'won' : 'lost';
  } else {
    headline = `${winner.username ?? 'A player'} won`;
  }

  const details = [endingText(message, isPlayer && viewerWon), playedText(message)];
  if (message.practice) details.push('A practice game: it does not count towards ratings.');
  else if (!message.counted && winner) details.push('Too short to count towards ratings.');

  return { headline, details, outcome };
}

function endingText(message: DuelFinishedMessage, viewerWon: boolean): string {
  switch (message.ending) {
    case 'surrender':
      return viewerWon ? 'Your opponent surrendered.' : 'A player surrendered.';
    case 'disconnect':
      return viewerWon
        ? 'Your opponent left and did not come back.'
        : 'A player left and did not come back.';
    case 'aborted':
      return 'The game ended without a winner.';
    default:
      return 'Won by the rules of the game.';
  }
}

function playedText(message: DuelFinishedMessage): string {
  const turns = message.turns === 1 ? '1 turn' : `${String(message.turns)} turns`;
  return `${turns} in ${minutesAndSeconds(message.durationSeconds)}.`;
}
