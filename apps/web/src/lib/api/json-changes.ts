import type { Middleware } from 'openapi-fetch';

/** Methods that only read, which the Go server accepts in any form. */
const READS = new Set(['GET', 'HEAD', 'OPTIONS']);

/**
 * Marks every request that changes something as JSON, even one without a body such as a delete or
 * logging out. The Go server refuses changes not sent as JSON, which other sites cannot send
 * without permission (api/openapi.yaml), and the API client only marks requests that have a body.
 */
export const jsonChanges: Middleware = {
  onRequest({ request }) {
    if (READS.has(request.method) || request.headers.has('Content-Type')) return undefined;

    const marked = new Request(request);
    marked.headers.set('Content-Type', 'application/json');
    return marked;
  },
};
