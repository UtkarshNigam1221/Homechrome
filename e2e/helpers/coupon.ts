import { APIRequestContext } from '@playwright/test';

import { json } from '../fixtures/api';

export interface Coupon {
  id: string;
  code: string;
  description: string;
  type: 'PERCENTAGE' | 'FIXED';
  value: number;
  min_order_value: number;
  max_discount: number;
  usage_limit: number;
  usage_per_user: number;
  usage_count: number;
  status: string;
  valid_from: string;
  valid_until?: string | null;
}

export interface CouponSpec {
  type?: 'PERCENTAGE' | 'FIXED';
  /** Percentage × 100 (1000 = 10%), or paise for FIXED. */
  value?: number;
  minOrderValue?: number;
  maxDiscount?: number;
  usageLimit?: number;
  usagePerUser?: number;
  validFrom?: Date;
  validUntil?: Date | null;
}

/**
 * A code no other run can collide with. The suite creates real coupons against
 * shared dev, and a fixed code would make two runs fight over one usage limit.
 */
function uniqueCode(prefix: string): string {
  const run = process.env.E2E_RUN_ID ?? 'local';
  const salt = Math.random().toString(36).slice(2, 8).toUpperCase();
  return `${prefix}${run}${salt}`.replace(/[^A-Z0-9]/gi, '').toUpperCase().slice(0, 24);
}

/** Creates a coupon and returns it. Defaults to 10% off with no limits. */
export async function createCoupon(
  admin: APIRequestContext,
  spec: CouponSpec = {},
  prefix = 'E2E'
): Promise<Coupon> {
  const code = uniqueCode(prefix);
  const from = spec.validFrom ?? new Date(Date.now() - 60 * 60 * 1000);
  // valid_until omitted entirely when null — an open-ended coupon, which is a
  // different stored shape from one with a far-future date.
  const until = spec.validUntil === null ? undefined : (spec.validUntil ?? new Date(Date.now() + 30 * 24 * 60 * 60 * 1000));

  const body: Record<string, unknown> = {
    code,
    name: code,
    description: 'created by the e2e suite',
    type: spec.type ?? 'PERCENTAGE',
    value: spec.value ?? 1000,
    min_order_value: spec.minOrderValue ?? 0,
    max_discount: spec.maxDiscount ?? 0,
    usage_limit: spec.usageLimit ?? 0,
    usage_per_user: spec.usagePerUser ?? 0,
    audience: 'ALL',
    combines_with_offers: false,
    valid_from: from.toISOString(),
  };
  if (until) body.valid_until = until.toISOString();

  const res = await admin.post('/admin/coupons', { data: body });
  if (!res.ok()) {
    throw new Error(`create coupon failed: ${res.status()} ${await res.text()}`);
  }
  return json<Coupon>(res);
}

/** Best-effort teardown. A leftover coupon is inert, but it clutters the admin list. */
export async function deleteCoupon(
  admin: APIRequestContext,
  coupon: Coupon | undefined
): Promise<void> {
  if (!coupon) return;
  await admin.delete(`/admin/coupons/${coupon.id}`).catch(() => undefined);
}

/** Reads a coupon back, which is how usage_count is observed. */
export async function getCoupon(admin: APIRequestContext, id: string): Promise<Coupon> {
  return json<Coupon>(await admin.get(`/admin/coupons/${id}`));
}

export interface CouponPreview {
  valid: boolean;
  code: string;
  coupon_id?: string;
  discount_amount?: number;
  error_message?: string;
  notice?: string;
}

/**
 * Previews a code against the customer's current cart. The cart total is read
 * server-side, so this is the same figure checkout will price against.
 */
export async function previewCoupon(
  store: APIRequestContext,
  code: string
): Promise<CouponPreview> {
  return json<CouponPreview>(
    await store.post('/api/v1/store/checkout/validate-coupon', { data: { code } })
  );
}
