import { expect, test } from '@playwright/test';

import { adminClient, json, Refund } from '../../fixtures/api';
import { loginToAdmin } from '../../pages/admin/login';
import { buyProducts, PaidFixture, releaseFixture } from '../../helpers/paid-order';

/**
 * The fourth manual check, UI half. The API half — that OPERATOR is refused
 * over HTTP — is in specs/refunds/rbac.spec.ts; hiding a button is a courtesy,
 * the 403 is the control.
 *
 * OrderDetailPage gates the button on three things at once
 * (OrderDetailPage.tsx): ADMIN, a payment that actually arrived, and at least
 * one line with unrefunded units. Each is asserted separately, because a gate
 * that passes for the wrong reason is a gate that will fail silently later.
 */
test.describe('the Refund button', () => {
  let fx: PaidFixture | undefined;

  test.afterEach(async () => {
    await releaseFixture(fx);
    fx = undefined;
  });

  test('is offered to an admin on a paid order', async ({ page }) => {
    fx = await buyProducts(await adminClient(), [{ stock: 8, buy: 2 }]);

    await loginToAdmin(page, 'admin');
    await page.goto(`/orders/${fx.order.id}`);

    await expect(page.getByRole('button', { name: 'Refund', exact: true })).toBeVisible();
  });

  test('is hidden from an operator on the same order', async ({ page }) => {
    fx = await buyProducts(await adminClient(), [{ stock: 8, buy: 2 }]);

    await loginToAdmin(page, 'operator');
    await page.goto(`/orders/${fx.order.id}`);

    // The page itself must still render — an operator has a legitimate reason
    // to read an order; they simply cannot refund it.
    await expect(page.getByText(fx.order.order_number)).toBeVisible();
    await expect(page.getByRole('button', { name: 'Refund', exact: true })).toBeHidden();
  });

  test('disappears once every unit has been refunded', async ({ page }) => {
    const admin = await adminClient();
    fx = await buyProducts(admin, [{ stock: 8, buy: 2 }]);
    const line = fx.order.items[0]!;

    const refund = await json<Refund>(
      await admin.post(`/admin/orders/${fx.order.id}/refunds`, {
        data: {
          reason: 'OUT_OF_STOCK',
          items: [{ order_item_id: line.id, quantity: 2, restock: false }],
        },
      })
    );
    await admin.post(`/admin/orders/${fx.order.id}/refunds/${refund.id}/recheck`);

    await loginToAdmin(page, 'admin');
    await page.goto(`/orders/${fx.order.id}`);

    await expect(
      page.getByRole('button', { name: 'Refund', exact: true }),
      'nothing left to refund'
    ).toBeHidden();
  });
});
