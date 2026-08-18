import { describe, expect, it } from 'vitest';

import { formatCurrency, formatCurrencyExact } from '../currency';

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

// Rounding is fine for a total in a list. It is not fine on the figure someone
// authorises: a refund of 235882 paise must not read as ₹2,359.
describe('formatCurrencyExact', () => {
  it('keeps both paise digits', () => {
    expect(formatCurrencyExact(235882)).toBe('₹2,358.82');
  });

  it('pads a whole-rupee amount rather than dropping the decimals', () => {
    expect(formatCurrencyExact(150000)).toBe('₹1,500.00');
  });

  it('shows an amount under a rupee as itself', () => {
    expect(formatCurrencyExact(50)).toBe('₹0.50');
  });

  it('handles zero', () => {
    expect(formatCurrencyExact(0)).toBe('₹0.00');
  });
});
