'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { DeckWorkbench, useWarnOnLeaving } from '@/components/decks/deck-workbench';
import { FormatPicker } from '@/components/lobby/format-picker';
import { Button } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { SelectField } from '@/components/ui/select-field';
import { TextField } from '@/components/ui/text-field';
import { createOpponent, saveOpponent } from '@/lib/admin-actions';
import type { CardFilters, Format } from '@/lib/api/client';
import {
  CIVILIZATION_LIMITS,
  formOf,
  newOpponentForm,
  OPPONENT_DESCRIPTION_MAX_LENGTH,
  opponentDraft,
  rulesBroken,
  sameForm,
  toOpponentInput,
  withCardType,
  type AdminComputerOpponent,
  type ComputerOpponentOptions,
  type OpponentForm,
} from '@/lib/computer-opponents';
import { USERNAME_MAX_LENGTH, USERNAME_MIN_LENGTH } from '@/lib/credentials';
import { sameDeck } from '@/lib/deck-draft';
import { notify } from '@/lib/notifications';
import { OpponentOfferButton } from './opponent-offer-button';

interface OpponentEditorProps {
  /** The opponent to change, or null to add one. */
  readonly opponent: AdminComputerOpponent | null;
  readonly options: ComputerOpponentOptions;
  readonly formats: readonly Format[];
  readonly filters: CardFilters;
}

/**
 * Adds or changes a computer opponent: who it is and how it plays, the rules its deck keeps to, and
 * the deck, built as players build theirs. A new one starts retired, to be offered once its deck is
 * ready; an offered one must keep a deck it can play, so retire it to work on the deck.
 */
