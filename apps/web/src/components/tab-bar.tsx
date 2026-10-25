'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { isCurrent, MAIN_LINKS } from './navigation';

/**
 * The main menu on phones: a bar along the bottom of the screen, within reach of a thumb. A spacer
 * of the bar's height keeps the end of the page from hiding behind it.
 */
export function TabBar() {
  const pathname = usePathname();

  if (MAIN_LINKS.length === 0) return null;

  return (
    <>
      <div aria-hidden className="h-14 md:hidden" />
      <nav
        aria-label="Main"
        className="fixed inset-x-0 bottom-0 z-30 border-t border-slate-600/50 bg-header-bottom/95 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
      >
        <ul className="flex">
          {MAIN_LINKS.map((link) => {
            const current = isCurrent(link, pathname);
            return (
              <li key={link.href} className="flex-1">
                <Link
                  href={link.href}
                  aria-current={current ? 'page' : undefined}
                  className={`flex min-h-14 items-center justify-center border-t-2 font-display text-sm font-semibold tracking-wider uppercase ${
                    current
                      ? 'border-accent text-white'
                      : 'border-transparent text-slate-400 active:bg-white/5'
                  }`}
                >
                  {link.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </>
  );
}
