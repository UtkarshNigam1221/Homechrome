import type { InventoryTransaction } from '@/features/inventory/types';

// A reservation settles on COMMIT, RELEASE or WRITE_OFF; one that does none is the
// drift signature. Settling rows sit on earlier pages, so only the newest page counts.
export function openReservationIDs(rows: InventoryTransaction[], anchored: boolean): Set<string> {
  if (!anchored) return new Set();

  // Netted, not counted by presence: one unit released out of three leaves two held,
  // and treating any settlement as the whole story hides the partial drift.
  const outstanding = new Map<string, number>();

  for (const row of rows) {
    if (row.reference_type !== 'ORDER' || !row.reference_id) continue;

    const held = outstanding.get(row.reference_id) ?? 0;
    if (row.type === 'RESERVE') {
      outstanding.set(row.reference_id, held + row.quantity);
    } else if (row.type === 'COMMIT' || row.type === 'RELEASE' || row.type === 'WRITE_OFF') {
      outstanding.set(row.reference_id, held - row.quantity);
    }
  }

  return new Set([...outstanding].filter(([, held]) => held > 0).map(([id]) => id));
}
