import Link from 'next/link';
import { buttonClasses } from '@/components/ui/button';
import type { CardFilters } from '@/lib/api/client';
import { type CardSearch, SEARCH_TEXT_MAX_LENGTH } from '@/lib/card-search';
import { CIVILIZATIONS, civilizationColour } from '@/lib/civilizations';

interface CardFiltersFormProps {
  readonly filters: CardFilters;
  readonly search: CardSearch;
}

const LABEL = 'block text-[11px] font-semibold tracking-wider text-slate-400 uppercase';
const FIELD =
  'mt-1 block w-full bg-field px-3 py-2 text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset hover:ring-slate-400/70 focus:ring-accent focus:outline-none';

/**
 * The card search. A plain form sent as the page address, so a search can be bookmarked or shared
 * and works before any script has loaded.
 */
export function CardFiltersForm({ filters, search }: CardFiltersFormProps) {
  return (
    <form action="/cards" role="search" aria-label="Cards" className="space-y-4 p-4">
      <div>
        <label htmlFor="card-search-q" className={LABEL}>
          Name or text
        </label>
        <input
          id="card-search-q"
          type="search"
          name="q"
          defaultValue={search.q}
          maxLength={SEARCH_TEXT_MAX_LENGTH}
          placeholder="Bolshack, blocker, draw…"
          className={FIELD}
        />
      </div>

      <fieldset>
        <legend className={LABEL}>Civilization</legend>
        <div className="mt-1 flex flex-wrap gap-1.5">
          {CIVILIZATIONS.filter(({ name }) => filters.civilizations.includes(name)).map(
            ({ name, key }) => (
              <label key={name} className="cursor-pointer">
                <input
                  type="checkbox"
                  name="civilization"
                  value={name}
                  defaultChecked={search.civilization?.includes(name)}
                  className="peer sr-only"
                />
                <span
                  className={`inline-block border-l-4 bg-slate-800/80 px-2.5 py-1 text-xs font-semibold text-slate-300 ring-1 ring-slate-600/60 ring-inset transition peer-checked:bg-slate-600 peer-checked:text-white peer-checked:ring-accent peer-focus-visible:outline-2 peer-focus-visible:outline-accent hover:text-white ${civilizationColour(key).border}`}
                >
                  {name}
                </span>
              </label>
            ),
          )}
        </div>
      </fieldset>

      <Choice label="Type" name="type" options={filters.types} chosen={search.type} />
      <Choice label="Race" name="race" options={filters.races} chosen={search.race} />
      <Choice label="Keyword" name="keyword" options={filters.keywords} chosen={search.keyword} />
      <Choice
        label="Set"
        name="set"
        options={filters.sets.map((set) => set.code)}
        chosen={search.set}
      />

      <fieldset>
        <legend className={LABEL}>Cost</legend>
        <div className="mt-1 flex items-center gap-2">
          <input
            type="number"
            name="minCost"
            min={0}
            inputMode="numeric"
            aria-label="Lowest cost"
            placeholder="From"
            defaultValue={search.minCost}
            className={`${FIELD} mt-0`}
          />
          <span aria-hidden className="text-slate-500">
            –
          </span>
          <input
            type="number"
            name="maxCost"
            min={0}
            inputMode="numeric"
            aria-label="Highest cost"
            placeholder="To"
            defaultValue={search.maxCost}
            className={`${FIELD} mt-0`}
          />
        </div>
      </fieldset>

      <label className="flex cursor-pointer items-center gap-2 text-sm text-slate-300 hover:text-white">
        <input
          type="checkbox"
          name="playable"
          value="true"
          defaultChecked={search.playable === true}
          className="size-4 accent-sky-400"
        />
        Only cards that can be played
      </label>

      <div className="flex gap-2 pt-1">
        <button type="submit" className={buttonClasses('primary', 'flex-1')}>
          Search
        </button>
        <Link href="/cards" className={buttonClasses('secondary')}>
          Clear
        </Link>
      </div>
    </form>
  );
}

interface ChoiceProps {
  readonly label: string;
  readonly name: string;
  readonly options: readonly string[];
  readonly chosen: readonly string[] | undefined;
}

/** One value from a long list, or any. */
function Choice({ label, name, options, chosen }: ChoiceProps) {
  const id = `card-search-${name}`;

  return (
    <div>
      <label htmlFor={id} className={LABEL}>
        {label}
      </label>
      <select id={id} name={name} defaultValue={chosen?.[0] ?? ''} className={FIELD}>
        <option value="">Any</option>
        {options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </div>
  );
}
