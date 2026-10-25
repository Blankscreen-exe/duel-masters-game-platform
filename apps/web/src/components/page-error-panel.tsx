'use client';

import type { ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';

/** What a failed page shows, most often because the game server cannot be reached. */
export function PageErrorPanel({
  reset,
  children,
}: {
  readonly reset: () => void;
  /** More ways out, beside trying again. */
  readonly children?: ReactNode;
}) {
  return (
    <Panel accent="fire" className="mx-auto max-w-md p-6 text-center">
      <h1 className="font-display text-3xl font-bold tracking-widest text-white uppercase">
        Something went wrong
      </h1>
      <p className="mt-3 text-slate-400">
        The page could not be loaded. The game server may be restarting; try again in a moment.
      </p>
      <div className="mt-6 flex justify-center gap-2">
        <Button variant="secondary" onClick={reset}>
          Try again
        </Button>
        {children}
      </div>
    </Panel>
  );
}
