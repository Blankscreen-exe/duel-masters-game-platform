'use client';

import { useRouter } from 'next/navigation';
import { useId, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Modal, ModalClose } from '@/components/ui/modal';
import { TextField } from '@/components/ui/text-field';
import type { DeckListReading } from '@/lib/api/client';
import { createDeck, readDeckList } from '@/lib/deck-actions';
import { DECK_LIST_MAX_LENGTH, DECK_NAME_MAX_LENGTH, DECK_SIZE } from '@/lib/deck-draft';
import { notify } from '@/lib/notifications';

/** Creates a deck from a list pasted as text, after showing what was found. */
export function ImportDeckButton() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button
        variant="secondary"
        onClick={() => {
          setOpen(true);
        }}
      >
        Import
      </Button>
      {open && (
        <ImportDeckDialog
          onClose={() => {
            setOpen(false);
          }}
        />
      )}
    </>
  );
}

function ImportDeckDialog({ onClose }: { readonly onClose: () => void }) {
  const router = useRouter();
  const textId = useId();
  const [name, setName] = useState('Imported deck');
  const [text, setText] = useState('');
  const [reading, setReading] = useState<DeckListReading | null>(null);
  const [pending, setPending] = useState(false);
  const total = reading?.cards.reduce((sum, { count }) => sum + count, 0) ?? 0;
  const tooMany = total > DECK_SIZE;

  async function read() {
    setPending(true);
    const outcome = await readDeckList(text);
    setPending(false);
    if (outcome.ok) setReading(outcome.value);
    else notify(outcome.message);
  }

  async function create() {
    if (!reading) return;
    setPending(true);
    const outcome = await createDeck({
      name,
      cards: reading.cards.map(({ cardId, count }) => ({ cardId, count })),
    });
    if (!outcome.ok) {
      setPending(false);
      notify(outcome.message);
      return;
    }
    notify(`Created “${outcome.value.name}”`);
    router.push(`/decks/${encodeURIComponent(outcome.value.id)}`);
  }

  return (
    <Modal title="Import a deck" onClose={onClose} className="w-[min(100%-2rem,36rem)]">
      <div className="space-y-4">
        <TextField
          label="Deck name"
          value={name}
          required
          maxLength={DECK_NAME_MAX_LENGTH}
          onChange={(event) => {
            setName(event.target.value);
          }}
        />
        <div>
          <label
            htmlFor={textId}
            className="block text-[11px] font-semibold tracking-wider text-slate-400 uppercase"
          >
            Deck list
          </label>
          <textarea
            id={textId}
            value={text}
            maxLength={DECK_LIST_MAX_LENGTH}
            rows={8}
            placeholder={'4 Bolshack Dragon\n4 Aqua Hulcus\n…'}
            onChange={(event) => {
              setText(event.target.value);
              setReading(null);
            }}
            className="mt-1 block w-full bg-field px-3 py-2 font-mono text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset placeholder:text-slate-600 hover:ring-slate-400/70 focus:ring-accent focus:outline-none"
          />
          <p className="mt-1 text-xs text-slate-500">
            One card per line, with how many copies: “4 Bolshack Dragon”.
          </p>
        </div>

        {reading && (
          <div aria-live="polite" className="space-y-2 text-sm">
            <p className={tooMany ? 'text-amber-300' : 'text-slate-200'}>
              Found {total} {total === 1 ? 'card' : 'cards'}
              {tooMany &&
                `, but a deck can have at most ${String(DECK_SIZE)}. Remove some lines first.`}
            </p>
            {reading.problems.length > 0 && (
              <ul className="max-h-40 space-y-1 overflow-y-auto bg-amber-500/10 px-3 py-2 text-amber-100 ring-1 ring-amber-400/40 ring-inset">
                {reading.problems.map((problem) => (
                  <li key={`${String(problem.line)}-${problem.message}`}>
                    <span className="text-amber-300">Line {problem.line}:</span> {problem.message}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}

        <div className="flex justify-end gap-2 pt-2">
          <ModalClose variant="secondary" disabled={pending}>
            Cancel
          </ModalClose>
          {reading ? (
            <Button
              disabled={pending || tooMany || name.trim() === ''}
              onClick={() => void create()}
            >
              {pending ? 'Creating…' : 'Create deck'}
            </Button>
          ) : (
            <Button disabled={pending || text.trim() === ''} onClick={() => void read()}>
              {pending ? 'Reading…' : 'Read list'}
            </Button>
          )}
        </div>
      </div>
    </Modal>
  );
}
