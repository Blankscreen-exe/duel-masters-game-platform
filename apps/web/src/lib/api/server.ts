import 'server-only';
import { cookies } from 'next/headers';
import createClient from 'openapi-fetch';
import { goServerUrl } from '../go-server-url';
import type { paths } from './schema';

/**
 * The API from pages rendered on the server. It calls the Go server directly and passes on the
 * visitor's cookies, so the Go server sees the same session as it would from the browser.
 */
export async function serverApi() {
  const cookie = (await cookies()).toString();

  return createClient<paths>({
    baseUrl: `${goServerUrl()}/api`,
    headers: cookie === '' ? {} : { cookie },
    // Answers depend on the visitor and change over time; nothing is cached between requests.
    cache: 'no-store',
  });
}
