'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { isCurrent } from '@/components/navigation';
import { ADMIN_SECTIONS } from '@/lib/admin';

/** The admin panel's menu: a row of its sections, which scrolls sideways on phones. */
export function AdminNav() {
  const pathname = usePathname();

  return (
    <nav aria-label="Admin panel" className="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
      <ul className="flex min-w-max gap-1 border-b border-slate-700/60">
        <li>
          <AdminLink href="/admin" label="Overview" current={pathname === '/admin'} />
        </li>
        {ADMIN_SECTIONS.map((section) => (
          <li key={section.href}>
            <AdminLink
              href={section.href}
              label={section.label}
              current={isCurrent(section, pathname)}
            />
          </li>
        ))}
      </ul>
    </nav>
  );
}

function AdminLink({
  href,
  label,
  current,
}: {
  readonly href: string;
  readonly label: string;
  readonly current: boolean;
}) {
  return (
    <Link
      href={href}
      aria-current={current ? 'page' : undefined}
      className={`-mb-px block min-h-11 border-b-2 px-3 py-2.5 font-display text-sm font-semibold tracking-wider uppercase transition ${
        current
          ? 'border-accent text-white'
          : 'border-transparent text-slate-400 hover:border-slate-500 hover:text-white'
      }`}
    >
      {label}
    </Link>
  );
}
