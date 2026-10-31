import type { Verification } from './api/client';
import { hrefWithQuery } from './search-params';

export const VERIFY_EMAIL_PATH = '/verify-email';

/** The longest email address, as the API contract allows. */
export const EMAIL_MAX_LENGTH = 254;

/** How many digits a code has; only a hint for the form, as the Go server checks it. */
export const CODE_DIGITS = 6;

/** Where a player goes to confirm their email address, and then on to next. */
export function verifyEmailHref(next: string): string {
  return hrefWithQuery(VERIFY_EMAIL_PATH, { next: next === '/' ? undefined : next });
}

/**
 * What a player can do about their address right now: wait while the site cannot send email, ask
 * for a first code, or enter the one sent.
 */
export type VerificationStep = 'waiting' | 'send' | 'enter';

export function verificationStep(status: Verification): VerificationStep {
  if (status.codeSentAt) return 'enter';
  return status.mailWorks ? 'send' : 'waiting';
}

/** A wait as it reads on the resend button: "45 s" or "2 min 5 s". */
export function waitText(seconds: number): string {
  if (seconds < 60) return `${String(seconds)} s`;

  const rest = seconds % 60;
  const minutes = `${String(Math.floor(seconds / 60))} min`;
  return rest === 0 ? minutes : `${minutes} ${String(rest)} s`;
}
