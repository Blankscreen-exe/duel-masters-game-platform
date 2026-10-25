import type { NextConfig } from 'next';
import { goServerUrl } from './src/lib/go-server-url';

// Checked when the site starts, so a wrong address fails at once.
const goServer = goServerUrl();

const nextConfig: NextConfig = {
  poweredByHeader: false,
  rewrites() {
    // Every /api request goes to the Go server, so the browser sees one site with one login
    // cookie (design/SPEC.md, section 1).
    return Promise.resolve([{ source: '/api/:path*', destination: `${goServer}/api/:path*` }]);
  },
};

export default nextConfig;
