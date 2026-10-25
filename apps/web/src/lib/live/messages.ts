import type { components } from '../api/schema';

/**
 * The close code of a live connection the server closes as it puts the player out of the site, by
 * a ban or a new password: the page is then logged out.
 */
export const SIGNED_OUT_CLOSE_CODE = 4001;

/** A message on the live connection, told apart by its type (api/openapi.yaml, RealtimeMessage). */
export type LiveMessage = components['schemas']['RealtimeMessage'];
export type LiveMessageType = LiveMessage['type'];
export type LiveMessageOf<T extends LiveMessageType> = Extract<LiveMessage, { type: T }>;

/**
 * A message as the server sent it, or null for anything that is not one: not JSON, or without a
 * type. Messages of a type this page does not know are passed on, and simply have no listener.
 */
export function parseLiveMessage(data: unknown): LiveMessage | null {
  if (typeof data !== 'string') return null;

  try {
    const message: unknown = JSON.parse(data);
    if (
      typeof message === 'object' &&
      message !== null &&
      'type' in message &&
      typeof message.type === 'string'
    ) {
      return message as LiveMessage;
    }
  } catch {
    // Not JSON: not a message.
  }
  return null;
}

/** Who listens to which messages. */
export class LiveListeners {
  private readonly byType = new Map<string, Set<(message: LiveMessage) => void>>();

  /** Listens to messages of one type, until the returned function is called. */
  on<T extends LiveMessageType>(
    type: T,
    listener: (message: LiveMessageOf<T>) => void,
  ): () => void {
    const wrapped = listener as (message: LiveMessage) => void;
    let listeners = this.byType.get(type);
    if (!listeners) {
      listeners = new Set();
      this.byType.set(type, listeners);
    }
    listeners.add(wrapped);

    return () => {
      listeners.delete(wrapped);
    };
  }

  /** Passes a message to everyone listening to its type. */
  emit(message: LiveMessage): void {
    for (const listener of [...(this.byType.get(message.type) ?? [])]) {
      listener(message);
    }
  }
}

/** The first wait before reconnecting, and the longest. */
const FIRST_DELAY_MS = 1000;
const LONGEST_DELAY_MS = 10_000;

/**
 * How long to wait before the next attempt to reconnect: doubling from a second up to ten, with a
 * little randomness so a whole lobby does not reconnect at the same instant after a restart.
 */
export function reconnectDelay(attempt: number, random: () => number = Math.random): number {
  const base = Math.min(FIRST_DELAY_MS * 2 ** attempt, LONGEST_DELAY_MS);
  return Math.round(base * (0.8 + 0.4 * random()));
}

/**
 * The address of one of this site's websockets, such as /api/realtime, for the page's own
 * location: secure when the page is.
 */
export function socketUrl(location: Pick<Location, 'protocol' | 'host'>, path: string): string {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
  return `${scheme}://${location.host}${path}`;
}
