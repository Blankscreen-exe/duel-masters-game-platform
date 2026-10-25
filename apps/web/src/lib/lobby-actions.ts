import { api } from './api/client';
import type { components } from './api/schema';
import { attempt, attemptEmpty, type Outcome } from './api/outcome';

type Schemas = components['schemas'];
export type Lobby = Schemas['Lobby'];
export type OpenGame = Schemas['OpenGame'];
export type OpenGameInput = Schemas['OpenGameInput'];
export type RunningGame = Schemas['RunningGame'];
export type OnlinePlayer = Schemas['OnlinePlayer'];
export type QuickMatchStatus = Schemas['QuickMatchStatus'];
export type QuickMatchEntered = Schemas['QuickMatchEntered'];
export type QueueCount = Schemas['QueueCount'];

export function openGame(body: OpenGameInput): Promise<Outcome<OpenGame>> {
  return attempt(() => api.POST('/lobby/games', { body }));
}

export function cancelOpenGame(openGameId: string): Promise<Outcome<null>> {
  return attemptEmpty(() =>
    api.DELETE('/lobby/games/{openGameId}', { params: { path: { openGameId } } }),
  );
}

/** Joins an open game; once it starts, the live connection takes both players to it. */
export function joinOpenGame(
  openGameId: string,
  deckId: string,
): Promise<Outcome<{ gameId: string }>> {
  return attempt(() =>
    api.POST('/lobby/games/{openGameId}/join', {
      params: { path: { openGameId } },
      body: { deckId },
    }),
  );
}

export function enterQuickMatch(
  formatId: string,
  deckId: string,
): Promise<Outcome<QuickMatchEntered>> {
  return attempt(() => api.POST('/quick-match', { body: { formatId, deckId } }));
}

export function leaveQuickMatch(): Promise<Outcome<null>> {
  return attemptEmpty(() => api.DELETE('/quick-match'));
}
