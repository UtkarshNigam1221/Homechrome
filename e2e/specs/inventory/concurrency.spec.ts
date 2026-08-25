import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getInventory } from '../../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { expectLedgerBalances } from '../../fixtures/reconcile';
import { customerClient, prepareCheckout } from '../../helpers/order';

/**
 * #230 case 10 — oversell under concurrency.
 *
 * `SELECT ... FOR UPDATE` serialises the reserve, but the availability check
 * and the reserve are separate steps across two datastores, so the interesting
 * question is whether the loser is refused cleanly rather than deadlocking or
 * driving available_qty negative.
 *
 * Used to race two POST /admin/orders calls for one customer and product. That
 * path is retired, so this now races two POST /checkout/initiate calls against
 * one cart instead — a double-submitted Pay button, a real scenario the admin
 * race never was. Both calls read the same cart (3 units) and independently
 * attempt to reserve it against the same 5-unit stock at
 * ReserveOrderStock/inventory_repository.go's per-product `FOR UPDATE`, so the
 * property under test is unchanged: two attempts, one lock, one winner.
 */
test.describe('concurrent orders for the last units', () => {
  let api: APIRequestContext;
  let store: APIRequestContext;
  let catalog: SeededCatalog | undefined;

  test.beforeAll(async () => {
    api = await adminClient();
    store = await customerClient();
  });

  test.afterEach(async () => {
    if (catalog) await expectLedgerBalances(api, catalog.products[0]!, 'after concurrency spec');
    await destroyCatalog(api, catalog);
    catalog = undefined;
  });

  test('exactly one of two racing checkouts wins, and stock never goes negative', async () => {
    // 5 on hand, one cart of 3, initiated twice at once. Only one can win.
    catalog = await seedCatalog(api, [5]);
    const product = catalog.products[0]!;

    const addressId = await prepareCheckout(store, [{ productId: product.id, quantity: 3 }]);

    const [a, b] = await Promise.all([
      store.post('/api/v1/store/checkout/initiate', { data: { shipping_address_id: addressId } }),
      store.post('/api/v1/store/checkout/initiate', { data: { shipping_address_id: addressId } }),
    ]);

    const statuses = [a.status(), b.status()].sort();
    const winners = statuses.filter((s) => s < 400);
    const losers = statuses.filter((s) => s >= 400);

    expect(winners, `exactly one checkout may win (got ${statuses.join(', ')})`).toHaveLength(1);
    expect(losers).toHaveLength(1);

    const loserBody = a.status() >= 400 ? await a.text() : await b.text();
    expect(loserBody, 'the loser must be refused for stock, not a deadlock').toMatch(
      /stock|INSUFFICIENT/i
    );
    expect(loserBody).not.toMatch(/deadlock/i);

    const inventory = await getInventory(api, product.id);
    expect(inventory.reserved_qty, 'only the winner holds stock').toBe(3);
    expect(inventory.available_qty, 'available must never go negative').toBe(2);
    expect(inventory.available_qty).toBeGreaterThanOrEqual(0);
  });
});
