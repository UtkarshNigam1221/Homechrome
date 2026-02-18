import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import { DollarSign, ShoppingCart, TrendingUp, Users } from 'lucide-react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { analyticsApi } from '@/features/analytics/api';
import { PageLoading } from '@/shared/components/loading';
import { Card, CardHeader, StatCard } from '@/shared/components/ui';
import { CHART_COLORS, PIE_COLORS } from '@/shared/utils/chartColors';
import { formatCurrency } from '@/shared/utils/currency';

export function AnalyticsPage() {
  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ['analytics-dashboard'],
    queryFn: analyticsApi.getDashboard,
  });

  const { data: salesData } = useQuery({
    queryKey: ['analytics-sales'],
    queryFn: () => analyticsApi.getSales({ period: 'last_30_days' }),
  });

  const { data: topProducts } = useQuery({
    queryKey: ['analytics-top-products'],
    queryFn: () => analyticsApi.getTopProducts({ limit: 10 }),
  });

  const { data: topCategories } = useQuery({
    queryKey: ['analytics-top-categories'],
    queryFn: () => analyticsApi.getTopCategories({ limit: 5 }),
  });

  if (statsLoading) {
    return <PageLoading />;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="page-title">Analytics</h1>
        <p className="page-subtitle">Track your store performance and insights</p>
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
          title="Avg Order Value"
          value={formatCurrency((stats?.total_revenue || 0) / (stats?.total_orders || 1))}
          icon={<TrendingUp className="w-6 h-6" />}
        />
      </div>

      {/* Revenue Chart */}
      <Card>
        <CardHeader title="Revenue Over Time" subtitle="Last 30 days" />
        <div className="h-80">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={salesData?.data || []}>
              <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.grid} />
              <XAxis
                dataKey="date"
                tickFormatter={(value) => format(new Date(value), 'MMM d')}
                stroke={CHART_COLORS.axis}
                fontSize={12}
              />
              <YAxis
                tickFormatter={(value) => `₹${(value / 100000).toFixed(0)}K`}
                stroke={CHART_COLORS.axis}
                fontSize={12}
              />
              <Tooltip
                formatter={(value) => [formatCurrency(Number(value) || 0), 'Revenue']}
                labelFormatter={(label) => format(new Date(label), 'MMM d, yyyy')}
              />
              <Line
                type="monotone"
                dataKey="revenue"
                stroke={CHART_COLORS.primary}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, strokeWidth: 0 }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Top Products */}
        <Card>
          <CardHeader title="Top Selling Products" subtitle="By revenue" />
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={topProducts || []} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.grid} />
                <XAxis
                  type="number"
                  tickFormatter={(value) => `₹${(value / 100000).toFixed(0)}K`}
                  stroke={CHART_COLORS.axis}
                  fontSize={12}
                />
                <YAxis
                  type="category"
                  dataKey="product_name"
                  width={120}
                  stroke={CHART_COLORS.axis}
                  fontSize={12}
                  tickFormatter={(value) =>
                    value.length > 15 ? `${value.slice(0, 15)}...` : value
                  }
                />
                <Tooltip formatter={(value) => [formatCurrency(Number(value) || 0), 'Revenue']} />
                <Bar dataKey="revenue" fill={CHART_COLORS.primary} radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>

        {/* Top Categories */}
        <Card>
          <CardHeader title="Revenue by Category" subtitle="Top 5 categories" />
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={topCategories || []}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={100}
                  paddingAngle={5}
                  dataKey="revenue"
                  nameKey="category_name"
                  label={({ name, percent }) => `${name} (${((percent || 0) * 100).toFixed(0)}%)`}
                  labelLine={false}
                >
                  {(topCategories || []).map((_, index) => (
                    <Cell key={`cell-${index}`} fill={PIE_COLORS[index % PIE_COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(value) => [formatCurrency(Number(value) || 0), 'Revenue']} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>

      {/* Orders Chart */}
      <Card>
        <CardHeader title="Orders Over Time" subtitle="Last 30 days" />
        <div className="h-80">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={salesData?.data || []}>
              <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.grid} />
              <XAxis
                dataKey="date"
                tickFormatter={(value) => format(new Date(value), 'MMM d')}
                stroke={CHART_COLORS.axis}
                fontSize={12}
              />
              <YAxis stroke={CHART_COLORS.axis} fontSize={12} />
              <Tooltip
                formatter={(value) => [Number(value) || 0, 'Orders']}
                labelFormatter={(label) => format(new Date(label), 'MMM d, yyyy')}
              />
              <Bar dataKey="orders" fill={CHART_COLORS.blue} radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </Card>
    </div>
  );
}
