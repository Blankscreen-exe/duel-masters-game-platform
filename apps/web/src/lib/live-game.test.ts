import { describe, expect, it } from 'vitest';
import { playerNamed, type GamePlayer, type LiveGamePlayers } from './live-game';

function player(username: string): GamePlayer {
  return { id: username, username, deckName: 'A deck', computer: false };
}

describe('playerNamed', () => {
  const players: LiveGamePlayers = [player('Shobu'), player('Kokujo')];

  it('finds either player', () => {
    expect(playerNamed(players, 'Shobu')).toBe(players[0]);
    expect(playerNamed(players, 'Kokujo')).toBe(players[1]);
  });

  it('finds no one else', () => {
    expect(playerNamed(players, 'Mimi')).toBeUndefined();
  });
});
