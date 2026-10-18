'use client';

import { BellAlertIcon } from '@heroicons/react/24/outline';
import { Button, Group, Modal, Stack, Text, ThemeIcon } from '@mantine/core';
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
    <Modal
      opened={shown && !dismissed}
      onClose={handleDismiss}
      centered
      withCloseButton={false}
      radius="lg"
      size={380}
      padding="xl"
      aria-label="Turn on handloom drop alerts"
      // Dimmed and blurred so the panel is the only thing in focus; Mantine
      // traps focus and closes on Escape or an outside click.
      overlayProps={{ backgroundOpacity: 0.6, blur: 4 }}
      transitionProps={{ transition: 'pop', duration: 260, timingFunction: 'ease-out' }}
    >

      <Stack align="center" gap="sm">
        <span className="hc-push-bell">
          <ThemeIcon size={60} radius="xl" color="brand">
            <BellAlertIcon width={30} height={30} />
          </ThemeIcon>
        </span>

        <Text fw={700} size="lg" c="navy.8" ta="center">
          Handloom drop alerts
        </Text>

        <Text size="sm" c="gray.7" ta="center" lh={1.55}>
          Never miss a genuine handloom release. Turn on notifications for new weaver collections,
          festive drops, and private offers.
        </Text>

        <Group gap="sm" justify="center" mt="xs" w="100%">
          <Button variant="subtle" color="gray" onClick={handleDismiss}>
            Not now
          </Button>
          <Button
            color="brand"
            loading={loading}
            onClick={handleSubscribe}
            leftSection={<BellAlertIcon width={16} height={16} />}
          >
            Notify me
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
