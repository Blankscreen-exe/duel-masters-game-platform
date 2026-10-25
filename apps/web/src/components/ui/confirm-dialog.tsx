'use client';

import { useState, type ReactNode } from 'react';
import { Button } from './button';
import { Modal, ModalClose } from './modal';

interface ConfirmDialogProps {
  readonly title: string;
  readonly children: ReactNode;
  /** The confirming button's text, such as "Delete deck". */
  readonly confirmLabel: string;
  /** Does the risky thing; the dialog stays open, showing it is busy, until it finishes. */
  readonly onConfirm: () => Promise<void>;
  readonly onClose: () => void;
}

/** Asks before something that cannot be undone, such as deleting a deck. */
export function ConfirmDialog({
  title,
  children,
  confirmLabel,
  onConfirm,
  onClose,
}: ConfirmDialogProps) {
  const [pending, setPending] = useState(false);

  async function confirm() {
    setPending(true);
    try {
      await onConfirm();
    } finally {
      setPending(false);
    }
  }

  return (
    <Modal title={title} onClose={onClose}>
      <div className="text-sm text-slate-300">{children}</div>
      <div className="mt-6 flex justify-end gap-2">
        <ModalClose variant="secondary" disabled={pending}>
          Cancel
        </ModalClose>
        <Button variant="danger" disabled={pending} onClick={() => void confirm()}>
          {pending ? 'Working…' : confirmLabel}
        </Button>
      </div>
    </Modal>
  );
}
