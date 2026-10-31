'use client';

import { useRouter } from 'next/navigation';
import { useId, useState, type SubmitEvent } from 'react';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { FIELD_CONTROL, FIELD_HINT, FIELD_LABEL } from '@/components/ui/field';
import { Modal, ModalClose } from '@/components/ui/modal';
import { TextField } from '@/components/ui/text-field';
import { createAnnouncement, deleteAnnouncement, updateAnnouncement } from '@/lib/admin-actions';
import {
  BODY_MAX_LENGTH,
  TITLE_MAX_LENGTH,
  type Announcement,
  type AnnouncementInput,
} from '@/lib/announcements';
import { textField } from '@/lib/form-data';
import { notify } from '@/lib/notifications';

/** The list's button to post a new notification. */
export function NewAnnouncementButton() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button
        onClick={() => {
          setOpen(true);
        }}
      >
        Post a notification
      </Button>
      {open && (
        <AnnouncementEditor
          onClose={() => {
            setOpen(false);
          }}
        />
      )}
    </>
  );
}

/** A notification's buttons in the list: change it, or delete it after asking. */
export function AnnouncementRowActions({ announcement }: { readonly announcement: Announcement }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function remove() {
    const outcome = await deleteAnnouncement(announcement.id);
    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    notify('Notification deleted.');
    setDeleting(false);
    router.refresh();
  }

  return (
    <>
      <Button
        variant="secondary"
        className="px-3 py-1.5 text-xs"
        aria-label={`Change ${announcement.title}`}
        onClick={() => {
          setEditing(true);
        }}
      >
        Change
      </Button>
      <Button
        variant="danger"
        className="px-3 py-1.5 text-xs"
        aria-label={`Delete ${announcement.title}`}
        onClick={() => {
          setDeleting(true);
        }}
      >
        Delete
      </Button>
      {editing && (
        <AnnouncementEditor
          announcement={announcement}
          onClose={() => {
            setEditing(false);
          }}
        />
      )}
      {deleting && (
        <ConfirmDialog
          title="Delete this notification?"
          confirmLabel="Delete"
          onConfirm={remove}
          onClose={() => {
            setDeleting(false);
          }}
        >
          “{announcement.title}” goes from everyone&apos;s page, and its popup goes for those who
          have not closed it. This cannot be undone.
        </ConfirmDialog>
      )}
    </>
  );
}

/** Writing a notification, new or already posted. */
function AnnouncementEditor({
  announcement,
  onClose,
}: {
  readonly announcement?: Announcement;
  readonly onClose: () => void;
}) {
  const router = useRouter();
  const bodyId = useId();
  const [important, setImportant] = useState(announcement?.important ?? false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const alreadyImportant = announcement?.important ?? false;

  async function save(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const input: AnnouncementInput = {
      title: textField(form, 'title'),
      body: textField(form, 'body'),
      important,
      showAgain: form.has('showAgain'),
    };

    setPending(true);
    setError(null);
    const outcome = announcement
      ? await updateAnnouncement(announcement.id, input)
      : await createAnnouncement(input);
    setPending(false);
    if (!outcome.ok) {
      setError(outcome.message);
      return;
    }

    notify(announcement ? 'Notification changed.' : 'Notification posted.');
    onClose();
    router.refresh();
  }

  return (
    <Modal title={announcement ? 'Change notification' : 'Post a notification'} onClose={onClose}>
      <form onSubmit={(event) => void save(event)} className="space-y-4">
        <TextField
          label="Title"
          name="title"
          required
          maxLength={TITLE_MAX_LENGTH}
          defaultValue={announcement?.title}
        />
        <div>
          <label htmlFor={bodyId} className={FIELD_LABEL}>
            Text
          </label>
          <textarea
            id={bodyId}
            name="body"
            required
            rows={6}
            maxLength={BODY_MAX_LENGTH}
            defaultValue={announcement?.body}
            className={FIELD_CONTROL}
          />
          <p className={FIELD_HINT}>Plain text; line breaks are kept.</p>
        </div>
        <label className="flex cursor-pointer items-start gap-2 text-sm text-slate-200">
          <input
            type="checkbox"
            className="mt-0.5"
            checked={important}
            onChange={(event) => {
              setImportant(event.target.checked);
            }}
          />
          <span>
            Important
            <span className={`block ${FIELD_HINT}`}>
              Also pops up in the lobby for every player, until they close it.
            </span>
          </span>
        </label>
        {alreadyImportant && important && (
          <label className="flex cursor-pointer items-start gap-2 text-sm text-slate-200">
            <input type="checkbox" name="showAgain" className="mt-0.5" />
            <span>
              Show again
              <span className={`block ${FIELD_HINT}`}>
                Pops it up again for players who have closed it. Leave this off for small fixes.
              </span>
            </span>
          </label>
        )}

        {error && (
          <p
            role="alert"
            className="bg-red-500/10 px-3 py-2 text-sm text-red-200 ring-1 ring-red-400/40 ring-inset"
          >
            {error}
          </p>
        )}

        <div className="flex justify-end gap-2">
          <ModalClose variant="secondary" disabled={pending}>
            Cancel
          </ModalClose>
          <Button type="submit" disabled={pending}>
            {announcement ? 'Save' : 'Post'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
