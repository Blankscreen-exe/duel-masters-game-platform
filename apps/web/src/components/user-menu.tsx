import Link from 'next/link';
import { profileHref } from '@/lib/players';
import { getCurrentAccount } from '@/lib/session';
import { LogOutButton } from './log-out-button';
import { RankBadge } from './players/rank-badge';
import { RankRefresh } from './players/rank-refresh';
import { buttonClasses } from './ui/button';

/** Log in and sign up for visitors; once logged in, the player's name, rank and log out. */
export async function UserMenu() {
  const account = await getCurrentAccount();

  if (!account) {
    return (
      <div className="flex items-center gap-2">
        <Link href="/login" className={buttonClasses('quiet')}>
          Log in
        </Link>
        <Link href="/signup" className={buttonClasses('primary', 'px-4 py-1.5 text-sm')}>
          Sign up
        </Link>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-3">
      <RankRefresh accountId={account.id} />
      <span className="flex items-center gap-2 text-sm font-semibold text-slate-100">
        <Link
          href={profileHref(account.username)}
          title="Your profile"
          className="group flex items-center gap-2"
        >
          <span
            aria-hidden
            className="cut-corners-sm grid size-8 place-items-center bg-linear-to-br from-slate-200 to-slate-500 font-display text-base font-bold text-slate-900 transition group-hover:from-white group-hover:to-slate-400"
          >
            {account.username.charAt(0).toUpperCase()}
          </span>
          <span className="sr-only sm:not-sr-only group-hover:text-accent">{account.username}</span>
        </Link>
        <span className="hidden sm:inline-flex">
          <RankBadge rank={account.rank} />
        </span>
        {account.role === 'admin' && (
          <span className="px-1.5 py-0.5 text-[10px] font-bold tracking-wider text-accent uppercase ring-1 ring-accent/50 ring-inset">
            Admin
          </span>
        )}
      </span>
      <LogOutButton />
    </div>
  );
}
