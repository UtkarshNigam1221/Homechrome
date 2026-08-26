import { APIRequestContext, request } from '@playwright/test';

import { adminClient, json, Order } from '../fixtures/api';
import { testPhone } from '../fixtures/test-phone';
import { payWithSandbox } from '../pages/phonepe-sandbox';
import { TARGETS } from '../playwright.config';

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

export async function prepareCheckout(
  store: APIRequestContext,
  lines: { productId: string; quantity: number }[]
): Promise<string> {
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
  return address.id;
}

interface CheckoutResult {
  order: { id: string };
  redirect_url?: string;
  merchant_txn_id?: string;
}

async function initiateCheckout(
  store: APIRequestContext,
  addressId: string
): Promise<CheckoutResult> {
  // CheckoutResult is {order, redirect_url, merchant_txn_id} — the order comes
  // back whole, there is no top-level order_id.
  const checkout = await json<CheckoutResult>(
    await store.post('/api/v1/store/checkout/initiate', {
      data: { shipping_address_id: addressId },
    })
  );
  if (!checkout.order?.id) {
    throw new Error(`checkout returned no order: ${JSON.stringify(checkout).slice(0, 200)}`);
  }
  return checkout;
}

export async function placeUnpaidOrder(
  store: APIRequestContext,
  lines: { productId: string; quantity: number }[]
): Promise<Order> {
  const addressId = await prepareCheckout(store, lines);
  const checkout = await initiateCheckout(store, addressId);
  return json<Order>(await store.get(`/api/v1/store/orders/${checkout.order.id}`));
}

export interface PaidOrderResult {
  order: Order;
  autoPaid: boolean;
}

export async function placePaidOrder(
  store: APIRequestContext,
  lines: { productId: string; quantity: number }[]
): Promise<PaidOrderResult> {
  const addressId = await prepareCheckout(store, lines);
  const checkout = await initiateCheckout(store, addressId);

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

/**
 * The storefront's payment-status is a pure read, so only PhonePe's webhook
 * advances a payment — and against UAT that delivery is unreliable. Waiting
 * longer cannot fix a webhook that never arrives, so this reconciles from the
 * provider instead, through the admin re-check.
 */
export async function pollUntilPaid(
  store: APIRequestContext,
  orderId: string,
  timeoutMs = 120_000
): Promise<Order> {
  const deadline = Date.now() + timeoutMs;
  let last = '';
  let consecutiveErrors = 0;
  let polls = 0;
  let admin: APIRequestContext | undefined;

  while (Date.now() < deadline) {
    // Give the webhook 10s of grace, then reconcile from the provider every
    // 10s rather than waiting out a delivery that may never come.
    if (++polls > 5 && polls % 5 === 1) {
      admin ??= await adminClient();
      await admin.get(`/admin/orders/${orderId}/payment-status`).catch(() => undefined);
    }

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
