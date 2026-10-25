'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useEffect, useRef, useState, type SubmitEvent } from 'react';
import { useLiveMessage } from '@/components/live/live-provider';
import { Button } from '@/components/ui/button';
import { Modal, ModalClose } from '@/components/ui/modal';
import { Panel } from '@/components/ui/panel';
import { api, errorMessage, UNREACHABLE } from '@/lib/api/client';
import type { components } from '@/lib/api/schema';
import { inForce, mutedMessage, type Penalty } from '@/lib/moderation';

type ChatMessage = components['schemas']['ChatMessage'];

/** The most messages kept on screen, as the server keeps. */
const HISTORY_LENGTH = 100;
/** The longest message, as the server allows. */
const MAX_LENGTH = 300;

interface LobbyChatPanelProps {
  readonly initial: readonly ChatMessage[];
  /** Whether the viewer may post: players can, visitors only read. */
  readonly canPost: boolean;
  /** The player's mute, while an admin has muted them: they read, but do not post. */
  readonly mute?: Penalty;
  readonly className?: string;
}

/**
 * The lobby chat in a panel: everyone reads it, players write in it, and new messages arrive live.
 * It expands into a modal that fills most of the screen, showing the same conversation and keeping
 * what the player was typing.
 */
export function LobbyChatPanel({ initial, canPost, mute, className = '' }: LobbyChatPanelProps) {
  const chat = useLobbyChat(initial);
  const muted = useMuted(mute);
  const [expanded, setExpanded] = useState(false);
  const body = <ChatBody chat={chat} canPost={canPost} muted={muted} />;

  return (
    <Panel
      title="Lobby chat"
      actions={
        <Button
          variant="quiet"
          aria-label="Expand the chat"
          title="Expand"
          onClick={() => {
            setExpanded(true);
          }}
        >
          <ExpandIcon />
        </Button>
      }
      className={`flex flex-col ${className}`}
    >
      {expanded ? (
        <p className="flex-1 px-4 py-3 text-sm text-slate-500">
          The chat is open in a larger view.
        </p>
      ) : (
        body
      )}
      {expanded && (
        <Modal
          title="Lobby chat"
          size="full"
          onClose={() => {
            setExpanded(false);
          }}
          actions={
            <ModalClose variant="quiet" aria-label="Shrink the chat" title="Shrink">
              <ShrinkIcon />
            </ModalClose>
          }
        >
          {body}
        </Modal>
      )}
    </Panel>
  );
}

/** A lobby chat's conversation and the message being written, shared by every view of it. */
interface LobbyChat {
  readonly messages: readonly ChatMessage[];
  readonly text: string;
  readonly setText: (text: string) => void;
  readonly error: string | null;
  readonly sending: boolean;
  readonly post: () => Promise<void>;
}

/** The lobby chat's state: the messages so far, kept up to date live, and posting a new one. */
function useLobbyChat(initial: readonly ChatMessage[]): LobbyChat {
  const [messages, setMessages] = useState<readonly ChatMessage[]>(initial);
  const [text, setText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);

  useLiveMessage('chat_message', ({ message }) => {
    setMessages((current) =>
      current.some((shown) => shown.id === message.id)
        ? current
        : [...current, message].slice(-HISTORY_LENGTH),
    );
  });

  async function post() {
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

  return { messages, text, setText, error, sending, post };
}

interface ChatBodyProps {
  readonly chat: LobbyChat;
  readonly canPost: boolean;
  readonly muted: Penalty | null;
}

/** The messages, newest at the bottom, and below them the box to write one in. */
function ChatBody({ chat, canPost, muted }: ChatBodyProps) {
  const list = useRef<HTMLOListElement>(null);
  const pathname = usePathname();

  // The newest message stays in view.
  useEffect(() => {
    const element = list.current;
    if (element) element.scrollTop = element.scrollHeight;
  }, [chat.messages]);

  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    void chat.post();
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ol
        ref={list}
        aria-label="Chat messages"
        aria-live="polite"
        className="min-h-48 flex-1 space-y-1.5 overflow-y-auto px-4 py-3 text-sm"
      >
        {chat.messages.length === 0 ? (
          <li className="text-slate-500">No messages yet. Say hello!</li>
        ) : (
          chat.messages.map((message) => (
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
        <form onSubmit={submit} className="border-t border-slate-700/60 p-3">
          {chat.error && (
            <p role="alert" className="mb-2 text-xs text-red-300">
              {chat.error}
            </p>
          )}
          <div className="flex gap-2">
            <input
              aria-label="Message"
              value={chat.text}
              maxLength={MAX_LENGTH}
              placeholder="Say something…"
              onChange={(event) => {
                chat.setText(event.target.value);
              }}
              className="min-w-0 flex-1 bg-field px-3 py-2 text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset placeholder:text-slate-500 focus:ring-accent focus:outline-none"
            />
            <Button
              type="submit"
              disabled={chat.sending || chat.text.trim() === ''}
              className="px-4"
            >
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

/** Four arrows pointing out to the corners. */
function ExpandIcon() {
  return (
    <svg aria-hidden viewBox="0 0 20 20" className="size-4" fill="none" stroke="currentColor">
      <path
        d="M12 3h5v5M8 17H3v-5M17 3l-6 6M3 17l6-6"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/** Four arrows pointing in from the corners. */
function ShrinkIcon() {
  return (
    <svg aria-hidden viewBox="0 0 20 20" className="size-4" fill="none" stroke="currentColor">
      <path
        d="M16 9h-5V4M4 11h5v5M11 9l6-6M9 11l-6 6"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
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
