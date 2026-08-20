/**
 * What an order's refunds already account for, and what that leaves refundable.
 *
 * Quantity bookkeeping only. The money — proration, rounding, which refund earns
 * the shipping — is derived server-side and fetched, so there is one implementation
 * of it rather than two that have to be kept agreeing.
 */

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

/**
 * Mirrors claimedByLive in internal/service/refund_service.go: a failed refund
 * returned nothing, so its units are free again.
 *
 * settledAmount is the payment's own figure, which wins when it is larger. A
 * settlement that half-completed leaves the rows ahead of the money or behind it,
 * and the refundable remainder has to be the pessimistic reading of both.
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
