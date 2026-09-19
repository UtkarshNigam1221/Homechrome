"use client";

import { BellAlertIcon, XMarkIcon } from "@heroicons/react/24/outline";
import {
  ActionIcon,
  Box,
  Group,
  Paper,
  Text,
  ThemeIcon,
  Transition,
} from "@mantine/core";
import { useEffect, useState } from "react";

const STORAGE_KEY = "hc_ios_install_hint_dismissed_at";
const DISMISS_DAYS = 14;
const APPEAR_DELAY_MS = 3500;

/** iPhone, iPod, or an iPad — which reports itself as a Mac with a touchscreen. */
function isIos(): boolean {
  if (typeof window === "undefined") return false;
  const ua = navigator.userAgent;
  return (
    /iPad|iPhone|iPod/.test(ua) ||
    (/Macintosh/.test(ua) && navigator.maxTouchPoints > 1)
  );
}

/** Already launched from the Home Screen, so push is available and this is moot. */
function isStandalone(): boolean {
  if (typeof window === "undefined") return false;
  return (
    window.matchMedia("(display-mode: standalone)").matches ||
    (window.navigator as Navigator & { standalone?: boolean }).standalone ===
      true
  );
}

/**
 * Web Push landed on iOS in 16.4. Below that, installing changes nothing, so
 * promising alerts would be a lie. An unreadable version string is treated as
 * new enough — showing the hint to a handful of old devices beats hiding it
 * from everyone whose UA we failed to parse.
 */
function iosSupportsPush(): boolean {
  const match = /OS (\d+)_(\d+)/.exec(navigator.userAgent);
  if (!match) return true;
  const major = Number(match[1]);
  const minor = Number(match[2]);
  return major > 16 || (major === 16 && minor >= 4);
}

/**
 * On iOS, Web Push only reaches a site that has been added to the Home Screen —
 * in a normal Safari tab the Notification API does not exist at all, so the
 * opt-in prompt hides itself and an iPhone visitor has no way to discover the
 * feature. This is the one place that gap gets explained.
 */
export default function IosInstallHint() {
  const [shown, setShown] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    // 'Notification' in window means push is already reachable — either this is
    // the installed app, or a browser that never needed installing.
    if (
      !isIos() ||
      isStandalone() ||
      "Notification" in window ||
      !iosSupportsPush()
    )
      return;

    try {
      const dismissedAt = Number(localStorage.getItem(STORAGE_KEY));
      if (dismissedAt && (Date.now() - dismissedAt) / 86_400_000 < DISMISS_DAYS)
        return;
    } catch {
      // Private mode blocks localStorage; showing the hint is the safe default.
    }

    const timer = setTimeout(() => setShown(true), APPEAR_DELAY_MS);
    return () => clearTimeout(timer);
  }, []);

  const handleDismiss = () => {
    setDismissed(true);
    try {
      localStorage.setItem(STORAGE_KEY, String(Date.now()));
    } catch {
      // Nothing to persist to — the hint reappears next visit.
    }
  };

  return (
    <Transition
      mounted={shown && !dismissed}
      transition="slide-up"
      duration={300}
      timingFunction="ease"
    >
      {(styles) => (
        <Box
          style={{
            ...styles,
            position: "fixed",
            // Clears the phone tab bar, which is fixed to the bottom on this
            // breakpoint, plus the home indicator on notched devices.
            bottom: "calc(64px + env(safe-area-inset-bottom, 0px) + 12px)",
            left: 12,
            right: 12,
            zIndex: 200,
          }}
        >
          <Paper withBorder p="md" radius="lg" bg="white" shadow="xl">
            <Group
              align="flex-start"
              justify="space-between"
              wrap="nowrap"
              mb={6}
            >
              <Group gap="xs" wrap="nowrap" align="center">
                <ThemeIcon size="lg" radius="md" color="brand">
                  <BellAlertIcon width={20} height={20} />
                </ThemeIcon>
                <Text fw={700} size="sm" c="navy.8">
                  Get drop alerts on this iPhone
                </Text>
              </Group>
              <ActionIcon
                variant="subtle"
                color="gray"
                size="sm"
                onClick={handleDismiss}
                aria-label="Dismiss install hint"
              >
                <XMarkIcon width={16} height={16} />
              </ActionIcon>
            </Group>

            <Text size="xs" c="gray.7" lh={1.5}>
              Apple only sends notifications to apps on your Home Screen. Open
              your browser&apos;s share menu, choose{" "}
              <strong>Add to Home Screen</strong>, then open Homechrome from the
              icon — we&apos;ll ask about alerts there.
            </Text>
          </Paper>
        </Box>
      )}
    </Transition>
  );
}
