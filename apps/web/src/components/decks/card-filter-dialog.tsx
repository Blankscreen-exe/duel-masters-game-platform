'use client';

import { useState, type ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { Chips } from '@/components/ui/chips';
import { FIELD_CONTROL, FIELD_HINT, FIELD_LABEL } from '@/components/ui/field';
import { Modal, ModalClose } from '@/components/ui/modal';
import { SelectField } from '@/components/ui/select-field';
import { TextField } from '@/components/ui/text-field';
import type { CardFilters, Format } from '@/lib/api/client';
import {
  cardFiltersProblem,
  keywordLabel,
  NO_CARD_FILTERS,
  type CardFilterChoices,
} from '@/lib/card-filters';
import { SEARCH_TEXT_MAX_LENGTH } from '@/lib/card-search';
import { CIVILIZATIONS } from '@/lib/civilizations';
import { MAX_COPIES } from '@/lib/deck-draft';

interface CardFilterDialogProps {
  /** The choices in force, to start from. */
  readonly choices: CardFilterChoices;
  readonly filters: CardFilters;
  readonly formats: readonly Format[];
  /** Takes the choices; nothing changes until then, and closing without it drops them. */
  readonly onApply: (choices: CardFilterChoices) => void;
  readonly onClose: () => void;
}

/** Every way of narrowing and ordering the trunk's cards. */
export function CardFilterDialog({
  choices: initial,
  filters,
  formats,
  onApply,
  onClose,
}: CardFilterDialogProps) {
  const [choices, setChoices] = useState(initial);
  const problem = cardFiltersProblem(choices);

  function change(fields: Partial<CardFilterChoices>) {
    setChoices((current) => ({ ...current, ...fields }));
  }

  return (
    <Modal
      title="Filter cards"
      onClose={onClose}
      className="w-[min(100%-2rem,50rem)]"
      // First in the dialog, which takes the focus: anything lower scrolls past the first filters.
      actions={<ModalClose variant="secondary">Cancel</ModalClose>}
    >
      <div className="space-y-5">
        <TextField
          label="Card text"
          type="search"
          value={choices.text}
          maxLength={SEARCH_TEXT_MAX_LENGTH}
          placeholder="draw a card, destroy, shield…"
          hint="Looks through what cards do, in any word order. Small typing mistakes are fine."
          onChange={(event) => {
            change({ text: event.target.value });
          }}
        />

        <fieldset className="space-y-3">
          <legend className={FIELD_LABEL}>Civilization</legend>
          <Chips
            label="Civilization"
            options={CIVILIZATIONS.filter(({ name }) => filters.civilizations.includes(name)).map(
              ({ name }) => ({ value: name, label: name }),
            )}
            chosen={choices.civilizations}
            onChange={(civilizations) => {
              change({ civilizations });
            }}
          />
          <div className="grid gap-3 sm:grid-cols-2">
            <SelectField
              label="Cards with"
              value={choices.civilizationMatch}
              disabled={choices.civilizations.length === 0}
              hint={choices.civilizations.length === 0 ? 'Choose a civilization first.' : undefined}
              onChange={(event) => {
                change({
                  civilizationMatch: event.target.value as CardFilterChoices['civilizationMatch'],
                });
              }}
            >
              <option value="any">Any of these</option>
              <option value="exactly">Exactly these, and no others</option>
              <option value="only">Only these, one or more of them</option>
            </SelectField>
            <SelectField
              label="How many civilizations"
              value={choices.civilizationCount}
              onChange={(event) => {
                change({
                  civilizationCount: event.target.value as CardFilterChoices['civilizationCount'],
                });
              }}
            >
              <option value="">Any number</option>
              <option value="single">One civilization</option>
              <option value="multi">Two or more</option>
            </SelectField>
          </div>
        </fieldset>

        <fieldset>
          <legend className={FIELD_LABEL}>Type</legend>
          <Chips
            label="Type"
            className="mt-1.5"
            options={filters.types.map((type) => ({ value: type, label: type }))}
            chosen={choices.types}
            onChange={(types) => {
              change({ types });
            }}
          />
        </fieldset>

        <div className="grid gap-5 sm:grid-cols-2">
          <Range
            label="Cost"
            from={choices.minCost}
            to={choices.maxCost}
            step={1}
            onChange={(minCost, maxCost) => {
              change({ minCost, maxCost });
            }}
          />
          <Range
            label="Power"
            from={choices.minPower}
            to={choices.maxPower}
            step={500}
            hint="Leaves out cards without power, such as spells."
            onChange={(minPower, maxPower) => {
              change({ minPower, maxPower });
            }}
          />
        </div>

        <div className="grid gap-5 sm:grid-cols-2">
          <CheckList
            label="Ability"
            plural="abilities"
            options={filters.keywords
              .map((keyword) => ({ value: keyword, label: keywordLabel(keyword) }))
              .sort((a, b) => a.label.localeCompare(b.label))}
            chosen={choices.keywords}
            onChange={(keywords) => {
              change({ keywords });
            }}
          />
          <CheckList
            label="Race"
            plural="races"
            options={filters.races.map((race) => ({ value: race, label: race }))}
            chosen={choices.races}
            onChange={(races) => {
              change({ races });
            }}
          />
        </div>

        <fieldset>
          <legend className={FIELD_LABEL}>Rarity</legend>
          <Chips
            label="Rarity"
            className="mt-1.5"
            options={filters.rarities.map(({ code, name }) => ({ value: code, label: name }))}
            chosen={choices.rarities}
            onChange={(rarities) => {
              change({ rarities });
            }}
          />
        </fieldset>

        <fieldset>
          <legend className={FIELD_LABEL}>Set</legend>
          <Chips
            label="Set"
            className="mt-1.5"
            options={filters.sets.map(({ code }) => ({ value: code, label: code }))}
            chosen={choices.sets}
            onChange={(sets) => {
              change({ sets });
            }}
          />
        </fieldset>

        <fieldset className="space-y-3 border-t border-slate-700/60 pt-4">
          <legend className="sr-only">Deck</legend>
          <div className="grid gap-3 sm:grid-cols-2">
            <SelectField
              label="Allowed in format"
              value={choices.formatId}
              onChange={(event) => {
                change({ formatId: event.target.value });
              }}
            >
              <option value="">Any format</option>
              {formats.map((format) => (
                <option key={format.id} value={format.id}>
                  {format.name}
                </option>
              ))}
            </SelectField>
            <SelectField
              label="In this deck"
              value={choices.deckCards}
              onChange={(event) => {
                change({ deckCards: event.target.value as CardFilterChoices['deckCards'] });
              }}
            >
              <option value="all">All cards</option>
              <option value="in">Only cards in this deck</option>
              <option value="out">Only cards not in this deck</option>
            </SelectField>
          </div>
          <label className="flex cursor-pointer items-center gap-2 text-sm text-slate-300 hover:text-white">
            <input
              type="checkbox"
              checked={choices.hideFullCards}
              onChange={(event) => {
                change({ hideFullCards: event.target.checked });
              }}
              className="size-4 accent-sky-400"
            />
            Hide cards the deck already has {MAX_COPIES} of
          </label>
          <p className={FIELD_HINT}>
            These go by the deck as it is when you press Apply, so a card does not vanish while you
            are adding it. Apply again to update them.
          </p>
        </fieldset>

        <div className="grid gap-3 border-t border-slate-700/60 pt-4 sm:grid-cols-2">
          <SelectField
            label="Sort by"
            value={choices.sort}
            onChange={(event) => {
              change({ sort: event.target.value as CardFilterChoices['sort'] });
            }}
          >
            <option value="name">Name</option>
            <option value="cost">Cost</option>
            <option value="power">Power</option>
          </SelectField>
          <SelectField
            label="Order"
            value={choices.order}
            onChange={(event) => {
              change({ order: event.target.value as CardFilterChoices['order'] });
            }}
          >
            <option value="asc">Lowest or A first</option>
            <option value="desc">Highest or Z first</option>
          </SelectField>
        </div>

        {/* Stays in view while the rest scrolls, out across the padding to the frame's thin edge. */}
        <div className="sticky bottom-0 -mx-[calc(1.25rem-1px)] -mb-[calc(1.25rem-1px)] flex flex-wrap items-center justify-end gap-2 border-t border-slate-600/40 bg-panel px-5 py-3">
          {problem !== null && (
            <p role="alert" className="mr-auto text-sm text-red-200">
              {problem}
            </p>
          )}
          <Button
            variant="secondary"
            onClick={() => {
              setChoices(NO_CARD_FILTERS);
            }}
          >
            Clear all
          </Button>
          <Button
            disabled={problem !== null}
            onClick={() => {
              onApply(choices);
            }}
          >
            Apply
          </Button>
        </div>
      </div>
    </Modal>
  );
}

interface Option {
  readonly value: string;
  readonly label: string;
}

interface ManyChoiceProps {
  readonly label: string;
  readonly options: readonly Option[];
  readonly chosen: readonly string[];
  readonly onChange: (chosen: readonly string[]) => void;
}

function toggled(chosen: readonly string[], value: string): readonly string[] {
  return chosen.includes(value) ? chosen.filter((c) => c !== value) : [...chosen, value];
}

/** Many values to pick any number of: a list that scrolls, with a box to find one. */
function CheckList({
  label,
  plural,
  options,
  chosen,
  onChange,
}: ManyChoiceProps & { readonly plural: string }) {
  const [finding, setFinding] = useState('');
  const found = finding.trim().toLowerCase();
  const shown = options.filter((option) => option.label.toLowerCase().includes(found));

  return (
    <fieldset>
      <legend className={FIELD_LABEL}>
        {label}
        {chosen.length > 0 && <span className="text-accent"> · {chosen.length} chosen</span>}
      </legend>
      <input
        type="search"
        aria-label={`Find ${plural}`}
        placeholder={`Find ${plural}…`}
        value={finding}
        onChange={(event) => {
          setFinding(event.target.value);
        }}
        className={FIELD_CONTROL}
      />
      <ul className="mt-1 h-40 overflow-y-auto bg-field/60 p-1 ring-1 ring-slate-700/60 ring-inset">
        {shown.map(({ value, label: text }) => (
          <li key={value}>
            <label className="flex cursor-pointer items-center gap-2 px-2 py-1 text-sm text-slate-300 hover:bg-white/[0.04] hover:text-white">
              <input
                type="checkbox"
                checked={chosen.includes(value)}
                onChange={() => {
                  onChange(toggled(chosen, value));
                }}
                className="size-4 shrink-0 accent-sky-400"
              />
              {text}
            </label>
          </li>
        ))}
        {shown.length === 0 && <li className="px-2 py-1 text-sm text-slate-500">None found.</li>}
      </ul>
    </fieldset>
  );
}

interface RangeProps {
  readonly label: string;
  readonly from: string;
  readonly to: string;
  readonly step: number;
  readonly hint?: ReactNode;
  readonly onChange: (from: string, to: string) => void;
}

/** A lowest and highest number, either or both left empty. */
function Range({ label, from, to, step, hint, onChange }: RangeProps) {
  return (
    <fieldset>
      <legend className={FIELD_LABEL}>{label}</legend>
      <div className="flex items-center gap-2">
        <input
          type="number"
          min={0}
          step={step}
          inputMode="numeric"
          aria-label={`Lowest ${label.toLowerCase()}`}
          placeholder="From"
          value={from}
          onChange={(event) => {
            onChange(event.target.value, to);
          }}
          className={FIELD_CONTROL}
        />
        <span aria-hidden className="mt-1 text-slate-500">
          –
        </span>
        <input
          type="number"
          min={0}
          step={step}
          inputMode="numeric"
          aria-label={`Highest ${label.toLowerCase()}`}
          placeholder="To"
          value={to}
          onChange={(event) => {
            onChange(from, event.target.value);
          }}
          className={FIELD_CONTROL}
        />
      </div>
      {hint && <p className={FIELD_HINT}>{hint}</p>}
    </fieldset>
  );
}
