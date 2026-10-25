import type { DuelFinishedMessage } from '@duel-masters/duel-board';
import { describe, expect, it } from 'vitest';
import { describeResult } from './game-result';

function finished(overrides: Partial<DuelFinishedMessage> = {}): DuelFinishedMessage {
  return {
    header: 'duel_finished',
    duelId: 'g1',
    winner: { uid: 'u1', username: 'Shobu' },
    ending: 'play',
    counted: true,
    turns: 12,
    durationSeconds: 754,
    ...overrides,
  };
}

describe('describeResult', () => {
  it('tells the winner they won', () => {
    expect(describeResult(finished(), 'u1', true)).toEqual({
      headline: 'You won',
      details: ['Won by the rules of the game.', '12 turns in 12:34.'],
      outcome: 'won',
    });
  });

  it('tells the other player they lost', () => {
    const result = describeResult(finished({ ending: 'surrender' }), 'u2', true);

    expect(result.headline).toBe('You lost');
    expect(result.outcome).toBe('lost');
    expect(result.details[0]).toBe('A player surrendered.');
  });

  it('tells the winner why their opponent lost', () => {
    expect(describeResult(finished({ ending: 'surrender' }), 'u1', true).details[0]).toBe(
      'Your opponent surrendered.',
    );
    expect(describeResult(finished({ ending: 'disconnect' }), 'u1', true).details[0]).toBe(
      'Your opponent left and did not come back.',
    );
  });

  it('names the winner to someone watching', () => {
    const result = describeResult(finished(), 'u3', false);

    expect(result.headline).toBe('Shobu won');
    expect(result.outcome).toBe('none');
  });

  it('says when nobody won', () => {
    const result = describeResult(
      finished({
        winner: undefined,
        ending: 'aborted',
        counted: false,
        turns: 1,
        durationSeconds: 5,
      }),
      'u1',
      true,
    );

    expect(result).toEqual({
      headline: 'The game was stopped',
      details: ['The game ended without a winner.', '1 turn in 0:05.'],
      outcome: 'none',
    });
  });

  it('says when a game was too short to count', () => {
    expect(describeResult(finished({ counted: false }), 'u1', true).details).toContain(
      'Too short to count towards ratings.',
    );
  });
});
