import { api, type Account, type Verification } from './api/client';
import { attempt, type Outcome } from './api/outcome';

/** Gives the player's account one of the profile pictures. */
export function setAvatar(avatarId: string): Promise<Outcome<Account>> {
  return attempt(() => api.PUT('/auth/avatar', { body: { avatarId } }));
}

/** Confirms the player's email address with the code sent to it. */
export function confirmEmail(code: string): Promise<Outcome<Account>> {
  return attempt(() => api.POST('/auth/verification', { body: { code } }));
}

/** Sends a new code to the player's email address. */
export function sendVerificationCode(): Promise<Outcome<Verification>> {
  return attempt(() => api.POST('/auth/verification/code'));
}

/** Corrects the email address of a player not yet confirmed, and sends a code to it. */
export function changeVerificationEmail(email: string): Promise<Outcome<Verification>> {
  return attempt(() => api.PUT('/auth/verification/email', { body: { email } }));
}
