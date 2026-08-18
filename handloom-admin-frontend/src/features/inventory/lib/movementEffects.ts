import type { InventoryTransaction, TransactionType } from '@/features/inventory/types';

export interface MovementEffect {
  label: string;
  variant: 'success' | 'warning' | 'danger' | 'info' | 'gray' | 'primary';
  // Signed change to each counter, omitted where the movement leaves it alone.
  onHand?: number;
  reserved?: number;
}

// A row records a before/after for one counter, but a dispatch moves both — which
// is why available_qty holds still. Deriving the second shows the whole movement.
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
    case 'WRITE_OFF':
      // Same arithmetic as a dispatch — both counters fall, available holds —
      // but the goods went nowhere, so the ledger must not say "shipped".
      return { label: 'Written off', variant: 'danger', onHand: -q, reserved: -q };
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

// Reconstructs the counter a row does not record by undoing each movement back
// from current levels. Only valid while `anchored` — the first page — holds.
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
