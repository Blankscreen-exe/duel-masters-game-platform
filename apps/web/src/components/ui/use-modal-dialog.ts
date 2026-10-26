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
 *
 * The browser fires a dialog's close event a moment after closing it. When the component is shown
 * again straight away, as when React mounts it twice, the close from the cleanup arrives after the
 * dialog has opened again. Only a close that leaves the dialog closed is the player's, so only that
 * one is reported.
 */
export function useModalDialog(onClose: () => void) {
  const ref = useRef<HTMLDialogElement>(null);
  const handleClose = useEffectEvent(onClose);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    const listener = () => {
      if (!dialog.open) handleClose();
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