export function OpponentEditor({ opponent, options, formats, filters }: OpponentEditorProps) {
  const router = useRouter();
  const [initialForm] = useState(() =>
    opponent ? formOf(opponent) : newOpponentForm(options, formats[0]?.id ?? ''),
  );
  const [initialDraft] = useState(() => opponentDraft(opponent));
  const [saved, setSaved] = useState({ opponent, form: initialForm, draft: initialDraft });
  const [form, setForm] = useState(initialForm);
  const [draft, setDraft] = useState(initialDraft);
  const [saving, setSaving] = useState(false);

  const unsaved = !saved.opponent || !sameForm(form, saved.form) || !sameDeck(draft, saved.draft);
  const broken = rulesBroken(form, draft);
  useWarnOnLeaving(unsaved && (saved.opponent !== null || form.name !== ''));

  function change(fields: Partial<OpponentForm>) {
    setForm((current) => ({ ...current, ...fields }));
  }

  /** Shows the opponent as the server left it. */
  function show(changed: AdminComputerOpponent) {
    const shownForm = formOf(changed);
    const shownDraft = opponentDraft(changed);
    setSaved({ opponent: changed, form: shownForm, draft: shownDraft });
    setForm(shownForm);
    setDraft(shownDraft);
  }

  async function save() {
    setSaving(true);
    const input = toOpponentInput(form, draft);
    const outcome = await (saved.opponent
      ? saveOpponent(saved.opponent.id, input)
      : createOpponent(input));
    setSaving(false);

    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    notify('Opponent saved');
    if (saved.opponent) show(outcome.value);
    else router.replace(`/admin/computer-opponents/${encodeURIComponent(outcome.value.id)}`);
  }

  return (
    <div className="space-y-4">
      <Link
        href="/admin/computer-opponents"
        className="font-display text-sm font-semibold tracking-wider text-slate-400 uppercase hover:text-white"
      >
        ← Computer opponents
      </Link>

      <Panel title={saved.opponent ? saved.opponent.name : 'New computer opponent'} accent="fire">
        <div className="space-y-4 p-4">
          <div className="grid gap-4 md:grid-cols-2">
            <TextField
              label="Name"
              value={form.name}
              minLength={USERNAME_MIN_LENGTH}
              maxLength={USERNAME_MAX_LENGTH}
              hint="As a player's: letters, numbers, underscores and hyphens, and nobody's already."
              onChange={(event) => {
                change({ name: event.target.value });
              }}
            />
            <TextField
              label="Description"
              value={form.description}
              maxLength={OPPONENT_DESCRIPTION_MAX_LENGTH}
              hint="One line, shown to players choosing an opponent."
              onChange={(event) => {
                change({ description: event.target.value });
              }}
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <SelectField
              label="Play style"
              value={form.style}
              onChange={(event) => {
                change({ style: event.target.value });
              }}
            >
              {options.styles.map((style) => (
                <option key={style.name} value={style.name}>
                  {style.label}
                </option>
              ))}
            </SelectField>
            <SelectField
              label="Strength"
              value={form.strength}
              onChange={(event) => {
                change({ strength: event.target.value === 'easy' ? 'easy' : 'normal' });
              }}
            >
              <option value="easy">Easy: makes small mistakes</option>
              <option value="normal">Normal: plays its best</option>
            </SelectField>
            <FormatPicker
              formats={formats}
              value={form.formatId}
              onChange={(formatId) => {
                change({ formatId });
              }}
            />
            <SelectField
              label="Civilizations in its deck"
              value={form.maxCivilizations}
              onChange={(event) => {
                change({ maxCivilizations: Number(event.target.value) });
              }}
            >
              {CIVILIZATION_LIMITS.map((limit) => (
                <option key={limit.value} value={limit.value}>
                  {limit.label}
                </option>
              ))}
            </SelectField>
          </div>

          <fieldset>
            <legend className="text-sm font-semibold text-slate-200">Card types in its deck</legend>
            <div className="mt-2 flex flex-wrap gap-x-6 gap-y-2 text-sm text-slate-300">
              {options.cardTypes.map((cardType) => (
                <label
                  key={cardType}
                  className="flex cursor-pointer items-center gap-2 hover:text-white"
                >
                  <input
                    type="checkbox"
                    checked={form.cardTypes.includes(cardType)}
                    onChange={(event) => {
                      const allowed = event.target.checked;
                      setForm((current) => withCardType(current, cardType, allowed));
                    }}
                    className="size-4 accent-sky-400"
                  />
                  {cardType}
                </label>
              ))}
            </div>
            <p className="mt-2 text-xs text-slate-500">
              The game does not play castles or Cross Gear yet, so a deck cannot hold them for now.
            </p>
          </fieldset>

          <div className="flex flex-wrap items-center justify-end gap-3 border-t border-slate-700/60 pt-4">
            <span className="mr-auto text-sm text-slate-400">
              {saved.opponent
                ? saved.opponent.offered
                  ? 'Offered to players.'
                  : 'Retired: players cannot choose it.'
                : 'It starts retired, until you offer it to players.'}
            </span>
            {saved.opponent && (
              <OpponentOfferButton
                opponent={saved.opponent}
                onChanged={show}
                disabled={unsaved}
                title={unsaved ? 'Save your changes first' : undefined}
              />
            )}
            <Button disabled={!unsaved || saving} onClick={() => void save()}>
              {saving ? 'Saving…' : unsaved ? 'Save' : 'Saved'}
            </Button>
          </div>
        </div>
      </Panel>

      <DeckWorkbench
        draft={draft}
        onChange={setDraft}
        filters={filters}
        formats={formats}
        sidePanels={
          <Panel title="Can it play?">
            <div className="space-y-2 p-4 text-sm">
              {broken.length === 0 && (
                <p className="text-emerald-300">
                  The deck keeps to its rules
                  {unsaved ? '. Save to check it against the format.' : '.'}
                </p>
              )}
              {broken.length > 0 && (
                <ul className="list-disc space-y-1 pl-5 text-amber-300">
                  {broken.map((problem) => (
                    <li key={problem}>{problem}</li>
                  ))}
                </ul>
              )}
              {!unsaved && saved.opponent && saved.opponent.problems.length > 0 && (
                <ul className="list-disc space-y-1 pl-5 text-red-300">
                  {saved.opponent.problems.map((problem) => (
                    <li key={`${problem.code}-${problem.cardId ?? ''}`}>{problem.message}</li>
                  ))}
                </ul>
              )}
            </div>
          </Panel>
        }
      />
    </div>
  );
}
