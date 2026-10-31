'use client';

interface ChipOption {
  readonly value: string;
  readonly label: string;
}

interface ChipsProps {
  /** What the values are, for screen readers. */
  readonly label: string;
  readonly options: readonly ChipOption[];
  readonly chosen: readonly string[];
  readonly onChange: (chosen: readonly string[]) => void;
  readonly className?: string;
}

/** A few values to pick any number of, all in view. */
export function Chips({ label, options, chosen, onChange, className = '' }: ChipsProps) {
  return (
    <div role="group" aria-label={label} className={`flex flex-wrap gap-1.5 ${className}`}>
      {options.map(({ value, label: text }) => {
        const on = chosen.includes(value);
        return (
          <button
            key={value}
            type="button"
            aria-pressed={on}
            onClick={() => {
              onChange(on ? chosen.filter((c) => c !== value) : [...chosen, value]);
            }}
            className={`px-2.5 py-1 text-xs font-semibold ring-1 ring-inset transition ${
              on
                ? 'bg-slate-600 text-white ring-accent'
                : 'bg-slate-800/80 text-slate-300 ring-slate-600/60 hover:text-white'
            }`}
          >
            {text}
          </button>
        );
      })}
    </div>
  );
}
