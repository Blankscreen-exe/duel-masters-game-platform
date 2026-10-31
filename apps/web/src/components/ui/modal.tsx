'use client';

import type { ComponentProps, ReactNode } from 'react';
import { Button } from './button';
import { useModalDialog } from './use-modal-dialog';

/**
 * How big a modal is. A dialog is as tall as its content, up to most of the screen, and scrolls
 * past that. A full one is the height of the screen, less a thin margin, however tall the screen
 * is, and its content fills it edge to edge and scrolls itself, such as a chat.
 */
export type ModalSize = 'dialog' | 'full';

const SIZES: Record<ModalSize, { dialog: string; frame: string; body: string }> = {
  dialog: {
    dialog: 'max-h-[90dvh] w-[min(100%-2rem,28rem)] overflow-y-auto',
    frame: '',
    body: 'p-5',
  },
  full: {
    dialog: 'h-[calc(100dvh-2rem)] max-h-none w-[min(100%-2rem,72rem)] overflow-hidden',
    frame: 'flex h-full flex-col',
    body: 'flex min-h-0 flex-1 flex-col',
  },
};

interface ModalProps {
  /** The dialog's title, also its accessible name. */
  readonly title: string;
  readonly onClose: () => void;
  /** The content; use ModalClose for buttons that close it. */
  readonly children: ReactNode;
  readonly size?: ModalSize;
  /** Next to the title, such as a button that closes it. */
  readonly actions?: ReactNode;
  readonly className?: string;
}

/**
 * A modal dialog: the browser's own, so focus stays inside and Escape closes it. Clicking the
 * backdrop closes it too. Render it only while it should be open.
 */
export function Modal({
  title,
  onClose,
  children,
  size = 'dialog',
  actions,
  className = '',
}: ModalProps) {
  const ref = useModalDialog(onClose);
  const sized = SIZES[size];

  return (
    <dialog
      ref={ref}
      aria-label={title}
      onClick={(event) => {
        // A click on the backdrop lands on the dialog itself.
        if (event.target === event.currentTarget) event.currentTarget.close();
      }}
      className={`m-auto bg-transparent p-0 text-slate-200 backdrop:bz-backdrop ${sized.dialog} ${className}`}
    >
      <div className={`bz-dialog ${sized.frame}`}>
        <header className="bz-dialog-header flex items-center justify-between gap-3 px-5 py-3">
          <h2 className="bz-title text-lg">{title}</h2>
          {actions}
        </header>
        <div className={sized.body}>{children}</div>
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
