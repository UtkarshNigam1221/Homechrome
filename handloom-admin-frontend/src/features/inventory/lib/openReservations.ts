import type { InventoryTransaction } from '@/features/inventory/types';

// A RESERVE with no COMMIT or RELEASE is the drift signature. Settling rows sit on
// earlier pages, so only the newest page (`anchored`) can conclude anything.
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
