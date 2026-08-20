import { chromium, Page } from '@playwright/test';

/**
 * The only file that knows what PhonePe's hosted page looks like.
 *
 * Dev points at api-preprod.phonepe.com/apis/pg-sandbox, so checkout returns a
 * real mercury-uat.phonepe.com URL and the payment genuinely has to be driven.
 * Everything else in the suite is HTTP; this launches its own browser for the
 * payment leg only, so the refund specs stay in the api project rather than
 * being dragged into a browser project wholesale.
 *
 * Every selector below was read off the live UAT page, not guessed. The ids are
 * semantic (#card-number, #card-cvv) while the class names are content-hashed
 * build output, so ids are what this keys on.
 *
 * A DOM change here is a one-file fix. That is the whole reason it is one file.
 */

export interface TestCard {
  number: string;
  name: string;
  expiry: string; // MM/YY
  cvv: string;
  otp?: string; // 3-D Secure step, if the simulator asks
}

/** Reads the UAT instrument from the environment. Absent → callers should skip. */
export function testCardFromEnv(): TestCard | undefined {
  const number = process.env.E2E_PHONEPE_CARD_NUMBER;
  const expiry = process.env.E2E_PHONEPE_CARD_EXPIRY;
  const cvv = process.env.E2E_PHONEPE_CARD_CVV;
  if (!number || !expiry || !cvv) return undefined;
  return {
    number,
    expiry,
    cvv,
    name: process.env.E2E_PHONEPE_CARD_NAME ?? 'E2E Suite',
    otp: process.env.E2E_PHONEPE_CARD_OTP,
  };
}

/**
 * Drives the hosted page to completion and resolves once PhonePe redirects back
 * to the storefront. Does not assert the order is PAID — settlement is
 * asynchronous, so the caller polls payment-status afterwards.
 */
export async function payWithSandbox(redirectUrl: string, card: TestCard): Promise<void> {
  const browser = await chromium.launch();
  try {
    const page = await browser.newPage();
    await page.goto(redirectUrl, { waitUntil: 'domcontentloaded', timeout: 60_000 });

    // The page renders its options client-side; the card form does not exist
    // until this row is chosen.
    await page.getByText('Debit/Credit Card', { exact: false }).first().click();
    await page.locator('#card-number').waitFor({ state: 'visible', timeout: 20_000 });

    await page.locator('#card-number').fill(card.number);
    await page.locator('#card-name').fill(card.name);
    await page.locator('#card-validity').fill(card.expiry);
    await page.locator('#card-cvv').fill(card.cvv);

    await page.getByRole('button', { name: /proceed/i }).click();

    await settle3DS(page, card);

    // Back on the storefront confirmation route means PhonePe is done with us.
    await page.waitForURL(/homechrome\.in|confirmation|order/i, { timeout: 90_000 });
  } finally {
    await browser.close();
  }
}

/**
 * The 3-D Secure step. UAT simulators vary — some show an OTP field, some a
 * bare Success button, some nothing at all — so each shape is attempted and a
 * miss is not fatal: if the flow already completed, waitForURL above succeeds
 * regardless.
 */
async function settle3DS(page: Page, card: TestCard): Promise<void> {
  await page.waitForTimeout(4_000);

  if (card.otp) {
    const otpField = page
      .locator('input[type="tel"], input[type="text"], input[type="password"]')
      .filter({ hasNot: page.locator('#card-number, #card-name, #card-validity, #card-cvv') })
      .first();
    if (await otpField.isVisible().catch(() => false)) {
      await otpField.fill(card.otp).catch(() => undefined);
    }
  }

  for (const name of [/submit/i, /confirm/i, /success/i, /^pay$/i, /proceed/i]) {
    const button = page.getByRole('button', { name }).first();
    if (await button.isVisible().catch(() => false)) {
      await button.click().catch(() => undefined);
      await page.waitForTimeout(3_000);
      return;
    }
  }
}
