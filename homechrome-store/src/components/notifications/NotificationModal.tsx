'use client';

import dynamic from 'next/dynamic';
import { useState } from 'react';

import { useUIStore } from '@/stores/uiStore';

const NotificationSettings = dynamic(() => import('./NotificationSettings'), { ssr: false });

// Mounted once from the layout. The panel, its chunk and its service-worker
// lookups load on the first open rather than on every page.
export default function NotificationModal() {
  const opened = useUIStore((s) => s.notificationsOpen);
  const close = useUIStore((s) => s.closeNotifications);
  // Latched during render, the sanctioned way to remember a past prop value.
  const [everOpened, setEverOpened] = useState(false);
  if (opened && !everOpened) setEverOpened(true);

  if (!everOpened && !opened) return null;
  return <NotificationSettings opened={opened} onClose={close} />;
}
