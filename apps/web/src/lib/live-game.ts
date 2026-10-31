import type { components } from './api/schema';

export type GamePlayer = components['schemas']['GamePlayer'];

/** The two players of a running game. */
export type LiveGamePlayers = readonly [GamePlayer, GamePlayer];

/** The player the board names, as the board knows players only by name. */
export function playerNamed(players: LiveGamePlayers, username: string): GamePlayer | undefined {
  return players.find((player) => player.username === username);
}
