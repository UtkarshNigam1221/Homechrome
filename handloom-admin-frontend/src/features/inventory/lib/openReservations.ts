import type { InventoryTransaction } from '@/features/inventory/types';

// A reservation settles when the same order dispatches (COMMIT), gives the stock
// back (RELEASE), or writes it off on a refund (WRITE_OFF). One that does none of
// those is stock held against an order that never shipped and never cancelled —
// the drift signature these movements are recorded to catch.
//
// Only the rows passed in are compared, so a caller showing one page of history
// sees open reservations within that page, not across the product's whole life.
export function openReservationIDs(rows: InventoryTransaction[]): Set<string> {
  const settled = new Set<string>();
  const reserved = new Set<string>();

  for (const row of rows) {
    if (row.reference_type !== 'ORDER' || !row.reference_id) continue;
    if (row.type === 'RESERVE') reserved.add(row.reference_id);
    if (row.type === 'COMMIT' || row.type === 'RELEASE' || row.type === 'WRITE_OFF') {
      settled.add(row.reference_id);
    }
  }

  return new Set([...reserved].filter((id) => !settled.has(id)));
}
