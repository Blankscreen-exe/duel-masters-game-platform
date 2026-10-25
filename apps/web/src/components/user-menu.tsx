import Link from 'next/link';
import { getCurrentAccount } from '@/lib/session';
import { LogOutButton } from './log-out-button';
import { buttonClasses } from './ui/button';

/** Log in and sign up for visitors; the player's name and log out once logged in. */
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
      <span className="flex items-center gap-2 text-sm font-semibold text-slate-100">
        <span
          aria-hidden
          className="cut-corners-sm grid size-8 place-items-center bg-linear-to-br from-slate-200 to-slate-500 font-display text-base font-bold text-slate-900"
        >
          {account.username.charAt(0).toUpperCase()}
        </span>
        <span className="hidden sm:inline">{account.username}</span>
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
