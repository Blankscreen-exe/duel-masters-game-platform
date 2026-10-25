'use client';

import { SelectField } from '@/components/ui/select-field';
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
  return (
    <SelectField
      label="Format"
      value={value}
      onChange={(event) => {
        onChange(event.target.value);
      }}
    >
      {formats.map((format) => (
        <option key={format.id} value={format.id}>
          {format.name}
          {note?.(format) ?? ''}
        </option>
      ))}
    </SelectField>
  );
}
