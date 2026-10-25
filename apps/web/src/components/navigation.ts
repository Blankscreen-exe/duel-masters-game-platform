/** A link in the site's menus. */
export interface NavLink {
  readonly href: string;
  readonly label: string;
}

/**
 * The main menu, on the header (desktop) and the bottom tab bar (phones). Pages join it as they
 * are built, so no link leads nowhere.
 */
export const MAIN_LINKS: readonly NavLink[] = [
  { href: '/', label: 'Home' },
  { href: '/play', label: 'Play' },
  { href: '/decks', label: 'Decks' },
  { href: '/cards', label: 'Cards' },
];

/** Whether a menu link belongs to the current page: its own path or one below it. */
export function isCurrent(link: NavLink, pathname: string): boolean {
  return pathname === link.href || pathname.startsWith(`${link.href}/`);
}
