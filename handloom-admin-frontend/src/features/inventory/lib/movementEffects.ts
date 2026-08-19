import type { InventoryTransaction, TransactionType } from '@/features/inventory/types';

export interface MovementEffect {
  label: string;
  variant: 'success' | 'warning' | 'danger' | 'info' | 'gray' | 'primary';
  // Signed change to each counter, omitted where the movement leaves it alone.
  onHand?: number;
  reserved?: number;
}

// A ledger row records a before/after pair for one counter only, but a dispatch
// moves both: quantity and reserved_qty fall by the same amount, which is what
// makes available_qty unchanged. Deriving the second figure here is what lets
// the table show the whole movement instead of half of it.
export function movementEffect(row: InventoryTransaction): MovementEffect {
  const q = row.quantity;

  switch (row.type) {
    case 'ADD':
      return { label: 'Restocked', variant: 'success', onHand: q };
    case 'RETURN':
      return { label: 'Returned', variant: 'success', onHand: q };
    case 'REMOVE':
      return { label: 'Removed', variant: 'danger', onHand: -q };
    case 'ADJUST':
      // The only type whose direction is not implied by the type itself.
      return { label: 'Recounted', variant: 'gray', onHand: row.new_qty - row.previous_qty };
    case 'RESERVE':
      return { label: 'Reserved', variant: 'warning', reserved: q };
    case 'RELEASE':
      return { label: 'Released', variant: 'info', reserved: -q };
    case 'COMMIT':
      return { label: 'Dispatched', variant: 'primary', onHand: -q, reserved: -q };
  }
}

// Which counter the recorded before/after pair belongs to.
export function recordedCounter(type: TransactionType): 'onHand' | 'reserved' {
  return type === 'RESERVE' || type === 'RELEASE' ? 'reserved' : 'onHand';
}

export interface Balance {
  onHand: number;
  reserved: number;
}

// A ledger row carries the before/after for one counter, so the other one has to
// be reconstructed: start from the product's current levels and undo each
// movement walking backwards through the list.
//
// `anchored` says the newest row shown is the product's newest movement, which is
// true on the first page alone. Checking the recorded counter is not a substitute:
// a reserve and its release on a newer page net to zero on that counter, so the
// check passes while the counter this function exists to derive is still wrong.
export function balancesAfter(
  rows: InventoryTransaction[],
  current: Balance,
  anchored: boolean
): (Balance | null)[] {
  if (!anchored) return rows.map(() => null);

  let { onHand, reserved } = current;

  return rows.map((row) => {
    const effect = movementEffect(row);
    const after: Balance = { onHand, reserved };

    onHand -= effect.onHand ?? 0;
    reserved -= effect.reserved ?? 0;

    // Belt and braces: `current` comes from the product row, which can be a
    // render behind the ledger it is being walked against.
    const recorded = recordedCounter(row.type);
    const expected = recorded === 'reserved' ? after.reserved : after.onHand;
    return expected === row.new_qty ? after : null;
  });
}
