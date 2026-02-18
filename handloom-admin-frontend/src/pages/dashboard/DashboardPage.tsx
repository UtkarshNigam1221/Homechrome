import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  AlertTriangle,
  ArrowRight,
  DollarSign,
  Package,
  ShoppingCart,
  TrendingUp,
  Users,
} from 'lucide-react';
import { Link } from 'react-router-dom';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { analyticsApi, inventoryApi, ordersApi } from '../../api';
import {
  Badge,
  Card,
  CardHeader,
  DashboardSkeleton,
  getStatusBadgeVariant,
  StatCard,
} from '../../components/common';
import { formatCurrency } from '@/utils/currency';

export function DashboardPage() {
  // Fetch dashboard stats
  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: analyticsApi.getDashboard,
  });

  // Fetch sales analytics
  const { data: salesData } = useQuery({
    queryKey: ['sales-analytics'],
    queryFn: () => analyticsApi.getSales({ period: 'last_30_days' }),
  });

  // Fetch top products
  const { data: topProducts } = useQuery({
    queryKey: ['top-products'],
    queryFn: () => analyticsApi.getTopProducts({ limit: 5 }),
  });

  // Fetch recent orders
  const { data: recentOrders } = useQuery({
    queryKey: ['recent-orders'],
    queryFn: () => ordersApi.list({ limit: 5 }),
  });

  // Fetch low stock items
  const { data: lowStockItems } = useQuery({
    queryKey: ['low-stock'],
    queryFn: () => inventoryApi.getLowStock({ limit: 5 }),
  });

  // Show skeleton loading state
  if (statsLoading) {
    return <DashboardSkeleton />;
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="page-header">
        <h1 className="page-title">Dashboard</h1>
        <p className="page-subtitle">Welcome back! Here's what's happening with your store.</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Total Revenue"
          value={formatCurrency(stats?.total_revenue || 0)}
          change={stats?.revenue_change}
          changeLabel="vs last month"
          icon={<DollarSign className="w-6 h-6" />}
        />
        <StatCard
          title="Total Orders"
          value={stats?.total_orders || 0}
          change={stats?.orders_change}
          changeLabel="vs last month"
          icon={<ShoppingCart className="w-6 h-6" />}
        />
        <StatCard
          title="Total Customers"
          value={stats?.total_customers || 0}
          icon={<Users className="w-6 h-6" />}
        />
        <StatCard
          title="Active Products"
          value={stats?.active_products || 0}
          icon={<Package className="w-6 h-6" />}
        />
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Sales Chart */}
        <Card>
          <CardHeader
            title="Sales Overview"
            subtitle="Revenue trend over the last 30 days"
            action={
              <Link
                to="/analytics"
                className="text-sm text-primary-600 hover:text-primary-700 flex items-center gap-1"
              >
                View details <ArrowRight className="w-4 h-4" />
              </Link>
            }
          />
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={salesData?.data || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis
                  dataKey="date"
                  tickFormatter={(value) => format(new Date(value), 'MMM d')}
                  stroke="#9ca3af"
                  fontSize={12}
                />
                <YAxis
                  tickFormatter={(value) => `₹${(value / 100000).toFixed(0)}K`}
                  stroke="#9ca3af"
                  fontSize={12}
                />
                <Tooltip
                  formatter={(value) => [formatCurrency(Number(value) || 0), 'Revenue']}
                  labelFormatter={(label) => format(new Date(label), 'MMM d, yyyy')}
                />
                <Line
                  type="monotone"
                  dataKey="revenue"
                  stroke="#ec7428"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4, strokeWidth: 0 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </Card>

        {/* Top Products Chart */}
        <Card>
          <CardHeader
            title="Top Products"
            subtitle="Best selling products by revenue"
            action={
              <Link
                to="/products"
                className="text-sm text-primary-600 hover:text-primary-700 flex items-center gap-1"
              >
                View all <ArrowRight className="w-4 h-4" />
              </Link>
            }
          />
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={topProducts || []} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis
                  type="number"
                  tickFormatter={(value) => `₹${(value / 100000).toFixed(0)}K`}
                  stroke="#9ca3af"
                  fontSize={12}
                />
                <YAxis
                  type="category"
                  dataKey="product_name"
                  width={150}
                  stroke="#9ca3af"
                  fontSize={12}
                  tickFormatter={(value) =>
                    value.length > 20 ? `${value.slice(0, 20)}...` : value
                  }
                />
                <Tooltip formatter={(value) => [formatCurrency(Number(value) || 0), 'Revenue']} />
                <Bar dataKey="revenue" fill="#ec7428" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>

      {/* Bottom Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Orders */}
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
                      {formatCurrency(order.total_price)}
                    </p>
                    <Badge variant={getStatusBadgeVariant(order.order_status)}>
                      {order.order_status}
                    </Badge>
                  </div>
                </div>
              </div>
            ))}
            {(!recentOrders?.items || recentOrders.items.length === 0) && (
              <div className="p-8 text-center text-gray-500">No recent orders</div>
            )}
          </div>
        </Card>

        {/* Low Stock Alert */}
        <Card padding="none">
          <div className="p-6 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-red-100 rounded-lg">
                  <AlertTriangle className="w-5 h-5 text-red-600" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">Low Stock Alert</h3>
                  <p className="text-sm text-gray-500">
                    {stats?.low_stock_count || 0} products need attention
                  </p>
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
      </div>
    </div>
  );
}
