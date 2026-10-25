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

/** The decks a player may play: the starter decks, every player's default first, and their own. */
export interface PlayableDecks {
  readonly starters: readonly DeckSummary[];
  readonly own: readonly DeckSummary[];
}

/** A player's decks as a picker shows them for one format. */
export interface DecksForFormat {
  /** The starter decks that can be played in the format, in their order. */
  readonly starters: DeckSummary[];
  /** The player's own decks that can be played in it, by name. */
  readonly legal: DeckSummary[];
  /** The player's own decks that cannot, by name, so a picker can show why. */
  readonly notLegal: DeckSummary[];
}

/** A player's decks for a format, as a picker shows them. */
export function decksForFormat(decks: PlayableDecks, formatId: string): DecksForFormat {
  const playable = (deck: DeckSummary) => deck.legalFormatIds.includes(formatId);
  const byName = [...decks.own].sort((a, b) => a.name.localeCompare(b.name));
  return {
    starters: decks.starters.filter(playable),
    legal: byName.filter(playable),
    notLegal: byName.filter((deck) => !playable(deck)),
  };
}

/**
 * The deck picked at first for a format: the first starter deck that can be played in it, every
 * player's default, or else the player's first own deck that can; empty when none can.
 */
export function defaultDeck(decks: PlayableDecks, formatId: string): string {
  const { starters, legal } = decksForFormat(decks, formatId);
  return (starters[0] ?? legal[0])?.id ?? '';
}

/** How many players wait for a quick match in a format. */
export function waitingIn(queue: readonly QueueCount[], formatId: string): number {
  return queue.find((count) => count.formatId === formatId)?.players ?? 0;
}

/** The address that leads a friend to an open game, listed or not. */
export function inviteLink(origin: string, openGameId: string): string {
  return `${origin}/play/join/${encodeURIComponent(openGameId)}`;
}
