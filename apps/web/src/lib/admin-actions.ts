import type { AdminSetting, AdminUser } from './admin';
import { api } from './api/client';
import { attempt, attemptEmpty, type Outcome } from './api/outcome';
import type { components } from './api/schema';

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
