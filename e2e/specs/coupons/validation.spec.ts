import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient } from '../../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { Coupon, createCoupon, deleteCoupon, previewCoupon } from '../../helpers/coupon';
import { customerClient, prepareCheckout } from '../../helpers/order';

/**
 * Every reason a coupon can be refused, over real HTTP against the deployed
 * validator. The service layer proves each branch against mocks; nothing proved
 * that the storefront's preview endpoint reaches them, or that the message a
 * customer sees is the message the branch produces.
 *
 * These drive no payment, so they cost nothing at the gateway.
 */
test.describe('coupon validation', () => {
  let admin: APIRequestContext;
  let store: APIRequestContext;
  let catalog: SeededCatalog | undefined;
  let coupon: Coupon | undefined;

  // One ₹300 cart for the whole file: validation reads the cart total and never
  // writes, so every case can share it.
  const unitPrice = 30000;

  test.beforeAll(async () => {
    admin = await adminClient();
    store = await customerClient();
    catalog = await seedCatalog(admin, [50]);
    await prepareCheckout(store, [{ productId: catalog.products[0]!.id, quantity: 1 }]);
  });

  test.afterEach(async () => {
    await deleteCoupon(admin, coupon);
    coupon = undefined;
  });

  test.afterAll(async () => {
    await destroyCatalog(admin, catalog);
  });

  test('a valid percentage code prices against the server cart', async () => {
    coupon = await createCoupon(admin, { value: 1000 });

    const preview = await previewCoupon(store, coupon.code);

    expect(preview.valid).toBeTruthy();
    expect(preview.discount_amount, '10% of ₹300').toBe(unitPrice / 10);
    expect(preview.coupon_id).toBe(coupon.id);
    expect(preview.error_message).toBeFalsy();
  });

  test('a fixed code takes its face value, in paise', async () => {
    coupon = await createCoupon(admin, { type: 'FIXED', value: 5000 });

    const preview = await previewCoupon(store, coupon.code);

    expect(preview.valid).toBeTruthy();
    expect(preview.discount_amount, '₹50 off, not ₹5000').toBe(5000);
  });

  test('max_discount caps a percentage code', async () => {
    coupon = await createCoupon(admin, { value: 5000, maxDiscount: 1000 });

    const preview = await previewCoupon(store, coupon.code);

    expect(preview.discount_amount, '50% of ₹300 capped at ₹10').toBe(1000);
  });

  test('an unknown code is refused', async () => {
    const preview = await previewCoupon(store, 'NOSUCHCODEEXISTS');

    expect(preview.valid).toBeFalsy();
    expect(preview.discount_amount ?? 0).toBe(0);
    expect(preview.error_message).toBeTruthy();
  });

  test('a code that is not active yet is refused', async () => {
    coupon = await createCoupon(admin, {
      validFrom: new Date(Date.now() + 24 * 60 * 60 * 1000),
      validUntil: new Date(Date.now() + 48 * 60 * 60 * 1000),
    });

    const preview = await previewCoupon(store, coupon.code);

    expect(preview.valid).toBeFalsy();
    expect(preview.error_message).toContain('active yet');
  });

  test('an expired code is refused', async () => {
    coupon = await createCoupon(admin, {
      validFrom: new Date(Date.now() - 48 * 60 * 60 * 1000),
      validUntil: new Date(Date.now() - 60 * 1000),
    });

    const preview = await previewCoupon(store, coupon.code);

    expect(preview.valid).toBeFalsy();
    expect(preview.error_message).toContain('expired');
  });

  // The message names the shortfall, so the customer knows what to add rather
  // than only that the code did not work.
  test('a cart below the minimum is told how much more it needs', async () => {
    coupon = await createCoupon(admin, { minOrderValue: unitPrice * 2 });

    const preview = await previewCoupon(store, coupon.code);

    expect(preview.valid).toBeFalsy();
    expect(preview.error_message).toContain('more to use this coupon');
  });

  // Applied and reduced, not refused: zeroing the total would make the gateway
  // reject the payment, so the coupon pays what it can and says so.
  test('a code worth more than the cart applies partially and says so', async () => {
    coupon = await createCoupon(admin, { type: 'FIXED', value: unitPrice * 10 });

    const preview = await previewCoupon(store, coupon.code);

    expect(preview.valid, 'refusing it outright would lose the sale').toBeTruthy();
    expect(preview.discount_amount, 'the cart less the ₹1 the gateway needs').toBe(unitPrice - 100);
    expect(preview.notice, 'the shortfall has to reach the customer').toBeTruthy();
  });

  test('an open-ended code stays valid', async () => {
    coupon = await createCoupon(admin, { validUntil: null });

    const preview = await previewCoupon(store, coupon.code);

    expect(preview.valid).toBeTruthy();
    expect(preview.discount_amount).toBe(unitPrice / 10);
  });
});
