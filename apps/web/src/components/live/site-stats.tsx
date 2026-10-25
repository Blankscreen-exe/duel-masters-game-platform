'use client';

import { useState } from 'react';
import type { components } from '@/lib/api/schema';
import { useLiveMessage } from './live-provider';

type SiteStats = components['schemas']['SiteStats'];

/** How many players are online and games running, kept up to date over the live connection. */
export function SiteStatsLine({ initial }: { readonly initial: SiteStats }) {
  const [stats, setStats] = useState(initial);

  useLiveMessage('site_stats', (message) => {
    setStats(message.stats);
  });

  return (
    <dl className="flex flex-wrap gap-x-8 gap-y-2 font-display tracking-wide">
      <Stat label="Players online" value={stats.playersOnline} />
      <Stat label="Games running" value={stats.gamesRunning} />
    </dl>
  );
}

function Stat({ label, value }: { readonly label: string; readonly value: number }) {
  return (
    <div className="flex items-baseline gap-2">
      <dt className="order-2 text-sm text-slate-400 uppercase">{label}</dt>
      <dd className="order-1 text-2xl font-bold text-white tabular-nums">{value}</dd>
    </div>
  );
}
