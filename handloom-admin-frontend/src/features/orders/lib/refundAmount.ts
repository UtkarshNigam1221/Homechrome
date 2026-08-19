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

  /** The three terms behind the total, so the screen can show why it is that
   * number rather than only what it is. */
  breakdown: {
    lineValue: number;
    discount: number;
    tax: number;
    /** Returned only by the refund that clears the order, and zero until then. */
    shipping: number;
  };

  /** What was actually priced, after clamping. This is what must be submitted —
   * sending the raw input would post a quantity the preview never costed. */
  lines: { orderItemId: string; quantity: number; amount: number }[];

  /** True when this refund clears the last unrefunded unit — the one that earns
   * back the shipping and absorbs the rounding residual. */
  isFinal: boolean;
}

/** Units of each line already spoken for by refunds that have not failed. */
export type RefundClaims = Record<string, number>;

/** The subset of a refund this module needs to count what is already claimed. */
export interface ClaimingRefund {
  status: string;
  amount: number;
  items: { order_item_id: string; quantity: number }[];
}

/**
 * What an order's refunds already account for — settled or still in flight.
 * Mirrors claimedByLive in internal/service/refund_service.go, and the two must
 * agree: a failed refund returned nothing, so its units are free again.
 *
 * settledAmount is the payment's own figure, which wins when it is larger. A
 * settlement that half-completed leaves the rows ahead of the money or behind
 * it, and the refundable remainder has to be the pessimistic reading of both.
 */
export function claimedByLiveRefunds(
  refunds: ClaimingRefund[],
  settledAmount = 0
): { claims: RefundClaims; amount: number } {
  const claims: RefundClaims = {};
  let amount = 0;

  for (const refund of refunds) {
    if (refund.status === 'FAILED') continue;
    amount += refund.amount;
    for (const item of refund.items) {
      claims[item.order_item_id] = (claims[item.order_item_id] ?? 0) + item.quantity;
    }
  }

  return { claims, amount: Math.max(amount, settledAmount) };
}

/**
 * What a line has left to refund.
 *
 * refunded_quantity counts settled refunds only, so a refund still in flight is
 * invisible to it — exactly the window in which the same units could go back
 * twice. claims carries those units, and the larger of the two wins.
 */
export function unrefundedQuantity(line: RefundableLine, claims?: RefundClaims): number {
  const settled = line.refunded_quantity ?? 0;
  const claimed = claims?.[line.id] ?? 0;
  return Math.max(0, line.quantity - Math.max(settled, claimed));
}

// value's share of the whole for a line worth lineSubtotal, rounded half-up.
// Integer paise throughout, matching the Go arithmetic exactly.
function prorate(value: number, lineSubtotal: number, subtotal: number): number {
  if (value === 0 || subtotal === 0) return 0;
  // BigInt because the product leaves a double's exact-integer range before it
  // leaves Go's int64: the server would then round differently, and the admin
  // authorises whatever this returns. The result is a paise figure well inside
  // Number.MAX_SAFE_INTEGER, so converting back is safe.
  const scaled = BigInt(value) * BigInt(lineSubtotal) + BigInt(subtotal) / 2n;
  return Number(scaled / BigInt(subtotal));
}

export function previewRefund(
  order: RefundableOrder,
  selection: RefundSelection,
  priorRefunded: number,
  claims?: RefundClaims
): RefundPreview {
  const lines: { orderItemId: string; quantity: number; amount: number }[] = [];
  const remainingAfter = new Map<string, number>();
  let runningTotal = 0;
  let lineValue = 0;
  let discount = 0;
  let tax = 0;

  for (const item of order.items) {
    const left = unrefundedQuantity(item, claims);

    // Clamped rather than rejected: the stepper already caps the input, and a
    // preview that throws would leave the admin staring at a blank panel.
    const quantity = Math.min(Math.max(selection[item.id] ?? 0, 0), left);
    if (quantity === 0) continue;

    remainingAfter.set(item.id, left - quantity);

    const lineSubtotal = item.unit_price * quantity;
    const lineDiscount = prorate(order.discount_amount, lineSubtotal, order.subtotal);
    const lineTax = prorate(order.tax_amount, lineSubtotal, order.subtotal);
    const amount = lineSubtotal - lineDiscount + lineTax;

    lineValue += lineSubtotal;
    discount += lineDiscount;
    tax += lineTax;
    runningTotal += amount;
    lines.push({ orderItemId: item.id, quantity, amount });
  }

  const isFinal = lines.length > 0 && clearsOrder(order, remainingAfter, claims);
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

  // Whatever the clearing refund absorbs beyond the per-line terms is the
  // shipping plus the rounding residual — shown as shipping, which is what it
  // overwhelmingly is.
  const shipping = isFinal ? total - (lineValue - discount + tax) : 0;

  return { total, lines, isFinal, breakdown: { lineValue, discount, tax, shipping } };
}

// Whether nothing would be left unrefunded once the selected lines go back.
function clearsOrder(
  order: RefundableOrder,
  remainingAfter: Map<string, number>,
  claims?: RefundClaims
): boolean {
  return order.items.every(
    (item) => (remainingAfter.get(item.id) ?? unrefundedQuantity(item, claims)) === 0
  );
}
