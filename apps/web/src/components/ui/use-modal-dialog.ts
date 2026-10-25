'use client';

import { useEffect, useEffectEvent, useRef } from 'react';

/**
 * Shows a <dialog> as a modal while the component is on screen, and calls onClose when the player
 * closes it (Escape, a method="dialog" form, or close()). Attach the returned ref.
 *
 * The dialog is also closed when the component is hidden rather than removed: Next.js keeps the
 * page you moved away from in the document, hidden, and a modal left open there would block every
 * click on the new page. That close is not reported, so the dialog's state survives; React runs
 * the same cleanup in development when it mounts components twice.
 */
export function useModalDialog(onClose: () => void) {
  const ref = useRef<HTMLDialogElement>(null);
  const handleClose = useEffectEvent(onClose);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    const listener = () => {
      handleClose();
    };
    dialog.addEventListener('close', listener);
    if (!dialog.open) dialog.showModal();
    return () => {
      dialog.removeEventListener('close', listener);
      dialog.close();
    };
  }, []);

  return ref;
}
