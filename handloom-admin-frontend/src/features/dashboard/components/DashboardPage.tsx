import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, ArrowRight, TrendingUp } from 'lucide-react';
import { Link } from 'react-router-dom';

import { inventoryApi } from '@/features/inventory/api';
import { ordersApi } from '@/features/orders/api';
import { DashboardSkeleton } from '@/shared/components/loading';
import { Badge, Card } from '@/shared/components/ui';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency } from '@/shared/utils/currency';

function CardSkeleton() {
  return (
    <div className="animate-pulse bg-white rounded-xl p-6 shadow-sm">
      <div className="h-4 bg-gray-200 rounded w-1/3 mb-4" />
      <div className="space-y-3">
        <div className="h-3 bg-gray-200 rounded" />
        <div className="h-3 bg-gray-200 rounded w-5/6" />
        <div className="h-3 bg-gray-200 rounded w-4/6" />
      </div>
    </div>
  );
}

export function DashboardPage() {
  // Fetch recent orders
  const { data: recentOrders, isLoading: recentOrdersLoading } = useQuery({
    queryKey: ['recent-orders'],
    queryFn: () => ordersApi.list({ limit: 5 }),
  });

  // Fetch low stock items
  const { data: lowStockItems, isLoading: lowStockLoading } = useQuery({
    queryKey: ['low-stock', { limit: 5 }],
    queryFn: () => inventoryApi.getLowStock({ limit: 5 }),
  });

  if (recentOrdersLoading && lowStockLoading) {
    return <DashboardSkeleton />;
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="page-header">
        <h1 className="page-title">Dashboard</h1>
        <p className="page-subtitle">Welcome back! Here's what's happening with your store.</p>
      </div>

      {/* Analytics link banner */}
      <div className="rounded-xl bg-primary-50 border border-primary-200 p-4 flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-primary-900">
            Analytics &amp; metrics have moved to Dashboards
          </p>
          <p className="text-xs text-primary-700 mt-0.5">
            Revenue, orders, funnel, geography and RUM are now powered by Neon Data API.
          </p>
        </div>
        <Link
          to="/dashboards"
          className="text-sm text-primary-600 hover:text-primary-700 flex items-center gap-1 shrink-0"
        >
          View Dashboards <ArrowRight className="w-4 h-4" />
        </Link>
      </div>

      {/* Bottom Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Orders */}
        {recentOrdersLoading ? (
          <CardSkeleton />
        ) : (
          <Card padding="none">
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">Recent Orders</h3>
                  <p className="text-sm text-gray-500">Latest orders from customers</p>
                </div>
                <Link
                  to="/orders"
                  className="text-sm text-primary-600 hover:text-primary-700 flex items-center gap-1"
                >
                  View all <ArrowRight className="w-4 h-4" />
                </Link>
              </div>
            </div>
            <div className="divide-y divide-gray-200">
              {(recentOrders?.items || []).slice(0, 5).map((order) => (
                <div key={order.id} className="p-4 hover:bg-gray-50 transition-colors">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {order.order_number || order.id.slice(0, 8)}
                      </p>
                      <p className="text-sm text-gray-500">
                        {order.customer_name || order.customer_email}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-medium text-gray-900">
                        {formatCurrency(order.total_amount)}
                      </p>
                      <Badge variant={getStatusBadgeVariant(order.status)}>{order.status}</Badge>
                    </div>
                  </div>
                </div>
              ))}
              {(!recentOrders?.items || recentOrders.items.length === 0) && (
                <div className="p-8 text-center text-gray-500">No recent orders</div>
              )}
            </div>
          </Card>
        )}

        {/* Low Stock Alert */}
        {lowStockLoading ? (
          <CardSkeleton />
        ) : (
          <Card padding="none">
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-red-100 rounded-lg">
                    <AlertTriangle className="w-5 h-5 text-red-600" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">Low Stock Alert</h3>
                    <p className="text-sm text-gray-500">Products that need attention</p>
                  </div>
                </div>
                <Link
                  to="/inventory"
                  className="text-sm text-primary-600 hover:text-primary-700 flex items-center gap-1"
                >
                  View all <ArrowRight className="w-4 h-4" />
                </Link>
              </div>
            </div>
            <div className="divide-y divide-gray-200">
              {(lowStockItems?.items || []).slice(0, 5).map((item) => (
                <div key={item.product_id} className="p-4 hover:bg-gray-50 transition-colors">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-900">{item.product_name}</p>
                      <p className="text-sm text-gray-500">SKU: {item.sku}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-medium text-red-600">{item.available_qty} left</p>
                      <p className="text-xs text-gray-500">Threshold: {item.low_stock_threshold}</p>
                    </div>
                  </div>
                </div>
              ))}
              {(!lowStockItems?.items || lowStockItems.items.length === 0) && (
                <div className="p-8 text-center text-gray-500">
                  <TrendingUp className="w-8 h-8 mx-auto mb-2 text-green-500" />
                  All products are well stocked!
                </div>
              )}
            </div>
          </Card>
        )}
      </div>
    </div>
  );
}
