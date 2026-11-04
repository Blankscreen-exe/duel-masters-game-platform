import type { NextConfig } from 'next';
import { goServerUrl } from './src/lib/go-server-url';

// Checked when the site starts, so a wrong Go server address fails at once. The /api forwarding
// itself is in src/proxy.ts.
goServerUrl();

const nextConfig: NextConfig = {
  poweredByHeader: false,
  // The dev server serves its scripts only to pages opened from localhost; this lets it serve them
  // to quick Cloudflare tunnels too (such as `kit serve --app 3000 --internet`), whose address
  // changes every run. It has no effect on a production build. The Go server's live connections
  // need the tunnel in SITE_ORIGINS as well (README.md).
  allowedDevOrigins: ['*.trycloudflare.com'],
  // The Docker image (apps/web/Dockerfile) sets NEXT_STANDALONE=1 to build a self-contained server
  // with only the files it needs, so the image carries no node_modules folder. Left off otherwise,
  // because `next start` does not run a standalone build.
  ...(process.env['NEXT_STANDALONE'] === '1' ? { output: 'standalone' as const } : {}),
};

export default nextConfig;
