import { OrderStatus } from '@/types';

// Two formatters, one entry point. maximumFractionDigits defaults to 3, so a lone
// minimumFractionDigits: 0 trims trailing zeros and renders 29990 paise as "₹299.9".
// Percentage coupons make non-round paise routine, so a rupee figure gets no decimals
// and anything with paise gets exactly two.
const wholeRupeeFormatter = new Intl.NumberFormat('en-IN', {
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});
const withPaiseFormatter = new Intl.NumberFormat('en-IN', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

export function formatPrice(paise: number): string {
  const formatter = paise % 100 === 0 ? wholeRupeeFormatter : withPaiseFormatter;
  return `₹${formatter.format(paise / 100)}`;
}

export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

export function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function calculateDiscountPercent(mrp: number, sellingPrice: number): number {
  if (mrp <= sellingPrice) return 0;
  return Math.round(((mrp - sellingPrice) / mrp) * 100);
}

// Preview only — checkout/initiate re-prices authoritatively. Both operands
// already came from the server, so this only composes them for display.
export function previewTotal(subtotal: number, discount: number): number {
  return subtotal - discount;
}

export const statusBadgeColor: Record<OrderStatus, string> = {
  PENDING: 'yellow',
  CONFIRMED: 'blue',
  PROCESSING: 'blue',
  SHIPPED: 'brand',
  DELIVERED: 'teal',
  CANCELLED: 'red',
  RETURNED: 'orange',
  REFUNDED: 'gray',
};
