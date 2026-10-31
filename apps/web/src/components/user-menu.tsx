import Link from 'next/link';
import { profileHref } from '@/lib/players';
import { getCurrentAccount, getMyAnnouncements } from '@/lib/session';
import { LogOutButton } from './log-out-button';
import { NotificationsLink } from './notifications-link';
import { PlayerAvatar } from './players/player-avatar';
import { RankBadge } from './players/rank-badge';
import { AccountRefresh } from './players/account-refresh';
import { buttonClasses } from './ui/button';

/**
 * Log in and sign up for visitors; once logged in, the player's name, rank and log out, and the way
 * into the admin panel for admins.
 */
export async function UserMenu() {
  const account = await getCurrentAccount();

  if (!account) {
    return (
      <div className="flex items-center gap-2">
        <NotificationsLink unread={0} />
        <Link href="/login" className={buttonClasses('quiet')}>
          Log in
        </Link>
        <Link href="/signup" className={buttonClasses('primary', 'px-4 py-1.5 text-sm')}>
          Sign up
        </Link>
      </div>
    );
  }

  const { unread } = await getMyAnnouncements();

  return (
    <div className="flex items-center gap-3">
      <AccountRefresh accountId={account.id} />
      <NotificationsLink unread={unread} />
      <span className="flex items-center gap-2 text-sm font-semibold text-slate-100">
        <Link
          href={profileHref(account.username)}
          title="Your profile"
          className="group flex items-center gap-2"
        >
          <PlayerAvatar username={account.username} src={account.avatarUrl} size="small" />
          <span className="sr-only sm:not-sr-only group-hover:text-accent">{account.username}</span>
        </Link>
        <span className="hidden sm:inline-flex">
          <RankBadge rank={account.rank} />
        </span>
        {account.role === 'admin' && (
          <Link
            href="/admin"
            title="The admin panel"
            className="px-1.5 py-0.5 text-[10px] font-bold tracking-wider text-accent uppercase ring-1 ring-accent/50 transition ring-inset hover:bg-accent/15 hover:text-white"
          >
            Admin
          </Link>
        )}
      </span>
      <LogOutButton />
    </div>
  );
}
