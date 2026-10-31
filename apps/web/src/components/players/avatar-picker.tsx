'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Modal, ModalClose } from '@/components/ui/modal';
import { setAvatar } from '@/lib/account-actions';
import type { Avatar } from '@/lib/api/client';
import { notify } from '@/lib/notifications';

interface AvatarPickerProps {
  readonly avatars: readonly Avatar[];
  /** The picture the player has now, if any. */
  readonly current: string | undefined;
}

/** Lets a player change their profile picture: pick one from a grid, then update. */
export function AvatarPicker({ avatars, current }: AvatarPickerProps) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button
        variant="secondary"
        onClick={() => {
          setOpen(true);
        }}
      >
        Change picture
      </Button>
      {open && (
        <AvatarGrid
          avatars={avatars}
          current={current}
          onClose={() => {
            setOpen(false);
          }}
        />
      )}
    </>
  );
}

function AvatarGrid({
  avatars,
  current,
  onClose,
}: AvatarPickerProps & { readonly onClose: () => void }) {
  const router = useRouter();
  const [chosen, setChosen] = useState(
    () => avatars.find(({ imageUrl }) => imageUrl === current)?.id ?? null,
  );
  const [saving, setSaving] = useState(false);
  const unchanged = avatars.find(({ id }) => id === chosen)?.imageUrl === current;

  async function update() {
    if (chosen === null) return;
    setSaving(true);
    const outcome = await setAvatar(chosen);
    setSaving(false);

    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    notify('Profile picture updated');
    router.refresh();
    onClose();
  }

  return (
    <Modal title="Choose your picture" size="full" onClose={onClose}>
      <ul className="grid min-h-0 flex-1 grid-cols-[repeat(auto-fill,minmax(4.5rem,1fr))] content-start gap-2 overflow-y-auto p-4">
        {avatars.map(({ id, imageUrl }, index) => {
          const selected = id === chosen;
          return (
            <li key={id}>
              <button
                type="button"
                aria-pressed={selected}
                aria-label={`Picture ${String(index + 1)}`}
                onClick={() => {
                  setChosen(id);
                }}
                className={`block w-full p-0.5 ring-2 transition ${
                  selected ? 'bg-accent/30 ring-accent' : 'ring-transparent hover:ring-slate-400'
                }`}
              >
                {/* eslint-disable-next-line @next/next/no-img-element -- tiny pixel art, served ready to use */}
                <img
                  src={imageUrl}
                  alt=""
                  loading="lazy"
                  decoding="async"
                  className="aspect-square w-full object-cover [image-rendering:pixelated]"
                />
              </button>
            </li>
          );
        })}
      </ul>
      <footer className="flex justify-end gap-3 border-t border-slate-600/40 px-5 py-3">
        <ModalClose variant="quiet">Cancel</ModalClose>
        <Button disabled={chosen === null || unchanged || saving} onClick={() => void update()}>
          {saving ? 'Updating…' : 'Update'}
        </Button>
      </footer>
    </Modal>
  );
}
