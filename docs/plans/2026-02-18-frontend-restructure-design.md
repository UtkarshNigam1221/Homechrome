# Frontend Directory Restructure — Design Document

**Date**: 2026-02-18
**Scope**: `handloom-admin-frontend/src/` — feature-based directory restructure, aggressive file splitting, dead code removal
**Strategy**: Big-bang migration in a single branch

---

## Context

The frontend codebase (~80 files, ~12,800 LOC) works correctly but has structural issues:
- `api/index.ts` (816 lines) and `types/index.ts` (681 lines) are monoliths
- `Loading.tsx` (319 lines) contains 15 components in one file
- Largest pages (`ProductFormModal` 722 lines, `OrderDetailPage` 535 lines) need splitting
- Dead code (`utils/api.ts`), empty dirs (`components/forms/`, `components/tables/`, `contexts/`), unused assets (`react.svg`)
- Domain-specific components (`AttributeFilterSidebar`, `ImageUpload`) incorrectly placed in shared `common/`

## Target Structure

```
src/
├── app/                            # App shell
│   ├── App.tsx                     # Thin shell: providers + router
│   ├── routes.tsx                  # Route definitions, lazy imports, guards
│   └── providers.tsx               # QueryClient, Toaster, ErrorBoundary
│
├── features/                       # One dir per business domain
│   ├── auth/
│   │   ├── api.ts
│   │   ├── components/
│   │   │   └── LoginPage.tsx
│   │   ├── types.ts
│   │   └── index.ts
│   ├── products/
│   │   ├── api.ts
│   │   ├── components/
│   │   │   ├── ProductsPage.tsx
│   │   │   ├── ProductFormModal/
│   │   │   │   ├── ProductFormModal.tsx
│   │   │   │   ├── AttributeFields.tsx
│   │   │   │   └── index.ts
│   │   │   └── AttributeFilterSidebar.tsx
│   │   ├── types.ts
│   │   └── index.ts
│   ├── orders/
│   │   ├── api.ts
│   │   ├── components/
│   │   │   ├── OrdersPage.tsx
│   │   │   └── OrderDetailPage/
│   │   │       ├── OrderDetailPage.tsx
│   │   │       ├── OrderNotes.tsx
│   │   │       ├── OrderTimeline.tsx
│   │   │       └── index.ts
│   │   ├── types.ts
│   │   └── index.ts
│   ├── categories/
│   │   ├── api.ts
│   │   ├── components/
│   │   │   ├── CategoriesPage.tsx
│   │   │   └── CategoryFormModal.tsx
│   │   ├── types.ts
│   │   └── index.ts
│   ├── customers/                  # Same pattern as above
│   │   ├── api.ts
│   │   ├── components/
│   │   │   ├── CustomersPage.tsx
│   │   │   └── CustomerFormModal.tsx
│   │   ├── types.ts
│   │   └── index.ts
│   ├── artisans/                   # Same pattern
│   ├── coupons/                    # Same pattern
│   ├── pricing/                    # Same pattern
│   ├── inventory/                  # Same pattern
│   ├── dashboard/
│   │   ├── api.ts                  # analyticsApi (dashboard stats subset)
│   │   ├── components/
│   │   │   └── DashboardPage.tsx
│   │   ├── types.ts
│   │   └── index.ts
│   ├── analytics/                  # Same pattern
│   ├── reports/                    # Same pattern
│   ├── bulk/                       # Same pattern
│   ├── notifications/              # Same pattern
│   └── settings/
│       ├── api.ts                  # usersApi
│       ├── components/
│       │   ├── SettingsPage.tsx
│       │   ├── UsersPage.tsx
│       │   └── UserFormModal.tsx
│       ├── types.ts
│       └── index.ts
│
├── shared/                         # Cross-feature shared code
│   ├── api/
│   │   └── client.ts              # Axios instance, interceptors, normalizeListResponse
│   ├── components/
│   │   ├── ui/                    # Primitives
│   │   │   ├── Button.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Select.tsx
│   │   │   ├── Badge.tsx
│   │   │   ├── Card.tsx
│   │   │   ├── Modal.tsx
│   │   │   ├── Table.tsx
│   │   │   ├── Tabs.tsx
│   │   │   ├── Pagination.tsx
│   │   │   ├── ImageUpload.tsx
│   │   │   └── index.ts
│   │   ├── loading/               # Split from monolithic Loading.tsx
│   │   │   ├── LoadingSpinner.tsx
│   │   │   ├── LoadingOverlay.tsx
│   │   │   ├── PageLoading.tsx
│   │   │   ├── Skeleton.tsx       # Base Skeleton + all domain skeletons
│   │   │   └── index.ts
│   │   ├── layout/
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   ├── MainLayout.tsx
│   │   │   └── index.ts
│   │   └── ErrorBoundary.tsx
│   ├── hooks/
│   │   ├── useDebounce.ts
│   │   ├── useCursorPagination.ts
│   │   └── index.ts
│   ├── stores/
│   │   ├── authStore.ts
│   │   └── uiStore.ts
│   ├── types/
│   │   └── common.ts              # ApiResponse, ListResponse, PaginationParams, PaginationResponse
│   └── utils/
│       ├── currency.ts
│       ├── badge.ts
│       ├── chartColors.ts
│       └── index.ts
│
├── test/
│   └── setup.ts
├── main.tsx
└── vite-env.d.ts
```

