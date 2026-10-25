import type { ReactNode } from 'react';
import { AdminNav } from '@/components/admin/admin-nav';
import { getCurrentAccount } from '@/lib/session';

/**
 * The admin panel's frame: its title and menu, for admins only. Each page checks the visitor
 * itself, sending visitors to log in and answering players as if it did not exist; the frame
 * only stays out of the way of anyone else.
 */
export default async function AdminLayout({ children }: { readonly children: ReactNode }) {
  const account = await getCurrentAccount();
  if (account?.role !== 'admin') return children;

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <p className="font-display text-sm font-semibold tracking-[0.25em] text-accent uppercase">
          Admin panel
        </p>
        <AdminNav />
      </div>
      {children}
    </div>
  );
}
