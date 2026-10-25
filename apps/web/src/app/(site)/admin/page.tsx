import type { Metadata } from 'next';
import Link from 'next/link';
import { Panel } from '@/components/ui/panel';
import { ADMIN_SECTIONS } from '@/lib/admin';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Admin panel' };

/** The admin panel's front page: its sections. */
export default async function AdminPage() {
  await requireAdmin('/admin');

  return (
    <div className="space-y-6">
      <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
        Overview
      </h1>
      <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {ADMIN_SECTIONS.map((section) => (
          <li key={section.href}>
            <Link href={section.href} className="group block h-full">
              <Panel className="h-full p-5 transition group-hover:ring-accent/60">
                <h2 className="font-display text-xl font-bold tracking-wider text-slate-100 uppercase group-hover:text-accent">
                  {section.label}
                </h2>
                <p className="mt-2 text-sm text-slate-400">{section.description}</p>
              </Panel>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
