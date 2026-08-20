import { expect, test } from '@playwright/test';

import { adminClient, getInventory, json, Refund } from '../../fixtures/api';
import { createProduct, destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { loginToAdmin } from '../../pages/admin/login';
import { createAdminOrder, resolveTestCustomerId } from '../../helpers/order';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';

/**
 * #230 cases 12, 15, 16 and 32 — the #227 ledger UI.
 *
 * Cases 13 and 14 (the orphan badge and the reconciliation endpoint) are
 * fixme'd at the bottom: `FindOrphanReservations` exists in the repository and
 * has no HTTP route and no service caller, so there is nothing to drive yet.
 */
test.describe('the ledger UI', () => {
  let catalog: SeededCatalog | undefined;
  let fx: PaidFixture | undefined;

  test.afterEach(async () => {
    const admin = await adminClient();
    await destroyCatalog(admin, catalog);
    await releaseFixture(fx);
    catalog = undefined;
    fx = undefined;
  });

  test('a dispatch shows against both counters (#230 case 12)', async ({ page }) => {
    const admin = await adminClient();
    const customerId = await resolveTestCustomerId(admin);
    catalog = await seedCatalog(admin, [10]);
    const product = catalog.products[0]!;

    const order = await createAdminOrder(admin, customerId, [
      { productId: product.id, quantity: 3 },
    ]);
    for (const status of ['CONFIRMED', 'PROCESSING', 'SHIPPED']) {
      await admin.patch(`/admin/orders/${order.id}/status`, { data: { status } });
    }

    await loginToAdmin(page, 'admin');
    await page.goto(`/products/${product.id}`);

    // A row records a before/after for one counter only; showing half of it is
    // what made the history unreadable.
    await expect(page.getByText('Dispatched').first()).toBeVisible();
    const row = page.locator('tr', { hasText: 'Dispatched' }).first();
    await expect(row, 'a dispatch moves on-hand and reserved together').toContainText('-3');
  });

  test('Remove Stock removes after Add Stock was opened first (#230 case 15)', async ({ page }) => {
    const admin = await adminClient();
    catalog = await seedCatalog(admin, [20]);
    const product = catalog.products[0]!;

    await loginToAdmin(page, 'admin');
    await page.goto(`/products/${product.id}`);

    // react-hook-form evaluates defaultValues once and the modal stays mounted,
    // which is what made the second open reuse the first operation's type.
    await page.getByRole('button', { name: /add stock/i }).click();
    await page.keyboard.press('Escape');
    await page.getByRole('button', { name: /remove stock/i }).click();

    await page.getByLabel(/quantity/i).fill('4');
    await page.getByLabel(/reason/i).fill('e2e remove');
    await page.getByRole('button', { name: /^remove|confirm|save$/i }).click();

    await expect(page.getByText('Removed').first()).toBeVisible();

    const inventory = await getInventory(admin, product.id);
    expect(inventory.quantity, 'the second modal must remove, not add').toBe(16);
  });

  test('the ledger names the actor, not their id (#230 case 16)', async ({ page }) => {
    const admin = await adminClient();
    catalog = await seedCatalog(admin, [5]);
    const product = catalog.products[0]!;

    await admin.post(`/admin/products/${product.id}/inventory/add`, {
      data: { quantity: 2, reason: 'e2e actor check' },
    });

    await loginToAdmin(page, 'admin');
    await page.goto(`/products/${product.id}`);

    await expect(page.getByText('Restocked').first()).toBeVisible();
    await expect(
      page.getByText(/usr_[a-z0-9_]+/),
      'a raw user id is not an actor name'
    ).toHaveCount(0);
  });

  test('the refund modal caps a partly refunded line at its remainder (#230 case 26)', async ({
    page,
  }) => {
    const admin = await adminClient();
    fx = await buyProducts(admin, [{ stock: 10, buy: 3 }]);
    const line = fx.order.items[0]!;

    const refund = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OUT_OF_STOCK',
          items: [{ order_item_id: line.id, quantity: 1, restock: false }],
        },
      })
    );
    await admin.post(`/admin/orders/${fx.order.id}/refunds/${refund.id}/recheck`);

    await loginToAdmin(page, 'admin');
    await page.goto(`/orders/${fx.order.id}`);
    await page.getByRole('button', { name: 'Refund', exact: true }).click();

    const stepper = page.locator('input[type="number"]').first();
    await expect(stepper, '3 bought, 1 refunded, 2 left').toHaveAttribute('max', '2');
  });

  test('an empty reason is refused (#230 case 32)', async ({ page }) => {
    const admin = await adminClient();
    fx = await buyProducts(admin, [{ stock: 6, buy: 1 }]);

    await loginToAdmin(page, 'admin');
    await page.goto(`/orders/${fx.order.id}`);
    await page.getByRole('button', { name: 'Refund', exact: true }).click();

    await page.locator('input[type="number"]').first().fill('1');
    // Reason left at its empty default.
    const submit = page.getByRole('button', { name: /refund/i }).last();
    await submit.click();

    await expect(
      page.getByText(/reason/i).first(),
      'the modal must not submit without a bounded reason'
    ).toBeVisible();
  });

  /**
   * #230 cases 13 and 14 are not browser tests, and the issue's phrasing
   * predates what shipped. Reconciliation is not an endpoint or a panel: it is
   * scripts/reconcile-inventory, a weekly Go CLI driven by
   * .github/workflows/inventory-reconciliation.yml that exits non-zero when
   * stock is held against nothing. A failing run is the alert.
   *
   * Its semantics — minAge, limit, and a settled reservation not counting as an
   * orphan — are covered deterministically against a disposable database in
   * handloom-admin/internal/repository/postgres/reconciliation_integration_test.go,
   * which is where they belong. There is no orphan badge in the admin UI to
   * assert, by design.
   */
});
