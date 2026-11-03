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
};

export default nextConfig;
