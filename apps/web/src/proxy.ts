import { NextResponse, type NextRequest } from 'next/server';
import { goServerTarget, goServerUrl } from './lib/go-server-url';

/**
 * Every /api request goes to the Go server, so the browser sees one site with one login cookie
 * (design/SPEC.md, section 1). Done here rather than as a rewrite in next.config.ts because those
 * are fixed when the site is built, and the server's address is a setting of the running site.
 */
export function proxy(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  return NextResponse.rewrite(goServerTarget(pathname, search, goServerUrl()));
}

export const config = {
  matcher: '/api/:path*',
};
