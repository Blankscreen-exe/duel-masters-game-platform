import type { PlayableDecks } from '@/lib/lobby';

/** Who is looking at a game: one of its players, or someone watching. */
export interface GameViewer {
  readonly id: string;
  readonly username: string;
  readonly isPlayer: boolean;
  /** The decks the player may play, for a rematch; none for someone watching. */
  readonly decks: PlayableDecks;
}
