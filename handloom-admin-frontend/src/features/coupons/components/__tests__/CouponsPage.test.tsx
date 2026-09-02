import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { type ReactNode, useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { couponsApi } from '@/features/coupons/api';

import type { Coupon } from '../../types';
import { CouponsPage } from '../CouponsPage';

vi.mock('@/features/coupons/api', () => ({
  couponsApi: {
    list: vi.fn(),
    get: vi.fn(),
    getByCode: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}));

function Wrapper({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } })
  );
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

// A SPECIFIC_CUSTOMER coupon: the exact shape the list never returns, per
// SetKeys' GSI1 partitioning, which is why this page needs a code lookup at all.
function targetedCoupon(): Coupon {
  return {
    id: 'coupon_9',
    code: 'APOLOGY50',
    name: 'Apology',
    type: 'FIXED',
    value: 50000,
    min_order_value: 0,
    usage_limit: 0,
    usage_per_user: 0,
    usage_count: 0,
    audience: 'SPECIFIC_CUSTOMER',
    customer_id: 'cust_42',
    combines_with_offers: false,
    valid_from: '2026-01-01T00:00:00.000Z',
    status: 'ACTIVE',
    created_at: '2026-01-01T00:00:00.000Z',
    updated_at: '2026-01-01T00:00:00.000Z',
  };
}

function stubEmptyList() {
  vi.mocked(couponsApi.list).mockResolvedValue({
    items: [],
    pagination: { limit: 10, has_more: false },
  });
}

describe('CouponsPage — find by code', () => {
  it('opens the edit modal with the coupon getByCode returns', async () => {
    stubEmptyList();
    vi.mocked(couponsApi.getByCode).mockResolvedValue(targetedCoupon());

    render(<CouponsPage />, { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/find a coupon by code/i), {
      target: { value: 'APOLOGY50' },
    });
    fireEvent.click(screen.getByRole('button', { name: /^find$/i }));

    await waitFor(() => expect(screen.getByText('Edit Coupon')).toBeInTheDocument());
    expect(couponsApi.getByCode).toHaveBeenCalledWith('APOLOGY50');
  });

  // A targeted coupon vanishing from the list is the defect this affordance fixes;
  // a lookup miss must read as a message, not send the operator to an error page.
  it('shows an inline message on a miss, not an error page', async () => {
    stubEmptyList();
    vi.mocked(couponsApi.getByCode).mockRejectedValue(new Error('Coupon not found'));

    render(<CouponsPage />, { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/find a coupon by code/i), {
      target: { value: 'NOPE' },
    });
    fireEvent.click(screen.getByRole('button', { name: /^find$/i }));

    await waitFor(() => expect(screen.getByText('Coupon not found')).toBeInTheDocument());
    expect(screen.queryByText('Edit Coupon')).not.toBeInTheDocument();
  });
});
