import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getInventory, getLedger, json, Refund, rowsForOrder } from '../../fixtures/api';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';

/**
 * The sixth manual check, API half. The UI half — that the ledger renders
 * "Written off" and the reconciliation panel agrees — lives in
 * specs/admin-ui/ledger.spec.ts.
 *
 * A write-off is arithmetically identical to a dispatch: both drop quantity and
 * reserved_qty together. The ledger is the only thing that can tell them apart,
 * which is why the type and the source matter as much as the numbers.
 */
test.describe('the ledger records a write-off as itself', () => {
  let admin: APIRequestContext;
  let fx: PaidFixture | undefined;

  test.beforeAll(async () => {
    admin = await adminClient();
  });

  test.afterEach(async () => {
    await releaseFixture(fx);
    fx = undefined;
  });

  test('write-off rows name the refund, reference the order, and reconcile', async () => {
    fx = await buyProducts(admin, [{ stock: 12, buy: 5 }]);
    const product = fx.products[0]!;
    const line = fx.order.items[0]!;

    const refund = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'DAMAGED',
          items: [{ order_item_id: line.id, quantity: 2, restock: false }],
        },
      })
    );

    const ledger = await getLedger(admin, product.id);
    const writeOffs = rowsForOrder(ledger, fx.order.id, 'WRITE_OFF');
    expect(writeOffs).toHaveLength(1);

    const row = writeOffs[0]!;
    expect(row.quantity).toBe(2);
    expect(row.reference_id, 'the reference is the order').toBe(fx.order.id);
    expect(row.source_id, 'the source is the refund that caused it').toBe(refund.id);
    expect(row.type, 'never COMMIT — a write-off is not a dispatch').toBe('WRITE_OFF');
    expect(row.previous_qty - row.new_qty, 'the row states the movement it made').toBe(2);

    // The ledger must reconcile against the row it describes: replaying every
    // movement for this product from its opening stock must land on the live
    // figure. This is what the reconciliation panel claims, asserted directly.
    const inventory = await getInventory(admin, product.id);
    const newest = ledger
      .slice()
      .sort((a, b) => a.created_at.localeCompare(b.created_at))
      .filter((r) => ['ADD', 'REMOVE', 'ADJUST', 'COMMIT', 'WRITE_OFF', 'RETURN'].includes(r.type))
      .at(-1);
    expect(newest, 'some movement should have touched on-hand').toBeTruthy();
    expect(
      newest!.new_qty,
      'the newest on-hand movement must agree with live inventory'
    ).toBe(inventory.quantity);
  });

  test('a dispatch and a write-off on one order stay distinguishable', async () => {
    fx = await buyProducts(admin, [{ stock: 12, buy: 5 }]);
    const product = fx.products[0]!;
    const line = fx.order.items[0]!;

    await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OUT_OF_STOCK',
          items: [{ order_item_id: line.id, quantity: 1, restock: false }],
        },
      })
    );

    for (const status of ['CONFIRMED', 'PROCESSING', 'SHIPPED']) {
      await admin.patch(`/admin/orders/${fx.order.id}/status`, { data: { status } });
    }

    const ledger = await getLedger(admin, product.id);
    expect(rowsForOrder(ledger, fx.order.id, 'WRITE_OFF'), 'the refund').toHaveLength(1);
    expect(rowsForOrder(ledger, fx.order.id, 'COMMIT'), 'the dispatch').toHaveLength(1);

    const inventory = await getInventory(admin, product.id);
    expect(
      inventory.quantity,
      '12 on hand, 1 written off, 4 dispatched'
    ).toBe(7);
    expect(inventory.reserved_qty, 'nothing left reserved for this order').toBe(0);
  });
});
