'use client';

import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Modal, ModalClose } from '@/components/ui/modal';
import type { Outcome } from '@/lib/api/outcome';
import { deckText } from '@/lib/deck-actions';
import { notify } from '@/lib/notifications';

interface ExportDeckDialogProps {
  readonly deckId: string;
  readonly deckName: string;
  readonly onClose: () => void;
}

/** The saved deck as text, to copy or download and import elsewhere. */
export function ExportDeckDialog({ deckId, deckName, onClose }: ExportDeckDialogProps) {
  const [text, setText] = useState<Outcome<string> | null>(null);

  useEffect(() => {
    let current = true;
    void deckText(deckId).then((outcome) => {
      if (current) setText(outcome);
    });
    return () => {
      current = false;
    };
  }, [deckId]);

  async function copy(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      notify('Deck list copied');
    } catch {
      notify('Copying was not allowed; select the text and copy it instead.');
    }
  }

  function download(value: string) {
    const link = document.createElement('a');
    link.href = URL.createObjectURL(new Blob([value], { type: 'text/plain' }));
    link.download = `${deckName.replace(/[^\w -]+/g, '').trim() || 'deck'}.txt`;
    link.click();
    // Released once the browser has started saving the file.
    setTimeout(() => {
      URL.revokeObjectURL(link.href);
    }, 0);
  }

  return (
    <Modal title="Export deck" onClose={onClose} className="w-[min(100%-2rem,32rem)]">
      {text === null ? (
        <p className="text-sm text-slate-400">Loading…</p>
      ) : text.ok ? (
        <>
          <textarea
            readOnly
            aria-label="Deck list"
            value={text.value}
            rows={Math.min(16, Math.max(4, text.value.split('\n').length))}
            className="block w-full bg-field px-3 py-2 font-mono text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset focus:ring-accent focus:outline-none"
            onFocus={(event) => {
              event.currentTarget.select();
            }}
          />
          <div className="mt-4 flex justify-end gap-2">
            <ModalClose variant="secondary">Close</ModalClose>
            <Button
              variant="secondary"
              onClick={() => {
                download(text.value);
              }}
            >
              Download
            </Button>
            <Button onClick={() => void copy(text.value)}>Copy</Button>
          </div>
        </>
      ) : (
        <>
          <p role="alert" className="text-sm text-red-200">
            {text.message}
          </p>
          <div className="mt-4 flex justify-end">
            <ModalClose variant="secondary">Close</ModalClose>
          </div>
        </>
      )}
    </Modal>
  );
}
