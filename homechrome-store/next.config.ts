import type { NextConfig } from 'next';

const apiBase = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

const isDev = process.env.NODE_ENV === 'development';

const nextConfig: NextConfig = {
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: '*.s3.amazonaws.com' },
      { protocol: 'http', hostname: 'localhost', port: '4566' },
    ],
    // Next.js blocks images resolving to private IPs — skip optimization locally
    unoptimized: isDev,
  },
  async rewrites() {
    return [
      { source: '/api/:path*', destination: `${apiBase}/api/:path*` },
    ];
  },
};

export default nextConfig;
