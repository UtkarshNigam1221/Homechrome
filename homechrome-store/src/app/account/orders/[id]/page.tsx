'use client';

import { CheckIcon } from '@heroicons/react/24/outline';
import Image from 'next/image';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import ReturnStatusBadge from '@/components/orders/ReturnStatusBadge';
import Button from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import Skeleton from '@/components/skeleton/Skeleton';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatDateTime, formatPrice, statusColors } from '@/lib/utils';
import { Order, OrderStatus } from '@/types';

const statusTimeline: OrderStatus[] = [
  'PENDING',
  'CONFIRMED',
  'PROCESSING',
  'SHIPPED',
  'DELIVERED',
];

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
      const { data } = await api.get<Order>(ROUTES.ORDERS.DETAIL(orderId));
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

    setCancelling(true);
    setCancelError(null);

    try {
      await api.post(ROUTES.ORDERS.CANCEL(orderId));
      await fetchOrder();
    } catch {
      setCancelError('Failed to cancel order. Please try again or contact support.');
    } finally {
      setCancelling(false);
    }
  };

  if (loading) {
    return (
      <div role="status" aria-label="Loading order details" className="space-y-6">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-1">
            <Skeleton className="h-4 w-28" />
            <Skeleton className="h-6 w-44" />
            <Skeleton className="h-4 w-40" />
          </div>
          <Skeleton variant="rectangular" className="h-7 w-24 self-start rounded-full" />
        </div>
        <Card>
          <CardContent>
            <div className="flex items-center justify-between">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="flex flex-1 items-center">
                  <div className="flex flex-col items-center">
                    <Skeleton variant="circular" className="h-8 w-8" />
                    <Skeleton className="mt-1 h-3 w-12" />
                  </div>
                  {i < 5 && <Skeleton className="mx-1 h-0.5 flex-1" />}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="space-y-4">
            {[1, 2].map((i) => (
              <div key={i} className="flex gap-4">
                <Skeleton variant="rectangular" className="h-20 w-20 flex-shrink-0 rounded-md" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-5 w-48" />
                  <Skeleton className="h-4 w-24" />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
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
          <p className="text-sm text-muted-foreground">Placed on {formatDateTime(order.created_at)}</p>
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
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Order Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              {statusTimeline.map((status, idx) => (
                <div key={status} className="flex flex-1 items-center">
                  <div className="flex flex-col items-center">
                    <div
                      className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-bold ${
                        idx <= currentStatusIdx
                          ? 'bg-primary text-white'
                          : 'bg-gray-200 text-muted-foreground'
                      }`}
                    >
                      {idx < currentStatusIdx ? (
                        <CheckIcon className="h-4 w-4" strokeWidth={2.5} />
                      ) : (
                        idx + 1
                      )}
                    </div>
                    <span className="mt-1 text-center text-[10px] text-muted-foreground sm:text-xs">
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
          </CardContent>
        </Card>
      )}

      {/* Tracking Info */}
      {order.tracking_number && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Tracking Information</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              {order.shipping_carrier && (
                <div>
                  <p className="text-xs text-muted-foreground">Courier</p>
                  <p className="font-medium text-foreground">{order.shipping_carrier}</p>
                </div>
              )}
              <div>
                <p className="text-xs text-muted-foreground">AWB Number</p>
                <p className="font-medium text-foreground">{order.tracking_number}</p>
              </div>
              {order.tracking_url && (
                <div>
                  <p className="text-xs text-muted-foreground">Track Shipment</p>
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
              <p className="mt-2 text-xs text-muted-foreground">
                Shipped on {formatDateTime(order.shipped_at)}
              </p>
            )}
            {order.delivered_at && (
              <p className="mt-1 text-xs text-muted-foreground">
                Delivered on {formatDateTime(order.delivered_at)}
              </p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Items */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Items</CardTitle>
        </CardHeader>
        <CardContent>
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
                    <div className="flex h-full w-full items-center justify-center text-xs text-muted-foreground">
                      No image
                    </div>
                  )}
                </div>
                <div className="flex flex-1 flex-col justify-between">
                  <div>
                    <p className="font-medium text-foreground">{item.product_name}</p>
                    <p className="text-sm text-muted-foreground">SKU: {item.product_sku}</p>
                  </div>
                  <div className="flex items-center justify-between">
                    <p className="text-sm text-muted-foreground">Qty: {item.quantity}</p>
                    <p className="font-medium text-foreground">
                      {formatPrice(item.total_price)}
                    </p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Returns */}
      {order.returns && order.returns.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Returns</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-3">
              {order.returns.map((ret) => (
                <li
                  key={ret.id}
                  className="flex items-start justify-between gap-4"
                >
                  <div className="text-sm">
                    <p className="font-medium text-foreground">
                      Return #{ret.id.slice(0, 8)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      Requested {formatDateTime(ret.created_at)}
                    </p>
                    {ret.reason && (
                      <p className="mt-1 text-xs text-muted-foreground">
                        Reason: {ret.reason}
                      </p>
                    )}
                  </div>
                  <ReturnStatusBadge status={ret.status} />
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {/* Order Summary + Shipping Address */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Payment Summary</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Subtotal</span>
                <span className="text-foreground">{formatPrice(order.subtotal)}</span>
              </div>
              {order.discount_amount > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Discount</span>
                  <span className="text-green-600">-{formatPrice(order.discount_amount)}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-muted-foreground">Shipping</span>
                <span className="text-foreground">
                  {order.shipping_amount === 0 ? 'FREE' : formatPrice(order.shipping_amount)}
                </span>
              </div>
              {order.tax_amount > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Tax</span>
                  <span className="text-foreground">{formatPrice(order.tax_amount)}</span>
                </div>
              )}
              <Separator />
              <div className="flex justify-between font-semibold">
                <span className="text-foreground">Total</span>
                <span className="text-foreground">{formatPrice(order.total_amount)}</span>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Shipping Address</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-sm">
              <p className="font-medium text-foreground">
                {order.shipping_address.first_name} {order.shipping_address.last_name}
              </p>
              <p className="mt-1 text-muted-foreground">{order.shipping_address.address_line1}</p>
              {order.shipping_address.address_line2 && (
                <p className="text-muted-foreground">{order.shipping_address.address_line2}</p>
              )}
              <p className="text-muted-foreground">
                {order.shipping_address.city}, {order.shipping_address.state} -{' '}
                {order.shipping_address.postal_code}
              </p>
              <p className="text-muted-foreground">Phone: {order.shipping_address.phone}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Cancel button */}
      {canCancel && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Cancel Order</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="mb-4 text-sm text-muted-foreground">
              You can cancel this order since it has not been processed yet.
            </p>
            <AlertDialog>
              <AlertDialogTrigger
                render={<Button variant="destructive" loading={cancelling} />}
              >
                Cancel Order
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Cancel Order</AlertDialogTitle>
                  <AlertDialogDescription>
                    Are you sure you want to cancel this order? This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Keep Order</AlertDialogCancel>
                  <AlertDialogAction variant="destructive" onClick={handleCancel}>
                    Cancel Order
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
