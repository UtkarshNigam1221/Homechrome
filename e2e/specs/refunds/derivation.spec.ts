import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, json, Refund, RefundPreview } from '../../fixtures/api';
import { expectLedgerBalances } from '../../fixtures/reconcile';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';

/**
 * #230 cases 18, 19, 23 and 31.
 *
 * The amount is derived in Go and mirrored in TypeScript for the admin preview.
 * A second implementation can drift, and money is the one thing where drift is
 * not survivable — so the derivation is pinned from outside, per line.
 */
test.describe('refund amount derivation', () => {
  let admin: APIRequestContext;
  let fx: PaidFixture | undefined;

  test.beforeAll(async () => {
    admin = await adminClient();
  });

  test.afterEach(async () => {
    if (fx) {
      for (const p of fx.products) await expectLedgerBalances(admin, p, 'after derivation spec');
    }
    await releaseFixture(fx);
    fx = undefined;
  });

  test('a client-supplied amount is ignored (#230 case 18)', async () => {
    fx = await buyProducts(admin, [{ stock: 10, buy: 2 }]);
    const line = fx.order.items[0]!;
    const items = [{ order_item_id: line.id, quantity: 1 }];

    const preview = await json<RefundPreview>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds/preview`, { data: { items } })
    );

    // A hostile client naming its own figure. The server must derive, not read.
    const refund = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OTHER',
          amount: 99_999_999,
          total: 99_999_999,
          items: [{ order_item_id: line.id, quantity: 1, restock: false, amount: 99_999_999 }],
        },
      })
    );

    expect(refund.amount, 'money must never be a client input').toBe(preview.total);
    expect(refund.items[0]!.amount).toBe(preview.lines[0]!.amount);
  });

  test('each line is its value less its prorated discount share (#230 case 19)', async () => {
    // Two products at different prices, so the proration split is uneven.
    fx = await buyProducts(admin, [
      { stock: 10, buy: 2, price: 30_000 },
      { stock: 10, buy: 1, price: 40_000 },
    ]);

    const order = fx.order;

    for (const line of order.items) {
      const preview = await json<RefundPreview>(
        await admin.post(`/admin/orders/${order.id}/refunds/preview`, {
          data: { items: [{ order_item_id: line.id, quantity: 1 }] },
        })
      );

      // Mirrors prorate() in refund_amount.go: the line's own discount, split over
      // its units. Prices are tax-inclusive, so no tax is added on top.
      const lineSubtotal = line.unit_price;
      const roundHalfUp = (amount: number, part: number, whole: number) =>
        whole <= 0 || amount === 0 ? 0 : Math.floor((amount * part + Math.floor(whole / 2)) / whole);

      const lineDiscount = Math.min(
        Math.max(roundHalfUp(line.discount_amount, 1, line.quantity), 0),
        lineSubtotal
      );
      const expected = lineSubtotal - lineDiscount;

      expect(
        preview.total,
        `line ${line.id}: value less its prorated discount, to the paise`
      ).toBe(expected);
      expect(preview.is_final, 'a single unit of a multi-line order is not final').toBe(false);
    }
  });

  test('the old singular refund route is gone (#230 case 23)', async () => {
    fx = await buyProducts(admin, [{ stock: 6, buy: 1 }]);

    // It set PaymentStatus = REFUNDED with no gateway call — a route that lied
    // about money having moved. It must not answer.
    const res = await admin.post(`/admin/orders/${fx.order.id}/refund`, {
      data: { amount: 100, reason: 'should not exist' },
    });
    expect([404, 405]).toContain(res.status());
  });

  test('a refund in flight bounds the next one (#230 case 31)', async () => {
    fx = await buyProducts(admin, [{ stock: 10, buy: 2 }]);
    const line = fx.order.items[0]!;

    // Both units, deliberately left unsettled.
    const first = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OUT_OF_STOCK',
          items: [{ order_item_id: line.id, quantity: 2, restock: false }],
        },
      })
    );
    expect(first.status, 'left in flight on purpose').toBe('PENDING');

    // The same units again. Bounding only on settled refunds would let this
    // through and send the money back twice.
    const second = await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
      data: {
        reason: 'DAMAGED',
        items: [{ order_item_id: line.id, quantity: 1, restock: false }],
      },
    });

    expect(
      second.status(),
      'units held by a PENDING refund are already spoken for'
    ).toBeGreaterThanOrEqual(400);
  });
});
