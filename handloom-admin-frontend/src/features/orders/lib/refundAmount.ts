/**
 * Mirrors deriveRefundAmount in internal/service/refund_amount.go.
 *
 * The server derives the refund and ignores any amount a client sends, so this
 * is a preview and never an input. It exists because an admin should see what
 * leaves the account before authorising it, not after.
 *
 * Keep in sync with the Go function — it is the source of truth. The tests use
 * the same figures as TestDeriveRefundAmount for exactly that reason.
 */

export interface RefundableLine {
  id: string;
  unit_price: number;
  quantity: number;
  refunded_quantity: number;
}

export interface RefundableOrder {
  items: RefundableLine[];
  subtotal: number;
  discount_amount: number;
  tax_amount: number;
  total_amount: number;
}

/** Quantities to refund, keyed by order item id. */
export type RefundSelection = Record<string, number>;

export interface RefundPreview {
  total: number;
  lines: { orderItemId: string; amount: number }[];

  /** True when this refund clears the last unrefunded unit — the one that earns
   * back the shipping and absorbs the rounding residual. */
  isFinal: boolean;
}

/** What a line has left to refund. */
export function unrefundedQuantity(line: RefundableLine): number {
  return Math.max(0, line.quantity - (line.refunded_quantity ?? 0));
}

// value's share of the whole for a line worth lineSubtotal, rounded half-up.
// Integer paise throughout, matching the Go arithmetic exactly.
function prorate(value: number, lineSubtotal: number, subtotal: number): number {
  if (value === 0 || subtotal === 0) return 0;
  return Math.floor((value * lineSubtotal + Math.floor(subtotal / 2)) / subtotal);
}

export function previewRefund(
  order: RefundableOrder,
  selection: RefundSelection,
  priorRefunded: number
): RefundPreview {
  const lines: { orderItemId: string; amount: number }[] = [];
  const remainingAfter = new Map<string, number>();
  let runningTotal = 0;

  for (const item of order.items) {
    const left = unrefundedQuantity(item);

    // Clamped rather than rejected: the stepper already caps the input, and a
    // preview that throws would leave the admin staring at a blank panel.
    const quantity = Math.min(Math.max(selection[item.id] ?? 0, 0), left);
    if (quantity === 0) continue;

    remainingAfter.set(item.id, left - quantity);

    const lineSubtotal = item.unit_price * quantity;
    const amount =
      lineSubtotal -
      prorate(order.discount_amount, lineSubtotal, order.subtotal) +
      prorate(order.tax_amount, lineSubtotal, order.subtotal);

    runningTotal += amount;
    lines.push({ orderItemId: item.id, amount });
  }

  const isFinal = lines.length > 0 && clearsOrder(order, remainingAfter);
  let total = runningTotal;

  if (isFinal) {
    // Whatever is left of the order is what is left to refund — shipping and
    // rounding residual included. Deriving it this way is what makes the
    // refunds sum to the order exactly.
    total = order.total_amount - priorRefunded;
    const adjust = total - runningTotal;
    if (adjust !== 0) {
      lines[lines.length - 1].amount += adjust;
    }
  }

  return { total, lines, isFinal };
}

// Whether nothing would be left unrefunded once the selected lines go back.
function clearsOrder(order: RefundableOrder, remainingAfter: Map<string, number>): boolean {
  return order.items.every(
    (item) => (remainingAfter.get(item.id) ?? unrefundedQuantity(item)) === 0
  );
}
