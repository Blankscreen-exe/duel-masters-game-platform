import type { Rank, Tier } from '@/lib/players';

/** Each tier's colours as style classes, written out in full so the stylesheet includes them. */
const TIER_CLASSES: Readonly<Record<Tier, string>> = {
  Bronze: 'text-amber-500 ring-amber-700/70',
  Silver: 'text-slate-200 ring-slate-400/70',
  Gold: 'text-yellow-300 ring-yellow-500/70',
  Platinum: 'text-cyan-200 ring-cyan-400/70',
  Diamond: 'text-sky-300 ring-sky-400/80 shadow-[0_0_10px_rgb(56_189_248/0.45)]',
};

interface RankBadgeProps {
  readonly rank: Rank;
  /** Shows the rating beside the tier. */
  readonly withRating?: boolean;
  readonly large?: boolean;
}

/** A player's rank, such as Gold II, in its tier's colours; the rating shows on hover. */
export function RankBadge({ rank, withRating = false, large = false }: RankBadgeProps) {
  const size = large ? 'px-2.5 py-1 text-sm' : 'px-1.5 py-0.5 text-[10px]';

  return (
    <span
      title={`Rating ${String(rank.rating)}`}
      aria-label={`${rank.label}, rating ${String(rank.rating)}`}
      className={`inline-flex shrink-0 items-center gap-1.5 bg-black/30 font-display font-bold tracking-wider whitespace-nowrap uppercase ring-1 ring-inset ${size} ${TIER_CLASSES[rank.tier]}`}
    >
      <span aria-hidden>{rank.label}</span>
      {withRating && (
        <span aria-hidden className="text-slate-400 tabular-nums">
          {rank.rating}
        </span>
      )}
    </span>
  );
}
