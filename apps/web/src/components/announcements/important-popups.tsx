'use client';

import { useState } from 'react';
import { Modal, ModalClose } from '@/components/ui/modal';
import { dismissAnnouncement } from '@/lib/account-actions';
import { nextPopup, popupKey, type Announcement } from '@/lib/announcements';
import { dateTimeText } from '@/lib/dates';

/**
 * The lobby's popups: each important notification the player has yet to close, one at a time, the
 * oldest first. Only closing one counts, by any means, so one that was never closed comes back. If
 * telling the server fails, it comes back next time too: better twice than never.
 */
export function ImportantPopups({ popups }: { readonly popups: readonly Announcement[] }) {
  const [closed, setClosed] = useState<ReadonlySet<string>>(new Set());
  const showing = nextPopup(popups, closed);

  if (!showing) return null;

  function close(popup: Announcement) {
    setClosed((current) => new Set([...current, popupKey(popup)]));
    void dismissAnnouncement(popup.id);
  }

  return (
    <Modal
      key={popupKey(showing)}
      title="Important"
      onClose={() => {
        close(showing);
      }}
    >
      <h3 className="font-display text-xl font-bold tracking-wide break-words text-white">
        {showing.title}
      </h3>
      <p className="text-xs text-slate-500">
        <time dateTime={showing.postedAt}>{dateTimeText(showing.postedAt)}</time>
      </p>
      <p className="mt-3 text-sm break-words whitespace-pre-line text-slate-200">{showing.body}</p>
      <div className="mt-6 flex justify-end">
        <ModalClose autoFocus>Got it</ModalClose>
      </div>
    </Modal>
  );
}
