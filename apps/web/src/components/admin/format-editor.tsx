'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useWarnOnLeaving } from '@/components/decks/deck-workbench';
import { Button } from '@/components/ui/button';
import { Chips } from '@/components/ui/chips';
import { FIELD_HINT, FIELD_LABEL } from '@/components/ui/field';
import { Panel } from '@/components/ui/panel';
import { SelectField } from '@/components/ui/select-field';
import { TextField } from '@/components/ui/text-field';
import { createFormat, saveFormat } from '@/lib/admin-actions';
import type { CardFilters } from '@/lib/api/client';
import {
  CIVILIZATION_COUNTS,
  civilizationRuleText,
  FORMAT_DESCRIPTION_MAX_LENGTH,
  FORMAT_NAME_MAX_LENGTH,
  formatFormProblem,
  formOfFormat,
  NEW_FORMAT,
  sameFormatForm,
  toFormatInput,
  type AdminFormat,
  type FormatForm,
} from '@/lib/formats';
import { notify } from '@/lib/notifications';
import { FormatCardLimits } from './format-card-limits';
import { FormatRetireButton } from './format-retire-button';

interface FormatEditorProps {
  /** The format to change, or null to add one. */
  readonly format: AdminFormat | null;
  readonly filters: CardFilters;
}

/** Adds or changes a format: its words, the sets it allows, and the cards it bans or limits. */
export function FormatEditor({ format, filters }: FormatEditorProps) {
  const router = useRouter();
  const [initial] = useState(() => (format ? formOfFormat(format) : NEW_FORMAT));
  const [saved, setSaved] = useState({ format, form: initial });
  const [form, setForm] = useState(initial);
  const [saving, setSaving] = useState(false);

  const unsaved = !saved.format || !sameFormatForm(form, saved.form);
  const problem = formatFormProblem(form);
  useWarnOnLeaving(unsaved && (saved.format !== null || form.name !== ''));

  function change(fields: Partial<FormatForm>) {
    setForm((current) => ({ ...current, ...fields }));
  }

  /** Shows the format as the server left it. */
  function show(changed: AdminFormat) {
    const shown = formOfFormat(changed);
    setSaved({ format: changed, form: shown });
    setForm(shown);
  }

  async function save() {
    setSaving(true);
    const input = toFormatInput(form);
    const outcome = await (saved.format ? saveFormat(saved.format.id, input) : createFormat(input));
    setSaving(false);

    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    notify('Format saved');
    if (saved.format) show(outcome.value);
    else router.replace(`/admin/formats/${encodeURIComponent(outcome.value.id)}`);
  }

  return (
    <div className="space-y-4">
      <Link
        href="/admin/formats"
        className="font-display text-sm font-semibold tracking-wider text-slate-400 uppercase hover:text-white"
      >
        ← Formats
      </Link>

      <Panel title={saved.format ? saved.format.name : 'New format'} accent="fire">
        <div className="space-y-5 p-4">
          <div className="grid gap-4 md:grid-cols-2">
            <TextField
              label="Name"
              value={form.name}
              maxLength={FORMAT_NAME_MAX_LENGTH}
              hint="What players pick it by; no two formats can share one."
              onChange={(event) => {
                change({ name: event.target.value });
              }}
            />
            <TextField
              label="Description"
              value={form.description}
              maxLength={FORMAT_DESCRIPTION_MAX_LENGTH}
              hint="One line saying what the format is."
              onChange={(event) => {
                change({ description: event.target.value });
              }}
            />
          </div>

          <fieldset className="space-y-2">
            <legend className={FIELD_LABEL}>Sets</legend>
            <div className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-slate-300">
              <label className="flex cursor-pointer items-center gap-2 hover:text-white">
                <input
                  type="radio"
                  name="format-sets"
                  checked={form.allSets}
                  onChange={() => {
                    change({ allSets: true });
                  }}
                  className="size-4 accent-sky-400"
                />
                Every set, including any added later
              </label>
              <label className="flex cursor-pointer items-center gap-2 hover:text-white">
                <input
                  type="radio"
                  name="format-sets"
                  checked={!form.allSets}
                  onChange={() => {
                    change({ allSets: false });
                  }}
                  className="size-4 accent-sky-400"
                />
                Only the sets I choose
              </label>
            </div>
            {!form.allSets && (
              <Chips
                label="Sets"
                options={filters.sets.map(({ code }) => ({ value: code, label: code }))}
                chosen={form.sets}
                onChange={(sets) => {
                  change({ sets });
                }}
              />
            )}
          </fieldset>

          <fieldset className="space-y-2">
            <legend className={FIELD_LABEL}>Civilizations in a deck</legend>
            <div className="grid gap-4 sm:max-w-md sm:grid-cols-2">
              <SelectField
                label="Fewest"
                value={form.minCivilizations}
                onChange={(event) => {
                  change({ minCivilizations: Number(event.target.value) });
                }}
              >
                {CIVILIZATION_COUNTS.map((count) => (
                  <option key={count} value={count}>
                    {count === 0 ? 'Any' : count}
                  </option>
                ))}
              </SelectField>
              <SelectField
                label="Most"
                value={form.maxCivilizations}
                onChange={(event) => {
                  change({ maxCivilizations: Number(event.target.value) });
                }}
              >
                {CIVILIZATION_COUNTS.map((count) => (
                  <option key={count} value={count}>
                    {count === 0 ? 'Any' : count}
                  </option>
                ))}
              </SelectField>
            </div>
            <p className={FIELD_HINT}>
              {civilizationRuleText(form) ?? 'Any number'}. A card of two civilizations counts as
              both. Set both the same for an exact number.
            </p>
          </fieldset>

          <div className="border-t border-slate-700/60 pt-4">
            <FormatCardLimits form={form} onChange={setForm} />
          </div>

          <div className="flex flex-wrap items-center justify-end gap-3 border-t border-slate-700/60 pt-4">
            <span className="mr-auto text-sm text-slate-400">
              {problem ??
                (saved.format?.retired
                  ? 'Retired: players cannot choose it.'
                  : 'Decks and new games follow these rules as soon as you save. Games running keep theirs.')}
            </span>
            {saved.format && (
              <FormatRetireButton
                format={saved.format}
                onChanged={show}
                disabled={unsaved}
                title={unsaved ? 'Save your changes first' : undefined}
              />
            )}
            <Button disabled={!unsaved || saving || problem !== null} onClick={() => void save()}>
              {saving ? 'Saving…' : unsaved ? 'Save' : 'Saved'}
            </Button>
          </div>
        </div>
      </Panel>
    </div>
  );
}
