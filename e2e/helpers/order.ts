import { APIRequestContext, request } from '@playwright/test';

import { json, Order } from '../fixtures/api';
import { testPhone } from '../fixtures/test-phone';
import { payWithSandbox } from '../pages/phonepe-sandbox';
import { TARGETS } from '../playwright.config';

/**
 * Two ways to get an order, because they exercise different code and only one
 * of them can be refunded.
 *
 * `createAdminOrder` goes through POST /admin/orders. It is the only path that
 * can put the same product on two lines, which is the oversell #226 fixes. It
 * produces a PENDING order with no payment, so it cannot be refunded.
 *
 * `placePaidOrder` drives the storefront: OTP login, cart, address, checkout,
 * payment. Slower, but a refund needs a real payment row to exist.
 */

// ---------------------------------------------------------------------------
// Admin path
// ---------------------------------------------------------------------------

export interface AdminOrderLine {
  productId: string;
  quantity: number;
}

/**
 * Admin order creation needs a customer to hang the order off. Reuses the
 * storefront test customer so the suite does not accrete customers:
 * CustomerService.Delete is a soft delete that refuses once a customer has an
 * order, so a fresh one per run would be permanently undeletable.
 */
export async function resolveTestCustomerId(api: APIRequestContext): Promise<string> {
  const phone = testPhone();
  const body = await json<{ customers?: { id: string; phone: string }[] }>(
    await api.get(`/admin/customers?search=${encodeURIComponent(phone)}&limit=10`)
  );
  const match = (body.customers ?? []).find((c) => c.phone === phone);
  if (!match) {
    throw new Error(
      `no customer in dev with phone ${phone}. Run a storefront purchase spec ` +
        `first, or seed the customer — admin orders need one to attach to.`
    );
  }
  return match.id;
}

export async function createAdminOrder(
  api: APIRequestContext,
  customerId: string,
  lines: AdminOrderLine[]
): Promise<Order> {
  return json<Order>(
    await api.post('/admin/orders', {
      data: {
        customer_id: customerId,
        items: lines.map((l) => ({ product_id: l.productId, quantity: l.quantity })),
        shipping_address: {
          first_name: 'E2E',
          last_name: 'Suite',
          phone: '9999900001',
          address_line1: '1 Test Street',
          city: 'Mumbai',
          state: 'Maharashtra',
          postal_code: '400001',
          country: 'India',
        },
      },
    })
  );
}

/** POST /admin/orders as a raw response, for specs asserting a rejection. */
export async function attemptAdminOrder(
  api: APIRequestContext,
  customerId: string,
  lines: AdminOrderLine[]
) {
  return api.post('/admin/orders', {
    data: {
      customer_id: customerId,
      items: lines.map((l) => ({ product_id: l.productId, quantity: l.quantity })),
      shipping_address: {
        first_name: 'E2E',
        last_name: 'Suite',
        phone: '9999900001',
        address_line1: '1 Test Street',
        city: 'Mumbai',
        state: 'Maharashtra',
        postal_code: '400001',
        country: 'India',
      },
    },
  });
}

// ---------------------------------------------------------------------------
// Storefront path
// ---------------------------------------------------------------------------

function requiredEnv(name: string): string {
  const v = process.env[name];
  if (!v) throw new Error(`${name} must be set for storefront specs`);
  return v;
}

/**
 * Logs the allowlisted test phone in against the real auth Lambda. The OTP is
 * not random for this number: SendOTP short-circuits above the SMS gateway and
 * stores the hash of STORE_TEST_OTP instead, so no SMS is sent and no message
 * is billed. Everything else — hashing, TTL, attempt limits, verification —
 * is the production path.
 */
export async function customerClient(): Promise<APIRequestContext> {
  // Same number resolveTestCustomerId uses, or an admin order would attach
  // to a different worker's customer.
  const phone = testPhone();
  const otp = requiredEnv('E2E_STORE_OTP');
  const ctx = await request.newContext({ baseURL: TARGETS.api });

  const sent = await ctx.post('/api/v1/store/auth/otp/send', { data: { phone } });
  if (!sent.ok()) {
    throw new Error(`otp/send failed: ${sent.status()} ${await sent.text()}`);
  }

  const verified = await ctx.post('/api/v1/store/auth/otp/verify', {
    data: { phone, code: otp },
  });
  if (!verified.ok()) {
    throw new Error(
      `otp/verify failed: ${verified.status()} ${await verified.text()}\n` +
        `Is ${phone} on the deployment's STORE_TEST_PHONES, and does ` +
        `E2E_STORE_OTP match its STORE_TEST_OTP?`
    );
  }
  return ctx;
}

