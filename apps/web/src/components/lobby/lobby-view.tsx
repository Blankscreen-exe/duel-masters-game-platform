'use client';

import { useState } from 'react';
import { LobbyChat } from '@/components/chat/lobby-chat';
import { useLiveMessage } from '@/components/live/live-provider';
import { Panel } from '@/components/ui/panel';
import type { Account, DeckSummary, Format } from '@/lib/api/client';
import type { components } from '@/lib/api/schema';
import { activityOf } from '@/lib/lobby';
import type { Lobby, QueueCount, QuickMatchStatus } from '@/lib/lobby-actions';
import { OnlinePlayers, OpenGamesList, RunningGamesList } from './game-lists';
import { OpenGameForm } from './open-game-form';
import { QuickMatchPanel } from './quick-match-panel';
import { YourActivity } from './your-activity';

interface LobbyViewProps {
  readonly account: Account;
  readonly lobby: Lobby;
  readonly quickMatch: QuickMatchStatus;
  readonly decks: readonly DeckSummary[];
  readonly formats: readonly Format[];
  readonly chat: readonly components['schemas']['ChatMessage'][];
}

/**
 * The lobby: open a game or find a quick match, join an open game, watch a running one, and chat.
 * Everything on it stays up to date over the live connection.
 */
export function LobbyView({
  account,
  lobby: initialLobby,
  quickMatch,
  decks,
  formats,
  chat,
}: LobbyViewProps) {
  const [lobby, setLobby] = useState(initialLobby);
  const [queue, setQueue] = useState<readonly QueueCount[]>(quickMatch.waiting);

  useLiveMessage('lobby', (message) => {
    setLobby(message.lobby);
  });
  useLiveMessage('quick_match_queue', (message) => {
    setQueue(message.waiting);
  });

  const activity = activityOf(lobby, account.id);
  const busy = activity !== null;
  const hosted =
    activity?.kind === 'hosting'
      ? (lobby.openGames.find((game) => game.id === activity.gameId) ?? null)
      : null;
  const waitingFormatName = activity?.kind === 'queued' ? (activity.formatName ?? '') : null;

  return (
    <div className="space-y-6">
      <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">Play</h1>

      {lobby.paused && (
        <p
          role="status"
          className="bg-amber-500/10 px-4 py-3 text-sm text-amber-200 ring-1 ring-amber-400/40 ring-inset"
        >
          New games are paused for a moment. Games already running carry on.
        </p>
      )}

      {activity && activity.kind !== 'queued' && (
        <YourActivity activity={activity} hosted={hosted} />
      )}

      <div className="grid gap-6 lg:grid-cols-[22rem_1fr]">
        <div className="space-y-6">
          <Panel title="Quick match" accent="fire">
            <QuickMatchPanel
              decks={decks}
              formats={formats}
              queue={queue}
              waitingFormatName={waitingFormatName}
              disabled={lobby.paused || (busy && activity.kind !== 'queued')}
            />
          </Panel>
          {!busy && (
            <Panel title="Open a game" accent="water">
              <OpenGameForm decks={decks} formats={formats} disabled={lobby.paused} />
            </Panel>
          )}
        </div>

        <div className="space-y-6">
          <Panel title="Open games" actions={<Count n={lobby.openGames.length} />}>
            <OpenGamesList
              games={lobby.openGames}
              decks={decks}
              userId={account.id}
              canJoin={!busy && !lobby.paused}
            />
          </Panel>
          <div className="grid gap-6 xl:grid-cols-2">
            <Panel title="Lobby chat" className="flex max-h-[28rem] flex-col">
              <LobbyChat initial={chat} canPost mute={account.mute} className="flex-1" />
            </Panel>
            <div className="space-y-6">
              <Panel title="Games running" actions={<Count n={lobby.runningGames.length} />}>
                <RunningGamesList games={lobby.runningGames} />
              </Panel>
              <Panel title="Online" actions={<Count n={lobby.online.length} />}>
                <OnlinePlayers players={lobby.online} />
              </Panel>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function Count({ n }: { readonly n: number }) {
  return <span className="font-display text-sm font-semibold text-slate-400">{n}</span>;
}
