"use client";

import {
  BellIcon,
  CheckCircleIcon,
  PaperAirplaneIcon,
  ShieldCheckIcon,
  XCircleIcon,
} from "@heroicons/react/24/outline";
import {
  Alert,
  Badge,
  Box,
  Button,
  Group,
  Modal,
  Paper,
  Stack,
  Switch,
  Text,
  ThemeIcon,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useState } from "react";

import { usePushNotifications } from "@/hooks/usePushNotifications";

interface NotificationModalProps {
  opened: boolean;
  onClose: () => void;
}

export default function NotificationModal({
  opened,
  onClose,
}: NotificationModalProps) {
  const {
    isSupported,
    isConfigured,
    permission,
    isSubscribed,
    loading,
    error,
    subscribe,
    unsubscribe,
    sendTest,
  } = usePushNotifications();

  const [testSending, setTestSending] = useState(false);

  const handleToggle = async (checked: boolean) => {
    if (checked) {
      if (await subscribe()) {
        notifications.show({
          title: "Notifications on",
          message: "You are subscribed to Homechrome handloom updates.",
          color: "teal",
        });
      }
      return;
    }

    if (await unsubscribe()) {
      notifications.show({
        title: "Notifications off",
        message: "This device will no longer receive browser alerts.",
        color: "gray",
      });
    }
  };

  // Sends to this device only — the backend route is scoped to the endpoint
  // in the request body and cannot fan out.
  const handleSendTest = async () => {
    setTestSending(true);
    try {
      // sendTest reports a missing local subscription by returning false rather
      // than throwing, so the catch below would not see it.
      if (!(await sendTest())) {
        notifications.show({
          title: "Could not send test",
          message:
            "This device is no longer subscribed. Turn notifications off and on again.",
          color: "red",
        });
        return;
      }
      notifications.show({
        title: "Test notification sent",
        message: "Check your notification drawer for the alert.",
        color: "teal",
      });
    } catch {
      // Axios messages here read like "Request failed with status code 400",
      // which tells a shopper nothing actionable.
      notifications.show({
        title: "Could not send test",
        message: "Please try again in a moment.",
        color: "red",
      });
    } finally {
      setTestSending(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        <Group gap="xs">
          <ThemeIcon size="md" color="brand" radius="md">
            <BellIcon width={18} height={18} />
          </ThemeIcon>
          <Text fw={700} size="md" c="navy.7">
            Notification Settings
          </Text>
        </Group>
      }
      size="md"
      radius="md"
      padding="lg"
      centered
    >
      <Stack gap="md">
        <Paper withBorder p="md" radius="md" bg="gray.0">
          <Group justify="space-between" align="center" wrap="nowrap" gap="sm">
            {/* The copy carries the slack, the control keeps its size. A width
                percentage would have to be retuned every time this text edits. */}
            <Box style={{ minWidth: 0 }}>
              <Text fw={600} size="sm" c="navy.7">
                Browser notifications
              </Text>
              <Text size="xs" c="dimmed" lh={1.4} mt={2}>
                Instant alerts for new saree releases, bedding restocks, and
                private weaver offers.
              </Text>
            </Box>
            <Switch
              checked={isSubscribed}
              onChange={(e) => handleToggle(e.currentTarget.checked)}
              disabled={
                loading ||
                !isSupported ||
                !isConfigured ||
                permission === "denied"
              }
              size="md"
              color="teal"
              aria-label="Toggle push notifications"
              style={{ flexShrink: 0 }}
            />
          </Group>
        </Paper>

        <Group justify="space-between" align="center">
          <Text size="xs" c="dimmed">
            This device
          </Text>
          {isSubscribed ? (
            <Badge
              color="teal"
              variant="light"
              leftSection={<CheckCircleIcon width={14} height={14} />}
            >
              Subscribed
            </Badge>
          ) : permission === "denied" ? (
            <Badge
              color="red"
              variant="light"
              leftSection={<XCircleIcon width={14} height={14} />}
            >
              Blocked in browser
            </Badge>
          ) : (
            <Badge color="gray" variant="light">
              Not subscribed
            </Badge>
          )}
        </Group>

        {error && (
          <Alert color="red" title="Notice" radius="md">
            {error}
          </Alert>
        )}

        {permission === "denied" && (
          <Alert color="orange" title="Permission blocked" radius="md">
            Notifications are blocked for this site. Open the site settings icon
            in your address bar and choose &quot;Allow&quot; for notifications.
          </Alert>
        )}

        {!isSupported && (
          <Alert color="gray" title="Not supported" radius="md">
            This browser does not support push notifications. Chrome, Edge,
            Safari, or Firefox on desktop or mobile all do.
          </Alert>
        )}

        {isSupported && !isConfigured && (
          <Alert color="gray" title="Unavailable" radius="md">
            Notifications are not available right now. Please check back later.
          </Alert>
        )}

        {isSubscribed && (
          <Box
            p="sm"
            bg="teal.0"
            style={{
              borderRadius: 8,
              border: "1px solid var(--mantine-color-teal-2)",
            }}
          >
            <Group
              justify="space-between"
              align="center"
              wrap="nowrap"
              gap="sm"
            >
              {/* minWidth lets the copy shrink; without it the flex row steals
                  width from the button instead and clips its label. */}
              <Box style={{ minWidth: 0 }}>
                <Text size="xs" fw={600} c="teal.9">
                  Check it works
                </Text>
                <Text size="11px" c="teal.7">
                  Sends one notification to this device.
                </Text>
              </Box>
              <Button
                variant="light"
                color="brand"
                size="xs"
                loading={testSending}
                onClick={handleSendTest}
                leftSection={<PaperAirplaneIcon width={14} height={14} />}
                style={{ flexShrink: 0 }}
              >
                Send Test
              </Button>
            </Group>
          </Box>
        )}

        <Paper withBorder p="xs" radius="sm" bg="gray.0">
          <Group gap="xs" wrap="nowrap">
            <ShieldCheckIcon
              width={18}
              height={18}
              className="text-emerald-600 shrink-0"
            />
            <Text size="xs" c="dimmed">
              We only send collection drops and offers. You can turn this off at
              any time.
            </Text>
          </Group>
        </Paper>
      </Stack>
    </Modal>
  );
}
