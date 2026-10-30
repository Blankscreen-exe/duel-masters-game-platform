'use client';

import { useState } from 'react';
import { LobbyChat } from '@/components/chat/lobby-chat';
import { useLiveMessage } from '@/components/live/live-provider';
import { Panel } from '@/components/ui/panel';
import { TabbedPanel } from '@/components/ui/tabbed-panel';
import type { Account, Format } from '@/lib/api/client';
import type { components } from '@/lib/api/schema';
import { FILL_SCREEN } from '@/lib/fill-screen';
import { activityOf, type PlayableDecks } from '@/lib/lobby';
import type { ComputerOpponent, Lobby, QueueCount, QuickMatchStatus } from '@/lib/lobby-actions';
import { GamesList, OnlinePlayers } from './game-lists';
import { OpenGameForm } from './open-game-form';
import { QuickMatchPanel } from './quick-match-panel';
import { VsComputerPanel } from './vs-computer-panel';
import { YourActivity } from './your-activity';

interface LobbyViewProps {
  readonly account: Account;
  readonly lobby: Lobby;
  readonly quickMatch: QuickMatchStatus;
  readonly decks: PlayableDecks;
  readonly formats: readonly Format[];
  readonly opponents: readonly ComputerOpponent[];
  readonly chat: readonly components['schemas']['ChatMessage'][];
}

/**
 * The lobby: open a game or find a quick match, join an open game, watch a running one, and chat.
 * Everything on it stays up to date over the live connection.
 *
 * On a wide screen the whole page fits the screen, without scrolling, in four columns. Three hold
 * the title, starting a game, a tab each for quick match, opening a game and playing the computer,
 * and under it the games, open and running in one list that takes the height left and scrolls
 * inside. The fourth
 * holds the chat and who is online, a tab each, the full height. A little narrower, the fourth
 * keeps a fixed width; on a phone everything stacks and the page scrolls, the chat as tall as the
 * screen.
 */
export function LobbyView({
  account,
  lobby: initialLobby,
  quickMatch,
  decks,
  formats,
  opponents,
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
    <div
      {...FILL_SCREEN}
      className="grid gap-6 lg:min-h-0 lg:flex-1 lg:grid-cols-[1fr_20rem] lg:grid-rows-[minmax(0,1fr)] xl:grid-cols-4"
    >
      <div className="flex flex-col gap-6 lg:min-h-0 xl:col-span-3">
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
          className="shrink-0"
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
            {
              id: 'computer',
              label: 'Vs computer',
              content: (
                <VsComputerPanel
                  opponents={opponents}
                  decks={decks}
                  disabled={lobby.paused || busy}
                />
              ),
            },
          ]}
        />

        <Panel
          title="Games"
          className="flex flex-col lg:min-h-0 lg:flex-1"
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
          <div className="min-h-0 flex-1 overflow-y-auto">
            <GamesList
              openGames={lobby.openGames}
              runningGames={lobby.runningGames}
              openOnly={openOnly}
              decks={decks}
              userId={account.id}
              canJoin={!busy && !lobby.paused}
            />
          </div>
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
            className="max-lg:h-[calc(100dvh-6.75rem)] max-lg:min-h-96 lg:min-h-0"
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
