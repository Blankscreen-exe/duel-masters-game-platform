import type { Metadata } from 'next';
import {
  AnnouncementRowActions,
  NewAnnouncementButton,
} from '@/components/admin/announcement-manager';
import { AnnouncementCard } from '@/components/announcements/announcement-card';
import { Pagination } from '@/components/ui/pagination';
import { Panel } from '@/components/ui/panel';
import { ADMIN_NOTIFICATIONS_PATH, pageHref, pageParam } from '@/lib/announcements';
import { serverApi } from '@/lib/api/server';
import { pageCount } from '@/lib/search-params';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Notifications · Admin' };

/** What admins tell every player: post, change and delete. */
export default async function AdminNotificationsPage({
  searchParams,
}: PageProps<'/admin/notifications'>) {
  const number = pageParam(await searchParams);
  await requireAdmin(ADMIN_NOTIFICATIONS_PATH);
  const api = await serverApi();
  const { data: page } = await api.GET('/announcements', {
    params: { query: { page: number } },
  });

  if (!page) throw new Error('The notifications failed to load');

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          Notifications
        </h1>
        <NewAnnouncementButton />
      </header>
      <p className="text-sm text-slate-400">
        Everyone can read these on the Notifications page. An important one also pops up in the
        lobby for every player who joined before it, until they close it.
      </p>
      <Panel>
        {page.announcements.length === 0 ? (
          <p className="px-4 py-10 text-center text-slate-400">Nothing posted yet.</p>
        ) : (
          <ul aria-label={`Notifications, ${String(page.total)} in all`}>
            {page.announcements.map((announcement) => (
              <AnnouncementCard
                key={announcement.id}
                announcement={announcement}
                actions={<AnnouncementRowActions announcement={announcement} />}
              />
            ))}
          </ul>
        )}
      </Panel>
      <Pagination
        page={page.page}
        pages={pageCount(page.total, page.pageSize)}
        href={(target) => pageHref(ADMIN_NOTIFICATIONS_PATH, target)}
      />
    </div>
  );
}
