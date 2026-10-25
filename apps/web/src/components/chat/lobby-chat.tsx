'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useEffect, useRef, useState, type SubmitEvent } from 'react';
import { useLiveMessage } from '@/components/live/live-provider';
import { Button } from '@/components/ui/button';
import { api, errorMessage, UNREACHABLE } from '@/lib/api/client';
import type { components } from '@/lib/api/schema';
import { inForce, mutedMessage, type Penalty } from '@/lib/moderation';

type ChatMessage = components['schemas']['ChatMessage'];

/** The most messages kept on screen, as the server keeps. */
const HISTORY_LENGTH = 100;
/** The longest message, as the server allows. */
const MAX_LENGTH = 300;

interface LobbyChatProps {
  readonly initial: readonly ChatMessage[];
  /** Whether the viewer may post: players can, visitors only read. */
  readonly canPost: boolean;
  /** The player's mute, while an admin has muted them: they read, but do not post. */
  readonly mute?: Penalty;
  readonly className?: string;
}

/** The lobby chat: everyone reads it, players write in it, and new messages arrive live. */
export function LobbyChat({ initial, canPost, mute, className = '' }: LobbyChatProps) {
  const muted = useMuted(mute);
  const [messages, setMessages] = useState<readonly ChatMessage[]>(initial);
  const [text, setText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const list = useRef<HTMLOListElement>(null);
  const pathname = usePathname();

  useLiveMessage('chat_message', ({ message }) => {
    setMessages((current) =>
      current.some((shown) => shown.id === message.id)
        ? current
        : [...current, message].slice(-HISTORY_LENGTH),
    );
  });

  // The newest message stays in view.
  useEffect(() => {
    const element = list.current;
    if (element) element.scrollTop = element.scrollHeight;
  }, [messages]);

  async function post(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (text.trim() === '') return;

    setSending(true);
    setError(null);
    try {
      const { error: refusal, response } = await api.POST('/chat', { body: { text } });
      if (response.ok) setText('');
      else setError(errorMessage(refusal));
    } catch {
      setError(UNREACHABLE);
    } finally {
      setSending(false);
    }
  }

  return (
    <div className={`flex min-h-0 flex-col ${className}`}>
      <ol
        ref={list}
        aria-label="Chat messages"
        aria-live="polite"
        className="min-h-48 flex-1 space-y-1.5 overflow-y-auto px-4 py-3 text-sm"
      >
        {messages.length === 0 ? (
          <li className="text-slate-500">No messages yet. Say hello!</li>
        ) : (
          messages.map((message) => (
            <li key={message.id} className="break-words">
              <span className="font-semibold text-accent">{message.username}</span>{' '}
              <span className="text-slate-200">{message.text}</span>
            </li>
          ))
        )}
      </ol>

      {canPost && muted ? (
        <p role="status" className="border-t border-slate-700/60 px-4 py-3 text-sm text-slate-400">
          {mutedMessage(muted)}
        </p>
      ) : canPost ? (
        <form onSubmit={(event) => void post(event)} className="border-t border-slate-700/60 p-3">
          {error && (
            <p role="alert" className="mb-2 text-xs text-red-300">
              {error}
            </p>
          )}
          <div className="flex gap-2">
            <input
              aria-label="Message"
              value={text}
              maxLength={MAX_LENGTH}
              placeholder="Say something…"
              onChange={(event) => {
                setText(event.target.value);
              }}
              className="min-w-0 flex-1 bg-field px-3 py-2 text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset placeholder:text-slate-500 focus:ring-accent focus:outline-none"
            />
            <Button type="submit" disabled={sending || text.trim() === ''} className="px-4">
              Send
            </Button>
          </div>
        </form>
      ) : (
        <p className="border-t border-slate-700/60 px-4 py-3 text-sm text-slate-400">
          <Link
            href={`/login?next=${encodeURIComponent(pathname)}`}
            className="font-semibold text-accent hover:underline"
          >
            Log in
          </Link>{' '}
          to chat.
        </p>
      )}
    </div>
  );
}

/**
 * The player's mute while it lasts: a timed mute lifts itself on screen when it ends, as it does on
 * the server.
 */
function useMuted(mute: Penalty | undefined): Penalty | null {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (mute?.until === undefined) return;
    const timer = window.setTimeout(
      () => {
        setNow(Date.now());
      },
      Math.max(0, Date.parse(mute.until) - Date.now()),
    );
    return () => {
      window.clearTimeout(timer);
    };
  }, [mute?.until]);

  return inForce(mute, now) ? mute : null;
}
