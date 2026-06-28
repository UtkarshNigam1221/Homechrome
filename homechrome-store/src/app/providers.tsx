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

// Mantine base layers (order matters: reset → variables → globals)
import '@mantine/core/styles/baseline.css';
import '@mantine/core/styles/default-css-variables.css';
import '@mantine/core/styles/global.css';
// Per-component styles — only the components used across src/ (+ their deps).
// MAINTENANCE: add the matching import here when introducing a new @mantine/core component.
import '@mantine/core/styles/UnstyledButton.css';
import '@mantine/core/styles/VisuallyHidden.css';
import '@mantine/core/styles/Paper.css';
import '@mantine/core/styles/Input.css';
import '@mantine/core/styles/CloseButton.css';
import '@mantine/core/styles/ModalBase.css';
import '@mantine/core/styles/Popover.css';
import '@mantine/core/styles/Loader.css';
import '@mantine/core/styles/Overlay.css';
import '@mantine/core/styles/ActionIcon.css';
import '@mantine/core/styles/Affix.css';
import '@mantine/core/styles/Alert.css';
import '@mantine/core/styles/Anchor.css';
import '@mantine/core/styles/AspectRatio.css';
import '@mantine/core/styles/Avatar.css';
import '@mantine/core/styles/Badge.css';
import '@mantine/core/styles/Breadcrumbs.css';
import '@mantine/core/styles/Button.css';
import '@mantine/core/styles/Card.css';
import '@mantine/core/styles/Center.css';
import '@mantine/core/styles/Checkbox.css';
import '@mantine/core/styles/Container.css';
import '@mantine/core/styles/Divider.css';
import '@mantine/core/styles/Drawer.css';
import '@mantine/core/styles/Flex.css';
import '@mantine/core/styles/Grid.css';
import '@mantine/core/styles/Group.css';
import '@mantine/core/styles/Image.css';
import '@mantine/core/styles/Indicator.css';
import '@mantine/core/styles/Kbd.css';
import '@mantine/core/styles/NavLink.css';
import '@mantine/core/styles/NumberInput.css';
import '@mantine/core/styles/PinInput.css';
import '@mantine/core/styles/Radio.css';
import '@mantine/core/styles/ScrollArea.css';
import '@mantine/core/styles/SimpleGrid.css';
import '@mantine/core/styles/Skeleton.css';
import '@mantine/core/styles/Stack.css';
import '@mantine/core/styles/Stepper.css';
import '@mantine/core/styles/Text.css';
import '@mantine/core/styles/ThemeIcon.css';
import '@mantine/core/styles/Title.css';
import '@mantine/core/styles/Tooltip.css';
import '@mantine/notifications/styles.css';
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
