import type { Address } from '@/shared/types/common';

export const EMPTY_ADDRESS: Address = {
  first_name: '',
  last_name: '',
  phone: '',
  address_line1: '',
  address_line2: '',
  city: '',
  state: '',
  postal_code: '',
  country: 'India',
  is_default: false,
};

const REQUIRED: [keyof Address, string][] = [
  ['first_name', 'First name'],
  ['last_name', 'Last name'],
  ['address_line1', 'Address line 1'],
  ['city', 'City'],
  ['state', 'State'],
  ['postal_code', 'PIN code'],
  ['country', 'Country'],
];

/**
 * Returns the first problem with an address, or null when it is fine.
 *
 * domain.Address carries no `validate` tags, so the backend will happily store
 * a blank address — this is the only gate on the admin path.
 */
export function validateAddress(address: Address): string | null {
  for (const [field, label] of REQUIRED) {
    if (!String(address[field] ?? '').trim()) return `${label} is required`;
  }
  if (!/^\d{6}$/.test(address.postal_code.trim())) return 'PIN code must be 6 digits';
  return null;
}
