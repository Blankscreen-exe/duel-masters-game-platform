import type { AdminSetting, AdminUser } from './admin';
import { api } from './api/client';
import { attempt, attemptEmpty, type Outcome } from './api/outcome';
import type { components } from './api/schema';
import type { Announcement, AnnouncementInput } from './announcements';
import type { AdminComputerOpponent, ComputerOpponentInput } from './computer-opponents';
import type { MailSettings, MailSettingsInput } from './mail-settings';
import type { AdminFormat, FormatInput } from './formats';

type Role = components['schemas']['Role'];

const user = (userId: string) => ({ params: { path: { userId } } });

/** Bans a player for minutes, or for good without; the answer is the account as banned. */
export function banUser(
  userId: string,
  reason: string,
  durationMinutes?: number,
): Promise<Outcome<AdminUser>> {
  return attempt(() =>
    api.POST('/admin/users/{userId}/ban', {
      ...user(userId),
      body: durationMinutes === undefined ? { reason } : { reason, durationMinutes },
    }),
  );
}

export function createAnnouncement(input: AnnouncementInput): Promise<Outcome<Announcement>> {
  return attempt(() => api.POST('/admin/announcements', { body: input }));
}

const announcement = (announcementId: string) => ({ params: { path: { announcementId } } });

export function updateAnnouncement(
  announcementId: string,
  input: AnnouncementInput,
): Promise<Outcome<Announcement>> {
  return attempt(() =>
    api.PUT('/admin/announcements/{announcementId}', {
      ...announcement(announcementId),
      body: input,
    }),
  );
}

export function deleteAnnouncement(announcementId: string): Promise<Outcome<null>> {
  return attemptEmpty(() =>
    api.DELETE('/admin/announcements/{announcementId}', announcement(announcementId)),
  );
}

/** Lets a player in without confirming their email address. */
export function letUserIn(userId: string): Promise<Outcome<AdminUser>> {
  return attempt(() => api.POST('/admin/users/{userId}/let-in', user(userId)));
}

export function saveMailSettings(input: MailSettingsInput): Promise<Outcome<MailSettings>> {
  return attempt(() => api.PUT('/admin/mail', { body: input }));
}

export function clearMailSettings(): Promise<Outcome<MailSettings>> {
  return attempt(() => api.DELETE('/admin/mail'));
}

/** Sends a test email through the mail server; a failure says why. */
export function sendTestMail(to: string): Promise<Outcome<null>> {
  return attemptEmpty(() => api.POST('/admin/mail/test', { body: { to } }));
}

export function unbanUser(userId: string): Promise<Outcome<AdminUser>> {
  return attempt(() => api.DELETE('/admin/users/{userId}/ban', user(userId)));
}

/** Mutes a player in the lobby chat for minutes, or for good without. */
export function muteUser(userId: string, durationMinutes?: number): Promise<Outcome<AdminUser>> {
  return attempt(() =>
    api.POST('/admin/users/{userId}/mute', {
      ...user(userId),
      body: durationMinutes === undefined ? {} : { durationMinutes },
    }),
  );
}

export function unmuteUser(userId: string): Promise<Outcome<AdminUser>> {
  return attempt(() => api.DELETE('/admin/users/{userId}/mute', user(userId)));
}

export function setUserRole(userId: string, role: Role): Promise<Outcome<AdminUser>> {
  return attempt(() => api.PUT('/admin/users/{userId}/role', { ...user(userId), body: { role } }));
}

/** Gives a player a new password, which the admin passes on; it ends the player's sessions. */
export function setUserPassword(userId: string, password: string): Promise<Outcome<null>> {
  return attemptEmpty(() =>
    api.PUT('/admin/users/{userId}/password', { ...user(userId), body: { password } }),
  );
}

/** Changes a setting, which applies at once; the answer is the setting as changed. */
export function setSetting(key: string, value: string): Promise<Outcome<AdminSetting>> {
  return attempt(() =>
    api.PUT('/admin/settings/{key}', { params: { path: { key } }, body: { value } }),
  );
}

/** Puts a setting back to its default. */
export function resetSetting(key: string): Promise<Outcome<AdminSetting>> {
  return attempt(() => api.DELETE('/admin/settings/{key}', { params: { path: { key } } }));
}

export function createFormat(body: FormatInput): Promise<Outcome<AdminFormat>> {
  return attempt(() => api.POST('/admin/formats', { body }));
}

export function saveFormat(formatId: string, body: FormatInput): Promise<Outcome<AdminFormat>> {
  return attempt(() =>
    api.PUT('/admin/formats/{formatId}', { params: { path: { formatId } }, body }),
  );
}

/** Retires a format or brings it back. */
export function setFormatRetired(
  formatId: string,
  retired: boolean,
): Promise<Outcome<AdminFormat>> {
  return attempt(() =>
    api.PUT('/admin/formats/{formatId}/retired', {
      params: { path: { formatId } },
      body: { retired },
    }),
  );
}

/** Puts every format, by id, in the order given. */
export function reorderFormats(formatIds: string[]): Promise<Outcome<AdminFormat[]>> {
  return attempt(() => api.PUT('/admin/formats/order', { body: { formatIds } }));
}

/** Adds a computer opponent, which starts retired. */
export function createOpponent(
  body: ComputerOpponentInput,
): Promise<Outcome<AdminComputerOpponent>> {
  return attempt(() => api.POST('/admin/computer-opponents', { body }));
}

export function saveOpponent(
  opponentId: string,
  body: ComputerOpponentInput,
): Promise<Outcome<AdminComputerOpponent>> {
  return attempt(() =>
    api.PUT('/admin/computer-opponents/{opponentId}', { params: { path: { opponentId } }, body }),
  );
}

/** Offers a computer opponent to players, or retires it. */
export function setOpponentOffered(
  opponentId: string,
  offered: boolean,
): Promise<Outcome<AdminComputerOpponent>> {
  return attempt(() =>
    api.PUT('/admin/computer-opponents/{opponentId}/offered', {
      params: { path: { opponentId } },
      body: { offered },
    }),
  );
}
