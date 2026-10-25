'use client';

import type { ComponentProps, ReactNode } from 'react';
import { Button } from './button';
import { useModalDialog } from './use-modal-dialog';

interface ModalProps {
  /** The dialog's title, also its accessible name. */
  readonly title: string;
  readonly onClose: () => void;
  /** The content; use ModalClose for buttons that close it. */
  readonly children: ReactNode;
  readonly className?: string;
}

/**
 * A modal dialog: the browser's own, so focus stays inside and Escape closes it. Clicking the
 * backdrop closes it too. Render it only while it should be open.
 */
export function Modal({ title, onClose, children, className = '' }: ModalProps) {
  const ref = useModalDialog(onClose);

  return (
    <dialog
      ref={ref}
      aria-label={title}
      onClick={(event) => {
        // A click on the backdrop lands on the dialog itself.
        if (event.target === event.currentTarget) event.currentTarget.close();
      }}
      className={`m-auto max-h-[90dvh] w-[min(100%-2rem,28rem)] overflow-y-auto bg-transparent p-0 text-slate-200 backdrop:bg-night/80 backdrop:backdrop-blur-sm ${className}`}
    >
      <div className="cut-corners bg-panel shadow-[0_10px_30px_rgb(0_0_0/0.6)] ring-1 ring-slate-500/40 ring-inset">
        <h2 className="border-b border-slate-600/40 bg-linear-to-r from-white/[0.07] to-transparent px-5 py-3 font-display text-lg font-bold tracking-[0.15em] text-slate-100 uppercase">
          {title}
        </h2>
        <div className="p-5">{children}</div>
      </div>
    </dialog>
  );
}

/** A button that closes the modal it is in: a native method="dialog" form, no script needed. */
export function ModalClose(props: Omit<ComponentProps<typeof Button>, 'type'>) {
  return (
    <form method="dialog" className="contents">
      <Button {...props} type="submit" />
    </form>
  );
}
