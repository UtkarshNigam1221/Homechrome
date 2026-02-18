# Frontend Full Cleanup Refactor — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all 25 identified issues in handloom-admin-frontend — 5 critical, 11 quality, 9 nice-to-have.

**Architecture:** Module-by-module refactor: shared infrastructure first (test framework, utilities, error boundary, config), then routing/layout, then page-by-page fixes, then testing and polish. Each task produces a working codebase.

**Tech Stack:** React 19, TypeScript 5.9, Vite 7, Tailwind CSS 4, React Query 5, Zustand 5, react-hook-form 7, Zod 4, Vitest (new), @testing-library/react (new)

**Working directory:** `handloom-admin-frontend/`

---

## Task 1: Set Up Vitest + Testing Library

**Files:**
- Modify: `package.json` (add devDependencies and test scripts)
- Create: `vitest.config.ts`
- Create: `src/test/setup.ts`

**Step 1: Install test dependencies**

Run from `handloom-admin-frontend/`:
```bash
npm install -D vitest @testing-library/react @testing-library/jest-dom @testing-library/user-event jsdom
```

**Step 2: Create vitest.config.ts**

```ts
/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: false,
  },
});
```

**Step 3: Create src/test/setup.ts**

```ts
import '@testing-library/jest-dom/vitest';
```

**Step 4: Add test scripts to package.json**

Add to `"scripts"`:
```json
"test": "vitest run",
"test:watch": "vitest"
```

**Step 5: Run vitest to verify setup works**

Run: `npx vitest run`
Expected: "No test files found" (but no config errors)

**Step 6: Commit**

```bash
git add -A && git commit -m "chore: add vitest + testing-library setup"
```

---

## Task 2: Create formatCurrency Utility (TDD)

**Files:**
- Create: `src/utils/currency.ts`
- Create: `src/utils/__tests__/currency.test.ts`

**Step 1: Write the failing test**

Create `src/utils/__tests__/currency.test.ts`:
```ts
import { describe, expect, it } from 'vitest';

import { formatCurrency } from '../currency';

describe('formatCurrency', () => {
  it('converts paise to INR with no decimals', () => {
    expect(formatCurrency(150000)).toBe('₹1,500');
  });

  it('handles zero', () => {
    expect(formatCurrency(0)).toBe('₹0');
  });

  it('handles small amounts under 1 rupee', () => {
    expect(formatCurrency(50)).toBe('₹1'); // rounds up from 0.50
  });

  it('handles large amounts', () => {
    expect(formatCurrency(10000000)).toBe('₹1,00,000');
  });
});
```

**Step 2: Run test to verify it fails**

Run: `npx vitest run src/utils/__tests__/currency.test.ts`
Expected: FAIL — module not found

**Step 3: Write minimal implementation**

Create `src/utils/currency.ts`:
```ts
const formatter = new Intl.NumberFormat('en-IN', {
  style: 'currency',
  currency: 'INR',
  minimumFractionDigits: 0,
});

export function formatCurrency(paise: number): string {
  return formatter.format(paise / 100);
}
```

**Step 4: Run test to verify it passes**

Run: `npx vitest run src/utils/__tests__/currency.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add src/utils/currency.ts src/utils/__tests__/currency.test.ts && git commit -m "feat: add shared formatCurrency utility"
```

---

## Task 3: Create extractItems Utility (TDD)

**Files:**
- Create: `src/utils/api.ts`
- Create: `src/utils/__tests__/api.test.ts`

**Step 1: Write the failing test**

