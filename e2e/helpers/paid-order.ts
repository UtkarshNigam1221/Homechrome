import { APIRequestContext } from '@playwright/test';

import { Order } from '../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog, SeededProduct } from '../fixtures/catalog';
import { customerClient, placePaidOrder } from './order';

/**
 * A refund needs a real payment row, so refund specs cannot use the admin order
 * path. This wraps the storefront purchase into one call and hands back
 * everything the assertions need.
 */
export interface PaidFixture {
  order: Order;
  catalog: SeededCatalog;
  products: SeededProduct[];
  store: APIRequestContext;
  admin: APIRequestContext;
}

export async function buyProducts(
  admin: APIRequestContext,
  spec: { stock: number; buy: number; price?: number }[]
): Promise<PaidFixture> {
  const catalog = await seedCatalog(
    admin,
    spec.map((s) => s.stock)
  );

  const store = await customerClient();
  const { order } = await placePaidOrder(
    store,
    catalog.products.map((p, i) => ({ productId: p.id, quantity: spec[i]!.buy }))
  );

  return { order, catalog, products: catalog.products, store, admin };
}

export async function releaseFixture(fixture: PaidFixture | undefined): Promise<void> {
  if (!fixture) return;
  // Cancel first: it releases whatever the order still holds. Deleting the
  // product would take the reservation with it anyway, but cancelling keeps the
  // order rows honest, and orders cannot be deleted at all.
  try {
    await fixture.admin.post(`/admin/orders/${fixture.order.id}/cancel`, {
      data: { reason: 'e2e teardown' },
    });
  } catch {
    // best effort; scripts/cleanup.ts sweeps the products regardless
  }
  await destroyCatalog(fixture.admin, fixture.catalog);
  await fixture.store.dispose();
}
