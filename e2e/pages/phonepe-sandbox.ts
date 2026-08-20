import { chromium } from '@playwright/test';
import jsQR from 'jsqr';
import { PNG } from 'pngjs';

/**
 * The only file that knows what PhonePe's hosted page looks like.
 *
 * Dev points at api-preprod.phonepe.com/apis/pg-sandbox, so checkout returns a
 * real mercury-uat.phonepe.com URL and the payment genuinely has to be made.
 *
 * The route through it is the one a human already uses in dev: the page renders
 * a QR, and in UAT that QR does not encode a `upi://` intent a browser could
 * never follow — it encodes a plain, unauthenticated simulator URL:
 *
 *   https://merchant-simulator.phonepe.com/checkout/ui/v2/payment/status
 *     ?transactToken=…&amount=…
 *
 * which offers Success / Failure / Submitted. So the suite scans the QR the way
 * a phone would, then clicks the button a tester would. No card, no test VPA,
 * no credentials of any kind — nothing to configure and nothing to leak.
 *
 * Everything below was read off the live UAT pages, not guessed.
 */

/** What the simulator should answer. Failure exercises the failed-payment path. */
export type SandboxOutcome = 'Success' | 'Failure';

/**
 * Pays a checkout by decoding its QR and driving the simulator.
 *
 * Returns once the simulator has been submitted. It does **not** assert the
 * order is paid: settlement is asynchronous — PhonePe calls the webhook — so
 * the caller polls payment-status afterwards. The merchant page itself may well
 * render an error after this; it is a collect flow and stops polling once the
 * tab moves on. That is cosmetic, and asserting on it would be asserting on
 * PhonePe's UI rather than on our order.
 */
export async function payWithSandbox(
  redirectUrl: string,
  outcome: SandboxOutcome = 'Success'
): Promise<void> {
  const browser = await chromium.launch();
  try {
    const page = await browser.newPage();
    await page.goto(redirectUrl, { waitUntil: 'domcontentloaded', timeout: 60_000 });

    // The QR is the only PNG data-uri on the page; everything else is an SVG
    // icon or a CDN logo.
    const qr = page.locator('img[src^="data:image/png"]').first();
    // If the QR has not rendered in 15s the page is not the one we expect.
    await qr.waitFor({ state: 'visible', timeout: 15_000 });

    const src = await qr.getAttribute('src');
    if (!src) throw new Error('PhonePe rendered no QR image to scan');

    const simulatorUrl = decodeQR(src);
    if (!simulatorUrl.startsWith('https://merchant-simulator.phonepe.com/')) {
      throw new Error(
        `expected the UAT QR to encode a merchant-simulator URL, got ${simulatorUrl.slice(0, 80)}.\n` +
          `A upi:// intent here means dev is pointed at production PhonePe, ` +
          `where this suite must never transact.`
      );
    }

    await page.goto(simulatorUrl, { waitUntil: 'domcontentloaded', timeout: 60_000 });
    await page.getByRole('button', { name: outcome, exact: true }).click();
    await page.locator('input[type="submit"]').click();
    await page.waitForTimeout(3_000);
  } finally {
    await browser.close();
  }
}

/** Decodes a base64 PNG data-uri into the string its QR encodes. */
function decodeQR(dataUri: string): string {
  const base64 = dataUri.split(',')[1];
  if (!base64) throw new Error('QR image had no base64 payload');

  const png = PNG.sync.read(Buffer.from(base64, 'base64'));
  const decoded = jsQR(new Uint8ClampedArray(png.data), png.width, png.height);
  if (!decoded) throw new Error('could not decode the QR image PhonePe rendered');

  return decoded.data;
}
