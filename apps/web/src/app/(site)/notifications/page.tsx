import type { Metadata } from 'next';
import { AnnouncementCard } from '@/components/announcements/announcement-card';
import { MarkSeen } from '@/components/announcements/mark-seen';
import { Pagination } from '@/components/ui/pagination';
import { Panel } from '@/components/ui/panel';
import { NOTIFICATIONS_PATH, pageHref, pageParam } from '@/lib/announcements';
import { serverApi } from '@/lib/api/server';
import { pageCount } from '@/lib/search-params';
import { getMyAnnouncements } from '@/lib/session';

export const metadata: Metadata = { title: 'Notifications' };

/** What admins have told every player, for anyone to read. */
export default async function NotificationsPage({ searchParams }: PageProps<'/notifications'>) {
  const number = pageParam(await searchParams);
  const api = await serverApi();
  const [{ data: page }, mine] = await Promise.all([
    api.GET('/announcements', { params: { query: { page: number } } }),
    getMyAnnouncements(),
  ]);

  if (!page) throw new Error('The notifications failed to load');

  return (
    <div className="space-y-6">
      <MarkSeen unread={mine.unread} />
      <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
        Notifications
      </h1>
      <Panel>
        {page.announcements.length === 0 ? (
          <p className="px-4 py-10 text-center text-slate-400">Nothing yet. Check back soon.</p>
        ) : (
          <ul aria-label={`Notifications, ${String(page.total)} in all`}>
            {page.announcements.map((announcement) => (
              <AnnouncementCard key={announcement.id} announcement={announcement} />
            ))}
          </ul>
        )}
      </Panel>
      <Pagination
        page={page.page}
        pages={pageCount(page.total, page.pageSize)}
        href={(target) => pageHref(NOTIFICATIONS_PATH, target)}
      />
    </div>
  );
}
