import { describe, expect, it } from 'vitest';

import { formatCurrency } from '../currency';

describe('formatCurrency', () => {
  it('converts paise to INR with no decimals', () => {
    expect(formatCurrency(150000)).toBe('₹1,500');
  });

  it('handles zero', () => {
    expect(formatCurrency(0)).toBe('₹0');
  });

  it('handles small amounts under 1 rupee', () => {
    expect(formatCurrency(50)).toBe('₹1'); // rounds up from 0.50
  });

  it('handles large amounts', () => {
    expect(formatCurrency(10000000)).toBe('₹1,00,000');
  });
});
