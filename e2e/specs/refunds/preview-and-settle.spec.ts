import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getOrder, json, Refund, RefundPreview } from '../../fixtures/api';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';
import { expectAllLedgersBalance } from '../../fixtures/reconcile';

/**
 * The fifth manual check, plus #223 Tier 1.5.
 *
 * The refund amount is derived twice — once in Go, once in TypeScript for the
 * admin preview. Two of #229's bugs were found by running it, not by testing
 * it. The preview endpoint lets the two be compared server-side, to the paise,
 * without a browser.
 */
test.describe('refund amounts and settlement', () => {
  let admin: APIRequestContext;
  let fx: PaidFixture | undefined;

  test.beforeAll(async () => {
    admin = await adminClient();
  });

  test.afterEach(async () => {
    // #230 case 35, before teardown removes the evidence.
    if (fx) await expectAllLedgersBalance(fx.admin, fx.products.map((p) => p.id));
    await releaseFixture(fx);
    fx = undefined;
  });

  test('a partial refund matches its preview exactly, then settles to PARTIALLY_REFUNDED', async () => {
    fx = await buyProducts(admin, [{ stock: 10, buy: 3 }]);
    const line = fx.order.items[0]!;
    const items = [{ order_item_id: line.id, quantity: 1 }];

    const preview = await json<RefundPreview>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds/preview`, { data: { items } })
    );
    expect(preview.is_final, 'one of three units is not the last').toBe(false);

    const refund = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: { reason: 'OUT_OF_STOCK', items: items.map((i) => ({ ...i, restock: false })) },
      })
    );

    expect(refund.amount, 'the charged amount must equal the previewed one, to the paise').toBe(
      preview.total
    );

    // Settlement is asynchronous. Re-check is the escape hatch for a webhook
    // that never arrived, and drives the same settlement path.
    const rechecked = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds/${refund.id}/recheck`)
    );
    expect(['COMPLETED', 'PENDING']).toContain(rechecked.status);

    if (rechecked.status === 'COMPLETED') {
      const order = await getOrder(admin, fx.order.id);
      expect(order.payment_status).toBe('PARTIALLY_REFUNDED');
      expect(order.status, 'fulfilment status must not move — the rest still ships').toBe(
        fx.order.status
      );
      expect(order.items[0]!.refunded_quantity).toBe(1);
    }
  });

  test('refunds against one order sum to its total exactly, and the last flips it to REFUNDED', async () => {
    // Three units so proration has a residual to absorb.
    fx = await buyProducts(admin, [{ stock: 10, buy: 3 }]);
    const line = fx.order.items[0]!;
    const orderTotal = fx.order.total_amount;

    let sum = 0;
    let lastRefundId = '';

    for (let i = 0; i < 3; i++) {
      const items = [{ order_item_id: line.id, quantity: 1 }];
      const preview = await json<RefundPreview>(
        await admin.post(`/admin/orders/${fx.order.id}/refunds/preview`, { data: { items } })
      );
      expect(preview.is_final, `refund ${i + 1} of 3`).toBe(i === 2);

      const refund = await json<Refund>(
        await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
          data: { reason: 'OTHER', items: items.map((x) => ({ ...x, restock: false })) },
        })
      );
      expect(refund.amount, `refund ${i + 1} must match its preview`).toBe(preview.total);
      sum += refund.amount;
      lastRefundId = refund.id;

      await admin.post(`/admin/orders/${fx.order.id}/refunds/${refund.id}/recheck`);
    }

    expect(
      sum,
      'per-line rounding must not leave the order a few paise short of fully refunded'
    ).toBe(orderTotal);

    const order = await getOrder(admin, fx.order.id);
    expect(order.payment_status, `after settling ${lastRefundId}`).toBe('REFUNDED');
    expect(order.items[0]!.refunded_quantity).toBe(3);
  });

  test('a refund beyond the remaining units is refused', async () => {
    fx = await buyProducts(admin, [{ stock: 10, buy: 2 }]);
    const line = fx.order.items[0]!;

    const res = await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
      data: {
        reason: 'OTHER',
        items: [{ order_item_id: line.id, quantity: 3, restock: false }],
      },
    });
    expect(res.status(), 'cannot refund more units than were bought').toBeGreaterThanOrEqual(400);
  });
});
