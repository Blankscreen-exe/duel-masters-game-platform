import type { ReactNode } from 'react';
import type { Announcement } from '@/lib/announcements';
import { dateTimeText } from '@/lib/dates';

/** One notification in a list: its title, marked if important, when it went out, and its text. */
export function AnnouncementCard({
  announcement,
  actions,
}: {
  readonly announcement: Announcement;
  /** Next to the title, such as an admin's buttons. */
  readonly actions?: ReactNode;
}) {
  return (
    <li className="border-b border-slate-800/80 px-4 py-4 last:border-0">
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
        <div className="min-w-0 flex-1 basis-64">
          <h2 className="flex flex-wrap items-center gap-2 font-display text-lg font-bold tracking-wide break-words text-white">
            {announcement.title}
            {announcement.important && (
              <span className="px-1.5 py-0.5 font-sans text-[10px] font-bold tracking-wider text-amber-300 uppercase ring-1 ring-amber-400/50 ring-inset">
                Important
              </span>
            )}
          </h2>
          <p className="text-xs text-slate-500">
            <time dateTime={announcement.postedAt}>{dateTimeText(announcement.postedAt)}</time>
          </p>
        </div>
        {actions && <div className="flex shrink-0 gap-2">{actions}</div>}
      </div>
      <p className="mt-2 text-sm break-words whitespace-pre-line text-slate-300">
        {announcement.body}
      </p>
    </li>
  );
}
