'use client';

import Link from 'next/link';
import { PageErrorPanel } from '@/components/page-error-panel';
import { buttonClasses } from '@/components/ui/button';

export default function GameError({
  reset,
}: {
  readonly error: Error;
  readonly reset: () => void;
}) {
  return (
    <div className="flex flex-1 items-center justify-center p-4">
      <PageErrorPanel reset={reset}>
        <Link href="/play" className={buttonClasses('quiet')}>
          Back to the lobby
        </Link>
      </PageErrorPanel>
    </div>
  );
}
