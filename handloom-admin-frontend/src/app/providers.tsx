import { QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { Toaster } from 'react-hot-toast';

import { queryClient } from '@/app/queryClient';
import { ErrorBoundary } from '@/shared/components/ErrorBoundary';

export function Providers({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <ErrorBoundary>{children}</ErrorBoundary>
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          className: 'bg-white text-gray-800 shadow-lg rounded-lg px-4 py-3',
          success: { iconTheme: { primary: '#10b981', secondary: '#fff' } },
          error: { iconTheme: { primary: '#ef4444', secondary: '#fff' } },
        }}
      />
    </QueryClientProvider>
  );
}
