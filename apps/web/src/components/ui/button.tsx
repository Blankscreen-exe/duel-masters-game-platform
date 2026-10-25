import type { ComponentPropsWithoutRef } from 'react';

export type ButtonVariant = 'primary' | 'secondary' | 'quiet' | 'danger';

const BASE =
  'inline-flex items-center justify-center gap-2 font-display font-bold uppercase tracking-wider transition disabled:cursor-not-allowed disabled:opacity-60';

const VARIANTS: Record<ButtonVariant, string> = {
  primary:
    'cut-corners-sm bg-linear-to-b from-slate-100 via-slate-300 to-slate-500 px-5 py-2 text-slate-900 hover:from-white hover:shadow-[0_0_18px_rgb(56_189_248/0.35)]',
  secondary:
    'cut-corners-sm bg-slate-800/80 px-4 py-2 text-slate-200 ring-1 ring-slate-500/50 ring-inset hover:bg-slate-700 hover:text-white',
  quiet: 'px-2 py-1 text-sm tracking-wide text-slate-400 hover:text-white',
  danger:
    'cut-corners-sm bg-red-900/70 px-4 py-2 text-red-100 ring-1 ring-red-400/50 ring-inset hover:bg-red-800',
};

/** Button styles, also used for links that look like buttons. */
export function buttonClasses(variant: ButtonVariant = 'primary', className = ''): string {
  return `${BASE} ${VARIANTS[variant]} ${className}`;
}

interface ButtonProps extends ComponentPropsWithoutRef<'button'> {
  readonly variant?: ButtonVariant;
}

export function Button({ variant = 'primary', className, type = 'button', ...props }: ButtonProps) {
  return <button type={type} className={buttonClasses(variant, className)} {...props} />;
}
