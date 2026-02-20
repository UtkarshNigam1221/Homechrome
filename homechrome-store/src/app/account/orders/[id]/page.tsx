'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import Button from '@/components/common/Button';
import api from '@/lib/api';
import { Order, OrderStatus } from '@/types';

const statusColors: Record<OrderStatus, string> = {
  PENDING: 'bg-yellow-100 text-yellow-800',
  CONFIRMED: 'bg-blue-100 text-blue-800',
  PROCESSING: 'bg-indigo-100 text-indigo-800',
  SHIPPED: 'bg-purple-100 text-purple-800',
  DELIVERED: 'bg-green-100 text-green-800',
  CANCELLED: 'bg-red-100 text-red-800',
  RETURNED: 'bg-orange-100 text-orange-800',
  REFUNDED: 'bg-gray-100 text-gray-800',
};

const statusTimeline: OrderStatus[] = [
  'PENDING',
  'CONFIRMED',
  'PROCESSING',
  'SHIPPED',
  'DELIVERED',
];

function formatPrice(paise: number): string {
  return `₹${(paise / 100).toLocaleString('en-IN')}`;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function getStatusIndex(status: OrderStatus): number {
  const idx = statusTimeline.indexOf(status);
  return idx === -1 ? -1 : idx;
}

export default function OrderDetailPage() {
  const params = useParams();
  const router = useRouter();
  const orderId = params.id as string;

  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState<string | null>(null);

  const fetchOrder = useCallback(async () => {
    try {
      setLoading(true);
      const { data } = await api.get<Order>(`/api/v1/store/orders/${orderId}`);
      setOrder(data);
    } catch {
      router.replace('/account/orders');
    } finally {
      setLoading(false);
    }
  }, [orderId, router]);

  useEffect(() => {
    fetchOrder();
  }, [fetchOrder]);

  const handleCancel = async () => {
    if (!order) return;
    const confirmed = window.confirm(
      'Are you sure you want to cancel this order? This action cannot be undone.',
    );
    if (!confirmed) return;

    setCancelling(true);
    setCancelError(null);

    try {
      await api.post(`/api/v1/store/orders/${orderId}/cancel`);
      await fetchOrder();
    } catch {
      setCancelError('Failed to cancel order. Please try again or contact support.');
    } finally {
      setCancelling(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-48 animate-pulse rounded bg-gray-200" />
        <div className="h-64 animate-pulse rounded-lg border border-border bg-white" />
      </div>
    );
  }

  if (!order) return null;

  const canCancel = order.status === 'PENDING' || order.status === 'CONFIRMED';
  const currentStatusIdx = getStatusIndex(order.status);
  const isCancelled = ['CANCELLED', 'RETURNED', 'REFUNDED'].includes(order.status);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <Link
            href="/account/orders"
            className="mb-1 inline-block text-sm text-primary hover:text-primary-dark"
          >
            &larr; Back to Orders
          </Link>
          <h2 className="text-lg font-semibold text-foreground">
            Order #{order.order_number}
          </h2>
          <p className="text-sm text-muted">Placed on {formatDate(order.created_at)}</p>
        </div>
        <span
          className={`inline-block self-start rounded-full px-3 py-1 text-sm font-medium ${statusColors[order.status]}`}
        >
          {order.status}
        </span>
      </div>

      {cancelError && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {cancelError}
        </div>
      )}

      {/* Status Timeline */}
      {!isCancelled && (
        <div className="rounded-lg border border-border bg-white p-6">
          <h3 className="mb-4 text-sm font-semibold text-foreground">Order Status</h3>
          <div className="flex items-center justify-between">
            {statusTimeline.map((status, idx) => (
              <div key={status} className="flex flex-1 items-center">
                <div className="flex flex-col items-center">
                  <div
                    className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-bold ${
                      idx <= currentStatusIdx
                        ? 'bg-primary text-white'
                        : 'bg-gray-200 text-muted'
                    }`}
                  >
                    {idx < currentStatusIdx ? (
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        fill="none"
                        viewBox="0 0 24 24"
                        strokeWidth={2.5}
                        stroke="currentColor"
                        className="h-4 w-4"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="m4.5 12.75 6 6 9-13.5"
                        />
                      </svg>
                    ) : (
                      idx + 1
                    )}
                  </div>
                  <span className="mt-1 text-center text-[10px] text-muted sm:text-xs">
                    {status}
                  </span>
                </div>
                {idx < statusTimeline.length - 1 && (
                  <div
                    className={`mx-1 h-0.5 flex-1 ${
                      idx < currentStatusIdx ? 'bg-primary' : 'bg-gray-200'
                    }`}
                  />
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tracking Info */}
      {order.tracking_number && (
        <div className="rounded-lg border border-border bg-white p-6">
          <h3 className="mb-3 text-sm font-semibold text-foreground">
            Tracking Information
          </h3>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            {order.shipping_carrier && (
              <div>
                <p className="text-xs text-muted">Courier</p>
                <p className="font-medium text-foreground">{order.shipping_carrier}</p>
              </div>
            )}
            <div>
              <p className="text-xs text-muted">AWB Number</p>
              <p className="font-medium text-foreground">{order.tracking_number}</p>
            </div>
            {order.tracking_url && (
              <div>
                <p className="text-xs text-muted">Track Shipment</p>
                <a
                  href={order.tracking_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm font-medium text-primary hover:text-primary-dark"
                >
                  Track on courier website
                </a>
              </div>
            )}
          </div>
          {order.shipped_at && (
            <p className="mt-2 text-xs text-muted">
              Shipped on {formatDate(order.shipped_at)}
            </p>
          )}
          {order.delivered_at && (
            <p className="mt-1 text-xs text-muted">
              Delivered on {formatDate(order.delivered_at)}
            </p>
          )}
        </div>
      )}

      {/* Items */}
      <div className="rounded-lg border border-border bg-white p-6">
        <h3 className="mb-4 text-sm font-semibold text-foreground">Items</h3>
        <div className="divide-y divide-border">
          {order.items.map((item) => (
            <div key={item.id} className="flex gap-4 py-4 first:pt-0 last:pb-0">
              <div className="relative h-20 w-20 flex-shrink-0 overflow-hidden rounded-md bg-gray-100">
                {item.product_image ? (
                  <Image
                    src={item.product_image}
                    alt={item.product_name}
                    fill
                    className="object-cover"
                    sizes="80px"
                  />
                ) : (
                  <div className="flex h-full w-full items-center justify-center text-xs text-muted">
                    No image
                  </div>
                )}
              </div>
              <div className="flex flex-1 flex-col justify-between">
                <div>
                  <p className="font-medium text-foreground">{item.product_name}</p>
                  <p className="text-sm text-muted">SKU: {item.product_sku}</p>
                </div>
                <div className="flex items-center justify-between">
                  <p className="text-sm text-muted">Qty: {item.quantity}</p>
                  <p className="font-medium text-foreground">
                    {formatPrice(item.total_price)}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Order Summary + Shipping Address */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
        <div className="rounded-lg border border-border bg-white p-6">
          <h3 className="mb-3 text-sm font-semibold text-foreground">
            Payment Summary
          </h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted">Subtotal</span>
              <span className="text-foreground">{formatPrice(order.subtotal)}</span>
            </div>
            {order.discount_amount > 0 && (
              <div className="flex justify-between">
                <span className="text-muted">Discount</span>
                <span className="text-green-600">
                  -{formatPrice(order.discount_amount)}
                </span>
              </div>
            )}
            <div className="flex justify-between">
              <span className="text-muted">Shipping</span>
              <span className="text-foreground">
                {order.shipping_amount === 0
                  ? 'FREE'
                  : formatPrice(order.shipping_amount)}
              </span>
            </div>
            {order.tax_amount > 0 && (
              <div className="flex justify-between">
                <span className="text-muted">Tax</span>
                <span className="text-foreground">{formatPrice(order.tax_amount)}</span>
              </div>
            )}
            <div className="flex justify-between border-t border-border pt-2 font-semibold">
              <span className="text-foreground">Total</span>
              <span className="text-foreground">{formatPrice(order.total_amount)}</span>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border bg-white p-6">
          <h3 className="mb-3 text-sm font-semibold text-foreground">
            Shipping Address
          </h3>
          <div className="text-sm">
            <p className="font-medium text-foreground">
              {order.shipping_address.first_name} {order.shipping_address.last_name}
            </p>
            <p className="mt-1 text-muted">{order.shipping_address.address_line1}</p>
            {order.shipping_address.address_line2 && (
              <p className="text-muted">{order.shipping_address.address_line2}</p>
            )}
            <p className="text-muted">
              {order.shipping_address.city}, {order.shipping_address.state} -{' '}
              {order.shipping_address.postal_code}
            </p>
            <p className="text-muted">Phone: {order.shipping_address.phone}</p>
          </div>
        </div>
      </div>

      {/* Cancel button */}
      {canCancel && (
        <div className="rounded-lg border border-border bg-white p-6">
          <h3 className="mb-2 text-sm font-semibold text-foreground">
            Cancel Order
          </h3>
          <p className="mb-4 text-sm text-muted">
            You can cancel this order since it has not been processed yet.
          </p>
          <Button
            variant="outline"
            onClick={handleCancel}
            loading={cancelling}
            className="border-red-300 text-red-600 hover:bg-red-50 hover:text-red-700"
          >
            Cancel Order
          </Button>
        </div>
      )}
    </div>
  );
}
