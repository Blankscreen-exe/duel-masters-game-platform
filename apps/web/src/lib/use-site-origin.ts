import { useSyncExternalStore } from 'react';

function stopListening(): void {
  // The page's address never changes while it is open, so there was nothing to listen to.
}

function subscribe(): () => void {
  return stopListening;
}

/**
 * This site's address, such as https://duelmasters.example, for links to share. Empty while the
 * page is drawn on the server, which cannot know the address the visitor used.
 */
export function useSiteOrigin(): string {
  return useSyncExternalStore(
    subscribe,
    () => window.location.origin,
    () => '',
  );
}
