import { defineConfig, devices } from '@playwright/test';

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

  // Stop at the first failure in CI. With retries off, a broken run reports in
  // the time of one spec instead of grinding through the rest to tell you the
  // same thing.
  maxFailures: process.env.CI ? 1 : 0,

  // Order-scoped inventory is shared mutable state in one Postgres row per
  // product. Specs create their own products to stay independent, but the
  // refund money-arc specs walk one order through several states, so parallel
  // execution within a file would race. Files still run in parallel.
  fullyParallel: false,
  workers: process.env.CI ? 2 : 1,

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
