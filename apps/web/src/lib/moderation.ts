import type { ApiError } from './api/client';
import type { components } from './api/schema';
import { dateTimeText } from './dates';

export type Penalty = components['schemas']['Penalty'];

/** A length a ban or mute can last: minutes, or none for good. */
export interface PenaltyLength {
  readonly label: string;
  readonly minutes?: number;
}

const HOUR = 60;
const DAY = 24 * HOUR;

/** The lengths an admin bans a player for. */
export const BAN_LENGTHS: readonly PenaltyLength[] = [
  { label: '1 day', minutes: DAY },
  { label: '7 days', minutes: 7 * DAY },
  { label: '30 days', minutes: 30 * DAY },
  { label: 'For good' },
];

/** The lengths an admin mutes a player for. */
export const MUTE_LENGTHS: readonly PenaltyLength[] = [
  { label: '1 hour', minutes: HOUR },
  { label: '24 hours', minutes: DAY },
  { label: '7 days', minutes: 7 * DAY },
  { label: 'For good' },
];

/** The longest ban reason, as the server allows. */
export const BAN_REASON_MAX_LENGTH = 200;

/** How long a ban or mute lasts: "until 2 Nov 2026, 12:00 UTC", or "for good". */
export function penaltyEnd(until: string | undefined): string {
  return until === undefined ? 'for good' : `until ${dateTimeText(until)}`;
}

/** Whether a ban or mute is still on at now; a timed one lifts itself. */
export function inForce(penalty: Penalty | undefined, now: number): penalty is Penalty {
  return penalty !== undefined && (penalty.until === undefined || Date.parse(penalty.until) > now);
}

/** What a banned player is told when they log in, from the server's "banned" refusal. */
export function bannedMessage(refusal: ApiError): string {
  const reason = refusal.reason ? ` Reason: ${refusal.reason}` : '';
  return `This account is banned ${penaltyEnd(refusal.until)}.${reason}`;
}

/** What a muted player is told in place of the chat box. */
export function mutedMessage(mute: Penalty): string {
  return `An admin has muted you in the chat ${penaltyEnd(mute.until)}.`;
}
