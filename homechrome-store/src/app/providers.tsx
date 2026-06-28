'use client';

import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import { Notifications } from '@mantine/notifications';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import { useEffect, useState } from 'react';

import { initAnalytics, stopAnalytics } from '@/lib/analytics';
import { initRUM } from '@/lib/rum';
import { useAuthStore } from '@/stores/auth';

import { theme } from './theme';

import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import '@mantine/dates/styles.css';
import '@mantine/spotlight/styles.css';
import '@mantine/carousel/styles.css';

function AuthInit() {
  const checkAuth = useAuthStore((s) => s.checkAuth);
  useEffect(() => {
    if (document.cookie.includes('hc_session=')) {
      checkAuth();
    } else {
      useAuthStore.setState({ isAuthenticated: false, isLoading: false });
    }
  }, [checkAuth]);
  return null;
}

function AnalyticsProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    initAnalytics();
    initRUM();
    return () => stopAnalytics();
  }, []);
  return <>{children}</>;
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 5 * 60 * 1000,
            retry: (failureCount, error) => {
              if (isAxiosError(error) && error.response?.status === 429) return false;
              return failureCount < 1;
            },
          },
        },
      }),
  );

  return (
    <MantineProvider theme={theme}>
      <ModalsProvider>
        <Notifications position="top-right" />
        <QueryClientProvider client={queryClient}>
          <AuthInit />
          <AnalyticsProvider>{children}</AnalyticsProvider>
        </QueryClientProvider>
      </ModalsProvider>
    </MantineProvider>
  );
}
