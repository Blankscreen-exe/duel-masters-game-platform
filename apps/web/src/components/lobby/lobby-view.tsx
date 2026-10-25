'use client';

import { useState, type ReactNode } from 'react';
import { LobbyChatPanel } from '@/components/chat/lobby-chat';
import { useLiveMessage } from '@/components/live/live-provider';
import { Panel } from '@/components/ui/panel';
import { TabList } from '@/components/ui/tabs';
import type { Account, Format } from '@/lib/api/client';
import type { components } from '@/lib/api/schema';
import { activityOf, type PlayableDecks } from '@/lib/lobby';
import type { Lobby, QueueCount, QuickMatchStatus } from '@/lib/lobby-actions';
import { OnlinePlayers, OpenGamesList, RunningGamesList } from './game-lists';
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
 * On a wide screen it has four columns: starting a game on the left, the open games two columns
 * wide with the running games and who is online under them, and the chat on the right, the height
 * of the screen. Narrower, the chat goes under the rest; on a phone everything stacks.
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

      <div className="grid gap-6 lg:grid-cols-[20rem_1fr] xl:grid-cols-[20rem_1fr_22rem]">
        <StartGamePanel
          quickMatch={
            <QuickMatchPanel
              decks={decks}
              formats={formats}
              queue={queue}
              waitingFormatName={waitingFormatName}
              disabled={lobby.paused || (busy && activity.kind !== 'queued')}
            />
          }
          openGame={
            <OpenGameForm decks={decks} formats={formats} disabled={lobby.paused || busy} />
          }
        />

        <div className="space-y-6">
          <Panel title="Open games" actions={<Count n={lobby.openGames.length} />}>
            <OpenGamesList
              games={lobby.openGames}
              decks={decks}
              userId={account.id}
              canJoin={!busy && !lobby.paused}
            />
          </Panel>
          <div className="grid gap-6 md:grid-cols-2">
            <Panel title="Games running" actions={<Count n={lobby.runningGames.length} />}>
              <RunningGamesList games={lobby.runningGames} />
            </Panel>
            <Panel title="Online" actions={<Count n={lobby.online.length} />}>
              <OnlinePlayers players={lobby.online} />
            </Panel>
          </div>
        </div>

        <LobbyChatPanel
          initial={chat}
          canPost
          mute={account.mute}
          className="max-h-[32rem] lg:col-span-2 xl:sticky xl:top-20 xl:col-span-1 xl:self-start xl:h-[calc(100dvh-6.5rem)] xl:max-h-none"
        />
      </div>
    </div>
  );
}

type StartGameTab = 'quick' | 'open';

const START_GAME_PANELS = 'start-game';

interface StartGamePanelProps {
  readonly quickMatch: ReactNode;
  readonly openGame: ReactNode;
}

/**
 * Starting a game, in one box: finding a quick match or opening a game, one tab each. Both stay
 * mounted, so switching tabs keeps what was picked in each.
 */
function StartGamePanel({ quickMatch, openGame }: StartGamePanelProps) {
  const [tab, setTab] = useState<StartGameTab>('quick');
  const panels: Record<StartGameTab, ReactNode> = { quick: quickMatch, open: openGame };

  return (
    <Panel accent="fire" className="self-start">
      <TabList
        label="Start a game"
        panelIdPrefix={START_GAME_PANELS}
        value={tab}
        onChange={setTab}
        className="m-3 mb-0"
        tabs={[
          { id: 'quick', label: 'Quick match' },
          { id: 'open', label: 'Open a game' },
        ]}
      />
      {(Object.keys(panels) as StartGameTab[]).map((id) => (
        <div
          key={id}
          id={`${START_GAME_PANELS}-${id}`}
          role="tabpanel"
          aria-labelledby={`${START_GAME_PANELS}-tab-${id}`}
          hidden={tab !== id}
        >
          {panels[id]}
        </div>
      ))}
    </Panel>
  );
}

function Count({ n }: { readonly n: number }) {
  return <span className="font-display text-sm font-semibold text-slate-400">{n}</span>;
}
