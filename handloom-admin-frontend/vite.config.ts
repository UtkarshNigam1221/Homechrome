import react from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig, loadEnv } from 'vite';

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');

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
          drop_console: true, // Remove console.log in production
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
          manualChunks(id) {
            if (!id.includes('node_modules')) return undefined;
            if (/[\\/]node_modules[\\/](react|react-dom|react-router|react-router-dom)[\\/]/.test(id))
              return 'vendor-react';
            if (id.includes('zustand') || id.includes('@tanstack/react-query'))
              return 'vendor-state';
            if (id.includes('@headlessui/react') || id.includes('lucide-react') || id.includes('clsx'))
              return 'vendor-ui';
            if (id.includes('recharts')) return 'vendor-charts';
            if (id.includes('react-hook-form') || id.includes('@hookform/resolvers') || id.includes('zod'))
              return 'vendor-forms';
            if (id.includes('axios') || id.includes('date-fns')) return 'vendor-utils';
            return undefined;
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

      // Generate source maps for debugging (can disable in production if needed)
      sourcemap: false,

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
