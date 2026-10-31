import type { AdminSetting, AdminUser } from './admin';
import { api } from './api/client';
import { attempt, attemptEmpty, type Outcome } from './api/outcome';
import type { components } from './api/schema';
import type { AdminComputerOpponent, ComputerOpponentInput } from './computer-opponents';
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
