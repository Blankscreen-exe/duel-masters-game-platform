'use client';

import { useState } from 'react';
import { LobbyChat } from '@/components/chat/lobby-chat';
import { useLiveMessage } from '@/components/live/live-provider';
import { Panel } from '@/components/ui/panel';
import { TabbedPanel } from '@/components/ui/tabbed-panel';
import type { Account, Format } from '@/lib/api/client';
import type { components } from '@/lib/api/schema';
import { activityOf, type PlayableDecks } from '@/lib/lobby';
import type { Lobby, QueueCount, QuickMatchStatus } from '@/lib/lobby-actions';
import { GamesList, OnlinePlayers } from './game-lists';
import { OpenGameForm } from './open-game-form';
import { QuickMatchPanel } from './quick-match-panel';
import { YourActivity } from './your-activity';

interface LobbyViewProps {
  readonly account: Account;
  readonly lobby: Lobby;
  readonly quickMatch: QuickMatchStatus;
  readonly decks: PlayableDecks;
  readonly formats: readonly Format[];
  readonly chat: readonly components['schemas']['ChatMessage'][];
}

/**
 * The lobby: open a game or find a quick match, join an open game, watch a running one, and chat.
 * Everything on it stays up to date over the live connection.
 *
 * On a wide screen it has four columns. Three hold the title, starting a game, a tab each for quick
 * match and opening a game, and the games under it, open and running in one list. The fourth holds
 * the chat and who is online, a tab each: it starts level with the title and is as tall as the
 * screen, so it fits from the start and stays in view when the page scrolls. A little narrower, the
 * fourth keeps a fixed width; on a phone everything stacks.
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
  const [openOnly, setOpenOnly] = useState(false);

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_20rem] xl:grid-cols-4">
      <div className="space-y-6 xl:col-span-3">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          Play
        </h1>

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

        <TabbedPanel
          label="Start a game"
          idPrefix="start-game"
          accent="fire"
          tabs={[
            {
              id: 'quick',
              label: 'Quick match',
              content: (
                <QuickMatchPanel
                  decks={decks}
                  formats={formats}
                  queue={queue}
                  waitingFormatName={waitingFormatName}
                  disabled={lobby.paused || (busy && activity.kind !== 'queued')}
                />
              ),
            },
            {
              id: 'open',
              label: 'Open a game',
              content: (
                <OpenGameForm decks={decks} formats={formats} disabled={lobby.paused || busy} />
              ),
            },
          ]}
        />

        <Panel
          title="Games"
          actions={
            <div className="flex items-center gap-4">
              <label className="flex cursor-pointer items-center gap-2 text-sm text-slate-300 hover:text-white">
                <input
                  type="checkbox"
                  checked={openOnly}
                  onChange={(event) => {
                    setOpenOnly(event.target.checked);
                  }}
                  className="size-4 accent-sky-400"
                />
                Show open games only
              </label>
              <span className="font-display text-sm font-semibold text-slate-400">
                {lobby.openGames.length} open
                {openOnly ? '' : ` · ${String(lobby.runningGames.length)} running`}
              </span>
            </div>
          }
        >
          <GamesList
            openGames={lobby.openGames}
            runningGames={lobby.runningGames}
            openOnly={openOnly}
            decks={decks}
            userId={account.id}
            canJoin={!busy && !lobby.paused}
          />
        </Panel>
      </div>

      <LobbyChat
        initial={chat}
        canPost
        mute={account.mute}
        frame={({ expand, body }) => (
          <TabbedPanel
            label="Chat and players"
            idPrefix="lobby-side"
            className="h-[calc(100dvh-6.75rem)] min-h-96 lg:sticky lg:top-[5.25rem] lg:self-start"
            tabs={[
              { id: 'chat', label: 'Lobby chat', content: body, actions: expand },
              {
                id: 'online',
                label: `Online ${String(lobby.online.length)}`,
                content: <OnlinePlayers players={lobby.online} />,
              },
            ]}
          />
        )}
      />
    </div>
  );
}
