import type { NextConfig } from 'next';
import { goServerUrl } from './src/lib/go-server-url';

// Checked when the site starts, so a wrong Go server address fails at once. The /api forwarding
// itself is in src/proxy.ts.
goServerUrl();

const nextConfig: NextConfig = {
  poweredByHeader: false,
};

export default nextConfig;
