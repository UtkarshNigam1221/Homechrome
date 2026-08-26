import react from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig, loadEnv } from 'vite';

// Which vendor chunk each package belongs to, keyed by exact package name. Adding
// a package here is the only way it leaves the default chunk.
const VENDOR_CHUNKS: Record<string, string> = {
  react: 'vendor-react',
  'react-dom': 'vendor-react',
  'react-router': 'vendor-react',
  'react-router-dom': 'vendor-react',
  zustand: 'vendor-state',
  '@tanstack/react-query': 'vendor-state',
  '@headlessui/react': 'vendor-ui',
  'lucide-react': 'vendor-ui',
  clsx: 'vendor-ui',
  recharts: 'vendor-charts',
  'react-hook-form': 'vendor-forms',
  '@hookform/resolvers': 'vendor-forms',
  zod: 'vendor-forms',
  axios: 'vendor-utils',
  'date-fns': 'vendor-utils',
};

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');

  // Keep browser console.* in non-prod builds so the AWS dev deploy is
  // debuggable from devtools. `build:dev` / `build:prod` map to mode
  // 'dev' / 'prod'; the default `build` (no mode) lands as 'production'.
  // No source maps in any build: they were 80% of the dev upload and starved
  // the BucketDeployment Lambda's sync. Vite defaults sourcemap to false.
  const stripConsole = mode === 'prod' || mode === 'production';

  return {
    plugins: [react()],

    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },

    build: {
      // Enable minification
      minify: 'terser',
      terserOptions: {
        compress: {
          drop_console: stripConsole,
          drop_debugger: true,
        },
      },

      // Optimize chunk size
      chunkSizeWarningLimit: 500,

      // Code splitting configuration
      rollupOptions: {
        output: {
          // Manual chunk splitting for better caching.
          // Function form required since Rollup v4 (object form removed from TS types).
          //
          // Matched on the owning package name, never on a substring of the path.
          // `id.includes('lucide-react')` also matched
          // @neondatabase/auth-ui/node_modules/lucide-react, so a nested duplicate
          // joined the same chunk as the real one and the two collided on their
          // shared bindings. A nested copy now falls through to the default chunk,
          // which keeps every vendor chunk holding exactly one copy of a package.
          manualChunks(id) {
            const segments = id.replace(/\\/g, '/').split('node_modules/');
            // 1 = app code; >2 = a nested duplicate of something already resolved.
            if (segments.length !== 2) return undefined;

            const rest = segments[1];
            const pkg = rest.startsWith('@')
              ? rest.split('/').slice(0, 2).join('/')
              : rest.split('/')[0];

            return VENDOR_CHUNKS[pkg];
          },

          // Optimize asset file names for caching
          assetFileNames: (assetInfo) => {
            const info = assetInfo.name?.split('.') || [];
            const ext = info[info.length - 1];
            if (/png|jpe?g|svg|gif|tiff|bmp|ico/i.test(ext)) {
              return `assets/images/[name]-[hash][extname]`;
            }
            if (/woff|woff2|eot|ttf|otf/i.test(ext)) {
              return `assets/fonts/[name]-[hash][extname]`;
            }
            return `assets/[name]-[hash][extname]`;
          },

          chunkFileNames: 'assets/js/[name]-[hash].js',
          entryFileNames: 'assets/js/[name]-[hash].js',
        },
      },

      // Target modern browsers for smaller bundles
      target: 'es2020',
    },

    // Optimize dependencies
    optimizeDeps: {
      include: [
        'react',
        'react-dom',
        'react-router-dom',
        '@tanstack/react-query',
        'zustand',
        'axios',
        'clsx',
        'lucide-react',
      ],
    },

    // Development server config
    server: {
      port: 5173,
      open: true,
      proxy: {
        '/admin': {
          target: env.VITE_API_URL || 'http://localhost:8081',
          changeOrigin: true,
          secure: false,
          cookieDomainRewrite: { '*': '' },
          configure: (proxy) => {
            // Strip Secure flag + fix SameSite so cookies work on http://localhost
            proxy.on('proxyRes', (proxyRes) => {
              const sc = proxyRes.headers['set-cookie'];
              if (sc) {
                proxyRes.headers['set-cookie'] = sc.map((c) =>
                  c.replace(/;\s*Secure/gi, '').replace(/;\s*SameSite=None/gi, '; SameSite=Lax')
                );
              }
            });
          },
        },
        '/api': {
          target: env.VITE_API_URL || 'http://localhost:8081',
          changeOrigin: true,
          secure: false,
        },
      },
    },
  };
});
