'use client';

import { BellAlertIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { ActionIcon, Box, Button, Group, Paper, Text, ThemeIcon, Transition } from '@mantine/core';
import { useEffect, useState } from 'react';

import { usePushNotifications } from '@/hooks/usePushNotifications';

const STORAGE_KEY = 'hc_push_prompt_dismissed_at';
const DISMISS_DAYS = 7;
const APPEAR_DELAY_MS = 2500;

export default function PushOptInBanner() {
  const { isSupported, isConfigured, permission, isSubscribed, loading, subscribe } =
    usePushNotifications();
  const [dismissed, setDismissed] = useState(false);
  const [shown, setShown] = useState(false);

  useEffect(() => {
    // 'default' means the browser has never been asked. Once it has, the
    // answer stands until the visitor changes it in site settings.
    if (!isSupported || !isConfigured || isSubscribed || permission !== 'default') return;

    try {
      const dismissedAt = Number(localStorage.getItem(STORAGE_KEY));
      if (dismissedAt && (Date.now() - dismissedAt) / 86_400_000 < DISMISS_DAYS) return;
    } catch {
      // Private mode blocks localStorage; showing the prompt is the safe default.
    }

    const timer = setTimeout(() => setShown(true), APPEAR_DELAY_MS);
    return () => clearTimeout(timer);
  }, [isSupported, isConfigured, isSubscribed, permission]);

  const handleDismiss = () => {
    setDismissed(true);
    try {
      localStorage.setItem(STORAGE_KEY, String(Date.now()));
    } catch {
      // Nothing to persist to — the prompt reappears next visit.
    }
  };

  const handleSubscribe = async () => {
    if (await subscribe()) setDismissed(true);
  };

  if (!isSupported || !isConfigured || isSubscribed || permission !== 'default') return null;

  return (
    <Transition mounted={shown && !dismissed} transition="slide-up" duration={350} timingFunction="ease">
      {(styles) => (
        <Box
          style={{
            ...styles,
            position: 'fixed',
            bottom: 24,
            right: 24,
            zIndex: 999,
            maxWidth: 420,
            width: 'calc(100vw - 48px)',
          }}
        >
          <Paper
            withBorder
            p="md"
            radius="lg"
            bg="white"
            shadow="xl"
            style={{ borderColor: 'var(--mantine-color-brand-3)' }}
          >
            <Group align="flex-start" justify="space-between" wrap="nowrap" mb="xs">
              <Group gap="xs" wrap="nowrap" align="center">
                <ThemeIcon size="lg" radius="md" color="brand">
                  <BellAlertIcon width={20} height={20} />
                </ThemeIcon>
                <Box>
                  <Text fw={700} size="sm" c="navy.8">
                    Handloom Drop Alerts
                  </Text>
                  <Text size="11px" c="dimmed">
                    Browser notifications
                  </Text>
                </Box>
              </Group>
              <ActionIcon
                variant="subtle"
                color="gray"
                size="sm"
                onClick={handleDismiss}
                aria-label="Dismiss notification prompt"
              >
                <XMarkIcon width={16} height={16} />
              </ActionIcon>
            </Group>

            <Text size="xs" c="gray.7" mb="md" lh={1.5}>
              Never miss a genuine handloom release. Turn on notifications for new weaver
              collections, festive drops, and private offers.
            </Text>

            <Group gap="xs" justify="flex-end">
              <Button variant="subtle" color="gray" size="xs" onClick={handleDismiss}>
                Not Now
              </Button>
              <Button
                color="brand"
                size="xs"
                loading={loading}
                onClick={handleSubscribe}
                leftSection={<BellAlertIcon width={14} height={14} />}
              >
                Notify Me
              </Button>
            </Group>
          </Paper>
        </Box>
      )}
    </Transition>
  );
}
