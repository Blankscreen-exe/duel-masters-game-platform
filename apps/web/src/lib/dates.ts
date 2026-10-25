/**
 * Dates and times as the site shows them: in English and in UTC, so a page drawn on the server
 * reads the same once it runs in the browser, wherever the player is.
 */

const DAY = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  timeZone: 'UTC',
});

const DAY_AND_TIME = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  timeZone: 'UTC',
});

/** A day, such as 25 Oct 2026. */
export function dayText(iso: string): string {
  return DAY.format(new Date(iso));
}

/** A moment, such as 2 Nov 2026, 12:00 UTC. */
export function dateTimeText(iso: string): string {
  return `${DAY_AND_TIME.format(new Date(iso))} UTC`;
}
