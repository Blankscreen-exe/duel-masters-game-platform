'use client';

import Link from 'next/link';
import { useState } from 'react';
import { Button, buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import type { components } from '@/lib/api/schema';
import { inviteLink } from '@/lib/lobby';
import { cancelOpenGame, declineRematch, type OpenGame } from '@/lib/lobby-actions';
import { notify } from '@/lib/notifications';
import { useSiteOrigin } from '@/lib/use-site-origin';

type PlayerActivity = components['schemas']['PlayerActivity'];

interface YourActivityProps {
  readonly activity: PlayerActivity;
  /** The open game hosted, when hosting and known. */
  readonly hosted: OpenGame | null;
}

/** What the player is doing now, with the way to carry on or stop: their open game, or their game. */
export function YourActivity({ activity, hosted }: YourActivityProps) {
  if (activity.kind === 'playing' && activity.gameId) {
    return (
      <Panel accent="fire" className="flex flex-wrap items-center justify-between gap-4 p-4">
        <p className="font-display text-lg font-semibold text-white">You are in a game.</p>
        <Link
          href={`/games/${encodeURIComponent(activity.gameId)}`}
          className={buttonClasses('primary')}
        >
          Back to your game
        </Link>
      </Panel>
    );
  }

  if (activity.kind === 'hosting' && activity.gameId) {
    return <HostedGame openGameId={activity.gameId} hosted={hosted} />;
  }

  if (activity.kind === 'rematch' && activity.gameId) {
    return <UndecidedRematch gameId={activity.gameId} />;
  }

  return null;
}

/**
 * A rematch still being decided, as the lobby shows it: for a player who came back here without
 * declining it, such as with the browser's back button. Declining frees them to play.
 */
function UndecidedRematch({ gameId }: { readonly gameId: string }) {
  const [pending, setPending] = useState(false);

  async function decline() {
    setPending(true);
    const outcome = await declineRematch(gameId);
    setPending(false);
    if (!outcome.ok) notify(outcome.message);
  }

  return (
    <Panel accent="light" className="flex flex-wrap items-center justify-between gap-4 p-4">
      <p className="font-display text-lg font-semibold text-white">
        A rematch of your last game is waiting for your answer.
      </p>
      <Button variant="secondary" disabled={pending} onClick={() => void decline()}>
        Decline the rematch
      </Button>
    </Panel>
  );
}

function HostedGame({
  openGameId,
  hosted,
}: {
  readonly openGameId: string;
  readonly hosted: OpenGame | null;
}) {
  const origin = useSiteOrigin();
  const link = origin === '' ? '' : inviteLink(origin, openGameId);
  const [pending, setPending] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(link);
      notify('Invite link copied');
    } catch {
      notify('Copying was not allowed; select the link and copy it instead.');
    }
  }

  async function cancel() {
    setPending(true);
    const outcome = await cancelOpenGame(openGameId);
    setPending(false);
    if (!outcome.ok) notify(outcome.message);
  }

  return (
    <Panel accent="water" className="space-y-3 p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p aria-live="polite" className="font-display text-lg font-semibold text-white">
          {hosted
            ? `“${hosted.name}” is waiting for an opponent…`
            : 'Your game is waiting for an opponent…'}
        </p>
        <Button variant="danger" disabled={pending} onClick={() => void cancel()}>
          Cancel game
        </Button>
      </div>
      <div className="flex gap-2">
        <input
          readOnly
          aria-label="Invite link"
          value={link}
          onFocus={(event) => {
            event.currentTarget.select();
          }}
          className="min-w-0 flex-1 bg-field px-3 py-2 font-mono text-xs text-slate-300 ring-1 ring-slate-600/60 ring-inset"
        />
        <Button variant="secondary" disabled={link === ''} onClick={() => void copy()}>
          Copy invite link
        </Button>
      </div>
    </Panel>
  );
}
