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
  // Deployed infrastructure: a cold Lambda plus a Neon cold start is slow.
  timeout: 90_000,
  expect: { timeout: 15_000 },

  // Order-scoped inventory is shared mutable state in one Postgres row per
  // product. Specs create their own products to stay independent, but the
  // refund money-arc specs walk one order through several states, so parallel
  // execution within a file would race. Files still run in parallel.
  fullyParallel: false,
  workers: process.env.CI ? 2 : 1,

  // A flaky retry against real infrastructure hides real flakiness. One retry
  // in CI only, to absorb a genuine Lambda cold-start timeout.
  retries: process.env.CI ? 1 : 0,
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
