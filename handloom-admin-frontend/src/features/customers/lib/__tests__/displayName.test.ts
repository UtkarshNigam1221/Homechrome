import { describe, expect, it } from 'vitest';

import type { Address } from '@/shared/types/common';

import type { Customer } from '../../types';
import { addressFullName, customerDisplayName, customerInitial } from '../displayName';

function address(overrides: Partial<Address> = {}): Address {
  return {
    first_name: '',
    last_name: '',
    address_line1: '1 Test Road',
    city: 'Mumbai',
    state: 'Maharashtra',
    postal_code: '400001',
    country: 'India',
    ...overrides,
  };
}

function customer(overrides: Partial<Customer> = {}): Customer {
  return {
    id: 'cust-1',
    email: '',
    first_name: '',
    last_name: '',
    status: 'ACTIVE',
    order_count: 0,
    total_spent: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('addressFullName', () => {
  it('joins both parts', () => {
    expect(addressFullName(address({ first_name: 'Asha', last_name: 'Rao' }))).toBe('Asha Rao');
  });

  it('trims when only one part is set', () => {
    expect(addressFullName(address({ first_name: 'Asha' }))).toBe('Asha');
    expect(addressFullName(address({ last_name: 'Rao' }))).toBe('Rao');
  });

  it('returns empty for a nameless or missing address', () => {
    expect(addressFullName(address())).toBe('');
    expect(addressFullName(undefined)).toBe('');
  });
});

describe('customerDisplayName', () => {
  it('prefers the customer’s own name', () => {
    const c = customer({
      first_name: 'Asha',
      last_name: 'Rao',
      addresses: [address({ first_name: 'Someone', last_name: 'Else' })],
    });
    expect(customerDisplayName(c)).toBe('Asha Rao');
  });

  it('falls back to the first address that has a name', () => {
    const c = customer({
      addresses: [address(), address({ first_name: 'Asha', last_name: 'Rao' })],
    });
    expect(customerDisplayName(c)).toBe('Asha Rao');
  });

  it('falls back to email, then phone', () => {
    expect(customerDisplayName(customer({ email: 'a@b.com', phone: '9999999999' }))).toBe(
      'a@b.com'
    );
    expect(customerDisplayName(customer({ phone: '9999999999' }))).toBe('9999999999');
  });

  it('returns empty when nothing is known', () => {
    expect(customerDisplayName(customer())).toBe('');
  });
});

describe('customerInitial', () => {
  it('uses the resolved display name', () => {
    expect(customerInitial(customer({ addresses: [address({ first_name: 'asha' })] }))).toBe('A');
  });

  it('falls back to C when there is no name at all', () => {
    expect(customerInitial(customer())).toBe('C');
  });
});
