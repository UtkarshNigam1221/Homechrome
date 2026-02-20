'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useRef, useState } from 'react';

import Button from '@/components/common/Button';
import api from '@/lib/api';
import { PaymentStatus } from '@/types';

interface PaymentStatusResponse {
  order_id: string;
  order_number: string;
  payment_status: PaymentStatus;
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

  const checkPaymentStatus = useCallback(async () => {
    if (!orderId) return;

    try {
      const { data } = await api.get<PaymentStatusResponse>(
        `/api/v1/store/checkout/payment-status/${orderId}`,
      );

      setStatus(data.payment_status);
      if (data.order_number) {
        setOrderNumber(data.order_number);
      }

      // Stop polling on terminal states
      if (['PAID', 'FAILED', 'REFUNDED'].includes(data.payment_status)) {
        setPolling(false);
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      }

      // Stop after 60 polls (3 minutes)
      pollCountRef.current += 1;
      if (pollCountRef.current >= 60) {
        setPolling(false);
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      }
    } catch {
      setError('Unable to check payment status. Please check your orders page.');
      setPolling(false);
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    }
  }, [orderId]);

  useEffect(() => {
    if (!orderId) return;

    // Initial check
    checkPaymentStatus();

    // Poll every 3 seconds
    intervalRef.current = setInterval(checkPaymentStatus, 3000);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [orderId, checkPaymentStatus]);

  if (!orderId) {
    return (
      <div className="mx-auto max-w-lg px-4 py-20 text-center">
        <h1 className="mb-4 text-2xl font-bold text-foreground">Invalid Request</h1>
        <p className="mb-6 text-muted">No order ID was provided.</p>
        <Link href="/">
          <Button>Go to Home</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-16 text-center sm:px-6">
      {/* Pending / Processing */}
      {polling && (status === 'PENDING' || status === 'INITIATED') && (
        <>
          <div className="mx-auto mb-6 h-16 w-16 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          <h1 className="mb-2 text-2xl font-bold text-foreground">
            Processing Payment
          </h1>
          <p className="mb-2 text-muted">
            Please wait while we confirm your payment...
          </p>
          <p className="text-sm text-muted">
            Do not close this page. This may take a few moments.
          </p>
        </>
      )}

      {/* Success */}
      {status === 'PAID' && (
        <>
          <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-green-100">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={2}
              stroke="currentColor"
              className="h-10 w-10 text-green-600"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="m4.5 12.75 6 6 9-13.5"
              />
            </svg>
          </div>
          <h1 className="mb-2 text-2xl font-bold text-foreground">
            Payment Successful!
          </h1>
          <p className="mb-1 text-muted">Thank you for your order.</p>
          {orderNumber && (
            <p className="mb-6 text-sm text-muted">
              Order Number: <span className="font-semibold text-foreground">{orderNumber}</span>
            </p>
          )}
          <div className="flex flex-col items-center gap-3">
            <Link href={`/account/orders/${orderId}`}>
              <Button>View Order Details</Button>
            </Link>
            <Link
              href="/"
              className="text-sm font-medium text-primary hover:text-primary-dark"
            >
              Continue Shopping
            </Link>
          </div>
        </>
      )}

      {/* Failed */}
      {status === 'FAILED' && (
        <>
          <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-red-100">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={2}
              stroke="currentColor"
              className="h-10 w-10 text-red-600"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M6 18 18 6M6 6l12 12"
              />
            </svg>
          </div>
          <h1 className="mb-2 text-2xl font-bold text-foreground">Payment Failed</h1>
          <p className="mb-6 text-muted">
            Your payment could not be processed. No amount has been charged.
          </p>
          <div className="flex flex-col items-center gap-3">
            <Link href="/checkout">
              <Button>Try Again</Button>
            </Link>
            <Link
              href="/account/orders"
              className="text-sm font-medium text-primary hover:text-primary-dark"
            >
              View Orders
            </Link>
          </div>
        </>
      )}

      {/* Refunded */}
      {status === 'REFUNDED' && (
        <>
          <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-amber-100">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={2}
              stroke="currentColor"
              className="h-10 w-10 text-amber-600"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
              />
            </svg>
          </div>
          <h1 className="mb-2 text-2xl font-bold text-foreground">Payment Refunded</h1>
          <p className="mb-6 text-muted">
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
          <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-amber-100">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={2}
              stroke="currentColor"
              className="h-10 w-10 text-amber-600"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
              />
            </svg>
          </div>
          <h1 className="mb-2 text-2xl font-bold text-foreground">
            Something went wrong
          </h1>
          <p className="mb-6 text-muted">{error}</p>
          <div className="flex flex-col items-center gap-3">
            <Link href="/account/orders">
              <Button>Check My Orders</Button>
            </Link>
            <Link
              href="/"
              className="text-sm font-medium text-primary hover:text-primary-dark"
            >
              Go to Home
            </Link>
          </div>
        </>
      )}

      {/* Timeout */}
      {!polling && !error && (status === 'PENDING' || status === 'INITIATED') && (
        <>
          <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-amber-100">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={2}
              stroke="currentColor"
              className="h-10 w-10 text-amber-600"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
              />
            </svg>
          </div>
          <h1 className="mb-2 text-2xl font-bold text-foreground">
            Payment is still processing
          </h1>
          <p className="mb-6 text-muted">
            Your payment is taking longer than expected. You will receive an update
            once it is confirmed.
          </p>
          <div className="flex flex-col items-center gap-3">
            <Link href="/account/orders">
              <Button>Check My Orders</Button>
            </Link>
            <Link
              href="/"
              className="text-sm font-medium text-primary hover:text-primary-dark"
            >
              Go to Home
            </Link>
          </div>
        </>
      )}
    </div>
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
