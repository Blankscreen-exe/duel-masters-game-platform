'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { setFormatRetired } from '@/lib/admin-actions';
import type { AdminFormat } from '@/lib/formats';
import { notify } from '@/lib/notifications';

interface FormatRetireButtonProps {
  readonly format: Pick<AdminFormat, 'id' | 'name' | 'retired' | 'opponents'>;
  readonly onChanged: (changed: AdminFormat) => void;
  readonly disabled?: boolean;
  readonly title?: string;
}

/** Retires a format, saying first what that does, or brings it back. */
export function FormatRetireButton({
  format,
  onChanged,
  disabled = false,
  title,
}: FormatRetireButtonProps) {
  const [asking, setAsking] = useState(false);
  const [pending, setPending] = useState(false);

  async function change() {
    setPending(true);
    const outcome = await setFormatRetired(format.id, !format.retired);
    setPending(false);
    setAsking(false);
    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    notify(
      outcome.value.retired
        ? `${format.name} is retired`
        : `Players can choose ${format.name} again`,
    );
    onChanged(outcome.value);
  }

  return (
    <>
      <Button
        variant="secondary"
        disabled={disabled || pending}
        title={title}
        onClick={() => {
          if (format.retired) void change();
          else setAsking(true);
        }}
      >
        {format.retired ? 'Bring back' : 'Retire'}
      </Button>
      {asking && (
        <ConfirmDialog
          title={`Retire ${format.name}?`}
          confirmLabel="Retire format"
          onConfirm={change}
          onClose={() => {
            setAsking(false);
          }}
        >
          <p>
            Players will no longer be able to choose it for new games, quick matches or rematches,
            and decks will not be checked against it. Games running or already open carry on, and
            past games keep its name. You can bring it back at any time.
          </p>
          {format.opponents.length > 0 && (
            <p className="mt-3 text-amber-300">
              These computer opponents play in it, and will stop being offered until you bring it
              back or move them to another format: {format.opponents.join(', ')}.
            </p>
          )}
        </ConfirmDialog>
      )}
    </>
  );
}
