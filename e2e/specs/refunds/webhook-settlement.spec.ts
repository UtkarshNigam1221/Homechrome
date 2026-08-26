import { createHash } from 'node:crypto';

import { APIRequestContext, expect, request, test } from '@playwright/test';

import { adminClient, getInventory, getOrder, json, listRefunds, Refund } from '../../fixtures/api';
import { expectLedgerBalances } from '../../fixtures/reconcile';
import { TARGETS } from '../../playwright.config';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';

/**
 * #230 cases 21, 22 and 24, and #223 Tier 1.2 and 1.3.
 *
 * The service layer proves the conditional write against mocks. Nothing proved
 * the whole HTTP path is idempotent — and PhonePe retries, while Lambda can
 * process two deliveries at once.
 *
 * These post to the deployed handler, signed the way PhonePe signs. Dev has
 * PHONEPE_WEBHOOK_USERNAME and PASSWORD set, so the signature check is live —
 * and it answers a rejection with 200, because PhonePe must not retry a body it
 * cannot authenticate. From out here a rejected delivery is therefore
 * indistinguishable from an accepted one, which is how these specs spent runs
 * asserting 200 against deliveries that never reached the refund path at all.
 */

/**
 * The Authorization header PhonePe sends: hex SHA256 of "username:password",
 * compared verbatim by the handler. Undefined when the credentials are absent,
 * which skips the group rather than testing nothing.
 */
function webhookAuth(): string | undefined {
  const user = process.env.E2E_PHONEPE_WEBHOOK_USERNAME;
  const password = process.env.E2E_PHONEPE_WEBHOOK_PASSWORD;
  if (!user || !password) return undefined;
  return createHash('sha256').update(`${user}:${password}`).digest('hex');
}

async function deliverRefundWebhook(
  event: 'pg.refund.completed' | 'pg.refund.failed',
  payload: { originalMerchantOrderId: string; refundId: string; amount: number }
) {
  const auth = webhookAuth();
  const ctx = await request.newContext({
    baseURL: TARGETS.api,
    extraHTTPHeaders: auth ? { Authorization: auth } : {},
  });
  const res = await ctx.post('/api/v1/store/webhooks/phonepe', {
    data: { event, payload: { ...payload, state: event.endsWith('completed') ? 'COMPLETED' : 'FAILED' } },
  });
  const status = res.status();
  await ctx.dispose();
  return status;
}

test.describe('refund webhook settlement over HTTP', () => {
  let admin: APIRequestContext;
  let fx: PaidFixture | undefined;

  test.beforeAll(async () => {
    // Without the credentials every delivery below is rejected and answered
    // 200, so the specs would pass while proving nothing. Skip, do not pretend.
    test.skip(
      !webhookAuth(),
      'E2E_PHONEPE_WEBHOOK_USERNAME / E2E_PHONEPE_WEBHOOK_PASSWORD unset — an unsigned delivery is rejected with a 200 and these specs cannot tell'
    );
    admin = await adminClient();
  });

  test.afterEach(async () => {
    if (fx) await expectLedgerBalances(admin, fx.products[0]!, 'after webhook spec');
    await releaseFixture(fx);
    fx = undefined;
  });

  test('a repeated delivery settles once (#230 case 21)', async () => {
    fx = await buyProducts(admin, [{ stock: 10, buy: 3 }]);
    const line = fx.order.items[0]!;

    const refund = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OUT_OF_STOCK',
          items: [{ order_item_id: line.id, quantity: 1, restock: false }],
        },
      })
    );
    test.skip(!refund.provider_refund_id, 'no provider refund id — gateway did not return one');

    const merchantOrderId = await merchantTxnIdFor(admin, fx.order.id);
    const payload = {
      originalMerchantOrderId: merchantOrderId,
      refundId: refund.provider_refund_id!,
      amount: refund.amount,
    };

    const stockBefore = await getInventory(admin, fx.products[0]!.id);

    // Three deliveries of the same event, as PhonePe would retry.
    for (let i = 0; i < 3; i++) {
      expect(await deliverRefundWebhook('pg.refund.completed', payload)).toBe(200);
    }

    const order = await getOrder(admin, fx.order.id);
    expect(order.items[0]!.refunded_quantity, 'quantity must not accumulate per delivery').toBe(1);
    expect(order.payment_status).toBe('PARTIALLY_REFUNDED');

    const stockAfter = await getInventory(admin, fx.products[0]!.id);
    expect(stockAfter.quantity, 'settlement moves no stock — creation already did').toBe(
      stockBefore.quantity
    );

    const refunds = await listRefunds(admin, fx.order.id);
    expect(refunds.filter((r) => r.id === refund.id)).toHaveLength(1);
    expect(refunds.find((r) => r.id === refund.id)!.status).toBe('COMPLETED');
  });

  test('concurrent deliveries apply exactly once (#230 case 22)', async () => {
    fx = await buyProducts(admin, [{ stock: 10, buy: 4 }]);
    const line = fx.order.items[0]!;

    const refund = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'DAMAGED',
          items: [{ order_item_id: line.id, quantity: 2, restock: false }],
        },
      })
    );
    test.skip(!refund.provider_refund_id, 'no provider refund id — gateway did not return one');

    const payload = {
      originalMerchantOrderId: await merchantTxnIdFor(admin, fx.order.id),
      refundId: refund.provider_refund_id!,
      amount: refund.amount,
    };

    // Five in parallel. Exactly one must pass the conditional write; the losers
    // must be no-ops, not errors and not partial applications.
    const statuses = await Promise.all(
      Array.from({ length: 5 }, () => deliverRefundWebhook('pg.refund.completed', payload))
    );
    expect(statuses.every((s) => s === 200), 'the webhook always acknowledges').toBeTruthy();

    const order = await getOrder(admin, fx.order.id);
    expect(
      order.items[0]!.refunded_quantity,
      'five concurrent settlements must increment the line once'
    ).toBe(2);
  });

  test('re-check is safe to call twice (#230 case 24)', async () => {
    fx = await buyProducts(admin, [{ stock: 10, buy: 2 }]);
    const line = fx.order.items[0]!;

    const refund = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OTHER',
          items: [{ order_item_id: line.id, quantity: 1, restock: false }],
        },
      })
    );

    const first = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds/${refund.id}/recheck`)
    );
    const second = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds/${refund.id}/recheck`)
    );

    expect(second.status, 'a terminal refund stays terminal').toBe(first.status);

    const order = await getOrder(admin, fx.order.id);
    expect(
      order.items[0]!.refunded_quantity,
      're-checking twice must not double-apply'
    ).toBe(first.status === 'COMPLETED' ? 1 : 0);
  });
});

/** The webhook correlates on the payment's merchant transaction id. */
async function merchantTxnIdFor(api: APIRequestContext, orderId: string): Promise<string> {
  const status = await json<{ merchant_txn_id?: string }>(
    await api.get(`/admin/orders/${orderId}/payment-status`)
  );
  if (!status.merchant_txn_id) {
    throw new Error(`no merchant_txn_id for order ${orderId}`);
  }
  return status.merchant_txn_id;
}
