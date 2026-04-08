'use client';

import { ShoppingBagIcon } from '@heroicons/react/24/outline';
import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';

import { Card, CardContent } from '@/components/ui/card';
import { EmptyState } from '@/components/ui/empty-state';
import OrderCardSkeleton from '@/components/skeleton/OrderCardSkeleton';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatDate, formatPrice, statusColors } from '@/lib/utils';
import { Order } from '@/types';

// The API returns orders as a flat array (unwrapped by axios interceptor).
// Pagination meta is included at the top level of the API response envelope
// and extracted separately.
type OrdersResponse = Order[];

export default function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchOrders = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const params: Record<string, unknown> = { limit: 10 };

      const { data } = await api.get<OrdersResponse>(ROUTES.ORDERS.LIST, params);

      const ordersList = Array.isArray(data) ? data : [];
      setOrders(ordersList);

      // TODO: wire up cursor pagination when backend returns meta
    } catch {
      setError('Failed to load orders. Please try again.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOrders();
  }, [fetchOrders]);

  if (loading) {
    return <OrderCardSkeleton count={3} />;
  }

  if (error) {
    return (
      <Card className="p-6">
        <p className="text-center text-sm text-red-500">{error}</p>
      </Card>
    );
  }

  if (orders.length === 0) {
    return (
      <Card className="p-6">
        <EmptyState
          icon={<ShoppingBagIcon strokeWidth={1} className="h-16 w-16 text-muted-foreground" />}
          title="No orders yet"
          description="Looks like you have not placed any orders yet."
          actionLabel="Start Shopping"
          actionHref="/products"
        />
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold text-foreground">Order History</h2>

      {orders.map((order) => (
        <Link
          key={order.id}
          href={`/account/orders/${order.id}`}
        >
          <Card className="transition-colors hover:border-primary/50">
            <CardContent>
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
                  <p className="mt-1 text-sm text-muted-foreground">
                    {formatDate(order.created_at)} &middot; {order.item_count}{' '}
                    {order.item_count === 1 ? 'item' : 'items'}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-lg font-semibold text-foreground">
                    {formatPrice(order.total_amount)}
                  </p>
                  {order.tracking_number && (
                    <p className="text-xs text-muted-foreground">
                      Tracking: {order.tracking_number}
                    </p>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </Link>
      ))}
    </div>
  );
}
