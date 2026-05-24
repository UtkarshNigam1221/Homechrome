# Homechrome Store

B2C customer-facing storefront for the Homechrome handloom e-commerce platform.

## Tech Stack

- Next.js 16 (App Router), React 19, TypeScript
- Mantine v9 (UI components, forms, modals, notifications, dates)
- React Query (server state), Zustand (client state)
- Axios (API client)

## Quick Start

```bash
npm install
npm run dev         # Requires backend running on localhost:8081
```

Open http://localhost:3000 in your browser.

> The backend must be running (`cd handloom-admin && make setup-local && make run`).
> See the [root README](../README.md) for full setup instructions.

## Dev Modes

| Script | Target | Env File | Use Case |
|--------|--------|----------|----------|
| `npm run dev` | `localhost:8081` | `.env.local` | Daily dev against monolith backend |
| `npm run dev:lambda` | `localhost:4566` | `.env.local-lambda` | Test against local Lambda/LocalStack |

## Commands

```bash
# Development
npm run dev               # Next.js on :3000 -> localhost:8081
npm run dev:local         # Same as dev (explicit port 3000)
npm run dev:lambda        # Next.js on :3000 -> localhost:4566

# Quality
npm run lint              # ESLint

# Build
npm run build             # Next.js production build
npm run start             # Start production server
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_API_URL` | Backend API URL (used for server-side fetches and rewrites) | `http://localhost:8081` |
| `NEXT_PUBLIC_SITE_URL` | Public site URL (used for SEO/sitemap) | `http://localhost:3000` |

## Project Structure

```
src/
├── app/                    # Next.js App Router
│   ├── page.tsx            # Homepage (featured products, categories)
│   ├── layout.tsx          # Root layout (header, footer, providers)
│   ├── providers.tsx       # React Query + Zustand providers
│   ├── c/[slug]/           # Category pages (product grid)
│   ├── p/[slug]/           # Product detail pages (gallery, add-to-cart)
│   ├── categories/         # All categories listing
│   ├── cart/               # Shopping cart with quantity controls
│   ├── checkout/           # Checkout flow (address, payment)
│   ├── login/              # Phone number + OTP login
│   ├── account/            # Customer account (profile, orders)
│   ├── track/              # Order tracking by ID
│   ├── sitemap.ts          # Dynamic sitemap generation
│   └── robots.ts           # Robots.txt
├── components/
│   ├── common/             # Shared UI components
│   ├── layout/             # Header, footer, navigation
│   ├── catalog/            # Product cards, category grids
│   ├── cart/               # Cart items, summary
│   └── checkout/           # Checkout form components
├── lib/
│   └── api.ts              # Axios client with response unwrapping + token refresh
├── stores/                 # Zustand stores
├── hooks/                  # Custom React hooks
└── types/                  # TypeScript type definitions
```

## Architecture

- **Routing**: Next.js App Router with server and client components
- **API**: Axios client uses relative URLs; Next.js rewrites `/api/*` to the backend
- **Auth**: Phone OTP login via MSG91, customer JWT stored in HttpOnly cookies, auto token refresh on 401
- **SEO**: JSON-LD structured data on product/category pages, dynamic sitemap, robots.txt
- **Images**: Next.js Image component with S3 remote patterns (unoptimized in dev for LocalStack compatibility)

## Backend API Routes

The storefront consumes B2C store routes served by the backend at `/api/v1/store/*`:

| Route | Description |
|-------|-------------|
| `/api/v1/store/auth/*` | Phone OTP send/verify, token refresh, logout |
| `/api/v1/store/catalog/*` | Browse categories, products, search |
| `/api/v1/store/cart/*` | Add/remove/update cart items |
| `/api/v1/store/checkout/*` | Place order, initiate payment |
| `/api/v1/store/orders/*` | Customer order history |
| `/api/v1/store/me/*` | Customer profile |
| `/api/v1/store/track/*` | Order tracking |
| `/api/v1/pricing/*` | Price calculation (public) |
