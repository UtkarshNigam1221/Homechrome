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
