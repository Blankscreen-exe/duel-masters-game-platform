/**
 * The Go server's address, from GO_SERVER_URL. Next.js forwards the browser's /api requests there,
 * and pages rendered on the server call it directly.
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
