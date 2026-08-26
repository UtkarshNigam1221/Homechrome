import { APIRequestContext, request } from '@playwright/test';

import { TARGETS } from '../playwright.config';

/**
 * Authenticated clients against the deployed dev API. Auth is a real login
 * against the real auth Lambda; the JWT comes back in an HttpOnly cookie, and
 * Playwright's request context keeps the jar, so every later call is
 * indistinguishable from the admin SPA's.
 */

export type Role = 'admin' | 'operator';

interface Credentials {
  email: string;
  password: string;
}

function credentialsFor(role: Role): Credentials {
  const prefix = role === 'admin' ? 'E2E_ADMIN' : 'E2E_OPERATOR';
  const email = process.env[`${prefix}_EMAIL`];
  const password = process.env[`${prefix}_PASSWORD`];
  if (!email || !password) {
    throw new Error(
      `${prefix}_EMAIL and ${prefix}_PASSWORD must be set. ` +
        `The suite asserts that ${role.toUpperCase()} is ${
          role === 'admin' ? 'allowed to' : 'refused when it tries to'
        } refund, so the account has to exist in dev.`
    );
  }
  return { email, password };
}

export async function adminClient(): Promise<APIRequestContext> {
  return loginAs('admin');
}

export async function operatorClient(): Promise<APIRequestContext> {
  return loginAs('operator');
}

async function loginAs(role: Role): Promise<APIRequestContext> {
  const { email, password } = credentialsFor(role);
  const ctx = await request.newContext({ baseURL: TARGETS.api });

  const res = await ctx.post('/admin/auth/login', { data: { email, password } });
  if (!res.ok()) {
    throw new Error(
      `login failed for ${role} (${email}): ${res.status()} ${await res.text()}`
    );
  }
  return ctx;
}

/**
 * Unwraps the standard `{success, data}` envelope. Throws on a non-2xx with the
 * body attached, because a bare "expected 200 got 400" tells you nothing about
 * which validation rule fired.
 */
export async function json<T>(
  res: Awaited<ReturnType<APIRequestContext['get']>>
): Promise<T> {
  const body = await res.text();
  if (!res.ok()) {
    throw new Error(`${res.status()} ${res.url()}\n${body}`);
  }
  const parsed = JSON.parse(body) as { data?: T } & T;
  return (parsed.data ?? parsed) as T;
}

// ---------------------------------------------------------------------------
// Domain shapes. Only the fields the suite asserts on.
// ---------------------------------------------------------------------------

export interface Inventory {
  product_id: string;
  quantity: number;
  reserved_qty: number;
  available_qty: number;
}

export interface InventoryTransaction {
  id: string;
  product_id: string;
  type: 'ADD' | 'REMOVE' | 'RESERVE' | 'RELEASE' | 'ADJUST' | 'COMMIT' | 'RETURN' | 'WRITE_OFF';
  quantity: number;
  previous_qty: number;
  new_qty: number;
  reason: string;
  reference_id: string;
  source_id?: string;
  created_at: string;
}

export interface OrderItem {
  id: string;
  product_id: string;
  quantity: number;
  unit_price: number;
  refunded_quantity?: number;
  /** This line's whole share of the order discount, in paise. */
  discount_amount?: number;
}

export interface Order {
  id: string;
  order_number: string;
  status: string;
  payment_status: string;
  items: OrderItem[];
  subtotal: number;
  discount_amount: number;
  tax_amount: number;
  shipping_amount: number;
  total_amount: number;
}

export interface Refund {
  id: string;
  order_id: string;
  amount: number;
  status: 'PENDING' | 'COMPLETED' | 'FAILED';
  reason: string;
  provider_refund_id?: string;
  items: { order_item_id: string; product_id: string; quantity: number; amount: number; restock: boolean }[];
}

export interface RefundPreview {
  total: number;
  is_final: boolean;
  lines: { order_item_id: string; quantity: number; amount: number }[];
}

// ---------------------------------------------------------------------------
// Reads
// ---------------------------------------------------------------------------

export async function getInventory(
  api: APIRequestContext,
  productId: string
): Promise<Inventory> {
  return json<Inventory>(await api.get(`/admin/products/${productId}/inventory`));
}

export async function getLedger(
  api: APIRequestContext,
  productId: string
): Promise<InventoryTransaction[]> {
  const body = await json<{ transactions?: InventoryTransaction[] } | InventoryTransaction[]>(
    await api.get(`/admin/products/${productId}/inventory/transactions?limit=100`)
  );
  return Array.isArray(body) ? body : (body.transactions ?? []);
}

export async function getOrder(api: APIRequestContext, orderId: string): Promise<Order> {
  return json<Order>(await api.get(`/admin/orders/${orderId}`));
}

export async function listRefunds(
  api: APIRequestContext,
  orderId: string
): Promise<Refund[]> {
  const body = await json<{ refunds?: Refund[] } | Refund[]>(
    await api.get(`/admin/orders/${orderId}/refunds`)
  );
  return Array.isArray(body) ? body : (body.refunds ?? []);
}

/** Ledger rows for one order, newest last, optionally narrowed to one type. */
export function rowsForOrder(
  ledger: InventoryTransaction[],
  orderId: string,
  type?: InventoryTransaction['type']
): InventoryTransaction[] {
  return ledger
    .filter((r) => r.reference_id === orderId && (!type || r.type === type))
    .sort((a, b) => a.created_at.localeCompare(b.created_at));
}
