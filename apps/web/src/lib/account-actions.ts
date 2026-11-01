import { api, type Account, type Verification } from './api/client';
import { attempt, attemptEmpty, type Outcome } from './api/outcome';

/** Gives the player's account one of the profile pictures. */
export function setAvatar(avatarId: string): Promise<Outcome<Account>> {
  return attempt(() => api.PUT('/auth/avatar', { body: { avatarId } }));
}

/** Notes that the player has looked at the notifications. */
export function markAnnouncementsSeen(): Promise<Outcome<null>> {
  return attemptEmpty(() => api.POST('/announcements/seen'));
}

/** Closes the popup of an important notification, so it does not come back. */
export function dismissAnnouncement(announcementId: string): Promise<Outcome<null>> {
  return attemptEmpty(() =>
    api.POST('/announcements/{announcementId}/dismiss', { params: { path: { announcementId } } }),
  );
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