Create `src/utils/__tests__/api.test.ts`:
```ts
import { describe, expect, it } from 'vitest';

import { extractItems } from '../api';

describe('extractItems', () => {
  it('extracts from { items: [...] }', () => {
    expect(extractItems({ items: [1, 2, 3] })).toEqual([1, 2, 3]);
  });

  it('extracts from named key like { products: [...] }', () => {
    expect(extractItems({ products: [{ id: 1 }] }, 'products')).toEqual([{ id: 1 }]);
  });

  it('returns the data directly if it is an array', () => {
    expect(extractItems([1, 2, 3])).toEqual([1, 2, 3]);
  });

  it('extracts from { data: [...] }', () => {
    expect(extractItems({ data: [1, 2] })).toEqual([1, 2]);
  });

  it('returns empty array for null/undefined', () => {
    expect(extractItems(null)).toEqual([]);
    expect(extractItems(undefined)).toEqual([]);
  });

  it('prefers named key over items', () => {
    expect(extractItems({ items: [1], products: [2] }, 'products')).toEqual([2]);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `npx vitest run src/utils/__tests__/api.test.ts`
Expected: FAIL

**Step 3: Write minimal implementation**

Create `src/utils/api.ts`:
```ts
export function extractItems<T>(data: unknown, key?: string): T[] {
  if (data == null) return [];
  if (Array.isArray(data)) return data as T[];

  const obj = data as Record<string, unknown>;

  if (key && Array.isArray(obj[key])) return obj[key] as T[];
  if (Array.isArray(obj.items)) return obj.items as T[];
  if (Array.isArray(obj.data)) return obj.data as T[];

  return [];
}
```

**Step 4: Run test to verify it passes**

Run: `npx vitest run src/utils/__tests__/api.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add src/utils/api.ts src/utils/__tests__/api.test.ts && git commit -m "feat: add shared extractItems utility"
```

---

## Task 4: Add useDebounce Tests

**Files:**
- Create: `src/hooks/__tests__/useDebounce.test.ts`
- Reference: `src/hooks/useDebounce.ts` (no changes)

**Step 1: Write the test**

Create `src/hooks/__tests__/useDebounce.test.ts`:
```ts
import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { useDebounce } from '../useDebounce';

describe('useDebounce', () => {
  it('returns initial value immediately', () => {
    const { result } = renderHook(() => useDebounce('hello', 300));
    expect(result.current).toBe('hello');
  });

  it('debounces value changes', () => {
    vi.useFakeTimers();
    const { result, rerender } = renderHook(
      ({ value }) => useDebounce(value, 300),
      { initialProps: { value: 'a' } }
    );

    rerender({ value: 'ab' });
    expect(result.current).toBe('a');

    act(() => vi.advanceTimersByTime(300));
    expect(result.current).toBe('ab');

    vi.useRealTimers();
  });
});
```

**Step 2: Run test**

Run: `npx vitest run src/hooks/__tests__/useDebounce.test.ts`
Expected: PASS (existing hook should already work)

**Step 3: Commit**

```bash
git add src/hooks/__tests__/useDebounce.test.ts && git commit -m "test: add useDebounce hook tests"
```

---

## Task 5: Add Store Tests

**Files:**
- Create: `src/stores/__tests__/authStore.test.ts`
- Create: `src/stores/__tests__/uiStore.test.ts`

**Step 1: Write authStore test**

Create `src/stores/__tests__/authStore.test.ts`:
```ts
import { beforeEach, describe, expect, it } from 'vitest';

import { useAuthStore } from '../authStore';

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      isLoading: true,
    });
  });

  it('starts unauthenticated', () => {
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.user).toBeNull();
  });

  it('login sets user and isAuthenticated', () => {
    const user = { id: '1', email: 'test@test.com', name: 'Test', role: 'ADMIN' };
    useAuthStore.getState().login(user as never);
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.user?.email).toBe('test@test.com');
    expect(state.isLoading).toBe(false);
  });

  it('logout clears user', () => {
    useAuthStore.getState().login({ id: '1', email: 'test@test.com', name: 'Test', role: 'ADMIN' } as never);
    useAuthStore.getState().logout();
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.user).toBeNull();
  });
});
```

**Step 2: Write uiStore test**

Create `src/stores/__tests__/uiStore.test.ts`:
```ts
import { beforeEach, describe, expect, it } from 'vitest';

import { useUIStore } from '../uiStore';

