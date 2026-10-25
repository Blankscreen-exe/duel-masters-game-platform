'use client';

import {
  createContext,
  useContext,
  useEffect,
  useEffectEvent,
  useState,
  type ReactNode,
} from 'react';
import {
  LiveListeners,
  parseLiveMessage,
  reconnectDelay,
  socketUrl,
  type LiveMessageOf,
  type LiveMessageType,
} from '@/lib/live/messages';

interface LiveContextValue {
  listeners: LiveListeners;
  connected: boolean;
  /** Says who is logged in; the connection opens once known, and again whenever it changes. */
  setAccount: (accountId: string | null) => void;
}

const LiveContext = createContext<LiveContextValue | null>(null);

/** Before the session is known: not connected yet, so as not to connect twice. */
const UNKNOWN = Symbol('unknown account');

/**
 * Keeps this tab's live connection to the server, reconnecting whenever it drops, and passes its
 * messages to the components listening. Rendered once, in the root layout; LiveSession says who
 * is logged in, so logging in or out reconnects as the new account.
 */
export function LiveProvider({ children }: { readonly children: ReactNode }) {
  const [listeners] = useState(() => new LiveListeners());
  const [connected, setConnected] = useState(false);
  const [account, setAccount] = useState<string | null | typeof UNKNOWN>(UNKNOWN);

  useEffect(() => {
    if (account === UNKNOWN) return;

    let socket: WebSocket | null = null;
    let retry: number | undefined;
    let attempt = 0;
    let stopped = false;

    const connect = () => {
      socket = new WebSocket(socketUrl(window.location, '/api/realtime'));
      socket.onopen = () => {
        attempt = 0;
        setConnected(true);
      };
      socket.onmessage = (event: MessageEvent) => {
        const message = parseLiveMessage(event.data);
        if (message) listeners.emit(message);
      };
      socket.onclose = () => {
        setConnected(false);
        if (stopped) return;
        retry = window.setTimeout(connect, reconnectDelay(attempt));
        attempt += 1;
      };
    };
    connect();

    return () => {
      stopped = true;
      window.clearTimeout(retry);
      socket?.close();
    };
  }, [listeners, account]);

  return <LiveContext value={{ listeners, connected, setAccount }}>{children}</LiveContext>;
}

function useLiveContext(): LiveContextValue {
  const live = useContext(LiveContext);
  if (!live) throw new Error('Live updates need a LiveProvider, which the root layout renders');
  return live;
}

/** Tells the live connection who is logged in; rendered by LiveSession. */
export function LiveAccount({ accountId }: { readonly accountId: string | null }) {
  const { setAccount } = useLiveContext();

  useEffect(() => {
    setAccount(accountId);
  }, [setAccount, accountId]);

  return null;
}

/** Whether this tab's live connection is open now. */
export function useLiveConnected(): boolean {
  return useLiveContext().connected;
}

/** Calls the listener with every live message of the type, while the component is on screen. */
export function useLiveMessage<T extends LiveMessageType>(
  type: T,
  listener: (message: LiveMessageOf<T>) => void,
): void {
  const { listeners } = useLiveContext();
  const onMessage = useEffectEvent(listener);

  useEffect(
    () =>
      listeners.on(type, (message) => {
        onMessage(message);
      }),
    [listeners, type],
  );
}
