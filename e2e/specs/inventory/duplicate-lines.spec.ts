import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getInventory, getLedger, rowsForOrder } from '../../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { expectAllLedgersBalance } from '../../fixtures/reconcile';
import { attemptAdminOrder, createAdminOrder, resolveTestCustomerId } from '../../helpers/order';

/**
 * #230 case 2 and 3, and the first of the manual checks.
 *
 * Only reachable through the admin path: the storefront cart keys on product id
 * so it cannot produce two lines of one product. Before #226 each line reserved
 * independently against the product total, so an order for 2 + 3 reserved 2 and
 * oversold the remaining 3.
 */
test.describe('duplicate product lines', () => {
  let api: APIRequestContext;
  let customerId: string;
  let catalog: SeededCatalog | undefined;

  test.beforeAll(async () => {
    api = await adminClient();
    customerId = await resolveTestCustomerId(api);
  });

  test.afterEach(async () => {
    // #230 case 35: the ledger must replay to the live balance after every
    // scenario, not only the ones that thought to check.
    if (catalog) await expectAllLedgersBalance(api, catalog.products);
    await destroyCatalog(api, catalog);
    catalog = undefined;
  });

  test('reserves the summed quantity across both lines, once', async () => {
    catalog = await seedCatalog(api, [10]);
    const product = catalog.products[0]!;

    const order = await createAdminOrder(api, customerId, [
      { productId: product.id, quantity: 2 },
      { productId: product.id, quantity: 3 },
    ]);

    const inventory = await getInventory(api, product.id);
    expect(inventory.reserved_qty, 'both lines must reserve, not just the first').toBe(5);
    expect(inventory.quantity, 'reserving must not touch on-hand').toBe(10);
    expect(inventory.available_qty).toBe(5);

    // One movement per (product, order, type) — the 013 invariant. Two rows of
    // 2 and 3 would satisfy the arithmetic but violate the ledger contract that
    // idempotency is built on.
    const reserves = rowsForOrder(await getLedger(api, product.id), order.id, 'RESERVE');
    expect(reserves).toHaveLength(1);
    expect(reserves[0]!.quantity).toBe(5);
  });

  test('rejects on the aggregate, and persists no order when it does', async () => {
    catalog = await seedCatalog(api, [5]);
    const product = catalog.products[0]!;

    // 3 + 3 against 5 available. Per-line checks pass; the aggregate must not.
    const res = await attemptAdminOrder(api, customerId, [
      { productId: product.id, quantity: 3 },
      { productId: product.id, quantity: 3 },
    ]);

    expect(res.status(), await res.text()).toBeGreaterThanOrEqual(400);
    const body = await res.text();
    expect(body).toMatch(/stock|INSUFFICIENT/i);

    const inventory = await getInventory(api, product.id);
    expect(inventory.reserved_qty, 'a rejected order must reserve nothing').toBe(0);
    expect(inventory.quantity).toBe(5);

    const ledger = await getLedger(api, product.id);
    expect(
      ledger.filter((r) => r.type === 'RESERVE'),
      'a rejected order must leave no RESERVE row'
    ).toHaveLength(0);
  });

  test('commits and returns the summed quantity through the lifecycle', async () => {
    catalog = await seedCatalog(api, [10]);
    const product = catalog.products[0]!;

    const order = await createAdminOrder(api, customerId, [
      { productId: product.id, quantity: 2 },
      { productId: product.id, quantity: 3 },
    ]);

    for (const status of ['CONFIRMED', 'PROCESSING', 'SHIPPED']) {
      const res = await api.patch(`/admin/orders/${order.id}/status`, { data: { status } });
      expect(res.ok(), `${status}: ${await res.text()}`).toBeTruthy();
    }

    let inventory = await getInventory(api, product.id);
    expect(inventory.quantity, 'dispatch drops on-hand by the summed quantity').toBe(5);
    expect(inventory.reserved_qty).toBe(0);
    expect(inventory.available_qty, 'available must not move on dispatch').toBe(5);

    const commits = rowsForOrder(await getLedger(api, product.id), order.id, 'COMMIT');
    expect(commits).toHaveLength(1);
    expect(commits[0]!.quantity).toBe(5);

    const returned = await api.patch(`/admin/orders/${order.id}/status`, {
      data: { status: 'RETURNED' },
    });
    expect(returned.ok(), await returned.text()).toBeTruthy();

    inventory = await getInventory(api, product.id);
    expect(inventory.quantity, 'a return restores the whole order, both lines').toBe(10);
    expect(inventory.available_qty).toBe(10);
  });
});
