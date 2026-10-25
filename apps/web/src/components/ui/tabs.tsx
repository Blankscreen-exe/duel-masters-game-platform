'use client';

import type { ReactNode } from 'react';

export interface Tab<Id extends string> {
  readonly id: Id;
  readonly label: ReactNode;
}

interface TabListProps<Id extends string> {
  /** The tab list's accessible name. */
  readonly label: string;
  readonly tabs: readonly Tab<Id>[];
  readonly value: Id;
  readonly onChange: (id: Id) => void;
  /** Each tab controls the element with the id `${panelIdPrefix}-${tab id}`. */
  readonly panelIdPrefix: string;
  readonly className?: string;
}

/**
 * A row of tabs, such as to show one panel at a time on phones. The panels are the caller's: give
 * each the id `${panelIdPrefix}-${tab id}` and hide the ones not selected.
 */
export function TabList<Id extends string>({
  label,
  tabs,
  value,
  onChange,
  panelIdPrefix,
  className = '',
}: TabListProps<Id>) {
  const tabId = (id: Id) => `${panelIdPrefix}-tab-${id}`;
  const select = (offset: number) => {
    const index = tabs.findIndex((tab) => tab.id === value);
    const next = tabs[(index + offset + tabs.length) % tabs.length];
    if (!next) return;
    onChange(next.id);
    document.getElementById(tabId(next.id))?.focus();
  };

  return (
    <div
      role="tablist"
      aria-label={label}
      className={`grid auto-cols-fr grid-flow-col gap-1 bg-field/80 p-1 ring-1 ring-slate-600/50 ring-inset ${className}`}
      onKeyDown={(event) => {
        // Arrow keys move between tabs, as in any tab list.
        if (event.key === 'ArrowRight') select(1);
        else if (event.key === 'ArrowLeft') select(-1);
      }}
    >
      {tabs.map(({ id, label: tabLabel }) => {
        const selected = value === id;
        return (
          <button
            key={id}
            id={tabId(id)}
            type="button"
            role="tab"
            aria-selected={selected}
            aria-controls={`${panelIdPrefix}-${id}`}
            tabIndex={selected ? 0 : -1}
            onClick={() => {
              onChange(id);
            }}
            className={`flex min-h-10 items-center justify-center gap-2 px-3 py-2 font-display text-sm font-semibold tracking-wider uppercase transition ${
              selected
                ? 'bg-linear-to-b from-slate-100 to-slate-400 text-slate-900'
                : 'text-slate-300 hover:bg-white/10 hover:text-white'
            }`}
          >
            {tabLabel}
          </button>
        );
      })}
    </div>
  );
}
