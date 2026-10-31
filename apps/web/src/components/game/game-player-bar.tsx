import { PlayerAvatar } from '@/components/players/player-avatar';
import { RankBadge } from '@/components/players/rank-badge';
import type { GamePlayer } from '@/lib/live-game';

interface GamePlayerBarProps {
  /** The name the board knows them by, shown even if their details are missing. */
  readonly username: string;
  readonly player: GamePlayer | undefined;
  /** Whether this is the player looking at the screen. */
  readonly isViewer: boolean;
  /** Whether it is this player's turn. */
  readonly active: boolean;
}

/** A player on the board: picture, name and rank, lit up on their turn. Their deck stays hidden. */
export function GamePlayerBar({ username, player, isViewer, active }: GamePlayerBarProps) {
  return (
    <div
      className={`cut-corners-sm flex items-center gap-2 bg-panel/95 px-2 py-0.5 ring-1 ring-inset transition-shadow ${
        active ? 'ring-accent/70 shadow-[inset_3px_0_0_var(--color-accent)]' : 'ring-slate-600/40'
      }`}
    >
      <PlayerAvatar username={username} src={player?.avatarUrl} size="small" />
      <span className="min-w-0 truncate font-display text-sm font-bold tracking-wide text-white">
        {username}
      </span>
      {isViewer && <span className="shrink-0 text-xs text-slate-400">(you)</span>}
      {player?.computer ? (
        <span className="shrink-0 font-display text-[10px] font-bold tracking-wider text-slate-400 uppercase">
          Computer
        </span>
      ) : (
        player?.rank && <RankBadge rank={player.rank} />
      )}
      {active && (
        <span className="ml-auto shrink-0 font-display text-[10px] font-bold tracking-wider text-accent uppercase">
          {isViewer ? 'Your turn' : 'Their turn'}
        </span>
      )}
    </div>
  );
}
