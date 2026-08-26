import { defineConfig, devices } from '@playwright/test';

import { testPhones } from './fixtures/test-phone';

/**
 * Targets the deployed dev environment — real Lambdas, real DynamoDB, real
 * Neon, real PhonePe sandbox. Nothing here is mocked or stubbed; that is the
 * entire point of the tier. If a URL is missing the run fails loudly rather
 * than quietly falling back to localhost.
 */
function required(name: string): string {
  const v = process.env[name];
  if (!v) {
    throw new Error(
      `${name} is not set. This suite runs against deployed dev only — ` +
        `copy .env.example and fill it in, or supply it from CI secrets.`
    );
  }
  return v;
}

export const TARGETS = {
  api: required('API_URL'),
  admin: required('ADMIN_URL'),
  store: process.env.STORE_URL ?? '',
};

// Concurrent payment initiations PhonePe tolerates without throttling the
// merchant. Empirical, not documented: three was too many.
const MAX_PAYMENT_WORKERS = 2;

export default defineConfig({
  testDir: './specs',
  // Deployed infrastructure: a cold Lambda plus a Neon cold start is slow, and
  // a refund spec additionally seeds a catalog, launches a browser to pay the
  // PhonePe sandbox, and then polls for the webhook to settle. 90s covered none
  // of that; the first run against dev timed out mid-payment.
  // A passing refund spec genuinely needs this: seed a catalog, launch a
  // browser, load the hosted page, decode the QR, drive the simulator, wait for
  // the webhook. It is a ceiling for the happy path, not a budget to spend —
  // the individual awaits below fail far sooner when something is actually
  // wrong, so a broken run goes red in seconds rather than minutes.
  timeout: 240_000,
  expect: { timeout: 10_000 },

  // Run everything, even after a failure. While the suite is being stabilised
  // one dispatch that reports every problem beats several that each report one
  // — the last run stopped at the first failure and left 33 specs unrun.
  // Worth reinstating (maxFailures: 1) once it is reliably green, when the
  // point of a run flips from finding faults to gating a promotion.
  maxFailures: 0,

  // One worker per storefront customer, no more. A customer has exactly one
  // cart and placePaidOrder clears it before adding items, so two workers on
  // one customer clear each other mid-flight and checkout fails with "Cart is
  // empty". Deriving the count from the allowlist makes that unrepresentable:
  // add a number to E2E_STORE_PHONES (and to the backend's STORE_TEST_PHONES)
  // and the suite widens on its own; leave one and it stays serial.
  //
  // Products are per-spec and parallelise safely. The cart is the constraint.
  fullyParallel: false,

  // …and PhonePe is the tighter one. It throttles a burst of payment
  // initiations per merchant, so the suite's own parallelism is what trips it:
  // on four workers, three concurrent checkouts held the merchant throttled for
  // ~8s and lost three specs to a 500. Two keeps most of the wall-clock and
  // halves the burst. If 429s survive this, the next step is one.
  workers: Math.min(MAX_PAYMENT_WORKERS, Math.max(1, testPhones().length)),

  // No retries. Against real infrastructure a retry doubles the time to red and
  // buys a second opinion nobody asked for — the first run against dev spent
  // 90s failing, then 90s failing again for a different reason. Re-dispatch is
  // one click if a cold start really did cause it.
  retries: 0,
  forbidOnly: !!process.env.CI,

  reporter: process.env.CI
    ? [['github'], ['html', { open: 'never' }], ['list']]
    : [['list']],

  use: {
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    ignoreHTTPSErrors: false,
  },

  projects: [
    {
      // No browser. Real HTTP against the deployed API.
      name: 'api',
      testDir: './specs',
      testMatch: /specs\/(inventory|refunds)\/.*\.spec\.ts/,
      use: { baseURL: TARGETS.api },
    },
    {
      name: 'admin-ui',
      testDir: './specs/admin-ui',
      use: { ...devices['Desktop Chrome'], baseURL: TARGETS.admin },
    },
  ],
});
