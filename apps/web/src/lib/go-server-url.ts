/**
 * The Go server's address, from GO_SERVER_URL, read while the site runs (not when it is built), so
 * one build can be pointed at any server. The browser's /api requests are forwarded there (see
 * src/proxy.ts), and pages rendered on the server call it directly.
 */
export function goServerUrl(
  env: Readonly<Record<string, string | undefined>> = process.env,
): string {
  const value = env['GO_SERVER_URL'] ?? 'http://127.0.0.1:8080';
  const url = URL.parse(value);

  if (url === null || (url.protocol !== 'http:' && url.protocol !== 'https:')) {
    throw new Error(`GO_SERVER_URL must be a URL such as http://127.0.0.1:8080, got "${value}"`);
  }

  return value.replace(/\/+$/, '');
}

/** Where a request to this site's /api goes on the Go server: the same path and query. */
export function goServerTarget(path: string, search: string, server: string): URL {
  return new URL(`${server}${path}${search}`);
}
