import type { InventoryTransaction } from '@/features/inventory/types';

// A reservation settles when the same order dispatches (COMMIT) or gives the
// stock back (RELEASE). One that does neither is stock held against an order
// that never shipped and never cancelled — the drift signature these movements
// are recorded to catch.
//
// Rows arrive newest first, so a settling row is always on an earlier page than
// the RESERVE it settles. Only the newest page holds every row newer than its
// own oldest, so it is the only page where absence of settlement means anything;
// anywhere else it just means the COMMIT is on a page we are not looking at.
// `anchored` says this is that page. The reconciliation endpoint answers the
// question properly for the whole product.
export function openReservationIDs(rows: InventoryTransaction[], anchored: boolean): Set<string> {
  if (!anchored) return new Set();

  const settled = new Set<string>();
  const reserved = new Set<string>();

  for (const row of rows) {
    if (row.reference_type !== 'ORDER' || !row.reference_id) continue;
    if (row.type === 'RESERVE') reserved.add(row.reference_id);
    if (row.type === 'COMMIT' || row.type === 'RELEASE') settled.add(row.reference_id);
  }

  return new Set([...reserved].filter((id) => !settled.has(id)));
}
