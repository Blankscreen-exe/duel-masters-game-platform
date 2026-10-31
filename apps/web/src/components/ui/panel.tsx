import type { ComponentPropsWithoutRef, ReactNode } from 'react';
import { civilizationColour, type Civilization } from '@/lib/civilizations';

interface PanelProps extends Omit<ComponentPropsWithoutRef<'section'>, 'title'> {
  /** Shown in the panel's header bar. */
  readonly title?: ReactNode;
  /** Marks the top edge with a civilization's colour. */
  readonly accent?: Civilization;
  /** Next to the title, such as a count or a link. */
  readonly actions?: ReactNode;
  /** Added to the header bar's classes, such as to keep it in view while scrolling. */
  readonly headerClassName?: string;
}

/** A Battle Zone panel: dark, with angled corners and a thin light edge. */
export function Panel({
  title,
  accent,
  actions,
  headerClassName = '',
  className = '',
  children,
  ...props
}: PanelProps) {
  return (
    <section
      className={`cut-corners bg-panel/95 shadow-[0_10px_30px_rgb(0_0_0/0.5)] ring-1 ring-slate-500/40 ring-inset ${accent ? `border-t-2 ${civilizationColour(accent).border}` : ''} ${className}`}
      {...props}
    >
      {(title !== undefined || actions !== undefined) && (
        <header
          className={`flex items-center justify-between gap-3 border-b border-slate-600/40 bg-linear-to-r from-white/[0.07] to-transparent px-4 py-2.5 ${headerClassName}`}
        >
          {title !== undefined && (
            <h2 className="font-display text-base font-bold tracking-[0.15em] text-slate-100 uppercase">
              {title}
            </h2>
          )}
          {actions}
        </header>
      )}
      {children}
    </section>
  );
}
