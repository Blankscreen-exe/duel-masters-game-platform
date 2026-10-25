import { api, errorMessage, UNREACHABLE } from './api/client';
import type { components } from './api/schema';
import { attempt, attemptEmpty, type Outcome } from './api/outcome';
import type { Rematch } from './rematch';

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

/** The rematch of a game that just ended, or null when there is none to decide. */
export async function getRematch(gameId: string): Promise<Outcome<Rematch | null>> {
  try {
    const { data, error, response } = await api.GET('/rematches/{gameId}', {
      params: { path: { gameId } },
    });
    if (data) return { ok: true, value: data };
    if (response.status === 404) return { ok: true, value: null };
    return { ok: false, message: errorMessage(error) };
  } catch {
    return { ok: false, message: UNREACHABLE };
  }
}

export function chooseRematchDeck(gameId: string, deckId: string): Promise<Outcome<Rematch>> {
  return attempt(() =>
    api.PUT('/rematches/{gameId}/deck', { params: { path: { gameId } }, body: { deckId } }),
  );
}

/** Offers the rematch or accepts it: the rematch, waiting for the opponent, or the new game. */
export function agreeToRematch(gameId: string): Promise<Outcome<Rematch | { gameId: string }>> {
  return attempt(() => api.POST('/rematches/{gameId}/ready', { params: { path: { gameId } } }));
}

/**
 * Declines a rematch. Sent with keepalive, so it still arrives when the player is closing the
 * page.
 */
export function declineRematch(gameId: string): Promise<Outcome<null>> {
  return attemptEmpty(() =>
    api.DELETE('/rematches/{gameId}', { params: { path: { gameId } }, keepalive: true }),
  );
}
