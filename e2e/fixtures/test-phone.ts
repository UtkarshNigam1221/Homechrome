/**
 * The storefront customer this worker owns.
 *
 * A customer has exactly one cart, and placePaidOrder clears the cart before
 * adding its items — so two workers sharing a customer clear each other's carts
 * mid-flight and checkout fails with "Cart is empty". Each worker therefore
 * takes its own number.
 *
 * Playwright sets TEST_PARALLEL_INDEX per worker slot, and the backend's
 * STORE_TEST_PHONES is an allowlist, so the two lists line up by index. They
 * must hold the same numbers: a phone the backend does not allowlist gets a
 * random OTP and the suite cannot log in as it.
 */
export function testPhones(): string[] {
  // Singular is the fallback so a one-phone setup keeps working unchanged.
  const raw = process.env.E2E_STORE_PHONES ?? process.env.E2E_STORE_PHONE ?? '';
  return raw
    .split(',')
    .map((p) => p.trim())
    .filter(Boolean);
}

export function testPhone(): string {
  const phones = testPhones();
  if (phones.length === 0) {
    throw new Error(
      'E2E_STORE_PHONES (or E2E_STORE_PHONE) must be set — the suite cannot log ' +
        'a customer in without a number the backend allowlists.'
    );
  }
  // Modulo, not an error, when workers exceed phones: playwright.config caps
  // workers at the phone count, so this only wraps if that cap is bypassed.
  const slot = Number(process.env.TEST_PARALLEL_INDEX ?? 0);
  return phones[slot % phones.length]!;
}
