import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getInventory, getLedger, getOrder, json, Refund, rowsForOrder } from '../../fixtures/api';
import { createProduct } from '../../fixtures/catalog';
import { placeUnpaidOrder } from '../../helpers/order';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';
import { expectAllLedgersBalance } from '../../fixtures/reconcile';

/**
 * The third manual check: cancel must release only this order's reservation.
 *
 * Cancel releases "what this order still holds". Before order-scoping it
 * guarded on the product-level reserved_qty total, so cancelling one order
 * could eat another order's reservation whenever both held the same product —
 * the case unit tests structurally miss, because they cover a product with
 * exactly one outstanding order.
 */
test.describe('cancelling a partly refunded order', () => {
  let admin: APIRequestContext;
  let fx: PaidFixture | undefined;

  test.beforeAll(async () => {
    admin = await adminClient();
  });

  test.afterEach(async () => {
    // #230 case 35, before teardown removes the evidence.
    if (fx) await expectAllLedgersBalance(fx.admin, fx.products);
    await releaseFixture(fx);
    fx = undefined;
  });

  test('releases only its own remainder, leaving another order untouched', async () => {
    fx = await buyProducts(admin, [{ stock: 20, buy: 5 }]);
    const product = fx.products[0]!;
    const line = fx.order.items[0]!;

    const other = await placeUnpaidOrder(fx.store, [{ productId: product.id, quantity: 4 }]);

    let inv = await getInventory(admin, product.id);
    expect(inv.reserved_qty, 'both orders hold stock').toBe(9);

    // Refund 2 of the paid order's 5, written off.
    await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OUT_OF_STOCK',
          items: [{ order_item_id: line.id, quantity: 2, restock: false }],
        },
      })
    );

    inv = await getInventory(admin, product.id);
    expect(inv.reserved_qty, 'the write-off released 2 of the paid order').toBe(7);
    expect(inv.quantity, 'and destroyed 2 on hand').toBe(18);

    // Cancel the paid order. It still holds 3.
    const cancelled = await admin.post(`/admin/orders/${fx.order.id}/cancel`, {
      data: { reason: 'e2e' },
    });
    expect(cancelled.ok(), await cancelled.text()).toBeTruthy();

    inv = await getInventory(admin, product.id);
    expect(
      inv.reserved_qty,
      "cancel must release only this order's remaining 3, never the other order's 4"
    ).toBe(4);
    expect(inv.quantity, 'cancel does not restore written-off units').toBe(18);

    // The other order is intact and still shippable.
    const otherOrder = await getOrder(admin, other.id);
    expect(otherOrder.status).not.toBe('CANCELLED');

    const releases = rowsForOrder(await getLedger(admin, product.id), fx.order.id, 'RELEASE');
    expect(releases.length, 'the cancel wrote its own release row').toBeGreaterThanOrEqual(1);

    await admin.post(`/admin/orders/${other.id}/cancel`, { data: { reason: 'e2e teardown' } });
  });
});
