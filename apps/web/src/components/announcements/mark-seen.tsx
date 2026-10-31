'use client';

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { markAnnouncementsSeen } from '@/lib/account-actions';

/**
 * Notes that the player has looked at the notifications, whenever there are unread ones while the
 * page is open, then draws the page again, which clears the count in the top bar.
 */
export function MarkSeen({ unread }: { readonly unread: number }) {
  const router = useRouter();

  useEffect(() => {
    if (unread === 0) return;

    void markAnnouncementsSeen().then((outcome) => {
      if (outcome.ok) router.refresh();
    });
  }, [unread, router]);

  return null;
}
