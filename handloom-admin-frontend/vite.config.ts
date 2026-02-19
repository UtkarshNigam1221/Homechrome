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
          // Manual chunk splitting for better caching
          manualChunks: {
            // React core - rarely changes
            'vendor-react': ['react', 'react-dom', 'react-router-dom'],

            // State management and data fetching
            'vendor-state': ['zustand', '@tanstack/react-query'],

            // UI libraries
            'vendor-ui': ['@headlessui/react', 'lucide-react', 'clsx'],

            // Charts - large bundle, only needed on dashboard/analytics
            'vendor-charts': ['recharts'],

            // Form handling
            'vendor-forms': ['react-hook-form', '@hookform/resolvers', 'zod'],

            // Utilities
            'vendor-utils': ['axios', 'date-fns'],
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
