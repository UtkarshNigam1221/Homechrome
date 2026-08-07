import type { Address } from '@/shared/types/common';

import type { Customer } from '../types';

/** "First Last" off an address, or '' when neither part is set. */
export function addressFullName(address?: Address | null): string {
  if (!address) return '';
  return `${address.first_name || ''} ${address.last_name || ''}`.trim();
}

/**
 * Best available human name for a customer.
 *
 * Storefront signups are phone-OTP only, so first/last name are frequently
 * blank while the addresses they entered at checkout do carry a name. Falls
 * back through: own name → first address that has a name → email → phone.
 * Returns '' only when the customer has none of those.
 */
export function customerDisplayName(customer: Customer): string {
  const own = `${customer.first_name || ''} ${customer.last_name || ''}`.trim();
  if (own) return own;

  const fromAddress = customer.addresses?.map(addressFullName).find(Boolean);
  if (fromAddress) return fromAddress;

  return customer.email || customer.phone || '';
}

/** Single uppercase initial for an avatar bubble, or 'C' when unknown. */
export function customerInitial(customer: Customer): string {
  return customerDisplayName(customer).charAt(0).toUpperCase() || 'C';
}
