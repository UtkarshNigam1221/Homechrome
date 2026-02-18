# Frontend Full Cleanup Refactor — Design Document

**Date**: 2026-02-18
**Scope**: `handloom-admin-frontend/` — full cleanup of all 25 identified issues
**Approach**: Module-by-module (infrastructure first, then page-by-page)

---

## Context

Audit of the frontend codebase identified 5 critical issues, 11 quality issues, and 9 nice-to-have improvements. The codebase is in good shape architecturally (React 19, TypeScript strict mode, React Query for server state, Zustand for client state, react-hook-form + Zod for forms) but needs a consolidation pass to fix DRY violations, broken functionality, and missing infrastructure.

---

## Module 1: Shared Infrastructure & Utilities

### Error Boundary
- Create `src/components/common/ErrorBoundary.tsx` — React class component (error boundaries require class components)
- Catches render errors and chunk load failures
- Shows "Something went wrong" with a reload button
- Wrap all routes in `App.tsx` with this boundary

### Shared Utilities
- `src/utils/currency.ts` — `formatCurrency(paise: number): string` using `Intl.NumberFormat` for INR. Replaces 8 duplicate definitions across pages.
- `src/utils/api.ts` — `extractItems<T>(data: unknown, key?: string): T[]` for inconsistent API response shapes. Replaces 5 duplicate definitions.

### API Layer Normalization
- Move all response shape normalization into `src/api/index.ts` API functions
- Pages should receive `ListResponse<T>` consistently — never call `extractItems` directly
- Follow the pattern already established by `usersApi.list()` for all other APIs (categories, products, etc.)
- Replace `as unknown as T[]` casts in analytics API with runtime type checks

### Debounce All Search Inputs
- Apply existing `useDebounce` hook in all 8 list pages: ProductsPage, OrdersPage, CustomersPage, CategoriesPage, CouponsPage, ArtisansPage, PricingRulesPage, InventoryPage
- Pattern: `const debouncedSearch = useDebounce(searchQuery, 300)` — pass `debouncedSearch` to React Query key

### Tailwind Cleanup
- Delete dead `tailwind.config.js` (Tailwind v4 reads `@theme` from `index.css`)
- Update hardcoded hex values in chart components to reference CSS variables or a shared `CHART_COLORS` constant matching the theme palette

### Path Alias
- Add `"paths": { "@/*": ["./src/*"] }` to `tsconfig.app.json`
- Migrate all relative imports to `@/` prefix
- Vite alias already configured

### Remove Unused Dependency
- Remove `@heroicons/react` from `package.json` (project uses `lucide-react` exclusively)

---

## Module 2: Routing, Auth & Layout

### Fix Broken Routes
- Remove `/products/:id` route — product editing uses modals, this route renders ProductsPage without using the ID param
- Fix `OrderDetailPage` link to `/customers/:id` — either open a customer detail modal inline or remove the broken link

### 404 Page
- Create `src/pages/NotFoundPage.tsx` with "Page not found" message and link to dashboard
- Replace catch-all `<Navigate to="/" />` with this component

### Suspense Helper
- Create `withSuspense(LazyComponent)` utility
- Reduces 16 repetitive `<Suspense fallback={<LazyPageFallback />}>` wrappers to single-line route definitions

### Mobile Sidebar
- Wire up existing `sidebarOpen` state from `useUIStore`
- On screens < `lg`: sidebar renders as slide-over overlay controlled by `sidebarOpen`
- `Header` hamburger already calls `toggleSidebar` — `Sidebar` needs to read `sidebarOpen` and render overlay on mobile
- Add backdrop overlay that closes sidebar on click

### LoginPage Consistency
- Migrate from manual `useState`/`try-catch` to React Query `useMutation`
- Matches every other mutation in the codebase

---

## Module 3: Page-Level Fixes

### Settings Page
- **AppearanceSettings**: Replace local `useState` with `useUIStore`'s `theme`/`setTheme` (already has `persist` middleware)
- **NotificationSettings**: Wire toggles to localStorage via a new small Zustand store with `persist` middleware
- **SecuritySettings**: Migrate from inline `register` rules to `zodResolver` with Zod schema including `.refine()` for password confirmation match

### Pricing Proxy Fix
- Add `/api` to Vite dev proxy config alongside `/admin`, OR verify backend mounts pricing under `/admin/` too and update the API functions to use that prefix

### Notification Badge
- Query actual unread notification count from API
- Show red dot only when count > 0
- If no endpoint exists, hide the badge rather than showing a perpetually-active fake one

---

## Module 4: Testing & Accessibility

### Test Framework Setup
- Add Vitest + `@testing-library/react` + `@testing-library/jest-dom`
- Add npm scripts: `test`, `test:watch`

### Initial Test Coverage (non-UI logic)
- `utils/currency.ts` — formatCurrency edge cases
- `utils/api.ts` — extractItems with various response shapes
- `hooks/useDebounce.ts` — timing behavior
- `stores/authStore.ts` — login/logout state transitions
- `stores/uiStore.ts` — sidebar/theme toggle
- `api/client.ts` — interceptor behavior (envelope unwrapping, 401 refresh)

### ARIA Improvements
- Add `role="tab"`, `aria-selected`, `aria-controls` to custom tab implementations in SettingsPage and CategoryFormModal
- Headless UI components already handle this — only hand-rolled tabs need fixing

### Dashboard Loading States
- Add skeleton loaders for sub-queries (top products, recent orders, low stock)
- Replace empty states that flash before data arrives

---

## Module 5: Cleanup & Polish

### Toast Styles
- Extract inline `style` objects from `Toaster` in `App.tsx` to Tailwind classes

### Env Files
- Add `.env.dev`, `.env.development`, `.env.prod` to `.gitignore`
- Create `.env.example` with placeholder values

### Import Migration
- Batch-migrate all relative imports to `@/` path alias after tsconfig is updated

---

## Out of Scope
- Backend changes (all fixes are frontend-only)
- New features or pages
- E2E testing (unit/integration tests only for this pass)
- Design system overhaul (using existing Tailwind theme, just fixing inconsistencies)
