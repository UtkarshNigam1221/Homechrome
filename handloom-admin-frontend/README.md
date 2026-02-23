# Handloom Admin Frontend

React admin dashboard for the Homechrome handloom e-commerce platform. Manages products, orders, customers, pricing, and analytics.

## Tech Stack

- React 19, TypeScript, Vite 7
- Tailwind CSS 4, Headless UI, Lucide icons
- Zustand (auth/UI state), React Query (server state)
- react-hook-form + zod (form validation)
- Recharts (analytics)
- Axios (API client)

## Quick Start

```bash
npm install
npm run dev:local    # Requires backend running on localhost:8081
```

Open http://localhost:5173 in your browser.

> The backend must be running (`cd handloom-admin && make setup-local && make run`).
> See the [root README](../README.md) for full setup instructions.

## Dev Modes

| Script | Target | Env File | Use Case |
|--------|--------|----------|----------|
| `npm run dev:local` | `localhost:8081` | `.env.local-backend` | Daily dev against monolith backend |
| `npm run dev:lambda` | `localhost:4566` | `.env.local-lambda` | Test against local Lambda/LocalStack |
| `npm run dev` | AWS dev API | `.env.development` | Test against deployed AWS |

## Commands

```bash
# Development
npm run dev:local         # Vite on :5173 -> localhost:8081
npm run dev:lambda        # Vite on :5173 -> localhost:4566
npm run dev               # Vite on :5173 -> AWS dev API

# Quality
npm run check             # typecheck + lint + format:check
npm run typecheck         # tsc --noEmit
npm run lint              # ESLint
npm run lint:fix          # ESLint with auto-fix
npm run format            # Prettier write

# Testing
npm run test              # Vitest run
npm run test:watch        # Vitest watch mode

# Build
npm run build             # TypeScript check + Vite production build
npm run build:dev         # Build for dev environment
npm run build:prod        # Build for prod environment

# Deploy
npm run cdk:deploy:dev    # Build + CDK deploy to dev
npm run cdk:deploy:prod   # Build + CDK deploy to prod
```

## Project Structure

```
src/
├── app/                    # App.tsx (all routes + auth guards), providers
├── api/                    # Axios client, API functions
├── features/               # Feature modules (settings, etc.)
├── pages/                  # Page components by domain
│   └── <domain>/           # List page, form modal, barrel index.ts
└── shared/
    ├── components/
    │   ├── common/         # Button, Modal, Table, Input, etc.
    │   └── layout/         # MainLayout, Sidebar, Header
    ├── stores/             # Zustand (authStore, uiStore)
    ├── hooks/              # Custom hooks (useDebounce, etc.)
    └── utils/              # Utilities (currency, etc.)
```

## Conventions

- **Path alias**: `@/` maps to `src/` (configured in `vite.config.ts` and `tsconfig.app.json`)
- **Imports**: `eslint-plugin-simple-import-sort` enforces ordering
- **Env vars**: prefixed with `VITE_`
- **Lint-staged**: on commit, `.ts/.tsx` files get eslint --fix + prettier; `.css/.json/.md` get prettier
- **Dev proxy**: Vite proxies `/admin/*` and `/api/*` to backend so cookies work same-origin

## Infrastructure

CDK stacks in `infra/` (written in Go):
- **Dev**: S3 static website hosting (HTTP, free tier)
- **Prod**: CloudFront + S3 (HTTPS, custom domain via ACM cert)
