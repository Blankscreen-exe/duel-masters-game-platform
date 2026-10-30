import { describe, expect, it } from 'vitest';
import type { DeckSummary } from './api/client';
import {
  activityOf,
  decksForFormat,
  defaultDeck,
  describeActivity,
  inviteLink,
  waitingIn,
} from './lobby';

const lobby = {
  openGames: [],
  runningGames: [],
  paused: false,
  online: [
    {
      id: 'u1',
      username: 'Shobu',
      activity: { kind: 'hosting' as const, gameId: 'g1', computer: false },
    },
    { id: 'u2', username: 'Kokujo' },
  ],
};

function deck(id: string, name: string, legalFormatIds: string[]): DeckSummary {
  return {
    id,
    name,
    cardCount: 40,
    civilizations: [],
    legalFormatIds,
    updatedAt: '2026-10-25T00:00:00Z',
  };
}

describe('activityOf', () => {
  it('finds what the player is doing', () => {
    expect(activityOf(lobby, 'u1')).toEqual({ kind: 'hosting', gameId: 'g1', computer: false });
    expect(activityOf(lobby, 'u2')).toBeNull();
    expect(activityOf(lobby, 'nobody')).toBeNull();
  });
});

describe('describeActivity', () => {
  it('says what a player is doing', () => {
    expect(describeActivity({ kind: 'hosting', computer: false })).toBe('Waiting for an opponent');
    expect(describeActivity({ kind: 'queued', formatName: 'Open', computer: false })).toBe(
      'Quick match in Open',
    );
    expect(describeActivity({ kind: 'queued', computer: false })).toBe('Quick match');
    expect(describeActivity({ kind: 'playing', gameId: 'g1', computer: false })).toBe('Playing');
    expect(describeActivity({ kind: 'playing', gameId: 'g1', computer: true })).toBe(
      'Playing the computer',
    );
    expect(describeActivity({ kind: 'rematch', gameId: 'g1', computer: false })).toBe(
      'Deciding on a rematch',
    );
    expect(describeActivity(null)).toBe('In the lobby');
    expect(describeActivity(undefined)).toBe('In the lobby');
  });
});

describe('decksForFormat and defaultDeck', () => {
  const starters = [
    deck('fire', 'Fire: Bolshack Rush', ['open', 'classic']),
    deck('water', 'Water: Depthcon Tide', ['open']),
  ];
  const own = [
    deck('a', 'Zebra', ['open']),
    deck('b', 'Apple', []),
    deck('c', 'Mango', ['open', 'classic']),
  ];

  it('shows the starter decks in their order, then the own decks that can be played, by name', () => {
    const { starters: shown, legal, notLegal } = decksForFormat({ starters, own }, 'open');

    expect(shown.map((d) => d.id)).toEqual(['fire', 'water']);
    expect(legal.map((d) => d.name)).toEqual(['Mango', 'Zebra']);
    expect(notLegal.map((d) => d.name)).toEqual(['Apple']);
    expect(decksForFormat({ starters, own }, 'classic').starters.map((d) => d.id)).toEqual([
      'fire',
    ]);
  });

  it('picks the first starter deck at first, which is Fire', () => {
    expect(defaultDeck({ starters, own }, 'open')).toBe('fire');
  });

  it("falls back to the player's own deck, then to none", () => {
    expect(defaultDeck({ starters: [], own }, 'open')).toBe('c');
    expect(defaultDeck({ starters: [], own }, 'nowhere')).toBe('');
  });
});

describe('waitingIn', () => {
  it('counts the players waiting in a format', () => {
    const queue = [{ formatId: 'open', players: 2 }];
    expect(waitingIn(queue, 'open')).toBe(2);
    expect(waitingIn(queue, 'classic')).toBe(0);
  });
});

describe('inviteLink', () => {
  it('leads to the open game on this site', () => {
    expect(inviteLink('https://duelmasters.example', 'a b')).toBe(
      'https://duelmasters.example/play/join/a%20b',
    );
  });
});
