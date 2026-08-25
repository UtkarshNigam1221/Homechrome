'use client';

import {
  CheckCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import {
  Alert,
  Anchor,
  Button,
  Container,
  Stack,
  Text,
  ThemeIcon,
  Title,
} from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useRef, useState } from 'react';

import { LoadingBlock, LoadingSpinner } from '@/components/ui/loading-spinner';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { PaymentStatus } from '@/types';

interface PaymentStatusResponse {
  order_id: string;
  order_number: string;
  payment_status: PaymentStatus;
}

// How often to re-poll the order's payment status while it's still pending.
const POLL_INTERVAL_MS = 3000;

type IconVariant = 'success' | 'error' | 'warning';

function StatusIcon({ variant }: { variant: IconVariant }) {
  const config: Record<IconVariant, { color: string; Icon: typeof CheckCircleIcon }> = {
    success: { color: 'teal', Icon: CheckCircleIcon },
    error: { color: 'red', Icon: XCircleIcon },
    warning: { color: 'yellow', Icon: ExclamationTriangleIcon },
  };
  const { color, Icon } = config[variant];
  return (
    <ThemeIcon size={80} radius="xl" color={color} variant="light">
      <Icon width={40} height={40} />
    </ThemeIcon>
  );
}

function StatusActions({ children }: { children: React.ReactNode }) {
  return <Stack align="center" gap="sm">{children}</Stack>;
}

function SecondaryLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Anchor component={Link} href={href} size="sm" c="brand">
      {children}
    </Anchor>
  );
}

function ConfirmationContent() {
  const searchParams = useSearchParams();
  const orderId = searchParams.get('order_id');
  const queryClient = useQueryClient();

  const [status, setStatus] = useState<PaymentStatus>('PENDING');
  const [orderNumber, setOrderNumber] = useState<string>('');
  const [polling, setPolling] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [couponNotice, setCouponNotice] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollCountRef = useRef(0);
  const checkStatusRef = useRef<() => Promise<void>>(undefined);

  const stopPolling = useCallback(() => {
    setPolling(false);
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  useEffect(() => {
    // handlePayNow stashed this before redirecting to the payment gateway,
    // since that navigation drops the initiate response. One-shot read.
    if (!orderId) return;
    const key = `coupon_notice:${orderId}`;
    const stored = sessionStorage.getItem(key);
    if (stored) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- one-shot hydration, not a subscription
      setCouponNotice(stored);
      sessionStorage.removeItem(key);
    }
  }, [orderId]);

  useEffect(() => {
    checkStatusRef.current = async () => {
      if (!orderId) return;
      try {
        const { data } = await api.get<PaymentStatusResponse>(
          ROUTES.CHECKOUT.PAYMENT_STATUS(orderId),
        );
        setStatus(data.payment_status);
        if (data.order_number) setOrderNumber(data.order_number);
        if (['PAID', 'SUCCESS', 'FAILED', 'REFUNDED'].includes(data.payment_status)) {
          stopPolling();
          queryClient.invalidateQueries({ queryKey: ['orders'] });
        }
        pollCountRef.current += 1;
        if (pollCountRef.current >= 60) stopPolling();
      } catch (err: unknown) {
        const httpStatus = (err as { response?: { status?: number } })?.response?.status;
        if (httpStatus === 401) {
          setError('Your payment has been processed. Please log in to view your order details.');
        } else {
          setError('Unable to check payment status. Please check your orders page.');
        }
        stopPolling();
      }
    };
  }, [orderId, stopPolling, queryClient]);

  useEffect(() => {
    if (!orderId) return;
    checkStatusRef.current?.();
    intervalRef.current = setInterval(() => checkStatusRef.current?.(), POLL_INTERVAL_MS);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [orderId]);

  if (!orderId) {
    return (
      <Container size="sm" py="xl">
        <Stack align="center" gap="md">
          <Title order={1} size="h2">Invalid Request</Title>
          <Text c="dimmed">No order ID was provided.</Text>
          <Button component={Link} href="/" color="brand">Go to Home</Button>
        </Stack>
      </Container>
    );
  }

  return (
    <Container size="sm" py="xl">
      <Stack align="center" gap="md" ta="center">
        {couponNotice && (
          <Alert color="yellow" title="About your coupon" w="100%" ta="left">
            {couponNotice}
          </Alert>
        )}

        {polling && (status === 'PENDING' || status === 'INITIATED') && (
          <>
            <LoadingSpinner size="lg" />
            <Title order={1} size="h2">Processing Payment</Title>
            <Text c="dimmed">Please wait while we confirm your payment...</Text>
            <Text size="sm" c="dimmed">
              Do not close this page. This may take a few moments.
            </Text>
          </>
        )}

        {(status === 'PAID' || status === 'SUCCESS') && (
          <>
            <StatusIcon variant="success" />
            <Title order={1} size="h2">Payment Successful!</Title>
            <Text c="dimmed">Thank you for your order.</Text>
            {orderNumber && (
              <Text size="sm" c="dimmed">
                Order Number: <Text span fw={600} c="navy.7">{orderNumber}</Text>
              </Text>
            )}
            <StatusActions>
              <Button component={Link} href={`/account/orders/${orderId}`} color="brand">
                View Order Details
              </Button>
              <SecondaryLink href="/">Continue Shopping</SecondaryLink>
            </StatusActions>
          </>
        )}

        {status === 'FAILED' && (
          <>
            <StatusIcon variant="error" />
            <Title order={1} size="h2">Payment Failed</Title>
            <Text c="dimmed">
              Your payment could not be processed. No amount has been charged.
            </Text>
            <StatusActions>
              <Button component={Link} href="/checkout" color="brand">Try Again</Button>
              <SecondaryLink href="/account/orders">View Orders</SecondaryLink>
            </StatusActions>
          </>
        )}

        {status === 'REFUNDED' && (
          <>
            <StatusIcon variant="warning" />
            <Title order={1} size="h2">Payment Refunded</Title>
            <Text c="dimmed">
              Your payment has been refunded. The amount will be credited back shortly.
            </Text>
            <Button component={Link} href="/" color="brand">Go to Home</Button>
          </>
        )}

        {error && (
          <>
            <StatusIcon variant="warning" />
            <Title order={1} size="h2">Something went wrong</Title>
            <Text c="dimmed">{error}</Text>
            <StatusActions>
              <Button component={Link} href="/account/orders" color="brand">
                Check My Orders
              </Button>
              <SecondaryLink href="/">Go to Home</SecondaryLink>
            </StatusActions>
          </>
        )}

        {!polling && !error && (status === 'PENDING' || status === 'INITIATED') && (
          <>
            <ThemeIcon size={80} radius="xl" color="yellow" variant="light">
              <ClockIcon width={40} height={40} />
            </ThemeIcon>
            <Title order={1} size="h2">Payment is still processing</Title>
            <Text c="dimmed">
              Your payment is taking longer than expected. You will receive an update
              once it is confirmed.
            </Text>
            <StatusActions>
              <Button component={Link} href="/account/orders" color="brand">
                Check My Orders
              </Button>
              <SecondaryLink href="/">Go to Home</SecondaryLink>
            </StatusActions>
          </>
        )}
      </Stack>
    </Container>
  );
}

export default function ConfirmationPage() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <ConfirmationContent />
    </Suspense>
  );
}
