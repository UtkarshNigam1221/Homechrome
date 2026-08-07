import { describe, expect, it } from 'vitest';

import type { Address } from '@/shared/types/common';

import { EMPTY_ADDRESS, validateAddress } from '../addressValidation';

function valid(overrides: Partial<Address> = {}): Address {
  return {
    ...EMPTY_ADDRESS,
    first_name: 'Asha',
    last_name: 'Rao',
    address_line1: '1 Test Road',
    city: 'Mumbai',
    state: 'Maharashtra',
    postal_code: '400001',
    country: 'India',
    ...overrides,
  };
}

describe('validateAddress', () => {
  it('accepts a complete address', () => {
    expect(validateAddress(valid())).toBeNull();
  });

  it('does not require address line 2 or phone', () => {
    expect(validateAddress(valid({ address_line2: '', phone: '' }))).toBeNull();
  });

  it.each([
    ['first_name', 'First name is required'],
    ['last_name', 'Last name is required'],
    ['address_line1', 'Address line 1 is required'],
    ['city', 'City is required'],
    ['state', 'State is required'],
    ['country', 'Country is required'],
  ] as [keyof Address, string][])('rejects a missing %s', (field, message) => {
    expect(validateAddress(valid({ [field]: '' }))).toBe(message);
  });

  it('treats whitespace as missing', () => {
    expect(validateAddress(valid({ city: '   ' }))).toBe('City is required');
  });

  it('requires a 6-digit PIN code', () => {
    expect(validateAddress(valid({ postal_code: '' }))).toBe('PIN code is required');
    expect(validateAddress(valid({ postal_code: '4001' }))).toBe('PIN code must be 6 digits');
    expect(validateAddress(valid({ postal_code: '40000a' }))).toBe('PIN code must be 6 digits');
    expect(validateAddress(valid({ postal_code: ' 400001 ' }))).toBeNull();
  });

  it('rejects the empty template it ships', () => {
    expect(validateAddress({ ...EMPTY_ADDRESS })).toBe('First name is required');
  });
});
