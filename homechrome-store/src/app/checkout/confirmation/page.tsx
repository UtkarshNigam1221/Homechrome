'use client';

import {
  CheckCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useRef, useState } from 'react';

import Button from '@/components/ui/button';
import { Container } from '@/components/ui/container';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { PaymentStatus } from '@/types';

interface PaymentStatusResponse {
  order_id: string;
  order_number: string;
  payment_status: PaymentStatus;
}

function StatusIcon({ variant, className }: { variant: 'success' | 'error' | 'warning'; className?: string }) {
  const bg = { success: 'bg-green-100', error: 'bg-red-100', warning: 'bg-amber-100' }[variant];
  const Icon = { success: CheckCircleIcon, error: XCircleIcon, warning: ExclamationTriangleIcon }[variant];
  const color = { success: 'text-green-600', error: 'text-red-600', warning: 'text-amber-600' }[variant];

  return (
    <div className={`mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full ${bg} ${className}`}>
      <Icon className={`h-10 w-10 ${color}`} />
    </div>
  );
}

function StatusActions({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col items-center gap-3">{children}</div>;
}

function SecondaryLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Link href={href} className="text-sm font-medium text-primary hover:text-primary-dark">
      {children}
    </Link>
  );
}

function ConfirmationContent() {
  const searchParams = useSearchParams();
  const orderId = searchParams.get('order_id');

  const [status, setStatus] = useState<PaymentStatus>('PENDING');
  const [orderNumber, setOrderNumber] = useState<string>('');
  const [polling, setPolling] = useState(true);
  const [error, setError] = useState<string | null>(null);
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

  // Keep the ref updated with the latest closure — no dependency array issues
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
  }, [orderId, stopPolling]);

  useEffect(() => {
    if (!orderId) return;

    // Fire immediately, then poll every 3s
    checkStatusRef.current?.();
    intervalRef.current = setInterval(() => checkStatusRef.current?.(), 3000);

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [orderId]);

  if (!orderId) {
    return (
      <Container size="narrow" className="max-w-lg py-20 text-center">
        <h1 className="mb-4 text-2xl font-bold text-foreground">Invalid Request</h1>
        <p className="mb-6 text-muted-foreground">No order ID was provided.</p>
        <Link href="/">
          <Button>Go to Home</Button>
        </Link>
      </Container>
    );
  }

  return (
    <Container size="narrow" className="max-w-lg py-16 text-center">
      {/* Pending / Processing */}
      {polling && (status === 'PENDING' || status === 'INITIATED') && (
        <>
          <div className="mx-auto mb-6 h-16 w-16 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          <h1 className="mb-2 text-2xl font-bold text-foreground">Processing Payment</h1>
          <p className="mb-2 text-muted-foreground">
            Please wait while we confirm your payment...
          </p>
          <p className="text-sm text-muted-foreground">
            Do not close this page. This may take a few moments.
          </p>
        </>
      )}

      {/* Success */}
      {(status === 'PAID' || status === 'SUCCESS') && (
        <>
          <StatusIcon variant="success" />
          <h1 className="mb-2 text-2xl font-bold text-foreground">Payment Successful!</h1>
          <p className="mb-1 text-muted-foreground">Thank you for your order.</p>
          {orderNumber && (
            <p className="mb-6 text-sm text-muted-foreground">
              Order Number: <span className="font-semibold text-foreground">{orderNumber}</span>
            </p>
          )}
          <StatusActions>
            <Link href={`/account/orders/${orderId}`}>
              <Button>View Order Details</Button>
            </Link>
            <SecondaryLink href="/">Continue Shopping</SecondaryLink>
          </StatusActions>
        </>
      )}

      {/* Failed */}
      {status === 'FAILED' && (
        <>
          <StatusIcon variant="error" />
          <h1 className="mb-2 text-2xl font-bold text-foreground">Payment Failed</h1>
          <p className="mb-6 text-muted-foreground">
            Your payment could not be processed. No amount has been charged.
          </p>
          <StatusActions>
            <Link href="/checkout">
              <Button>Try Again</Button>
            </Link>
            <SecondaryLink href="/account/orders">View Orders</SecondaryLink>
          </StatusActions>
        </>
      )}

      {/* Refunded */}
      {status === 'REFUNDED' && (
        <>
          <StatusIcon variant="warning" />
          <h1 className="mb-2 text-2xl font-bold text-foreground">Payment Refunded</h1>
          <p className="mb-6 text-muted-foreground">
            Your payment has been refunded. The amount will be credited back shortly.
          </p>
          <Link href="/">
            <Button>Go to Home</Button>
          </Link>
        </>
      )}

      {/* Error */}
      {error && (
        <>
          <StatusIcon variant="warning" />
          <h1 className="mb-2 text-2xl font-bold text-foreground">Something went wrong</h1>
          <p className="mb-6 text-muted-foreground">{error}</p>
          <StatusActions>
            <Link href="/account/orders">
              <Button>Check My Orders</Button>
            </Link>
            <SecondaryLink href="/">Go to Home</SecondaryLink>
          </StatusActions>
        </>
      )}

      {/* Timeout */}
      {!polling && !error && (status === 'PENDING' || status === 'INITIATED') && (
        <>
          <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-amber-100">
            <ClockIcon className="h-10 w-10 text-amber-600" />
          </div>
          <h1 className="mb-2 text-2xl font-bold text-foreground">
            Payment is still processing
          </h1>
          <p className="mb-6 text-muted-foreground">
            Your payment is taking longer than expected. You will receive an update
            once it is confirmed.
          </p>
          <StatusActions>
            <Link href="/account/orders">
              <Button>Check My Orders</Button>
            </Link>
            <SecondaryLink href="/">Go to Home</SecondaryLink>
          </StatusActions>
        </>
      )}
    </Container>
  );
}

export default function ConfirmationPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center py-20">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      }
    >
      <ConfirmationContent />
    </Suspense>
  );
}
