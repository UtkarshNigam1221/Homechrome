import { expect, test } from '@playwright/test';

import { adminClient, getInventory, json, Refund } from '../../fixtures/api';
import { loginToAdmin } from '../../pages/admin/login';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';

/**
 * The sixth manual check, UI half.
 *
 * movementEffects.ts renders a WRITE_OFF row as "Written off"; the API half in
 * specs/refunds/ledger-writeoff.spec.ts asserts the underlying row. This checks
 * the two agree — that the screen an operator reads matches the database.
 */
test.describe('the stock ledger', () => {
  let fx: PaidFixture | undefined;

  test.afterEach(async () => {
    await releaseFixture(fx);
    fx = undefined;
  });

  test('shows the refund as "Written off" and agrees with live stock', async ({ page }) => {
    const admin = await adminClient();
    fx = await buyProducts(admin, [{ stock: 9, buy: 3 }]);
    const product = fx.products[0]!;
    const line = fx.order.items[0]!;

    await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'DAMAGED',
          items: [{ order_item_id: line.id, quantity: 2, restock: false }],
        },
      })
    );

    const inventory = await getInventory(admin, product.id);

    await loginToAdmin(page, 'admin');
    await page.goto(`/products/${product.id}`);

    await expect(
      page.getByText('Written off').first(),
      'a write-off must not render as a dispatch'
    ).toBeVisible();

    // The figure on screen must be the figure in the database. A ledger that
    // renders beautifully and disagrees with stock is worse than no ledger.
    await expect(
      page.getByText(String(inventory.quantity), { exact: false }).first()
    ).toBeVisible();
  });
});
