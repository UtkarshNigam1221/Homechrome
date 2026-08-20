import { expect, test } from '@playwright/test';

import { adminClient, getInventory, json, Refund } from '../../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { InventoryPage } from '../../pages/admin/inventory';
import { loginToAdmin } from '../../pages/admin/login';
import { createAdminOrder, resolveTestCustomerId } from '../../helpers/order';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';

/**
 * #230 cases 12, 15, 16, 26 and 32.
 *
 * The ledger lives in a modal on /inventory, opened per row — not on a product
 * page. Row actions are icon buttons with no text, identified by title.
 * pages/admin/inventory.ts owns both facts.
 *
 * Cases 13 and 14 are not browser tests: reconciliation is
 * scripts/reconcile-inventory, a Go CLI driven by a workflow, and its semantics
 * are pinned against a disposable database in internal/repository/postgres.
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
    const modal = await new InventoryPage(page).openLedger(product.sku);

    // A row records a before/after for one counter only; showing half of it is
    // what made the history unreadable.
    await expect(modal.getByText('Dispatched').first()).toBeVisible();
    await expect(modal.getByText('Reserved').first()).toBeVisible();
  });

  test('Remove Stock removes after Add Stock was opened first (#230 case 15)', async ({ page }) => {
    const admin = await adminClient();
    catalog = await seedCatalog(admin, [20]);
    const product = catalog.products[0]!;

    await loginToAdmin(page, 'admin');
    const inventory = new InventoryPage(page);

    // react-hook-form evaluates defaultValues once and the modal stays mounted,
    // which is what made the second open reuse the first operation's type.
    const row = await inventory.findProduct(product.sku);
    await row.locator('[title="Add stock"]').click();
    await inventory.closeModal();

    await inventory.adjustStock(product.sku, 'Remove stock', 4, 'e2e remove');

    const after = await getInventory(admin, product.id);
    expect(after.quantity, 'the second modal must remove, not add').toBe(16);
  });

  test('the ledger names the actor, not their id (#230 case 16)', async ({ page }) => {
    const admin = await adminClient();
    catalog = await seedCatalog(admin, [5]);
    const product = catalog.products[0]!;

    await admin.post(`/admin/products/${product.id}/inventory/add`, {
      data: { quantity: 2, reason: 'e2e actor check' },
    });

    await loginToAdmin(page, 'admin');
    const modal = await new InventoryPage(page).openLedger(product.sku);

    await expect(modal.getByText('Restocked').first()).toBeVisible();
    await expect(
      modal.getByText(/usr_[a-z0-9_]+/),
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

  test('a refund cannot be submitted without a reason (#230 case 32)', async ({ page }) => {
    const admin = await adminClient();
    fx = await buyProducts(admin, [{ stock: 6, buy: 1 }]);

    await loginToAdmin(page, 'admin');
    await page.goto(`/orders/${fx.order.id}`);
    await page.getByRole('button', { name: 'Refund', exact: true }).click();

    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible();
    await modal.locator('input[type="number"]').first().fill('1');

    // The submit is disabled while reason is empty
    // (RefundModal: disabled={selectedUnits === 0 || reason === '' || isPricing}),
    // so the modal refuses by not offering the action rather than by rejecting
    // it afterwards. Asserting the disabled state is the honest check — the
    // earlier version clicked it and waited out the test timeout.
    const submit = modal.getByRole('button', { name: /refund/i }).last();
    await expect(submit, 'no reason picked, so there is nothing to submit').toBeDisabled();

    await modal.getByLabel('Reason').selectOption('OUT_OF_STOCK');
    await expect(submit, 'with a reason it becomes available').toBeEnabled();
  });
});
