'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';

import Button from '@/components/common/Button';
import api from '@/lib/api';
import { formatDate, formatPrice, statusColors } from '@/lib/utils';
import { Order } from '@/types';

interface OrdersResponse {
  data: Order[];
  meta: {
    limit: number;
    next_cursor: string;
    has_more: boolean;
  };
}

export default function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [cursor, setCursor] = useState<string>('');
  const [hasMore, setHasMore] = useState(false);

  const fetchOrders = useCallback(async (nextCursor?: string) => {
    const isLoadMore = !!nextCursor;
    if (isLoadMore) {
      setLoadingMore(true);
    } else {
      setLoading(true);
    }

    try {
      const params: Record<string, unknown> = { limit: 10 };
      if (nextCursor) params.cursor = nextCursor;

      const { data } = await api.get<OrdersResponse>('/api/v1/store/orders', params);

      const ordersList = data.data || [];
      if (isLoadMore) {
        setOrders((prev) => [...prev, ...ordersList]);
      } else {
        setOrders(ordersList);
      }

      setCursor(data.meta?.next_cursor || '');
      setHasMore(data.meta?.has_more || false);
    } catch {
      // Failed to load orders
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, []);

  useEffect(() => {
    fetchOrders();
  }, [fetchOrders]);

  if (loading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-28 animate-pulse rounded-lg border border-border bg-white"
          />
        ))}
      </div>
    );
  }

  if (orders.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-white p-12 text-center">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={1}
          stroke="currentColor"
          className="mx-auto mb-4 h-16 w-16 text-muted"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M15.75 10.5V6a3.75 3.75 0 1 0-7.5 0v4.5m11.356-1.993 1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 0 1-1.12-1.243l1.264-12A1.125 1.125 0 0 1 5.513 7.5h12.974c.576 0 1.059.435 1.119 1.007ZM8.625 10.5a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm7.5 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
          />
        </svg>
        <h2 className="mb-2 text-lg font-semibold text-foreground">No orders yet</h2>
        <p className="mb-4 text-muted">
          Looks like you have not placed any orders yet.
        </p>
        <Link href="/products">
          <Button>Start Shopping</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold text-foreground">Order History</h2>

      {orders.map((order) => (
        <Link
          key={order.id}
          href={`/account/orders/${order.id}`}
          className="block rounded-lg border border-border bg-white p-4 transition-colors hover:border-primary/50 sm:p-5"
        >
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <div className="flex items-center gap-2">
                <p className="font-semibold text-foreground">
                  #{order.order_number}
                </p>
                <span
                  className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${statusColors[order.status]}`}
                >
                  {order.status}
                </span>
              </div>
              <p className="mt-1 text-sm text-muted">
                {formatDate(order.created_at)} &middot; {order.item_count}{' '}
                {order.item_count === 1 ? 'item' : 'items'}
              </p>
            </div>
            <div className="text-right">
              <p className="text-lg font-semibold text-foreground">
                {formatPrice(order.total_amount)}
              </p>
              {order.tracking_number && (
                <p className="text-xs text-muted">
                  Tracking: {order.tracking_number}
                </p>
              )}
            </div>
          </div>
        </Link>
      ))}

      {hasMore && (
        <div className="pt-2 text-center">
          <Button
            variant="outline"
            onClick={() => fetchOrders(cursor)}
            loading={loadingMore}
          >
            Load More Orders
          </Button>
        </div>
      )}
    </div>
  );
}
