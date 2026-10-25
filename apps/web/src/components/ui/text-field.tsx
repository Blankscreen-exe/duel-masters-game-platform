import { useId, type ComponentPropsWithoutRef } from 'react';

interface TextFieldProps extends ComponentPropsWithoutRef<'input'> {
  readonly label: string;
  readonly hint?: string;
}

export function TextField({ label, hint, className = '', ...inputProps }: TextFieldProps) {
  const id = useId();
  const hintId = `${id}-hint`;

  return (
    <div className={className}>
      <label
        htmlFor={id}
        className="block text-[11px] font-semibold tracking-wider text-slate-400 uppercase"
      >
        {label}
      </label>
      <input
        id={id}
        aria-describedby={hint ? hintId : undefined}
        className="mt-1 block w-full bg-field px-3 py-2 text-sm text-slate-100 ring-1 ring-slate-600/60 ring-inset placeholder:text-slate-500 hover:ring-slate-400/70 focus:ring-accent focus:outline-none"
        {...inputProps}
      />
      {hint && (
        <p id={hintId} className="mt-1 text-xs text-slate-500">
          {hint}
        </p>
      )}
    </div>
  );
}
