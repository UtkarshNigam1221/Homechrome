import { OrderStatus } from '@/types';

const priceFormatter = new Intl.NumberFormat('en-IN', { minimumFractionDigits: 0 });

export function formatPrice(paise: number): string {
  return `₹${priceFormatter.format(paise / 100)}`;
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
