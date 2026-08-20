// What an order's refunds account for, and what that leaves refundable. Quantity
// bookkeeping only: the money is derived server-side, so it exists in one place.

export interface RefundableLine {
  id: string;
  quantity: number;
  refunded_quantity: number;
}

/** Units of each line already spoken for by refunds that have not failed. */
export type RefundClaims = Record<string, number>;

/** The subset of a refund this module needs to count what is already claimed. */
export interface ClaimingRefund {
  status: string;
  amount: number;
  items: { order_item_id: string; quantity: number }[];
}

// Mirrors claimedByLive: a failed refund returned nothing, so its units are free.
// settledAmount is the payment's figure, which wins when a half settlement runs ahead.
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

// What a line has left to refund. refunded_quantity counts settled refunds only, so
// claims carries the in-flight units and the larger of the two wins.
export function unrefundedQuantity(line: RefundableLine, claims?: RefundClaims): number {
  const settled = line.refunded_quantity ?? 0;
  const claimed = claims?.[line.id] ?? 0;
  return Math.max(0, line.quantity - Math.max(settled, claimed));
}
