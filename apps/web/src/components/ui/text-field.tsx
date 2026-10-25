import { useId, type ComponentPropsWithoutRef } from 'react';
import { FIELD_CONTROL, FIELD_HINT, FIELD_LABEL } from './field';

interface TextFieldProps extends ComponentPropsWithoutRef<'input'> {
  readonly label: string;
  readonly hint?: string;
}

export function TextField({ label, hint, className = '', ...inputProps }: TextFieldProps) {
  const id = useId();
  const hintId = `${id}-hint`;

  return (
    <div className={className}>
      <label htmlFor={id} className={FIELD_LABEL}>
        {label}
      </label>
      <input
        id={id}
        aria-describedby={hint ? hintId : undefined}
        className={FIELD_CONTROL}
        {...inputProps}
      />
      {hint && (
        <p id={hintId} className={FIELD_HINT}>
          {hint}
        </p>
      )}
    </div>
  );
}
