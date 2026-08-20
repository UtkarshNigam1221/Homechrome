import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getInventory, getLedger, json, Refund, rowsForOrder } from '../../fixtures/api';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';

/**
 * The second manual check, and the reason migration 014 exists.
 *
 * 013 made "one movement per (product, order, type)" unique. That is right for
 * the order lifecycle — reserve once, dispatch once — but a refund is not part
 * of it: an order can be refunded line by line over days. The second refund of
 * the same product was deduped into the first, so the money went back and the
 * stock never moved. 014 adds source_id to the index; these assert it from
 * outside, against the deployed stack.
 */
test.describe('a second refund on the same product', () => {
  let admin: APIRequestContext;
  let fx: PaidFixture | undefined;

  test.beforeAll(async () => {
    admin = await adminClient();
  });

  test.afterEach(async () => {
    await releaseFixture(fx);
    fx = undefined;
  });

  test('moves stock again, and records itself separately in the ledger', async () => {
    // One product, 10 on hand, buy 4. Refund 1, then 1 again.
    fx = await buyProducts(admin, [{ stock: 10, buy: 4 }]);
    const product = fx.products[0]!;
    const line = fx.order.items[0]!;

    const before = await getInventory(admin, product.id);

    const first = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OUT_OF_STOCK',
          items: [{ order_item_id: line.id, quantity: 1, restock: false }],
        },
      })
    );
    const afterFirst = await getInventory(admin, product.id);

    expect(afterFirst.quantity, 'a write-off drops on-hand').toBe(before.quantity - 1);
    expect(afterFirst.reserved_qty, 'and releases the reservation with it').toBe(
      before.reserved_qty - 1
    );
    expect(afterFirst.available_qty, 'available must not move on a write-off').toBe(
      before.available_qty
    );

    const second = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'DAMAGED',
          items: [{ order_item_id: line.id, quantity: 1, restock: false }],
        },
      })
    );
    const afterSecond = await getInventory(admin, product.id);

    expect(
      afterSecond.quantity,
      'the second refund must move stock too — this is the 014 regression'
    ).toBe(before.quantity - 2);
    expect(afterSecond.reserved_qty).toBe(before.reserved_qty - 2);
    expect(afterSecond.available_qty).toBe(before.available_qty);

    // Two rows, same (product, order, WRITE_OFF), distinguished by source_id.
    const writeOffs = rowsForOrder(await getLedger(admin, product.id), fx.order.id, 'WRITE_OFF');
    expect(writeOffs, 'each refund writes its own ledger row').toHaveLength(2);

    const sources = writeOffs.map((r) => r.source_id).filter(Boolean);
    expect(new Set(sources).size, 'source_id must differ per refund').toBe(2);
    expect(sources).toEqual(expect.arrayContaining([first.id, second.id]));

    for (const row of writeOffs) {
      expect(row.reference_id, 'the reference stays the order — one story per order').toBe(
        fx!.order.id
      );
    }
  });

  test('a restock refund returns units to sale rather than destroying them', async () => {
    fx = await buyProducts(admin, [{ stock: 10, buy: 4 }]);
    const product = fx.products[0]!;
    const line = fx.order.items[0]!;

    const before = await getInventory(admin, product.id);

    await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'CUSTOMER_REQUEST',
          items: [{ order_item_id: line.id, quantity: 2, restock: true }],
        },
      })
    );

    const after = await getInventory(admin, product.id);
    expect(after.quantity, 'a restock leaves on-hand alone').toBe(before.quantity);
    expect(after.reserved_qty, 'it releases the reservation').toBe(before.reserved_qty - 2);
    expect(after.available_qty, 'so the units return to sale').toBe(before.available_qty + 2);

    const releases = rowsForOrder(await getLedger(admin, product.id), fx.order.id, 'RELEASE');
    expect(releases.length, 'the restock is recorded as a release').toBeGreaterThanOrEqual(1);
  });
});
