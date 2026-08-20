/**
 * One id per run, stamped onto every entity the suite creates so
 * scripts/cleanup.ts can find them again after a crash. Short enough to fit
 * inside a SKU, unique enough not to collide between concurrent runs.
 */
export const RUN_ID: string =
  process.env.E2E_RUN_ID ??
  `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;

/** Every entity the suite creates carries this prefix. Cleanup keys on it. */
export const E2E_PREFIX = 'E2E-';

/**
 * Per-process nonce plus a counter. RUN_ID alone is not enough: it is fixed for
 * the whole run (CI pins it to the workflow run id), so every seedCatalog
 * produced the same category name — and the second one 409'd on a duplicate
 * slug. The nonce covers parallel workers, which are separate processes and
 * would otherwise each start their counter at 1.
 */
const NONCE = Math.random().toString(36).slice(2, 6);
let sequence = 0;

export function tag(name: string): string {
  sequence += 1;
  return `${E2E_PREFIX}${RUN_ID}-${NONCE}${sequence}-${name}`;
}

/** True for anything this suite created, in any run. */
export function isE2EEntity(name: string | undefined | null): boolean {
  return typeof name === 'string' && name.startsWith(E2E_PREFIX);
}
