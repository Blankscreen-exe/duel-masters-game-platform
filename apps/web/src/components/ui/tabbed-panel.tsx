'use client';

import { useState, type ReactNode } from 'react';
import type { Civilization } from '@/lib/civilizations';
import { Panel } from './panel';
import { TabList } from './tabs';

export interface PanelTab<Id extends string> {
  readonly id: Id;
  readonly label: ReactNode;
  readonly content: ReactNode;
  /** Beside the tabs while this tab is shown, such as a button for its content. */
  readonly actions?: ReactNode;
}

interface TabbedPanelProps<Id extends string> {
  /** The tab list's accessible name. */
  readonly label: string;
  /** Each tab's panel has the id `${idPrefix}-${tab id}`. */
  readonly idPrefix: string;
  readonly tabs: readonly PanelTab<Id>[];
  readonly accent?: Civilization;
  readonly className?: string;
}

/**
 * A panel showing one of several tabs at a time, the first to begin with. Every tab's content stays
 * mounted, hidden while another is shown, so what was typed or picked in it, and what it keeps up
 * to date, carries on. The content fills the panel's height and scrolls when it is longer.
 */
export function TabbedPanel<Id extends string>({
  label,
  idPrefix,
  tabs,
  accent,
  className = '',
}: TabbedPanelProps<Id>) {
  const [selected, setSelected] = useState<Id | undefined>(tabs[0]?.id);
  const shown = tabs.find((tab) => tab.id === selected);

  return (
    <Panel accent={accent} className={`flex flex-col ${className}`}>
      <div className="m-3 mb-0 flex items-center gap-2">
        {selected !== undefined && (
          <TabList
            label={label}
            panelIdPrefix={idPrefix}
            value={selected}
            onChange={setSelected}
            className="flex-1"
            tabs={tabs}
          />
        )}
        {shown?.actions}
      </div>
      {tabs.map((tab) => (
        <div
          key={tab.id}
          id={`${idPrefix}-${tab.id}`}
          role="tabpanel"
          aria-labelledby={`${idPrefix}-tab-${tab.id}`}
          hidden={tab.id !== selected}
          className={tab.id === selected ? 'flex min-h-0 flex-1 flex-col overflow-y-auto' : ''}
        >
          {tab.content}
        </div>
      ))}
    </Panel>
  );
}
