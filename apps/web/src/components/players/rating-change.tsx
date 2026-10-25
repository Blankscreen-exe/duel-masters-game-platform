import { ratingChangeText, type MatchPlayer } from '@/lib/players';

/** What a match did to a player's rating, such as +16 to 1016; or that it did not count. */
export function RatingChange({ player }: { readonly player: MatchPlayer }) {
  if (player.ratingChange === undefined || player.rating === undefined) {
    return <span className="text-xs text-slate-500">Not counted</span>;
  }

  const colour = player.ratingChange >= 0 ? 'text-civ-nature' : 'text-civ-fire';
  return (
    <span className="whitespace-nowrap">
      <span className={`font-display text-base font-bold tabular-nums ${colour}`}>
        {ratingChangeText(player.ratingChange)}
      </span>{' '}
      <span className="text-xs text-slate-400 tabular-nums">to {player.rating}</span>
    </span>
  );
}
