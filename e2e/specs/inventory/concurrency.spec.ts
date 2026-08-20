import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getInventory } from '../../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { expectLedgerBalances } from '../../fixtures/reconcile';
import { attemptAdminOrder, resolveTestCustomerId } from '../../helpers/order';

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
  let customerId: string;
  let catalog: SeededCatalog | undefined;

  test.beforeAll(async () => {
    api = await adminClient();
    customerId = await resolveTestCustomerId(api);
  });

  test.afterEach(async () => {
    if (catalog) await expectLedgerBalances(api, catalog.products[0]!, 'after concurrency spec');
    await destroyCatalog(api, catalog);
    catalog = undefined;
  });

  test('exactly one of two racing orders wins, and stock never goes negative', async () => {
    // 5 on hand, two orders of 3. Only one can be satisfied.
    catalog = await seedCatalog(api, [5]);
    const product = catalog.products[0]!;

    const [a, b] = await Promise.all([
      attemptAdminOrder(api, customerId, [{ productId: product.id, quantity: 3 }]),
      attemptAdminOrder(api, customerId, [{ productId: product.id, quantity: 3 }]),
    ]);

    const statuses = [a.status(), b.status()].sort();
    const winners = statuses.filter((s) => s < 400);
    const losers = statuses.filter((s) => s >= 400);

    expect(winners, `exactly one order may win (got ${statuses.join(', ')})`).toHaveLength(1);
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
