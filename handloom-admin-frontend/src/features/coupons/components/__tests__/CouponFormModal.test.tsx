import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import { type ReactNode, useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import type { Coupon } from '../../types';
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

  // Regression guard: a maxLength on this input once truncated a pasted country-coded
  // number into a different, well-formed one that resolved to the wrong customer.
  it('keeps a full country-coded paste in the phone field, uncapped', () => {
    render(<CouponFormModal isOpen onClose={() => {}} />, { wrapper: Wrapper });

    fireEvent.change(audienceSelect(), { target: { value: 'SPECIFIC_CUSTOMER' } });
    const phoneInput = screen.getByLabelText(/customer mobile number/i) as HTMLInputElement;

    // jsdom doesn't enforce maxlength on a programmatic value set, so the attribute
    // itself — not just the resulting value — is what has to stay absent.
    expect(phoneInput).not.toHaveAttribute('maxlength');
    fireEvent.change(phoneInput, { target: { value: '919876543210' } });
    expect(phoneInput.value).toBe('919876543210');
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

  // The backend has no path to change audience post-creation; the design requires the
  // field to stay locked on edit even though the three options are now enabled.
  it('locks the audience select on edit', () => {
    const coupon: Coupon = {
      id: 'coupon_1',
      code: 'FEST20',
      name: 'Festive 20',
      type: 'PERCENTAGE',
      value: 2000,
      min_order_value: 0,
      usage_limit: 0,
      usage_per_user: 0,
      usage_count: 0,
      audience: 'ALL',
      combines_with_offers: false,
      valid_from: '2026-01-01T00:00:00.000Z',
      status: 'ACTIVE',
      created_at: '2026-01-01T00:00:00.000Z',
      updated_at: '2026-01-01T00:00:00.000Z',
    };

    render(<CouponFormModal isOpen onClose={() => {}} coupon={coupon} />, { wrapper: Wrapper });

    expect(audienceSelect()).toBeDisabled();
  });
});
