/**
 * Where to go after logging in or signing up: the page the player came from, given as ?next=, if
 * it is a page of this site, and the home page otherwise. Anything that could lead to another site
 * ("//evil.example", "https://…", "/\evil.example") is ignored.
 */
export function safeRedirect(next: string | string[] | undefined): string {
  const path = Array.isArray(next) ? next[0] : next;

  if (path?.startsWith('/') !== true || path.startsWith('//') || path.includes('\\')) {
    return '/';
  }

  // Resolved against a placeholder origin: a path that leaves it is not a page of this site.
  const resolved = new URL(path, 'https://site.invalid');
  if (resolved.origin !== 'https://site.invalid') return '/';

  return `${resolved.pathname}${resolved.search}${resolved.hash}`;
}
