import type { Address } from '@/shared/types/common';

// Mirrors domain.CustomerStatus.
export type CustomerStatus = 'ACTIVE' | 'INACTIVE' | 'BLOCKED';

export interface Customer {
  id: string;
  email: string;
  phone?: string;
  first_name: string;
  last_name: string;
  addresses?: Address[];
  status: CustomerStatus;
  order_count: number;
  total_spent: number;
  last_order_at?: string;
  created_at: string;
  updated_at: string;
}

// Mirrors domain.CreateCustomerRequest: the backend requires first/last name
// separately and takes a single `address`, not an `addresses` array.
export interface CreateCustomerRequest {
  email: string;
  phone?: string;
  first_name: string;
  last_name: string;
  address?: Address;
}

// Mirrors domain.UpdateCustomerRequest. Note it carries no address field —
// addresses are managed through the dedicated /customers/{id}/addresses routes.
export interface UpdateCustomerRequest {
  first_name?: string;
  last_name?: string;
  phone?: string;
  status?: CustomerStatus;
}
