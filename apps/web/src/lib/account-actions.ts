import { api, type Account } from './api/client';
import { attempt, type Outcome } from './api/outcome';

/** Gives the player's account one of the profile pictures. */
export function setAvatar(avatarId: string): Promise<Outcome<Account>> {
  return attempt(() => api.PUT('/auth/avatar', { body: { avatarId } }));
}
