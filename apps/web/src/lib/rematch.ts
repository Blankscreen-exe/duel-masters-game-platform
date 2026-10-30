import type { components } from './api/schema';

export type Rematch = components['schemas']['Rematch'];
type EndedReason = components['schemas']['RealtimeRematchEnded']['reason'];

/**
 * Where a rematch stands for the player: nobody has agreed, so they may offer it; they have, and
 * wait for their opponent; or their opponent has, and they may accept.
 */
export type RematchStep = 'offer' | 'waiting' | 'accept';

export function rematchStep(rematch: Rematch): RematchStep {
  if (rematch.you.ready) return 'waiting';
  if (rematch.opponent.ready) return 'accept';
  return 'offer';
}

/** The words on the button that agrees: the computer has always agreed, so nothing is accepted. */
export function agreeLabel(rematch: Rematch): string {
  switch (rematchStep(rematch)) {
    case 'offer':
      return 'Offer a rematch';
    case 'waiting':
      return 'Waiting for your opponent…';
    default:
      return rematch.opponent.computer ? 'Play again' : 'Accept the rematch';
  }
}

/** Who calls the new game's coin toss, in words for the player. */
export function tossCallerText(rematch: Rematch): string {
  switch (rematch.tossCallerId) {
    case rematch.you.id:
      return 'You call the coin toss, having lost the last game.';
    case rematch.opponent.id:
      return `${rematch.opponent.username} calls the coin toss, having lost the last game.`;
    default:
      return 'Whoever calls the coin toss is picked at random.';
  }
}

/**
 * Why a rematch will not be played, for the player, or null when the player ended it themselves
 * and needs no telling.
 */
export function endedText(
  reason: EndedReason,
  by: string | undefined,
  viewerName: string,
): string | null {
  if (by === viewerName) return null;

  switch (reason) {
    case 'declined':
      return `${by ?? 'Your opponent'} declined the rematch.`;
    case 'left':
      return `${by ?? 'Your opponent'} left, so there is no rematch.`;
    default:
      return 'The rematch offer ran out.';
  }
}

/** Seconds until a rematch offer runs out, never below none. */
export function secondsLeft(expiresAt: string, now: number): number {
  return Math.max(0, Math.ceil((Date.parse(expiresAt) - now) / 1000));
}
