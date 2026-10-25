/**
 * Short messages shown over any page for a few seconds, such as "Deck saved". The list lives in the
 * browser outside React, so a message posted just before moving to another page is shown there.
 */
export interface Notification {
  readonly id: number;
  readonly message: string;
}

const NONE: readonly Notification[] = [];
let notifications = NONE;
let nextId = 1;
const listeners = new Set<() => void>();

function changed(next: readonly Notification[]): void {
  notifications = next;
  for (const listener of listeners) listener();
}

/** Shows a message. A message that is already showing is not shown twice. */
export function notify(message: string): void {
  if (notifications.some((notification) => notification.message === message)) return;
  changed([...notifications, { id: nextId++, message }]);
}

export function dismissNotification(id: number): void {
  changed(notifications.filter((notification) => notification.id !== id));
}

export function subscribeToNotifications(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getNotifications(): readonly Notification[] {
  return notifications;
}

/** Nothing is ever shown in server-rendered pages. */
export function getServerNotifications(): readonly Notification[] {
  return NONE;
}
