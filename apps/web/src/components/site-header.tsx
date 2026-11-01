import Link from 'next/link';
import { Suspense } from 'react';
import { MainNav } from './main-nav';
import { UserMenu } from './user-menu';

/** The top of every page: logo, main menu and the account's menu, over a civilization stripe. */
export function SiteHeader() {
  return (
    <header className="sticky top-0 z-30 bg-linear-to-b from-header-top to-header-bottom shadow-[0_8px_24px_rgb(0_0_0/0.6)]">
      <div className="mx-auto flex h-14 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6">
        <div className="flex items-center gap-6">
          <Link
            href="/"
            className="group flex items-center gap-2.5"
            aria-label="Duel Masters, home"
          >
            <span
              aria-hidden
              className="size-5 rotate-45 bg-linear-to-br from-slate-100 via-slate-400 to-slate-700 shadow-[0_0_12px_rgb(148_163_184/0.6)] transition group-hover:shadow-[0_0_16px_rgb(56_189_248/0.7)]"
            />
            <span className="hidden bg-linear-to-b from-white to-slate-400 bg-clip-text font-display text-xl font-bold tracking-[0.18em] whitespace-nowrap text-transparent uppercase min-[30rem]:inline sm:text-2xl">
              Duel Masters
            </span>
          </Link>
          <MainNav />
        </div>
        {/* Depends on the session cookie, so it streams in after the rest of the header. */}
        <Suspense fallback={<div className="h-9 w-32" aria-hidden />}>
          <UserMenu />
        </Suspense>
      </div>
      <div aria-hidden className="civ-stripe h-[3px]" />
    </header>
  );
}
