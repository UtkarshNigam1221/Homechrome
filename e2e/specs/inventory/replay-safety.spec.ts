import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getInventory, getLedger, rowsForOrder } from '../../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { expectAllLedgersBalance } from '../../fixtures/reconcile';
import { customerClient, placePaidOrder, placeUnpaidOrder } from '../../helpers/order';

/**
 * #230 cases 6, 8 and 9.
 *
 * GetByID is a DynamoDB read without ConsistentRead and Update is an
 * unconditional PutItem, so a client that times out and retries can re-drive a
 * transition that already succeeded. Migration 013's unique partial index makes
 * the second movement a no-op rather than a second decrement; these assert that
 * from outside, over real HTTP.
 */
test.describe('replay safety', () => {
  let api: APIRequestContext;
  let store: APIRequestContext;
  let catalog: SeededCatalog | undefined;

  test.beforeAll(async () => {
    api = await adminClient();
    store = await customerClient();
  });

  test.afterEach(async () => {
    // #230 case 35: the ledger must replay to the live balance after every
    // scenario, not only the ones that thought to check.
    if (catalog) await expectAllLedgersBalance(api, catalog.products);
    await destroyCatalog(api, catalog);
    catalog = undefined;
  });

  test('a repeated dispatch moves stock once', async () => {
    catalog = await seedCatalog(api, [10]);
    const product = catalog.products[0]!;
    const { order } = await placePaidOrder(store, [{ productId: product.id, quantity: 4 }]);

    await api.patch(`/admin/orders/${order.id}/status`, { data: { status: 'PROCESSING' } });
    await api.patch(`/admin/orders/${order.id}/status`, { data: { status: 'SHIPPED' } });

    const afterFirst = await getInventory(api, product.id);

    // Same transition again. Whether the API accepts or refuses it, stock must
    // not move twice — that is the invariant, not the status code.
    await api.patch(`/admin/orders/${order.id}/status`, { data: { status: 'SHIPPED' } });

    const afterSecond = await getInventory(api, product.id);
    expect(afterSecond.quantity).toBe(afterFirst.quantity);
    expect(afterSecond.reserved_qty).toBe(afterFirst.reserved_qty);

    expect(
      rowsForOrder(await getLedger(api, product.id), order.id, 'COMMIT'),
      'one COMMIT row per (product, order) — the 013 invariant'
    ).toHaveLength(1);
  });

  for (const status of ['PENDING', 'CONFIRMED', 'PROCESSING'] as const) {
    test(`cancel releases once, from ${status}`, async () => {
      catalog = await seedCatalog(api, [10]);
      const product = catalog.products[0]!;
      const order =
        status === 'PENDING'
          ? await placeUnpaidOrder(store, [{ productId: product.id, quantity: 3 }])
          : (await placePaidOrder(store, [{ productId: product.id, quantity: 3 }])).order;

      if (status === 'PROCESSING') {
        await api.patch(`/admin/orders/${order.id}/status`, { data: { status: 'PROCESSING' } });
      }

      const cancelled = await api.post(`/admin/orders/${order.id}/cancel`, {
        data: { reason: 'e2e' },
      });
      expect(cancelled.ok(), `cancel from ${status}: ${await cancelled.text()}`).toBeTruthy();

      // Replay the cancel.
      await api.post(`/admin/orders/${order.id}/cancel`, { data: { reason: 'e2e replay' } });

      const inventory = await getInventory(api, product.id);
      expect(inventory.reserved_qty, `cancel from ${status} must release`).toBe(0);
      expect(inventory.quantity, `cancel from ${status} must not touch on-hand`).toBe(10);
      expect(
        rowsForOrder(await getLedger(api, product.id), order.id, 'RELEASE'),
        `cancel from ${status} must release exactly once`
      ).toHaveLength(1);
    });
  }

  test('a dispatched order refuses to cancel', async () => {
    catalog = await seedCatalog(api, [10]);
    const product = catalog.products[0]!;
    const { order } = await placePaidOrder(store, [{ productId: product.id, quantity: 2 }]);

    await api.patch(`/admin/orders/${order.id}/status`, { data: { status: 'PROCESSING' } });
    await api.patch(`/admin/orders/${order.id}/status`, { data: { status: 'SHIPPED' } });

    const res = await api.post(`/admin/orders/${order.id}/cancel`, { data: { reason: 'e2e' } });
    expect(res.status(), 'SHIPPED is past the point of cancelling').toBeGreaterThanOrEqual(400);
  });
});
