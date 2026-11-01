import Link from 'next/link';
import { notificationsLabel, NOTIFICATIONS_PATH, unreadBadge } from '@/lib/announcements';

/** The bell in the top bar, on every screen size, with how many notifications are unread. */
export function NotificationsLink({ unread }: { readonly unread: number }) {
  const badge = unreadBadge(unread);
  const label = notificationsLabel(unread);

  return (
    <Link
      href={NOTIFICATIONS_PATH}
      aria-label={label}
      title={label}
      className="relative flex size-9 shrink-0 items-center justify-center text-slate-300 transition hover:bg-white/10 hover:text-white"
    >
      <svg
        viewBox="0 0 24 24"
        className="size-5"
        fill="none"
        stroke="currentColor"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M18 8a6 6 0 10-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.7 21a2 2 0 01-3.4 0" />
      </svg>
      {badge && (
        <span
          aria-hidden="true"
          className="absolute top-0.5 right-0 min-w-4 bg-red-600 px-1 text-center text-[10px] leading-4 font-bold text-white"
        >
          {badge}
        </span>
      )}
    </Link>
  );
}
