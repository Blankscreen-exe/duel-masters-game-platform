import { describe, expect, it } from 'vitest';
import type { DeckSummary } from './api/client';
import { activityOf, decksForFormat, describeActivity, inviteLink, waitingIn } from './lobby';

const lobby = {
  openGames: [],
  runningGames: [],
  paused: false,
  online: [
    { id: 'u1', username: 'Shobu', activity: { kind: 'hosting' as const, gameId: 'g1' } },
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
    expect(activityOf(lobby, 'u1')).toEqual({ kind: 'hosting', gameId: 'g1' });
    expect(activityOf(lobby, 'u2')).toBeNull();
    expect(activityOf(lobby, 'nobody')).toBeNull();
  });
});

describe('describeActivity', () => {
  it('says what a player is doing', () => {
    expect(describeActivity({ kind: 'hosting' })).toBe('Waiting for an opponent');
    expect(describeActivity({ kind: 'queued', formatName: 'Open' })).toBe('Quick match in Open');
    expect(describeActivity({ kind: 'queued' })).toBe('Quick match');
    expect(describeActivity({ kind: 'playing', gameId: 'g1' })).toBe('Playing');
    expect(describeActivity(null)).toBe('In the lobby');
    expect(describeActivity(undefined)).toBe('In the lobby');
  });
});

describe('decksForFormat', () => {
  it('puts the decks that can be played first, by name', () => {
    const decks = [
      deck('a', 'Zebra', ['open']),
      deck('b', 'Apple', []),
      deck('c', 'Mango', ['open', 'classic']),
    ];

    const { legal, notLegal } = decksForFormat(decks, 'open');

    expect(legal.map((d) => d.name)).toEqual(['Mango', 'Zebra']);
    expect(notLegal.map((d) => d.name)).toEqual(['Apple']);
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
