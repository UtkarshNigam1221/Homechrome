import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, operatorClient } from '../../fixtures/api';
import { expectLedgerBalances } from '../../fixtures/reconcile';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';

/**
 * #223 Tier 1.1, and the API half of the fourth manual check.
 *
 * There are no handler tests in the Go codebase at all, so nothing below this
 * asserts the role gate. It was verified by hand once and never since.
 *
 * NOTE — a deliberate divergence from #223. That issue says the refund *list*
 * "must stay open or an operator's order page breaks on a 403". #232 moved the
 * GET inside the admin group: "admin-only end to end, the read included". This
 * asserts what #232 actually does. If the operator order page is meant to show
 * refunds, that is a product decision to settle on #232, not a test to soften.
 *
 * orderId below is a fixture, not a real order. RequireRole (order_handler.go)
 * is mounted with r.Use ahead of every handler in the group and checks only
 * the caller's role, and ListByOrder (refund_service.go) queries refunds by
 * order_id with no existence check — so every case here, refusal and success
 * alike, resolves without a real order behind the id. unauthenticated.spec.ts
 * uses the same trick one layer below this.
 */
test.describe('refund routes are admin-only', () => {
  let admin: APIRequestContext;
  let operator: APIRequestContext;
  let catalog: SeededCatalog | undefined;
  const orderId = 'order_rbac_probe';

  test.beforeAll(async () => {
    admin = await adminClient();
    operator = await operatorClient();
    catalog = await seedCatalog(admin, [5]);
  });

  test.afterAll(async () => {
    await destroyCatalog(admin, catalog);
    await operator.dispose();
  });

  test('operator is refused when creating a refund', async () => {
    const res = await operator.post(`/admin/orders/${orderId}/refunds`, {
      data: { reason: 'OUT_OF_STOCK', items: [{ order_item_id: 'x', quantity: 1, restock: false }] },
    });
    expect(res.status(), 'refunds move money; OPERATOR must not').toBe(403);
  });

  test('operator is refused when re-checking a refund', async () => {
    const res = await operator.post(`/admin/orders/${orderId}/refunds/refund_nonexistent/recheck`);
    expect(res.status(), 're-check can settle a refund, so it is a write').toBe(403);
  });

  test('operator is refused when previewing a refund', async () => {
    const res = await operator.post(`/admin/orders/${orderId}/refunds/preview`, {
      data: { items: [{ order_item_id: 'x', quantity: 1 }] },
    });
    expect(res.status()).toBe(403);
  });

  test('operator is refused when listing refunds', async () => {
    const res = await operator.get(`/admin/orders/${orderId}/refunds`);
    expect(
      res.status(),
      'per #232 the read is admin-only too — see the note above if this fails'
    ).toBe(403);
  });

  test('admin may list refunds', async () => {
    const res = await admin.get(`/admin/orders/${orderId}/refunds`);
    expect(res.ok(), await res.text()).toBeTruthy();
  });

  test('the ledger still balances after the refusals', async () => {
    // #230 case 35. A refused request must move nothing; asserting it here
    // catches a 403 that rejected the response but not the side effect.
    await expectLedgerBalances(admin, catalog!.products[0]!, 'after rbac spec');
  });
});
