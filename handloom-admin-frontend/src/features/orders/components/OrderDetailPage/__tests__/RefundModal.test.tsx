import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ordersApi } from '@/features/orders/api';

import type { Order } from '../../../types';
import { RefundModal } from '../RefundModal';

vi.mock('@/features/orders/api', () => ({
  ordersApi: { createRefund: vi.fn() },
}));

vi.mock('react-hot-toast', () => ({
  default: { success: vi.fn(), error: vi.fn() },
}));

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

// A two-line order: 2 bedsheets at ₹1,999 and 3 sarees at ₹100, no discount,
// no tax, no shipping — so every figure below is the line value itself.
function order(overrides: Partial<Order> = {}): Order {
  return {
    id: 'order_1',
    order_number: 'HL-1',
    customer_id: 'cust_1',
    customer_name: 'Asha Rao',
    customer_email: 'asha@example.com',
    items: [
      {
        id: 'item_a',
        product_id: 'prod_a',
        product_name: 'Bedsheet',
        product_sku: 'SKU-A',
        quantity: 2,
        refunded_quantity: 0,
        unit_price: 199900,
        total_price: 399800,
      },
      {
        id: 'item_b',
        product_id: 'prod_b',
        product_name: 'Saree',
        product_sku: 'SKU-B',
        quantity: 3,
        refunded_quantity: 0,
        unit_price: 10000,
        total_price: 30000,
      },
    ],
    subtotal: 429800,
    discount_amount: 0,
    tax_amount: 0,
    shipping_amount: 0,
    total_amount: 429800,
    currency: 'INR',
    payment_status: 'PAID',
    status: 'CONFIRMED',
    shipping_address: {} as Order['shipping_address'],
    created_at: '2026-08-19T00:00:00Z',
    updated_at: '2026-08-19T00:00:00Z',
    ...overrides,
  };
}

function open(props: Partial<React.ComponentProps<typeof RefundModal>> = {}) {
  const onClose = vi.fn();
  const view = render(
    <RefundModal
      isOpen
      onClose={onClose}
      order={order()}
      priorRefunded={0}
      claims={{}}
      hasPendingRefund={false}
      {...props}
    />,
    { wrapper }
  );
  return { onClose, view };
}

function qtyFor(name: string): HTMLInputElement {
  return screen.getByLabelText(`Refund quantity for ${name}`) as HTMLInputElement;
}

describe('RefundModal', () => {
  beforeEach(() => vi.clearAllMocks());

  it('caps a line at what it has left', () => {
    open();

    fireEvent.change(qtyFor('Bedsheet'), { target: { value: '9' } });

    expect(qtyFor('Bedsheet').value).toBe('2');
  });

  // The units of a refund still in flight are not off the order yet, and
  // offering them again is how the same money goes back twice.
  it('does not offer units an unsettled refund already claims', () => {
    open({ claims: { item_a: 2 }, priorRefunded: 399800, hasPendingRefund: true });

    expect(screen.queryByLabelText('Refund quantity for Bedsheet')).toBeNull();
    expect(qtyFor('Saree')).toBeTruthy();
    expect(screen.getByText(/still pending at the provider/i)).toBeTruthy();
  });

  it('submits the quantity it priced, not the raw input', async () => {
    vi.mocked(ordersApi.createRefund).mockResolvedValue({
      id: 'refund_1',
      amount: 399800,
    } as never);
    open();

    fireEvent.change(qtyFor('Bedsheet'), { target: { value: '9' } });
    fireEvent.change(screen.getByLabelText(/reason/i), { target: { value: 'DAMAGED' } });
    fireEvent.click(screen.getByRole('button', { name: /Refund ₹/ }));

    await waitFor(() => expect(ordersApi.createRefund).toHaveBeenCalled());
    expect(vi.mocked(ordersApi.createRefund).mock.calls[0][1]).toEqual({
      reason: 'DAMAGED',
      items: [{ order_item_id: 'item_a', quantity: 2, restock: false }],
    });
  });

  // The order refetches while the modal is open, so a pending refund settling
  // shrinks what is left underneath a selection already made. The priced figure
  // follows; the request has to follow with it.
  it('submits the shrunken quantity when the remainder drops under an open modal', async () => {
    vi.mocked(ordersApi.createRefund).mockResolvedValue({
      id: 'refund_1',
      amount: 199900,
    } as never);
    const props = {
      onClose: vi.fn(),
      order: order(),
      priorRefunded: 0,
      hasPendingRefund: false,
    };
    const { rerender } = render(<RefundModal isOpen claims={{}} {...props} />, { wrapper });

    fireEvent.change(qtyFor('Bedsheet'), { target: { value: '2' } });
    fireEvent.change(screen.getByLabelText(/reason/i), { target: { value: 'DAMAGED' } });

    // One of the two units settles elsewhere; only one is still refundable.
    rerender(<RefundModal isOpen claims={{ item_a: 1 }} {...props} priorRefunded={199900} />);

    expect(screen.getByRole('button', { name: /Refund ₹1,999.00/ })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: /Refund ₹/ }));

    await waitFor(() => expect(ordersApi.createRefund).toHaveBeenCalled());
    expect(vi.mocked(ordersApi.createRefund).mock.calls[0][1].items).toEqual([
      { order_item_id: 'item_a', quantity: 1, restock: false },
    ]);
  });

  it('defaults every line to written off', async () => {
    vi.mocked(ordersApi.createRefund).mockResolvedValue({ id: 'refund_1', amount: 10000 } as never);
    open();

    fireEvent.change(qtyFor('Saree'), { target: { value: '1' } });
    fireEvent.change(screen.getByLabelText(/reason/i), { target: { value: 'DAMAGED' } });
    fireEvent.click(screen.getByRole('button', { name: /Refund ₹/ }));

    await waitFor(() => expect(ordersApi.createRefund).toHaveBeenCalled());
    expect(vi.mocked(ordersApi.createRefund).mock.calls[0][1].items[0].restock).toBe(false);
  });

  it('will not send a refund with no reason', () => {
    open();

    fireEvent.change(qtyFor('Bedsheet'), { target: { value: '1' } });

    expect(screen.getByRole('button', { name: /Refund ₹/ })).toHaveProperty('disabled', true);
  });

  // The modal stays mounted between openings, so nothing resets on its own.
  it('forgets the previous selection when it reopens', () => {
    const onClose = vi.fn();
    const props = {
      onClose,
      order: order(),
      priorRefunded: 0,
      claims: {},
      hasPendingRefund: false,
    };
    const { rerender } = render(<RefundModal isOpen {...props} />, { wrapper });

    fireEvent.change(qtyFor('Bedsheet'), { target: { value: '2' } });
    rerender(<RefundModal isOpen={false} {...props} />);
    rerender(<RefundModal isOpen {...props} />);

    expect(qtyFor('Bedsheet').value).toBe('0');
  });

  it('says a refund clears the order when it does', () => {
    open();

    fireEvent.change(qtyFor('Bedsheet'), { target: { value: '2' } });
    fireEvent.change(qtyFor('Saree'), { target: { value: '3' } });

    expect(screen.getByText(/clears the order/i)).toBeTruthy();
  });

  // After dispatch RETURNED owns restocking, so the stock choice is not offered.
  it('drops the stock choice once the order has shipped', () => {
    open({ order: order({ status: 'SHIPPED' }) });

    expect(screen.queryByLabelText('Stock handling for Bedsheet')).toBeNull();
    expect(screen.getByText(/already been dispatched/i)).toBeTruthy();
  });
});