## File Splitting Plan

### api/index.ts (816 lines) → 14 feature-scoped api.ts files

The shared `normalizeListResponse()` helper moves to `shared/api/client.ts`. Each feature's `api.ts` imports it and contains only that domain's API functions.

Each feature `api.ts` follows this pattern:
```ts
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';
import type { Product, CreateProductRequest } from './types';

export const productsApi = { list, getById, create, update, delete };
```

### types/index.ts (681 lines) → per-feature types.ts + shared/types/common.ts

- `shared/types/common.ts` — PaginationParams, PaginationResponse, ApiResponse, ListResponse (~30 lines)
- Each `features/<domain>/types.ts` — domain-specific types

Cross-feature references: types stay in the feature that owns them. Types used by 3+ features move to shared.

### Loading.tsx (319 lines) → shared/components/loading/ (4 files)

| File | Components |
|------|-----------|
| `LoadingSpinner.tsx` | LoadingSpinner |
| `LoadingOverlay.tsx` | LoadingOverlay, InlineLoading, LoadingBar |
| `PageLoading.tsx` | PageLoading, DataLoading |
| `Skeleton.tsx` | Skeleton, CardSkeleton, TableSkeleton, FormSkeleton, ChartSkeleton, StatsSkeleton, DashboardSkeleton, ListItemSkeleton, TablePageSkeleton |

### ProductFormModal.tsx (722 lines) → subdirectory

Extract `AttributeFields.tsx` — renders dynamic form inputs for each category attribute type. Main form drops to ~500 lines.

### OrderDetailPage.tsx (535 lines) → subdirectory

Extract `OrderNotes.tsx` (notes list + add form) and `OrderTimeline.tsx` (status history). Main file drops to ~300 lines.

### App.tsx (224 lines) → app/ directory

- `app/providers.tsx` — QueryClientProvider, Toaster, ErrorBoundary setup
- `app/routes.tsx` — Route definitions, lazy imports, withSuspense, ProtectedRoute/AdminRoute/PublicRoute guards
- `app/App.tsx` — Thin shell: BrowserRouter + providers + routes

## Dead Code Removal

| Item | Reason | Action |
|------|--------|--------|
| `src/utils/api.ts` | `extractItems` only used by its own test; duplicated by `normalizeListResponse` in api layer | Delete |
| `src/utils/__tests__/api.test.ts` | Tests dead code | Delete |
| `src/components/forms/` | Empty directory | Delete |
| `src/components/tables/` | Empty directory | Delete |
| `src/contexts/` | Empty directory (Zustand used instead) | Delete |
| `src/assets/react.svg` | Vite scaffold leftover, not referenced anywhere | Delete |
| Barrel re-export of `getStatusBadgeVariant` from `components/common/index.ts` | Proxy re-export; consumers should import from `@/shared/utils/badge` directly | Remove proxy |

## Import Migration

All imports update to the new structure using the `@/` path alias:
- `@/components/common` → `@/shared/components/ui` or `@/shared/components/loading`
- `@/components/layout` → `@/shared/components/layout`
- `@/stores/authStore` → `@/shared/stores/authStore`
- `@/hooks` → `@/shared/hooks`
- `@/utils/*` → `@/shared/utils/*`
- `@/types` → `@/shared/types/common` (for shared types) or `@/features/<domain>/types` (for domain types)
- `@/api` → `@/features/<domain>/api` or `@/shared/api/client`
- `@/pages/<domain>` → `@/features/<domain>`

## Validation Strategy

Run `npm run check` (typecheck + lint + format:check) after completing each major step to catch import breakage immediately. Zero logic changes — this is purely structural.

## Out of Scope

- No logic changes, bug fixes, or feature additions
- No new dependencies
- No backend changes
- No test additions (existing tests move with their subjects)
