import type { ComponentType } from 'react';
import { lazy, Suspense } from 'react';
import { Navigate, Outlet, Route, Routes } from 'react-router-dom';

import { LoadingOverlay, PageLoading } from '@/shared/components/loading';
import { MainLayout } from '@/shared/components/layout';
import { useAuthStore } from '@/shared/stores/authStore';
import { LoginPage } from '@/features/auth';

// Suspense wrapper to reduce repetitive <Suspense> in routes
function withSuspense<P extends object>(LazyComponent: ComponentType<P>) {
  return function SuspenseWrapper(props: P) {
    return (
      <Suspense fallback={<PageLoading message="Loading page..." />}>
        <LazyComponent {...props} />
      </Suspense>
    );
  };
}

// Lazy loaded pages - only loaded when navigating to that route
const Dashboard = withSuspense(
  lazy(() => import('@/features/dashboard').then((m) => ({ default: m.DashboardPage })))
);
const Categories = withSuspense(
  lazy(() => import('@/features/categories').then((m) => ({ default: m.CategoriesPage })))
);
const Products = withSuspense(
  lazy(() => import('@/features/products').then((m) => ({ default: m.ProductsPage })))
);
const Orders = withSuspense(
  lazy(() => import('@/features/orders').then((m) => ({ default: m.OrdersPage })))
);
const OrderDetail = withSuspense(
  lazy(() => import('@/features/orders').then((m) => ({ default: m.OrderDetailPage })))
);
const Customers = withSuspense(
  lazy(() => import('@/features/customers').then((m) => ({ default: m.CustomersPage })))
);
const Artisans = withSuspense(
  lazy(() => import('@/features/artisans').then((m) => ({ default: m.ArtisansPage })))
);
const PricingRules = withSuspense(
  lazy(() => import('@/features/pricing').then((m) => ({ default: m.PricingRulesPage })))
);
const Coupons = withSuspense(
  lazy(() => import('@/features/coupons').then((m) => ({ default: m.CouponsPage })))
);
const Inventory = withSuspense(
  lazy(() => import('@/features/inventory').then((m) => ({ default: m.InventoryPage })))
);
const Analytics = withSuspense(
  lazy(() => import('@/features/analytics').then((m) => ({ default: m.AnalyticsPage })))
);
const Reports = withSuspense(
  lazy(() => import('@/features/reports').then((m) => ({ default: m.ReportsPage })))
);
const Notifications = withSuspense(
  lazy(() => import('@/features/notifications').then((m) => ({ default: m.NotificationsPage })))
);
const BulkOperations = withSuspense(
  lazy(() => import('@/features/bulk').then((m) => ({ default: m.BulkOperationsPage })))
);
const Users = withSuspense(
  lazy(() => import('@/features/settings').then((m) => ({ default: m.UsersPage })))
);
const Settings = withSuspense(
  lazy(() => import('@/features/settings').then((m) => ({ default: m.SettingsPage })))
);
const NotFound = withSuspense(
  lazy(() =>
    import('@/shared/components/NotFoundPage').then((m) => ({ default: m.NotFoundPage }))
  )
);

// Protected Route wrapper
function ProtectedRoute() {
  const { isAuthenticated, isLoading } = useAuthStore();

  if (isLoading) {
    return <LoadingOverlay message="Loading..." />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}

// Admin Route wrapper (for admin-only routes)
function AdminRoute() {
  const { user } = useAuthStore();

  if (user?.role !== 'ADMIN') {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}

// Public Route wrapper (redirect to home if already logged in)
function PublicRoute() {
  const { isAuthenticated, isLoading } = useAuthStore();

  if (isLoading) {
    return <LoadingOverlay message="Loading..." />;
  }

  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}

export function AppRoutes() {
  return (
    <Routes>
      {/* Public Routes */}
      <Route element={<PublicRoute />}>
        <Route path="/login" element={<LoginPage />} />
      </Route>

      {/* Protected Routes */}
      <Route element={<ProtectedRoute />}>
        <Route element={<MainLayout />}>
          {/* Dashboard */}
          <Route path="/" element={<Dashboard />} />

          {/* Catalog */}
          <Route path="/categories" element={<Categories />} />
          <Route path="/products" element={<Products />} />
          <Route path="/inventory" element={<Inventory />} />

          {/* Sales */}
          <Route path="/orders" element={<Orders />} />
          <Route path="/orders/:id" element={<OrderDetail />} />
          <Route path="/customers" element={<Customers />} />
          <Route path="/artisans" element={<Artisans />} />

          {/* Marketing */}
          <Route path="/pricing" element={<PricingRules />} />
          <Route path="/coupons" element={<Coupons />} />

          {/* Analytics & Reports */}
          <Route path="/analytics" element={<Analytics />} />
          <Route path="/reports" element={<Reports />} />

          {/* Operations */}
          <Route path="/bulk" element={<BulkOperations />} />
          <Route path="/notifications" element={<Notifications />} />

          {/* Admin Only Routes */}
          <Route element={<AdminRoute />}>
            <Route path="/users" element={<Users />} />
          </Route>

          {/* Settings */}
          <Route path="/settings" element={<Settings />} />
          <Route path="/settings/*" element={<Settings />} />
        </Route>
      </Route>

      {/* 404 */}
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}
