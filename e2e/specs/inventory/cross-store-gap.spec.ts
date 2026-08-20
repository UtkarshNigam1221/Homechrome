import { test } from '@playwright/test';

/**
 * #230 case 36 — the DynamoDB/Postgres saga has no shared transaction, so a
 * process death between the order write and the reserve leaves the two stores
 * disagreeing. The issue asks us to pin that the result is *detectable*, not
 * that it cannot happen.
 *
 * Left as a fixme deliberately, with the reason stated rather than a weaker
 * test substituted:
 *
 * A deployed Lambda cannot be killed mid-invocation from outside. There is no
 * fault-injection seam in the handler, and adding one purely for a test would
 * put a crash path into production code. Reproducing it needs either the
 * monolith under a supervisor that can SIGKILL it at a chosen point, or an
 * injected failure behind a build tag — both of which belong in the Go tier
 * against a local stack, not here against deployed dev.
 *
 * What *is* covered: the detection half. If such a divergence occurs, the
 * reservation is left held against an order that never dispatched, which is
 * exactly what scripts/reconcile-inventory reports and what
 * TestInventoryRepository_FindOrphanReservations pins. The gap is therefore
 * visible even though it is not reproducible from this tier.
 */
test.fixme(
  'a crash between the order write and the reserve is surfaced by reconciliation (#230 case 36)',
  async () => {}
);
