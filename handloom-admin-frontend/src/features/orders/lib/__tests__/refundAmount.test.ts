import { describe, expect, it } from 'vitest';

import type { RefundableLine, RefundableOrder } from '../refundAmount';
import { previewRefund, unrefundedQuantity } from '../refundAmount';

function line(id: string, unitPrice: number, quantity: number, refunded = 0): RefundableLine {
  return { id, unit_price: unitPrice, quantity, refunded_quantity: refunded };
}

// Builds an order whose money adds up: subtotal - discount + tax + shipping.
function order(
  discount: number,
  tax: number,
  shipping: number,
  ...items: RefundableLine[]
): RefundableOrder {
  const subtotal = items.reduce((sum, item) => sum + item.unit_price * item.quantity, 0);
  return {
    items,
    subtotal,
    discount_amount: discount,
    tax_amount: tax,
    total_amount: subtotal - discount + tax + shipping,
  };
}

// The figures here are the ones TestDeriveRefundAmount asserts in
// internal/service/refund_amount_test.go. They agree deliberately: this preview
// is only worth showing if it lands on the same number the server will.
describe('previewRefund', () => {
  it('refunds one line of a multi-line order at its own value', () => {
    const got = previewRefund(order(0, 0, 0, line('a', 10000, 2), line('b', 5000, 1)), { a: 1 }, 0);

    expect(got.total).toBe(10000);
    expect(got.isFinal).toBe(false);
    expect(got.lines).toEqual([{ orderItemId: 'a', quantity: 1, amount: 10000 }]);
  });

  it('keeps the shipping while a unit still ships', () => {
    const got = previewRefund(order(0, 0, 5000, line('a', 10000, 2)), { a: 1 }, 0);

    expect(got.total).toBe(10000);
  });

  it('returns the shipping with the refund that clears the order', () => {
    const subject = order(0, 0, 5000, line('a', 10000, 2));

    const got = previewRefund(subject, { a: 2 }, 0);

    expect(got.isFinal).toBe(true);
    expect(got.total).toBe(subject.total_amount);
  });

  it('prorates the discount by the line share of the subtotal', () => {
    // subtotal 30000, discount 3000 → a 10000 line carries 1000 of it.
    const got = previewRefund(
      order(3000, 0, 0, line('a', 10000, 1), line('b', 20000, 1)),
      { a: 1 },
      0
    );

    expect(got.total).toBe(9000);
  });

  it('prorates the tax the same way', () => {
    const got = previewRefund(
      order(0, 3000, 0, line('a', 10000, 1), line('b', 20000, 1)),
      { a: 1 },
      0
    );

    expect(got.total).toBe(11000);
  });

  it('absorbs the rounding residual into the refund that clears the order', () => {
    // subtotal 3, discount 1 → every line's prorated share rounds to zero.
    const subject = order(1, 0, 0, line('a', 1, 1), line('b', 1, 1), line('c', 1, 1));

    const first = previewRefund(subject, { a: 1 }, 0);
    subject.items[0].refunded_quantity = 1;
    const second = previewRefund(subject, { b: 1, c: 1 }, first.total);

    expect(second.isFinal).toBe(true);
    expect(first.total + second.total).toBe(subject.total_amount);
  });

  it('treats a line with nothing selected as not requested', () => {
    const got = previewRefund(order(0, 0, 0, line('a', 10000, 2), line('b', 5000, 1)), { a: 0 }, 0);

    expect(got.total).toBe(0);
    expect(got.lines).toEqual([]);
    expect(got.isFinal).toBe(false);
  });

  it('counts an already-refunded line as settled when deciding what clears the order', () => {
    const subject = order(0, 0, 0, line('a', 10000, 2, 1));

    const got = previewRefund(subject, { a: 1 }, 10000);

    expect(got.isFinal).toBe(true);
    expect(got.total).toBe(subject.total_amount - 10000);
  });

  // The clamped figure is what gets priced, so it is also what has to be sent —
  // submitting the raw input would post a quantity the preview never costed.
  it('reports the clamped quantity it actually priced', () => {
    const got = previewRefund(order(0, 0, 0, line('a', 10000, 2, 1)), { a: 5 }, 10000);

    expect(got.lines).toEqual([{ orderItemId: 'a', quantity: 1, amount: 10000 }]);
  });

  // claims and priorRefunded have to describe the same set of refunds. Passing
  // in-flight units while counting only settled money would price the clearing
  // refund at the whole order.
  it('subtracts units an unsettled refund already claims', () => {
    const subject = order(0, 0, 0, line('a', 10000, 3));

    const got = previewRefund(subject, { a: 3 }, 20000, { a: 2 });

    expect(got.lines).toEqual([{ orderItemId: 'a', quantity: 1, amount: 10000 }]);
    expect(got.isFinal).toBe(true);
    expect(got.total).toBe(10000, 'only the unclaimed unit is left to send back');
  });

  // Money is integer paise on both sides. JS doubles lose the last digits where
  // Go's int64 does not, and the admin authorises what this returns.
  it('stays exact where a double would not', () => {
    const got = previewRefund(order(2500000, 0, 0, line('a', 626862991, 1)), { a: 1 }, 0);

    expect(Number.isSafeInteger(got.total)).toBe(true);
    expect(got.total).toBe(626862991 - 2500000);
  });

  it('ignores a line the order does not have', () => {
    const got = previewRefund(order(0, 0, 0, line('a', 10000, 1)), { ghost: 1 }, 0);

    expect(got.total).toBe(0);
  });
});

describe('unrefundedQuantity', () => {
  it('is what the line has left to refund', () => {
    expect(unrefundedQuantity(line('a', 10000, 3, 1))).toBe(2);
  });

  it('never goes below zero', () => {
    expect(unrefundedQuantity(line('a', 10000, 1, 4))).toBe(0);
  });

  it('counts units a refund still in flight has claimed', () => {
    expect(unrefundedQuantity(line('a', 10000, 3, 1), { a: 2 })).toBe(1);
  });

  it('takes whichever claim is larger, settled or in flight', () => {
    expect(unrefundedQuantity(line('a', 10000, 3, 2), { a: 1 })).toBe(1);
  });
});

// The design asks for the terms, not just the total: a ₹10,000 line refunding
// ₹9,000 gives the admin no way to see the ₹1,000 is a discount share, not an
// error.
describe('previewRefund breakdown', () => {
  it('separates the line value from its prorated discount', () => {
    const got = previewRefund(
      order(3000, 0, 0, line('a', 10000, 1), line('b', 20000, 1)),
      { a: 1 },
      0
    );

    expect(got.breakdown.lineValue).toBe(10000);
    expect(got.breakdown.discount).toBe(1000);
    expect(got.breakdown.tax).toBe(0);
    expect(got.breakdown.shipping).toBe(0);
  });

  it('shows the shipping only on the refund that clears the order', () => {
    const subject = order(0, 0, 5000, line('a', 10000, 2));

    expect(previewRefund(subject, { a: 1 }, 0).breakdown.shipping).toBe(0);
    expect(previewRefund(subject, { a: 2 }, 0).breakdown.shipping).toBe(5000);
  });

  it('adds up to the total it reports', () => {
    const got = previewRefund(
      order(3000, 7000, 5000, line('a', 10000, 2), line('b', 20000, 1)),
      { a: 2, b: 1 },
      0
    );
    const { lineValue, discount, tax, shipping } = got.breakdown;

    expect(lineValue - discount + tax + shipping).toBe(got.total);
  });
});
