'use client';

import { Button } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';

/** Shown when a page fails, most often because the game server cannot be reached. */
export default function PageError({
  reset,
}: {
  readonly error: Error;
  readonly reset: () => void;
}) {
  return (
    <Panel accent="fire" className="mx-auto max-w-md p-6 text-center">
      <h1 className="font-display text-3xl font-bold tracking-widest text-white uppercase">
        Something went wrong
      </h1>
      <p className="mt-3 text-slate-400">
        The page could not be loaded. The game server may be restarting; try again in a moment.
      </p>
      <Button variant="secondary" className="mt-6" onClick={reset}>
        Try again
      </Button>
    </Panel>
  );
}
