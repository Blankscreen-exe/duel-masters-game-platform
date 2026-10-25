import type { DeckSummary } from './api/client';
import type { components } from './api/schema';

type Lobby = components['schemas']['Lobby'];
type PlayerActivity = components['schemas']['PlayerActivity'];
type QueueCount = components['schemas']['QueueCount'];

/** What the player is doing, as the lobby says, or null for nothing in particular. */
export function activityOf(lobby: Lobby, userId: string): PlayerActivity | null {
  return lobby.online.find((player) => player.id === userId)?.activity ?? null;
}

/** What a player is doing, in a few words for the online list. */
export function describeActivity(activity: PlayerActivity | null | undefined): string {
  switch (activity?.kind) {
    case 'hosting':
      return 'Waiting for an opponent';
    case 'queued':
      return activity.formatName ? `Quick match in ${activity.formatName}` : 'Quick match';
    case 'playing':
      return 'Playing';
    case 'rematch':
      return 'Deciding on a rematch';
    default:
      return 'In the lobby';
  }
}

/**
 * A player's decks for a format: the ones that can be played in it first, by name, and the
 * others after, so a picker can show why they cannot be chosen.
 */
export function decksForFormat(
  decks: readonly DeckSummary[],
  formatId: string,
): { legal: DeckSummary[]; notLegal: DeckSummary[] } {
  const byName = [...decks].sort((a, b) => a.name.localeCompare(b.name));
  return {
    legal: byName.filter((deck) => deck.legalFormatIds.includes(formatId)),
    notLegal: byName.filter((deck) => !deck.legalFormatIds.includes(formatId)),
  };
}

/** How many players wait for a quick match in a format. */
export function waitingIn(queue: readonly QueueCount[], formatId: string): number {
  return queue.find((count) => count.formatId === formatId)?.players ?? 0;
}

/** The address that leads a friend to an open game, listed or not. */
export function inviteLink(origin: string, openGameId: string): string {
  return `${origin}/play/join/${encodeURIComponent(openGameId)}`;
}
