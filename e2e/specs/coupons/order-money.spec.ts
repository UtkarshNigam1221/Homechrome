import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, getOrder } from '../../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { Coupon, createCoupon, deleteCoupon, getCoupon } from '../../helpers/coupon';
import { customerClient, placeUnpaidOrder } from '../../helpers/order';

/**
 * What a coupon does to the money on a real order.
 *
 * The allocator is swept in unit tests and the totals are asserted against
 * mocks. Neither proves that checkout writes the shares onto the lines it
 * persists, and every later refund reads those lines — a gap between the two
 * strands the order.
 *
 * Only the redemption case needs a payment; the rest read an unpaid order,
 * because checkout writes the lines and totals at initiate.
 */
test.describe('coupon money on the order', () => {
  let admin: APIRequestContext;
  let store: APIRequestContext;
  let catalog: SeededCatalog | undefined;
  let coupon: Coupon | undefined;

  test.beforeAll(async () => {
    admin = await adminClient();
    store = await customerClient();
  });

  test.afterEach(async () => {
    await deleteCoupon(admin, coupon);
    coupon = undefined;
    await destroyCatalog(admin, catalog);
    catalog = undefined;
  });

  // The invariant every refund depends on. Uneven line values on purpose, so a
  // naive per-line rounding would not sum back.
  test('the line shares sum to the order discount exactly', async () => {
    catalog = await seedCatalog(admin, [20, 20, 20]);
    coupon = await createCoupon(admin, { value: 1000 });

    const order = await placeUnpaidOrder(
      store,
      [
        { productId: catalog.products[0]!.id, quantity: 1 },
        { productId: catalog.products[1]!.id, quantity: 2 },
        { productId: catalog.products[2]!.id, quantity: 3 },
      ],
      coupon.code
    );

    expect(order.discount_amount, 'the coupon reached the order').toBeGreaterThan(0);

    const lineSum = order.items.reduce((n, i) => n + (i.discount_amount ?? 0), 0);
    expect(lineSum, 'a gap here strands every later refund').toBe(order.discount_amount);

    for (const item of order.items) {
      const lineValue = item.unit_price * item.quantity;
      expect(item.discount_amount ?? 0, 'no line is discounted past its own value')
        .toBeLessThanOrEqual(lineValue);
      expect(item.discount_amount ?? 0).toBeGreaterThanOrEqual(0);
    }
  });

  test('the total is the subtotal less the discount, with tax taken out of it', async () => {
    catalog = await seedCatalog(admin, [20]);
    coupon = await createCoupon(admin, { value: 1000 });

    const order = await placeUnpaidOrder(
      store,
      [{ productId: catalog.products[0]!.id, quantity: 2 }],
      coupon.code
    );

    expect(order.total_amount, 'shipping is free, so this is subtotal - discount').toBe(
      order.subtotal - order.discount_amount + order.shipping_amount
    );
    // Tax-inclusive: contained within the total, never added to it.
    expect(order.tax_amount).toBeGreaterThan(0);
    expect(order.tax_amount, 'GST is a slice of the total, not a surcharge').toBeLessThan(
      order.total_amount
    );
  });

  // Counting at order creation would let an abandoned cart burn a single-use
  // code permanently. The order exists here and is never paid.
  test('an unpaid order does not spend a usage slot', async () => {
    catalog = await seedCatalog(admin, [20]);
    coupon = await createCoupon(admin, { usageLimit: 1 });

    const order = await placeUnpaidOrder(
      store,
      [{ productId: catalog.products[0]!.id, quantity: 1 }],
      coupon.code
    );
    expect(order.discount_amount).toBeGreaterThan(0);

    const after = await getCoupon(admin, coupon.id);
    expect(after.usage_count, 'the slot is claimed at payment success, not here').toBe(0);
  });

  // An operator edit must not reset the counter, and must not resurrect a
  // coupon's remaining uses.
  test('an admin edit leaves the usage count alone', async () => {
    coupon = await createCoupon(admin, { usageLimit: 5 });

    const patched = await admin.patch(`/admin/coupons/${coupon.id}`, {
      data: { description: 'edited by the e2e suite' },
    });
    expect(patched.ok(), await patched.text()).toBeTruthy();

    const after = await getCoupon(admin, coupon.id);
    expect(after.usage_count).toBe(0);
    expect(after.usage_limit, 'the limit survives an unrelated edit').toBe(5);
    expect(after.description).toBe('edited by the e2e suite');
  });

  // Deleting has to remove the code pointer too, or the code is burnt forever.
  test('deleting a coupon frees its code', async () => {
    const first = await createCoupon(admin, {}, 'FREED');
    const code = first.code;

    const dup = await admin.post('/admin/coupons', {
      data: {
        code,
        name: code,
        type: 'PERCENTAGE',
        value: 1000,
        audience: 'ALL',
        combines_with_offers: false,
        valid_from: new Date().toISOString(),
      },
    });
    expect(dup.status(), 'the code is taken while the coupon lives').toBe(409);

    await deleteCoupon(admin, first);

    const again = await admin.post('/admin/coupons', {
      data: {
        code,
        name: code,
        type: 'PERCENTAGE',
        value: 1000,
        audience: 'ALL',
        combines_with_offers: false,
        valid_from: new Date().toISOString(),
      },
    });
    expect(again.status(), 'deleting must release the code pointer').toBe(201);
    coupon = await again.json();
  });
});
