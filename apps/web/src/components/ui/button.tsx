import type { ComponentPropsWithoutRef } from 'react';

export type ButtonVariant = 'primary' | 'secondary' | 'quiet' | 'danger';

const VARIANTS: Record<ButtonVariant, string> = {
  primary: 'bz-button-primary px-5 py-2',
  secondary: 'bz-button-secondary px-4 py-2',
  quiet: 'px-2 py-1 text-sm tracking-wide text-slate-400 hover:text-white',
  danger: 'bz-button-danger px-4 py-2',
};

/** Button styles, also used for links that look like buttons. */
export function buttonClasses(variant: ButtonVariant = 'primary', className = ''): string {
  return `bz-button ${VARIANTS[variant]} ${className}`;
}

interface ButtonProps extends ComponentPropsWithoutRef<'button'> {
  readonly variant?: ButtonVariant;
}

export function Button({ variant = 'primary', className, type = 'button', ...props }: ButtonProps) {
  return <button type={type} className={buttonClasses(variant, className)} {...props} />;
}
