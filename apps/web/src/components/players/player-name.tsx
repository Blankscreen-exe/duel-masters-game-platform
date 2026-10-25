import Link from 'next/link';
import { profileHref, type Rank } from '@/lib/players';
import { RankBadge } from './rank-badge';

/** A player's name, leading to their profile, with their rank when known. */
export function PlayerName({
  username,
  rank,
}: {
  readonly username: string;
  readonly rank?: Rank;
}) {
  return (
    <span className="inline-flex min-w-0 items-center gap-2">
      <Link
        href={profileHref(username)}
        className="truncate font-semibold text-slate-100 hover:text-accent"
      >
        {username}
      </Link>
      {rank && <RankBadge rank={rank} />}
    </span>
  );
}
