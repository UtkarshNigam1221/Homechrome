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
import { useDelayedPrompt } from "@/hooks/useDelayedPrompt";

const STORAGE_KEY = "hc_ios_install_hint_dismissed_at";
const DISMISS_DAYS = 14;
const APPEAR_DELAY_MS = 3500;

/** iPhone, iPod, or an iPad — which reports itself as a Mac with a touchscreen. */
function isIos(): boolean {
  const ua = navigator.userAgent;
  return (
    /iPad|iPhone|iPod/.test(ua) ||
    (/Macintosh/.test(ua) && navigator.maxTouchPoints > 1)
  );
}

/** Already launched from the Home Screen, so push is available and this is moot. */
function isStandalone(): boolean {
  return (
    window.matchMedia("(display-mode: standalone)").matches ||
    (window.navigator as Navigator & { standalone?: boolean }).standalone ===
      true
  );
}

// Web Push needs iOS 16.4; an unreadable version string counts as new enough,
// since a stray hint on an old device beats hiding it from everyone.
function iosSupportsPush(): boolean {
  const match = /OS (\d+)_(\d+)/.exec(navigator.userAgent);
  if (!match) return true;
  const major = Number(match[1]);
  const minor = Number(match[2]);
  return major > 16 || (major === 16 && minor >= 4);
}

// On iOS, push only reaches a Home Screen install and a Safari tab has no
// Notification API at all, so this is where that gap gets explained.
// 'Notification' in window means push is already reachable, installed or not.
function needsInstallHint(): boolean {
  return isIos() && !isStandalone() && !("Notification" in window) && iosSupportsPush();
}

export default function IosInstallHint() {
  const { shown, dismiss } = useDelayedPrompt(STORAGE_KEY, DISMISS_DAYS, APPEAR_DELAY_MS, needsInstallHint);

  return (
    <Transition
      mounted={shown}
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
                onClick={dismiss}
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
