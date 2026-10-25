'use client';

import { PageErrorPanel } from '@/components/page-error-panel';

export default function PageError({
  reset,
}: {
  readonly error: Error;
  readonly reset: () => void;
}) {
  return <PageErrorPanel reset={reset} />;
}
