import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import { type ReactNode, useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { CouponFormModal } from '../CouponFormModal';

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

function audienceSelect() {
  return screen.getByLabelText(/who can use this coupon/i) as HTMLSelectElement;
}

describe('CouponFormModal — audience targeting', () => {
  it('does not show the phone field for the default audience', () => {
    render(<CouponFormModal isOpen onClose={() => {}} />, { wrapper: Wrapper });
    expect(screen.queryByLabelText(/customer mobile number/i)).not.toBeInTheDocument();
  });

  it('reveals the phone field once SPECIFIC_CUSTOMER is selected', () => {
    render(<CouponFormModal isOpen onClose={() => {}} />, { wrapper: Wrapper });

    fireEvent.change(audienceSelect(), { target: { value: 'SPECIFIC_CUSTOMER' } });

    expect(screen.getByLabelText(/customer mobile number/i)).toBeInTheDocument();
  });

  it('leaves none of the three targeting options disabled', () => {
    render(<CouponFormModal isOpen onClose={() => {}} />, { wrapper: Wrapper });

    const options = Array.from(audienceSelect().options);
    for (const value of ['FIRST_ORDER', 'RETURNING', 'SPECIFIC_CUSTOMER']) {
      expect(options.find((o) => o.value === value)?.disabled).toBe(false);
    }
  });

  // Regression guard: the "(Phase 3)" suffix used to sit on every locked option label.
  it('mentions no phase in any audience option', () => {
    render(<CouponFormModal isOpen onClose={() => {}} />, { wrapper: Wrapper });
    expect(audienceSelect().textContent).not.toMatch(/Phase \d/);
  });
});