describe('uiStore', () => {
  beforeEach(() => {
    useUIStore.setState({
      sidebarOpen: true,
      sidebarCollapsed: false,
      theme: 'light',
    });
  });

  it('toggles sidebar collapsed state', () => {
    useUIStore.getState().toggleSidebarCollapse();
    expect(useUIStore.getState().sidebarCollapsed).toBe(true);
    useUIStore.getState().toggleSidebarCollapse();
    expect(useUIStore.getState().sidebarCollapsed).toBe(false);
  });

  it('toggles sidebar open state', () => {
    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().sidebarOpen).toBe(false);
  });

  it('sets theme', () => {
    useUIStore.getState().setTheme('dark');
    expect(useUIStore.getState().theme).toBe('dark');
  });
});
```

**Step 3: Run both tests**

Run: `npx vitest run src/stores/__tests__/`
Expected: PASS

**Step 4: Commit**

```bash
git add src/stores/__tests__/ && git commit -m "test: add authStore and uiStore tests"
```

---

## Task 6: Fix tsconfig Paths + Delete Dead Config

**Files:**
- Modify: `tsconfig.app.json` — add `paths` and `baseUrl`
- Delete: `tailwind.config.js` (dead file, Tailwind v4 uses `@theme` in `index.css`)
- Modify: `package.json` — remove `@heroicons/react`

**Step 1: Add paths to tsconfig.app.json**

Add inside `"compilerOptions"`:
```json
"baseUrl": ".",
"paths": {
  "@/*": ["./src/*"]
}
```

**Step 2: Delete tailwind.config.js**

```bash
rm tailwind.config.js
```

**Step 3: Remove @heroicons/react**

```bash
npm uninstall @heroicons/react
```

**Step 4: Verify build still works**

Run: `npx tsc --noEmit`
Expected: No errors

Run: `npx vitest run`
Expected: All existing tests still pass

**Step 5: Commit**

```bash
git add -A && git commit -m "chore: add tsconfig paths, remove dead tailwind.config.js and @heroicons/react"
```

---

## Task 7: Create Error Boundary

**Files:**
- Create: `src/components/common/ErrorBoundary.tsx`
- Modify: `src/App.tsx` — wrap routes with ErrorBoundary

**Step 1: Create ErrorBoundary component**

Create `src/components/common/ErrorBoundary.tsx`:
```tsx
import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;

      return (
        <div className="flex min-h-screen items-center justify-center bg-gray-50 p-4">
          <div className="text-center">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Something went wrong</h1>
            <p className="text-gray-600 mb-4">
              {this.state.error?.message || 'An unexpected error occurred.'}
            </p>
            <button
              onClick={() => window.location.reload()}
              className="rounded-lg bg-primary-500 px-4 py-2 text-white hover:bg-primary-600"
            >
              Reload page
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
```

**Step 2: Add ErrorBoundary export to common/index.ts**

In `src/components/common/index.ts`, add:
```ts
export { ErrorBoundary } from './ErrorBoundary';
```

**Step 3: Wrap routes in App.tsx**

In `src/App.tsx`, import `ErrorBoundary` and wrap the `<Routes>` block:
```tsx
import { ErrorBoundary } from './components/common/ErrorBoundary';

// In the JSX, wrap around the <Routes> block:
<ErrorBoundary>
  <Routes>
    {/* ... all routes ... */}
  </Routes>
</ErrorBoundary>
```

**Step 4: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: add ErrorBoundary to catch render errors and chunk load failures"
```

---

## Task 8: Create NotFoundPage + withSuspense Helper + Fix Routes

**Files:**
- Create: `src/pages/NotFoundPage.tsx`
- Modify: `src/App.tsx` — replace catch-all, add withSuspense helper, remove `/products/:id`, fix customer link handling

**Step 1: Create NotFoundPage**

Create `src/pages/NotFoundPage.tsx`:
```tsx
import { useNavigate } from 'react-router-dom';

export function NotFoundPage() {
  const navigate = useNavigate();

  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-gray-300 mb-4">404</h1>
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Page not found</h2>
        <p className="text-gray-600 mb-6">The page you are looking for does not exist.</p>
        <button
          onClick={() => navigate('/')}
          className="rounded-lg bg-primary-500 px-4 py-2 text-white hover:bg-primary-600"
        >
          Back to Dashboard
        </button>
      </div>
    </div>
  );
}
```

**Step 2: Create withSuspense helper in App.tsx**

Replace the `LazyPageFallback` and all 16 `<Suspense>` wrappers with:
```tsx
import type { ComponentType } from 'react';

function withSuspense<P extends object>(LazyComponent: ComponentType<P>) {
  return function SuspenseWrapper(props: P) {
    return (
      <Suspense fallback={<PageLoading message="Loading page..." />}>
        <LazyComponent {...props} />
      </Suspense>
    );
  };
}
```

Then for each lazy import, wrap with `withSuspense`:
```tsx
const Dashboard = withSuspense(lazy(() => import('./pages/dashboard/DashboardPage').then(m => ({ default: m.DashboardPage }))));
```

And routes become:
```tsx
<Route path="/" element={<Dashboard />} />
```

**Step 3: Fix routes**

- Remove the `/products/:id` route entirely (line 167-174 of current App.tsx)
- Replace catch-all `<Navigate to="/" replace />` with `<NotFoundPage />`
- Lazy load NotFoundPage too:
```tsx
const NotFound = withSuspense(lazy(() => import('./pages/NotFoundPage').then(m => ({ default: m.NotFoundPage }))));

// In Routes:
<Route path="*" element={<NotFound />} />
```

**Step 4: Fix customer link in OrderDetailPage**

In `src/pages/orders/OrderDetailPage.tsx` around line 332-338, replace the navigate-to-route behavior with navigating to the customers list page:
```tsx
<Button
  variant="secondary"
  size="sm"
  onClick={() => navigate('/customers')}
>
  View Customer
</Button>
```

**Step 5: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 6: Commit**

```bash
git add -A && git commit -m "feat: add 404 page, withSuspense helper, fix broken routes"
```

---

## Task 9: Fix Vite Proxy + Pricing API Paths

**Files:**
- Modify: `vite.config.ts` — add `/api` proxy

**Step 1: Add /api proxy**

In `vite.config.ts` line 98-104, add `/api` alongside `/admin`:
```ts
proxy: {
  '/admin': {
    target: env.VITE_API_URL || 'http://localhost:8080',
    changeOrigin: true,
    secure: true,
  },
  '/api': {
    target: env.VITE_API_URL || 'http://localhost:8080',
    changeOrigin: true,
    secure: true,
  },
},
```

**Step 2: Verify vite starts**

Run: `npx vite --port 5174 &` then kill it — just verify no config errors.

**Step 3: Commit**

```bash
git add vite.config.ts && git commit -m "fix: add /api proxy for pricing endpoints in vite dev server"
```

---

## Task 10: Normalize API Layer Responses

**Files:**
- Modify: `src/api/index.ts` — normalize all list APIs to return consistent `ListResponse<T>`

**Context:** Currently only `usersApi.list()` normalizes `{ users: [...] }` → `{ items: [...] }`. Other APIs return raw `response.data` which sometimes has `{ products: [...] }` instead of `{ items: [...] }`. Pages work around this with copy-pasted `extractItems()`.

**Step 1: Add a normalizeListResponse helper at the top of api/index.ts**

```ts
function normalizeListResponse<T>(
  data: Record<string, unknown>,
  key: string
): ListResponse<T> {
  const items = (data[key] || data.items || data.data || []) as T[];
  const pagination = (data.pagination as ListResponse<T>['pagination']) || {
    limit: 10,
    has_more: false,
  };
  return { items, pagination };
}
```

**Step 2: Apply to each API's list() function**

For each API that currently does `return response.data`, change to normalize with the appropriate key:

- `categoriesApi.list()` (line ~97): `return normalizeListResponse<Category>(response.data, 'categories');`
- `productsApi.list()` (line ~169): `return normalizeListResponse<Product>(response.data, 'products');`
- `ordersApi.list()` (line ~280): `return normalizeListResponse<Order>(response.data, 'orders');`
- `customersApi.list()` (line ~322): `return normalizeListResponse<Customer>(response.data, 'customers');`
- `artisansApi.list()` (line ~373): `return normalizeListResponse<Artisan>(response.data, 'artisans');`
- `couponsApi.list()` (line ~420): `return normalizeListResponse<Coupon>(response.data, 'coupons');`
- `pricingApi.listRules()` (line ~476): `return normalizeListResponse<PricingRule>(response.data, 'rules');`
- `notificationsApi.list()` (line ~529): `return normalizeListResponse<Notification>(response.data, 'notifications');`
- `bulkApi.list()` (line ~625): `return normalizeListResponse<BulkOperation>(response.data, 'operations');`
- `reportsApi.list()` (line ~692): `return normalizeListResponse<Report>(response.data, 'reports');`
- `auditApi.list()` (line ~787): `return normalizeListResponse<AuditLog>(response.data, 'logs');`
- `inventoryApi.getLowStock()` (line ~243): `return normalizeListResponse<Inventory>(response.data, 'inventories');`

Also simplify `usersApi.list()` to use the same helper.

**Step 3: Fix analytics API unsafe casts**

Replace `as unknown as T[]` patterns (lines ~586-604) with:
```ts
getTopProducts: async (...) => {
  const response = await apiClient.get<...>(...);
  const data = response.data;
  return Array.isArray(data) ? data : (data.products || []);
},
getTopCategories: async (...) => {
  const response = await apiClient.get<...>(...);
  const data = response.data;
  return Array.isArray(data) ? data : (data.categories || []);
},
```

**Step 4: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add src/api/index.ts && git commit -m "refactor: normalize all API list responses with consistent ListResponse<T>"
```

---

## Task 11: Replace Duplicated formatCurrency Across All Pages

**Files to modify (remove local `formatCurrency` and import from `@/utils/currency`):**
1. `src/pages/dashboard/DashboardPage.tsx` (line 71)
2. `src/pages/analytics/AnalyticsPage.tsx` (line 49)
3. `src/pages/products/ProductsPage.tsx` (line 100)
4. `src/pages/orders/OrdersPage.tsx` (line 91)
5. `src/pages/orders/OrderDetailPage.tsx` (line 124)
6. `src/pages/customers/CustomersPage.tsx` (line 66)
7. `src/pages/artisans/ArtisansPage.tsx` (line 67)
8. `src/pages/pricing/PricingRulesPage.tsx` (line 66)

**Step 1: In each file**

- Add `import { formatCurrency } from '@/utils/currency';` at the top
- Delete the local `const formatCurrency = (value: number) => { ... }` definition

**Step 2: Verify build and tests**

Run: `npx tsc --noEmit && npx vitest run`
Expected: PASS

**Step 3: Commit**

```bash
git add -A && git commit -m "refactor: replace 8 duplicated formatCurrency with shared utility"
```

---

## Task 12: Replace Duplicated extractItems Across All Pages

**Files to modify (remove local `extractItems` and use normalized API responses):**
1. `src/pages/products/ProductsPage.tsx` (lines 110-120)
2. `src/pages/products/ProductFormModal.tsx` (lines 340-350)
3. `src/pages/pricing/PricingRuleFormModal.tsx` (lines 170-180)
4. `src/pages/bulk/BulkOperationsPage.tsx` (lines 70-82)
5. `src/pages/inventory/InventoryPage.tsx` (lines 44-55)

**Step 1: In each file**

Since the API layer now returns normalized `ListResponse<T>` (Task 10), the `extractItems` calls can be replaced with direct `.items` access:

For example in ProductsPage, change:
```tsx
const products = extractItems<Product>(productsData, 'products');
```
to:
```tsx
const products = productsData?.items ?? [];
```

Delete the local `extractItems` function from each file. If any edge case still needs extraction, import from `@/utils/api` instead.

**Step 2: Verify build and tests**

Run: `npx tsc --noEmit && npx vitest run`
Expected: PASS

**Step 3: Commit**

```bash
git add -A && git commit -m "refactor: remove 5 duplicated extractItems, use normalized API responses"
```

---

## Task 13: Add Debounce to All Search Pages

**Files to modify (add `useDebounce` to search inputs):**
1. `src/pages/products/ProductsPage.tsx`
2. `src/pages/orders/OrdersPage.tsx`
3. `src/pages/customers/CustomersPage.tsx`
4. `src/pages/artisans/ArtisansPage.tsx`
5. `src/pages/inventory/InventoryPage.tsx`
6. `src/pages/coupons/CouponsPage.tsx`

Note: CategoriesPage and PricingRulesPage do client-side filtering only — debounce is optional for them but can be added for consistency.

**Step 1: In each file, apply the pattern**

```tsx
import { useDebounce } from '@/hooks';

// Existing:
const [searchQuery, setSearchQuery] = useState('');

// Add:
const debouncedSearch = useDebounce(searchQuery, 300);

// Then in the React Query useQuery call, replace `searchQuery` with `debouncedSearch` in:
// 1. The queryKey array
// 2. The queryFn params (e.g., { search: debouncedSearch })
```

For UsersPage — it already uses this pattern (line 95). No changes needed there.

**Step 2: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add -A && git commit -m "perf: add debounced search to 6 list pages to avoid API calls on every keystroke"
```

---

## Task 14: Fix Settings Page

**Files:**
- Modify: `src/pages/settings/SettingsPage.tsx`
- Modify: `src/stores/uiStore.ts` (add notification preferences)

**Step 1: Create notification preferences in uiStore**

Add to `src/stores/uiStore.ts`:
```ts
interface UIState {
  // ... existing fields ...
  notificationPrefs: {
    orderUpdates: boolean;
    inventoryAlerts: boolean;
    systemNotifications: boolean;
    emailNotifications: boolean;
  };
  setNotificationPref: (key: keyof UIState['notificationPrefs'], value: boolean) => void;
}

// In the store:
notificationPrefs: {
  orderUpdates: true,
  inventoryAlerts: true,
  systemNotifications: true,
  emailNotifications: false,
},
setNotificationPref: (key, value) => set((state) => ({
  notificationPrefs: { ...state.notificationPrefs, [key]: value },
})),
```

These are persisted via the existing `persist` middleware.

**Step 2: Fix AppearanceSettings**

In SettingsPage.tsx `AppearanceSettings` section (lines 284-314), replace local `useState`:
```tsx
// Before:
const [theme, setTheme] = useState<'light' | 'dark' | 'system'>('light');

// After:
const { theme, setTheme } = useUIStore();
```

Remove `'system'` option since the store only supports `'light' | 'dark'`. Or keep the UI option but map 'system' to checking `prefers-color-scheme`.

**Step 3: Fix NotificationSettings**

Replace local `useState` with store:
```tsx
const { notificationPrefs, setNotificationPref } = useUIStore();
```
Wire each toggle to `setNotificationPref(key, !notificationPrefs[key])`.

**Step 4: Fix SecuritySettings — migrate to Zod**

Replace manual validation with:
```tsx
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';

const passwordSchema = z.object({
  current_password: z.string().min(1, 'Current password is required'),
  new_password: z.string().min(8, 'Password must be at least 8 characters'),
  confirm_password: z.string(),
}).refine(data => data.new_password === data.confirm_password, {
  message: 'Passwords do not match',
  path: ['confirm_password'],
});

const { register, handleSubmit, formState: { errors } } = useForm({
  resolver: zodResolver(passwordSchema),
});
```

**Step 5: Add ARIA attributes to tabs**

In the tab container (lines 38-53), add:
```tsx
<div role="tablist" className="flex border-b ...">
  {tabs.map((tab) => (
    <button
      key={tab.id}
      role="tab"
      aria-selected={activeTab === tab.id}
      aria-controls={`tabpanel-${tab.id}`}
      onClick={() => setActiveTab(tab.id)}
      className={...}
    >
```

And on each tab content panel:
```tsx
<div role="tabpanel" id={`tabpanel-${activeTab}`}>
```

**Step 6: Verify build and tests**

Run: `npx tsc --noEmit && npx vitest run`
Expected: PASS

**Step 7: Commit**

```bash
git add -A && git commit -m "fix: wire settings persistence, add Zod validation, add ARIA to tabs"
```

---

## Task 15: Fix Mobile Sidebar

**Files:**
- Modify: `src/components/layout/Sidebar.tsx` — add mobile overlay mode
- Modify: `src/components/layout/MainLayout.tsx` — handle mobile layout

**Step 1: Update Sidebar to support mobile overlay**

In `Sidebar.tsx`, read `sidebarOpen` and `setSidebarOpen` from store:
```tsx
const { sidebarCollapsed, toggleSidebarCollapse, sidebarOpen, setSidebarOpen } = useUIStore();
```

Add a mobile overlay wrapper that renders on small screens:
```tsx
// Mobile overlay
<>
  {/* Backdrop */}
  {sidebarOpen && (
    <div
      className="fixed inset-0 z-40 bg-black/50 lg:hidden"
      onClick={() => setSidebarOpen(false)}
    />
  )}

  {/* Sidebar */}
  <aside className={clsx(
    'fixed top-0 left-0 z-50 h-full bg-white border-r transition-all duration-300',
    // Desktop: always visible, controlled by sidebarCollapsed
    'hidden lg:block',
    sidebarCollapsed ? 'lg:w-20' : 'lg:w-64',
  )}>
    {/* existing sidebar content */}
  </aside>

  {/* Mobile sidebar */}
  <aside className={clsx(
    'fixed top-0 left-0 z-50 h-full w-64 bg-white border-r transition-transform duration-300 lg:hidden',
    sidebarOpen ? 'translate-x-0' : '-translate-x-full',
  )}>
    {/* same sidebar content, always expanded on mobile */}
  </aside>
</>
```

**Step 2: Update MainLayout for mobile**

In `MainLayout.tsx`, the `ml-64`/`ml-20` margin should only apply on `lg+`:
```tsx
<main className={clsx(
  'pt-16 min-h-screen transition-all duration-300',
  sidebarCollapsed ? 'lg:ml-20' : 'lg:ml-64'
)}>
```

**Step 3: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add -A && git commit -m "feat: wire up mobile sidebar as slide-over overlay"
```

---

## Task 16: Migrate LoginPage to useMutation

**Files:**
- Modify: `src/pages/auth/LoginPage.tsx`

**Step 1: Replace manual async with useMutation**

```tsx
import { useMutation } from '@tanstack/react-query';
import { authApi, getErrorMessage } from '@/api';

// Replace manual onSubmit with:
const loginMutation = useMutation({
  mutationFn: authApi.login,
  onSuccess: (response) => {
    login(response.user);
    toast.success('Welcome back!');
    navigate(from, { replace: true });
  },
  onError: (error) => {
    toast.error(getErrorMessage(error));
  },
});

const onSubmit = (data: LoginFormData) => {
  loginMutation.mutate(data);
};
```

Replace `isLoading` state with `loginMutation.isPending`. Remove `useState` for `isLoading`.

**Step 2: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add src/pages/auth/LoginPage.tsx && git commit -m "refactor: migrate LoginPage to useMutation for consistency"
```

---

## Task 17: Fix Chart Colors + Create Color Constants

**Files:**
- Create: `src/utils/chartColors.ts`
- Modify: `src/pages/dashboard/DashboardPage.tsx`
- Modify: `src/pages/analytics/AnalyticsPage.tsx`

**Step 1: Create chart color constants**

Create `src/utils/chartColors.ts`:
```ts
// Colors aligned with Tailwind v4 @theme in index.css
export const CHART_COLORS = {
  primary: '#f97316',    // primary-500
  primaryLight: '#fb923c', // primary-400
  blue: '#3b82f6',       // blue-500
  amber: '#f59e0b',      // amber-500
  emerald: '#10b981',    // emerald-500
  violet: '#8b5cf6',     // violet-500
  grid: '#f0f0f0',
  axis: '#9ca3af',       // gray-400
} as const;

export const PIE_COLORS = [
  CHART_COLORS.primary,
  CHART_COLORS.amber,
  CHART_COLORS.emerald,
  CHART_COLORS.blue,
  CHART_COLORS.violet,
];
```

**Step 2: Update DashboardPage**

Replace all hardcoded `#ec7428` with `CHART_COLORS.primary`, `#f0f0f0` with `CHART_COLORS.grid`, `#9ca3af` with `CHART_COLORS.axis`:
```tsx
import { CHART_COLORS } from '@/utils/chartColors';

// Line 134: stroke={CHART_COLORS.grid}
// Line 138: stroke={CHART_COLORS.axis}
// Line 153: stroke={CHART_COLORS.primary}
// etc.
```

**Step 3: Update AnalyticsPage**

Same replacements. Also replace the local `COLORS` array with import:
```tsx
import { CHART_COLORS, PIE_COLORS } from '@/utils/chartColors';
```

**Step 4: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add -A && git commit -m "refactor: centralize chart colors, align with Tailwind v4 theme"
```

---

## Task 18: Fix Notification Badge + Add ARIA to CategoryFormModal Tabs

**Files:**
- Modify: `src/components/layout/Header.tsx` — conditional badge
- Modify: `src/pages/categories/CategoryFormModal.tsx` — ARIA tabs

**Step 1: Fix notification badge**

In Header.tsx (lines 47-56), wrap the red dot in a conditional. Since we don't have a real unread count API yet, hide the badge entirely:
```tsx
<button
  onClick={() => navigate('/notifications')}
  className="relative p-2.5 rounded-xl text-gray-500 hover:bg-gray-100 hover:text-gray-700 transition-all duration-200"
>
  <Bell className="w-5 h-5" />
</button>
```

Remove the hardcoded animated red dot. When a notifications count endpoint is added later, it can be conditionally rendered.

**Step 2: Add ARIA to CategoryFormModal tabs**

In CategoryFormModal.tsx (lines 191-219), add tab semantics:
```tsx
<div role="tablist" className="flex border-b border-gray-200 mb-4">
  <button
    type="button"
    role="tab"
    aria-selected={activeTab === 'basic'}
    aria-controls="tabpanel-basic"
    onClick={() => setActiveTab('basic')}
    className={...}
  >
    Basic Info
  </button>
  <button
    type="button"
    role="tab"
    aria-selected={activeTab === 'attributes'}
    aria-controls="tabpanel-attributes"
    onClick={() => setActiveTab('attributes')}
    className={...}
  >
    Attributes
    ...
  </button>
</div>

{/* Tab panels */}
<div role="tabpanel" id="tabpanel-basic" hidden={activeTab !== 'basic'}>
  ...
</div>
<div role="tabpanel" id="tabpanel-attributes" hidden={activeTab !== 'attributes'}>
  ...
</div>
```

**Step 3: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add -A && git commit -m "fix: remove fake notification badge, add ARIA to category form tabs"
```

---

## Task 19: Add Dashboard Skeleton Loaders

**Files:**
- Modify: `src/pages/dashboard/DashboardPage.tsx` — add loading skeletons for sub-queries

**Step 1: Create a skeleton component inline or in common/**

Add a simple skeleton to show while sub-queries (top products, recent orders, low stock) are loading:
```tsx
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
```

**Step 2: Use in DashboardPage**

For each sub-query section, show `<CardSkeleton />` when `isLoading` is true instead of showing empty state.

**Step 3: Verify build**

Run: `npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add src/pages/dashboard/DashboardPage.tsx && git commit -m "ux: add skeleton loaders for dashboard sub-queries"
```

---

## Task 20: Cleanup + Polish

**Files:**
- Modify: `src/App.tsx` — extract toast styles
- Modify: `.gitignore` — add env files
- Create: `.env.example`

**Step 1: Extract toast inline styles**

In `App.tsx` (lines 310-334), replace inline `style` with `className`:
```tsx
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
```

**Step 2: Update .gitignore**

Add to `.gitignore`:
```
# Environment files (use .env.example as template)
.env.dev
.env.development
.env.prod
```

**Step 3: Create .env.example**

Create `.env.example`:
```
# API URL (leave empty for local dev with Vite proxy)
VITE_API_URL=
VITE_APP_ENV=development
VITE_APP_NAME=Handloom Admin
```

**Step 4: Verify build and all tests**

Run: `npx tsc --noEmit && npx vitest run`
Expected: All PASS

**Step 5: Commit**

```bash
git add -A && git commit -m "chore: extract toast styles, add .env.example, gitignore env files"
```

---

## Task 21: Migrate All Imports to @/ Path Alias

**Files:** All `.ts` and `.tsx` files in `src/`

**Step 1: Migrate imports**

Replace all relative imports like `../../api`, `../stores/authStore`, etc. with `@/api`, `@/stores/authStore`.

Rules:
- `./` imports within the same directory stay relative (e.g., `./LoginPage` in `index.ts`)
- Cross-directory imports become `@/` (e.g., `../../components/common` → `@/components/common`)

This can be done with find-and-replace or by running eslint auto-fix if import-sort handles it.

**Step 2: Verify build and all tests**

Run: `npx tsc --noEmit && npx vitest run`
Expected: All PASS

**Step 3: Commit**

```bash
git add -A && git commit -m "refactor: migrate all imports to @/ path alias"
```

---

## Final Verification

After all tasks:

```bash
npx tsc --noEmit        # TypeScript check
npx vitest run           # All tests
npm run lint             # ESLint
npm run format:check     # Prettier
npm run build            # Full production build
```

All should pass with no errors.
