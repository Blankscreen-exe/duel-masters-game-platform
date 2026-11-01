import type { components } from './api/schema';
import { firstParam, hrefWithQuery, wholeNumberParam, type SearchParams } from './search-params';

type Schemas = components['schemas'];
export type Announcement = Schemas['Announcement'];
export type AnnouncementInput = Schemas['AnnouncementInput'];
export type MyAnnouncements = Schemas['MyAnnouncements'];

/**
 * What admins tell every player. The site calls these notifications; in the code that word already
 * means the short messages of lib/notifications.ts, such as "Deck saved".
 */
export const NOTIFICATIONS_PATH = '/notifications';
export const ADMIN_NOTIFICATIONS_PATH = '/admin/notifications';

/** The text limits, as the API contract has them. */
export const TITLE_MAX_LENGTH = 100;
export const BODY_MAX_LENGTH = 2000;

/** What a player with nothing to see has. */
export const NOTHING_NEW: MyAnnouncements = { unread: 0, popups: [] };

/** The most the top bar's count shows; past that it reads as "9+". */
const MOST_COUNTED = 9;

/** The top bar's count, or null when there is nothing to show. */
export function unreadBadge(unread: number): string | null {
  if (unread <= 0) return null;
  return unread > MOST_COUNTED ? `${String(MOST_COUNTED)}+` : String(unread);
}

/** The bell's name, for screen readers and its tooltip. */
export function notificationsLabel(unread: number): string {
  if (unread <= 0) return 'Notifications';
  return `Notifications, ${String(unread)} unread`;
}

/** The page number a page address asks for. */
export function pageParam(params: SearchParams): number {
  return wholeNumberParam(firstParam(params['page']), 1) ?? 1;
}

/** The address of a page of a list of notifications. */
export function pageHref(path: string, page: number): string {
  return hrefWithQuery(path, { page: page === 1 ? undefined : page });
}

/**
 * Which going-out of a notification a popup is for. An admin showing it again is a new one, which
 * pops up again even for a player who closed the last and has stayed on the page since.
 */
export function popupKey({ id, postedAt }: Announcement): string {
  return `${id} ${postedAt}`;
}

/** The first popup a player has yet to close, leaving out those they closed on this page. */
export function nextPopup(
  popups: readonly Announcement[],
  closed: ReadonlySet<string>,
): Announcement | null {
  return popups.find((popup) => !closed.has(popupKey(popup))) ?? null;
}
