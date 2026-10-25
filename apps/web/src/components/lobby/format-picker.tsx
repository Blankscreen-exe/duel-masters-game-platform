'use client';

import { useId } from 'react';
import type { Format } from '@/lib/api/client';

interface FormatPickerProps {
  readonly formats: readonly Format[];
  readonly value: string;
  readonly onChange: (formatId: string) => void;
  /** Extra words after a format's name, such as how many players wait in it. */
  readonly note?: (format: Format) => string;
}

/** One of the formats games are played in. */
export function FormatPicker({ formats, value, onChange, note }: FormatPickerProps) {
  const id = useId();

  return (
    <div>
      <label
        htmlFor={id}
        className="block text-[11px] font-semibold tracking-wider text-slate-400 uppercase"
      >
        Format
      </label>
      <select
        id={id}
        value={value}
        onChange={(event) => {
          onChange(event.target.value);
        }}
        className="mt-1 block w-full bg-field px-3 py-2 text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset hover:ring-slate-400/70 focus:ring-accent focus:outline-none"
      >
        {formats.map((format) => (
          <option key={format.id} value={format.id}>
            {format.name}
            {note?.(format) ?? ''}
          </option>
        ))}
      </select>
    </div>
  );
}
