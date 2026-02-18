import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ComponentType } from 'react';
import { lazy, Suspense, useEffect } from 'react';
import { Toaster } from 'react-hot-toast';
import { BrowserRouter, Navigate, Outlet, Route, Routes } from 'react-router-dom';

import { authApi } from './api';
import { ErrorBoundary, LoadingOverlay, PageLoading } from './components/common';
import { MainLayout } from './components/layout';
// Eagerly loaded pages (critical path)
import { LoginPage } from './pages/auth/LoginPage';
import { useAuthStore } from './stores/authStore';

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
const Dashboard = withSuspense(lazy(() => import('./pages/dashboard/DashboardPage').then((m) => ({ default: m.DashboardPage }))));
const Categories = withSuspense(lazy(() => import('./pages/categories').then((m) => ({ default: m.CategoriesPage }))));
const Products = withSuspense(lazy(() => import('./pages/products').then((m) => ({ default: m.ProductsPage }))));
const Orders = withSuspense(lazy(() => import('./pages/orders').then((m) => ({ default: m.OrdersPage }))));
const OrderDetail = withSuspense(lazy(() => import('./pages/orders/OrderDetailPage').then((m) => ({ default: m.OrderDetailPage }))));
const Customers = withSuspense(lazy(() => import('./pages/customers').then((m) => ({ default: m.CustomersPage }))));
const Artisans = withSuspense(lazy(() => import('./pages/artisans').then((m) => ({ default: m.ArtisansPage }))));
const PricingRules = withSuspense(lazy(() => import('./pages/pricing').then((m) => ({ default: m.PricingRulesPage }))));
const Coupons = withSuspense(lazy(() => import('./pages/coupons').then((m) => ({ default: m.CouponsPage }))));
const Inventory = withSuspense(lazy(() => import('./pages/inventory').then((m) => ({ default: m.InventoryPage }))));
const Analytics = withSuspense(lazy(() => import('./pages/analytics').then((m) => ({ default: m.AnalyticsPage }))));
const Reports = withSuspense(lazy(() => import('./pages/reports').then((m) => ({ default: m.ReportsPage }))));
const Notifications = withSuspense(lazy(() => import('./pages/notifications').then((m) => ({ default: m.NotificationsPage }))));
const BulkOperations = withSuspense(lazy(() => import('./pages/bulk').then((m) => ({ default: m.BulkOperationsPage }))));
const Users = withSuspense(lazy(() => import('./pages/settings/UsersPage').then((m) => ({ default: m.UsersPage }))));
const Settings = withSuspense(lazy(() => import('./pages/settings').then((m) => ({ default: m.SettingsPage }))));
const NotFound = withSuspense(lazy(() => import('./pages/NotFoundPage').then((m) => ({ default: m.NotFoundPage }))));

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
  },
});

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

function App() {
  // On mount, check if we have a valid session via HTTP-only cookie
  useEffect(() => {
    const { login, logout } = useAuthStore.getState();
    authApi
      .getCurrentUser()
      .then((user) => {
        login(user);
      })
      .catch(() => {
        logout();
      });
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ErrorBoundary>
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
        </ErrorBoundary>
      </BrowserRouter>

      {/* Toast notifications */}
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          className: 'bg-white text-gray-800 shadow-lg rounded-lg px-4 py-3',
          success: {
            iconTheme: {
              primary: '#10b981',
              secondary: '#fff',
            },
          },
          error: {
            iconTheme: {
              primary: '#ef4444',
              secondary: '#fff',
            },
          },
        }}
      />
    </QueryClientProvider>
  );
}

export default App;
