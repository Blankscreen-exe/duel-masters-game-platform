import { describe, expect, it } from 'vitest';
import {
  agreeLabel,
  endedText,
  rematchStep,
  secondsLeft,
  tossCallerText,
  type Rematch,
} from './rematch';

function rematch(
  overrides: { youReady?: boolean; opponentReady?: boolean; tossCallerId?: string } = {},
): Rematch {
  return {
    gameId: 'g1',
    name: 'Friendly',
    formatId: 'open',
    formatName: 'Open',
    you: {
      id: 'u1',
      username: 'Shobu',
      deck: { id: 'd1', name: 'Fire Rush', civilizations: ['Fire'] },
      ready: overrides.youReady ?? false,
      computer: false,
    },
    opponent: {
      id: 'u2',
      username: 'Kokujo',
      deck: { name: 'Water Wall', civilizations: ['Water'] },
      ready: overrides.opponentReady ?? false,
      computer: false,
    },
    tossCallerId: overrides.tossCallerId,
    expiresAt: '2026-10-25T12:02:00Z',
  };
}

describe('agreeLabel', () => {
  it('says what agreeing does at each step', () => {
    expect(agreeLabel(rematch())).toBe('Offer a rematch');
    expect(agreeLabel(rematch({ youReady: true }))).toBe('Waiting for your opponent…');
    expect(agreeLabel(rematch({ opponentReady: true }))).toBe('Accept the rematch');
  });

  it('just plays again against the computer, which has always agreed', () => {
    const offered = rematch({ opponentReady: true });

    expect(agreeLabel({ ...offered, opponent: { ...offered.opponent, computer: true } })).toBe(
      'Play again',
    );
  });
});

describe('rematchStep', () => {
  it('lets a player offer when nobody has agreed', () => {
    expect(rematchStep(rematch())).toBe('offer');
  });

  it('waits for the opponent once the player agreed', () => {
    expect(rematchStep(rematch({ youReady: true }))).toBe('waiting');
  });

  it('lets a player accept their opponent’s offer', () => {
    expect(rematchStep(rematch({ opponentReady: true }))).toBe('accept');
  });
});

describe('tossCallerText', () => {
  it('says who calls the next coin toss', () => {
    expect(tossCallerText(rematch({ tossCallerId: 'u1' }))).toBe(
      'You call the coin toss, having lost the last game.',
    );
    expect(tossCallerText(rematch({ tossCallerId: 'u2' }))).toBe(
      'Kokujo calls the coin toss, having lost the last game.',
    );
    expect(tossCallerText(rematch())).toBe('Whoever calls the coin toss is picked at random.');
  });
});

describe('endedText', () => {
  it('says why there is no rematch', () => {
    expect(endedText('declined', 'Kokujo', 'Shobu')).toBe('Kokujo declined the rematch.');
    expect(endedText('left', 'Kokujo', 'Shobu')).toBe('Kokujo left, so there is no rematch.');
    expect(endedText('expired', undefined, 'Shobu')).toBe('The rematch offer ran out.');
  });

  it('says nothing to the player who ended it', () => {
    expect(endedText('declined', 'Shobu', 'Shobu')).toBeNull();
  });
});

describe('secondsLeft', () => {
  it('counts the seconds until the offer runs out', () => {
    const expiresAt = '2026-10-25T12:02:00Z';
    expect(secondsLeft(expiresAt, Date.parse('2026-10-25T12:00:00Z'))).toBe(120);
    expect(secondsLeft(expiresAt, Date.parse('2026-10-25T12:01:59.5Z'))).toBe(1);
    expect(secondsLeft(expiresAt, Date.parse('2026-10-25T12:03:00Z'))).toBe(0);
  });
});