export interface PaidOrderResult {
  order: Order;
  /** Set when dev is running the PhonePe DevClient rather than the sandbox. */
  autoPaid: boolean;
}

/**
 * Cart → address → checkout → payment → PAID.
 *
 * Payment branches on what dev is actually configured with, detected from the
 * redirect the API returns rather than assumed:
 *
 *   - PhonePe DevClient (no PHONEPE_CLIENT_ID in the deployment) returns a
 *     local confirmation URL carrying `dev_payment=`. The payment completes
 *     server-side; there is nothing to drive.
 *   - The real sandbox returns a phonepe.com URL, which needs a browser. Those
 *     specs live in the admin-ui project and use pages/phonepe-sandbox.ts.
 *
 * Either way this is the deployed gateway, not a stub in the suite.
 */
export async function placePaidOrder(
  store: APIRequestContext,
  lines: { productId: string; quantity: number }[]
): Promise<PaidOrderResult> {
  await store.delete('/api/v1/store/cart');

  for (const line of lines) {
    const added = await store.post('/api/v1/store/cart/items', {
      data: { product_id: line.productId, quantity: line.quantity },
    });
    if (!added.ok()) {
      throw new Error(`add to cart failed: ${added.status()} ${await added.text()}`);
    }
  }

  const address = await json<{ id: string }>(
    await store.post('/api/v1/store/me/addresses', {
      data: {
        first_name: 'E2E',
        last_name: 'Suite',
        phone: '9999900001',
        address_line1: '1 Test Street',
        city: 'Mumbai',
        state: 'Maharashtra',
        postal_code: '400001',
        country: 'India',
        is_default: true,
      },
    })
  );

  // CheckoutResult is {order, redirect_url, merchant_txn_id} — the order comes
  // back whole, there is no top-level order_id.
  const checkout = await json<{
    order: { id: string };
    redirect_url?: string;
    merchant_txn_id?: string;
  }>(
    await store.post('/api/v1/store/checkout/initiate', {
      data: { shipping_address_id: address.id },
    })
  );
  if (!checkout.order?.id) {
    throw new Error(`checkout returned no order: ${JSON.stringify(checkout).slice(0, 200)}`);
  }

  const redirect = checkout.redirect_url ?? '';
  const autoPaid = redirect.includes('dev_payment=');

  if (!autoPaid && redirect) {
    // The real sandbox. Scan its QR and answer the simulator, in the one file
    // that knows those pages. Needs no credentials.
    await payWithSandbox(redirect);
  }

  const order = await pollUntilPaid(store, checkout.order.id);
  return { order, autoPaid };
}

/** Payment settles asynchronously even on the DevClient; poll, do not sleep. */
export async function pollUntilPaid(
  store: APIRequestContext,
  orderId: string,
  timeoutMs = 45_000
): Promise<Order> {
  const deadline = Date.now() + timeoutMs;
  let last = '';
  let consecutiveErrors = 0;

  while (Date.now() < deadline) {
    const res = await store.get(`/api/v1/store/checkout/payment-status/${orderId}`);

    if (!res.ok()) {
      // A 404 here means the order id is wrong, and no amount of waiting fixes
      // that. Three in a row is the endpoint saying so, not a slow webhook.
      if (++consecutiveErrors >= 3) {
        throw new Error(
          `payment-status kept returning ${res.status()} for order ${orderId} — ` +
            `the order id is probably wrong, not the payment slow`
        );
      }
    } else {
      consecutiveErrors = 0;
      const status = await json<{ payment_status?: string; local_status?: string }>(res);
      last = status.payment_status ?? status.local_status ?? '';
      if (last === 'PAID' || last === 'SUCCESS') {
        return json<Order>(await store.get(`/api/v1/store/orders/${orderId}`));
      }
      // Terminal states: stop immediately rather than waiting out the deadline.
      if (last === 'FAILED' || last === 'CANCELLED') {
        throw new Error(`payment reached ${last} for order ${orderId}`);
      }
    }
    await new Promise((r) => setTimeout(r, 2_000));
  }
  throw new Error(`order ${orderId} never reached PAID (last status: ${last || 'unknown'})`);
}
