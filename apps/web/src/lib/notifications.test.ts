import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  dismissNotification,
  getNotifications,
  getServerNotifications,
  notify,
  subscribeToNotifications,
} from './notifications';

afterEach(() => {
  for (const { id } of getNotifications()) dismissNotification(id);
});

describe('notifications', () => {
  it('shows each message until it is dismissed, telling subscribers', () => {
    const listener = vi.fn();
    const unsubscribe = subscribeToNotifications(listener);

    notify('Deck saved');
    notify('Deck deleted');
    const [first] = getNotifications();
    if (first) dismissNotification(first.id);

    expect(getNotifications().map(({ message }) => message)).toEqual(['Deck deleted']);
    expect(listener).toHaveBeenCalledTimes(3);
    unsubscribe();
    notify('Deck copied');
    expect(listener).toHaveBeenCalledTimes(3);
  });

  it('does not show a message twice while it is showing', () => {
    notify('Deck saved');
    notify('Deck saved');

    expect(getNotifications()).toHaveLength(1);
  });

  it('shows nothing in pages rendered on the server', () => {
    notify('Deck saved');

    expect(getServerNotifications()).toEqual([]);
  });
});
