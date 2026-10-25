import type { DeckSummary } from '@/lib/api/client';

/** Who is looking at a game: one of its players, or someone watching. */
export interface GameViewer {
  readonly id: string;
  readonly username: string;
  readonly isPlayer: boolean;
  /** The player's decks, for a rematch; empty for someone watching. */
  readonly decks: readonly DeckSummary[];
}
