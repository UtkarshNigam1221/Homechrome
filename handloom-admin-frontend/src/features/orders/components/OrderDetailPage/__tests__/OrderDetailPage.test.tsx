import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { ordersApi } from '@/features/orders/api';
import { formatCurrency } from '@/shared/utils/currency';

import type { Order } from '../../../types';
import { OrderDetailPage } from '../OrderDetailPage';

// Only what the page calls: OrderRefunds reads the same module, but the refunds
// list passed in from here is already empty, so it never has cause to call in.
vi.mock('@/features/orders/api', () => ({
  ordersApi: { get: vi.fn(), listRefunds: vi.fn() },
}));

vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>();
  return { ...actual, useNavigate: () => vi.fn(), useParams: () => ({ id: 'order_1' }) };
});

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

// The row div wrapping a label, for scoping an assertion or checking DOM order
// against another row — every summary line is a label span plus a value span.
function rowFor(label: string): HTMLElement {
  const row = screen.getByText(label).closest('div');
  if (!row) throw new Error(`No row div wraps "${label}"`);
  return row;
}

// Tax-inclusive: subtotal ₹1,000, discount ₹100, shipping ₹50, and a total of
// ₹950 — subtotal − discount + shipping, with no tax term. GST of ₹43 is
// contained inside that total, not a further amount on top of it.
function order(): Order {
  return {
    id: 'order_1',
    order_number: 'HL-1',
    customer_id: 'cust_1',
    customer_name: 'Asha Rao',
    customer_email: 'asha@example.com',
    items: [],
    subtotal: 100000,
    discount_amount: 10000,
    tax_amount: 4300,
    shipping_amount: 5000,
    total_amount: 95000,
    currency: 'INR',
    payment_status: 'PAID',
    status: 'CONFIRMED',
    shipping_address: {} as Order['shipping_address'],
    coupon_code: 'SAVE10',
    created_at: '2026-08-19T00:00:00Z',
    updated_at: '2026-08-19T00:00:00Z',
  };
}

describe('OrderDetailPage payment summary', () => {
  // Every existing RefundModal fixture prices tax at 0, which is exactly how an
  // additive tax row shipped on two screens unnoticed — a fixture that cannot
  // exhibit the defect can't catch it. This one prices both a discount and a
  // tax so the reconciliation, and the GST line's framing, both have something
  // to assert against.
  it('reconciles subtotal, discount and shipping to the total, with GST shown only as information', async () => {
    vi.mocked(ordersApi.get).mockResolvedValue(order());
    vi.mocked(ordersApi.listRefunds).mockResolvedValue([]);

    render(<OrderDetailPage />, { wrapper });

    await waitFor(() => expect(screen.getByText('Payment Summary')).toBeTruthy());

    expect(screen.getByText(formatCurrency(100000))).toBeTruthy(); // subtotal
    expect(screen.getByText(`-${formatCurrency(10000)}`)).toBeTruthy(); // discount
    expect(screen.getByText(formatCurrency(5000))).toBeTruthy(); // shipping

    // The additive stack reconciles without tax as a term: 1,000 − 100 + 50 = 950.
    const totalRow = rowFor('Total');
    expect(totalRow).toHaveTextContent(formatCurrency(95000));

    // GST is shown, but as information about the total rather than a further
    // addend — a distinct label ("Of which GST", not "Tax"), and positioned
    // after the total it describes rather than among the rows summing to it.
    const gstRow = rowFor('Of which GST');
    expect(gstRow).toHaveTextContent(formatCurrency(4300));
    expect(
      totalRow.compareDocumentPosition(gstRow) & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();
  });
});
