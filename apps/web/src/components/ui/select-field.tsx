import { useId, type ComponentPropsWithoutRef } from 'react';
import { FIELD_CONTROL, FIELD_HINT, FIELD_LABEL } from './field';

interface SelectFieldProps extends ComponentPropsWithoutRef<'select'> {
  readonly label: string;
  readonly hint?: string;
}

/** A labelled drop-down list, styled as TextField is; the options are its children. */
export function SelectField({
  label,
  hint,
  className = '',
  children,
  ...selectProps
}: SelectFieldProps) {
  const id = useId();
  const hintId = `${id}-hint`;

  return (
    <div className={className}>
      <label htmlFor={id} className={FIELD_LABEL}>
        {label}
      </label>
      <select
        id={id}
        aria-describedby={hint ? hintId : undefined}
        className={FIELD_CONTROL}
        {...selectProps}
      >
        {children}
      </select>
      {hint && (
        <p id={hintId} className={FIELD_HINT}>
          {hint}
        </p>
      )}
    </div>
  );
}
