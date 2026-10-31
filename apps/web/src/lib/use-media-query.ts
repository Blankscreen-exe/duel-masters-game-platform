'use client';

import { useSyncExternalStore } from 'react';

/** Whether the screen matches a media query, kept up to date; false when rendered on the server. */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (changed) => {
      const list = window.matchMedia(query);
      list.addEventListener('change', changed);
      return () => {
        list.removeEventListener('change', changed);
      };
    },
    () => window.matchMedia(query).matches,
    () => false,
  );
}
