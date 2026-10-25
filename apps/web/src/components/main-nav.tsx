'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { isCurrent, MAIN_LINKS } from './navigation';

/** The main menu in the header, on wider screens. */
export function MainNav() {
  const pathname = usePathname();

  return (
    <nav aria-label="Main" className="hidden items-center gap-1 md:flex">
      {MAIN_LINKS.map((link) => {
        const current = isCurrent(link, pathname);
        return (
          <Link
            key={link.href}
            href={link.href}
            aria-current={current ? 'page' : undefined}
            className={`border-b-2 px-3 py-1.5 font-display text-sm font-semibold tracking-wider uppercase transition ${
              current
                ? 'border-accent text-white'
                : 'border-transparent text-slate-400 hover:border-slate-500 hover:text-white'
            }`}
          >
            {link.label}
          </Link>
        );
      })}
    </nav>
  );
}
