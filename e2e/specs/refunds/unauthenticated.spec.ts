import { expect, request, test } from '@playwright/test';

import { TARGETS } from '../../playwright.config';

/**
 * #230 case 34 — every admin endpoint the refund and inventory work touches
 * must reject an unauthenticated caller.
 *
 * The role gate is asserted in rbac.spec.ts. This is the layer below it: the
 * one that stops the internet. It runs with no cookie jar at all, so a route
 * accidentally mounted outside the authenticated group shows up here rather
 * than in production.
 */
const GUARDED = [
  ['GET', '/admin/orders'],
  ['GET', '/admin/orders/order_probe'],
  ['GET', '/admin/orders/order_probe/refunds'],
  ['POST', '/admin/orders/order_probe/refunds'],
  ['POST', '/admin/orders/order_probe/refunds/preview'],
  ['POST', '/admin/orders/order_probe/refunds/refund_probe/recheck'],
  ['POST', '/admin/orders/order_probe/cancel'],
  ['PATCH', '/admin/orders/order_probe/status'],
  ['GET', '/admin/products'],
  ['GET', '/admin/products/prod_probe/inventory'],
  ['GET', '/admin/products/prod_probe/inventory/transactions'],
  ['POST', '/admin/products/prod_probe/inventory/add'],
  ['POST', '/admin/products/prod_probe/inventory/remove'],
  ['POST', '/admin/products/prod_probe/inventory/adjust'],
  ['GET', '/admin/inventory/low-stock'],
] as const;

test.describe('unauthenticated access', () => {
  for (const [method, path] of GUARDED) {
    test(`${method} ${path} is refused without a session`, async () => {
      const anon = await request.newContext({ baseURL: TARGETS.api });
      const res = await anon.fetch(path, { method, data: method === 'GET' ? undefined : {} });
      const status = res.status();
      await anon.dispose();

      // 401 is the intent. 403 is acceptable — still refused. Anything 2xx is a
      // hole, and a 404 would mean the route moved and this test rotted.
      expect(
        [401, 403],
        `${method} ${path} answered ${status} to an anonymous caller`
      ).toContain(status);
    });
  }
});
