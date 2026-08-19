import type { InventoryTransaction } from '@/features/inventory/types';

// A reservation settles when the same order dispatches (COMMIT), gives the stock
// back (RELEASE), or writes it off on a refund (WRITE_OFF). One that does none of
// those is stock held against an order that never shipped and never cancelled —
// the drift signature these movements are recorded to catch.
//
// Only the rows passed in are compared, so a caller showing one page of history
// sees open reservations within that page, not across the product's whole life.
export function openReservationIDs(rows: InventoryTransaction[]): Set<string> {
  // Netted rather than counted by presence: one unit released out of three
  // leaves two still held, and treating any settlement as the whole story hides
  // exactly the partial drift this is meant to catch.
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
