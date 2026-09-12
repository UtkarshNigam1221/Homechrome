import { networkInterfaces } from 'node:os';

import type { NextConfig } from 'next';

const apiBase = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

// Phones and tablets on the LAN reach the dev server by IP, and Next blocks
// cross-origin /_next/webpack-hmr by default — so they silently keep serving
// the stale bundle. Read the machine's own addresses rather than pinning one
// that changes with the network. Dev-only; the deployed build ignores it.
const lanOrigins = Object.values(networkInterfaces())
  .flat()
  .filter((n) => n && n.family === 'IPv4' && !n.internal)
  .map((n) => n!.address);

const nextConfig: NextConfig = {
  output: 'standalone',
  allowedDevOrigins: lanOrigins,
  async rewrites() {
    return [{ source: '/api/:path*', destination: `${apiBase}/api/:path*` }];
  },
};

export default nextConfig;
