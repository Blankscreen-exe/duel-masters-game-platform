import { describe, expect, it } from 'vitest';
import {
  nextPopup,
  notificationsLabel,
  pageHref,
  pageParam,
  popupKey,
  unreadBadge,
  type Announcement,
} from './announcements';

const announcement = (id: string): Announcement => ({
  id,
  title: id,
  body: 'Details.',
  important: true,
  postedAt: '2026-11-01T00:00:00Z',
});

describe('unreadBadge', () => {
  it('shows nothing with nothing unread', () => {
    expect(unreadBadge(0)).toBeNull();
  });

  it('counts up to a limit', () => {
    expect(unreadBadge(1)).toBe('1');
    expect(unreadBadge(9)).toBe('9');
    expect(unreadBadge(10)).toBe('9+');
  });
});

describe('notificationsLabel', () => {
  it('says how many are unread', () => {
    expect(notificationsLabel(0)).toBe('Notifications');
    expect(notificationsLabel(12)).toBe('Notifications, 12 unread');
  });
});

describe('pages', () => {
  it('reads the page asked for, falling back to the first', () => {
    expect(pageParam({ page: '3' })).toBe(3);
    expect(pageParam({})).toBe(1);
    expect(pageParam({ page: 'nonsense' })).toBe(1);
    expect(pageParam({ page: '0' })).toBe(1);
  });

  it('leaves the first page out of the address', () => {
    expect(pageHref('/notifications', 1)).toBe('/notifications');
    expect(pageHref('/notifications', 2)).toBe('/notifications?page=2');
  });
});

describe('nextPopup', () => {
  const popups = [announcement('a'), announcement('b')];

  it('is the first one', () => {
    expect(nextPopup(popups, new Set())?.id).toBe('a');
  });

  const closed = (...closing: Announcement[]) => new Set(closing.map(popupKey));
  const [first, second] = popups as [Announcement, Announcement];

  it('skips those closed on this page', () => {
    expect(nextPopup(popups, closed(first))?.id).toBe('b');
  });

  it('is none once all are closed', () => {
    expect(nextPopup(popups, closed(first, second))).toBeNull();
    expect(nextPopup([], new Set())).toBeNull();
  });

  it('shows one again when an admin sends it out again', () => {
    const again = { ...first, postedAt: '2026-11-02T00:00:00Z' };

    expect(nextPopup([again], closed(first))?.id).toBe('a');
  });
});
