import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { lazy, Suspense, useEffect } from 'react';
import { Toaster } from 'react-hot-toast';
import { BrowserRouter, Navigate, Outlet, Route, Routes } from 'react-router-dom';

import { authApi } from './api';
import { LoadingOverlay, PageLoading } from './components/common';
import { MainLayout } from './components/layout';
// Eagerly loaded pages (critical path)
import { LoginPage } from './pages/auth/LoginPage';
import { useAuthStore } from './stores/authStore';

// Lazy loaded pages - only loaded when navigating to that route
const DashboardPage = lazy(() =>
  import('./pages/dashboard/DashboardPage').then((m) => ({ default: m.DashboardPage }))
);
const CategoriesPage = lazy(() =>
  import('./pages/categories').then((m) => ({ default: m.CategoriesPage }))
);
const ProductsPage = lazy(() =>
  import('./pages/products').then((m) => ({ default: m.ProductsPage }))
);
const OrdersPage = lazy(() => import('./pages/orders').then((m) => ({ default: m.OrdersPage })));
const OrderDetailPage = lazy(() =>
  import('./pages/orders/OrderDetailPage').then((m) => ({ default: m.OrderDetailPage }))
);
const CustomersPage = lazy(() =>
  import('./pages/customers').then((m) => ({ default: m.CustomersPage }))
);
const ArtisansPage = lazy(() =>
  import('./pages/artisans').then((m) => ({ default: m.ArtisansPage }))
);
const DesignsPage = lazy(() => import('./pages/designs').then((m) => ({ default: m.DesignsPage })));
const PricingRulesPage = lazy(() =>
  import('./pages/pricing').then((m) => ({ default: m.PricingRulesPage }))
);
const CouponsPage = lazy(() => import('./pages/coupons').then((m) => ({ default: m.CouponsPage })));
const InventoryPage = lazy(() =>
  import('./pages/inventory').then((m) => ({ default: m.InventoryPage }))
);
const AnalyticsPage = lazy(() =>
  import('./pages/analytics').then((m) => ({ default: m.AnalyticsPage }))
);
const ReportsPage = lazy(() => import('./pages/reports').then((m) => ({ default: m.ReportsPage })));
const NotificationsPage = lazy(() =>
  import('./pages/notifications').then((m) => ({ default: m.NotificationsPage }))
);
const BulkOperationsPage = lazy(() =>
  import('./pages/bulk').then((m) => ({ default: m.BulkOperationsPage }))
);
const UsersPage = lazy(() =>
  import('./pages/settings/UsersPage').then((m) => ({ default: m.UsersPage }))
);
const SettingsPage = lazy(() =>
  import('./pages/settings').then((m) => ({ default: m.SettingsPage }))
);

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

// Suspense fallback for lazy-loaded pages
function LazyPageFallback() {
  return <PageLoading message="Loading page..." />;
}

function App() {
  // Validate persisted auth on startup
  useEffect(() => {
    const { accessToken, setUser, logout, setLoading } = useAuthStore.getState();
    if (!accessToken) {
      setLoading(false);
      return;
    }
    authApi
      .getCurrentUser()
      .then((user) => {
        setUser(user);
        setLoading(false);
      })
      .catch(() => {
        logout();
      });
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Public Routes */}
          <Route element={<PublicRoute />}>
            <Route path="/login" element={<LoginPage />} />
          </Route>

          {/* Protected Routes */}
          <Route element={<ProtectedRoute />}>
            <Route element={<MainLayout />}>
              {/* Dashboard */}
              <Route
                path="/"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <DashboardPage />
                  </Suspense>
                }
              />

              {/* Catalog */}
              <Route
                path="/categories"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <CategoriesPage />
                  </Suspense>
                }
              />
              <Route
                path="/designs"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <DesignsPage />
                  </Suspense>
                }
              />
              <Route
                path="/products"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <ProductsPage />
                  </Suspense>
                }
              />
              <Route
                path="/products/:id"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <ProductsPage />
                  </Suspense>
                }
              />
              <Route
                path="/inventory"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <InventoryPage />
                  </Suspense>
                }
              />

              {/* Sales */}
              <Route
                path="/orders"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <OrdersPage />
                  </Suspense>
                }
              />
              <Route
                path="/orders/:id"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <OrderDetailPage />
                  </Suspense>
                }
              />
              <Route
                path="/customers"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <CustomersPage />
                  </Suspense>
                }
              />
              <Route
                path="/artisans"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <ArtisansPage />
                  </Suspense>
                }
              />

              {/* Marketing */}
              <Route
                path="/pricing"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <PricingRulesPage />
                  </Suspense>
                }
              />
              <Route
                path="/coupons"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <CouponsPage />
                  </Suspense>
                }
              />

              {/* Analytics & Reports */}
              <Route
                path="/analytics"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <AnalyticsPage />
                  </Suspense>
                }
              />
              <Route
                path="/reports"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <ReportsPage />
                  </Suspense>
                }
              />

              {/* Operations */}
              <Route
                path="/bulk"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <BulkOperationsPage />
                  </Suspense>
                }
              />
              <Route
                path="/notifications"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <NotificationsPage />
                  </Suspense>
                }
              />

              {/* Admin Only Routes */}
              <Route element={<AdminRoute />}>
                <Route
                  path="/users"
                  element={
                    <Suspense fallback={<LazyPageFallback />}>
                      <UsersPage />
                    </Suspense>
                  }
                />
              </Route>

              {/* Settings */}
              <Route
                path="/settings"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <SettingsPage />
                  </Suspense>
                }
              />
              <Route
                path="/settings/*"
                element={
                  <Suspense fallback={<LazyPageFallback />}>
                    <SettingsPage />
                  </Suspense>
                }
              />
            </Route>
          </Route>

          {/* Catch all - redirect to home */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>

      {/* Toast notifications */}
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          style: {
            background: '#fff',
            color: '#363636',
            boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
            borderRadius: '0.5rem',
            padding: '12px 16px',
          },
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
