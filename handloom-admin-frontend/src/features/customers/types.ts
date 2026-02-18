import type { Address } from '@/shared/types/common';

export type CustomerStatus = 'ACTIVE' | 'INACTIVE' | 'SUSPENDED';

export interface Customer {
  id: string;
  email: string;
  phone?: string;
  name: string;
  first_name?: string;
  last_name?: string;
  addresses?: Address[];
  status: CustomerStatus;
  order_count: number;
  total_spent: number;
  last_order_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCustomerRequest {
  email: string;
  phone?: string;
  name: string;
  addresses?: Address[];
}

export interface UpdateCustomerRequest extends Partial<CreateCustomerRequest> {
  status?: CustomerStatus;
}
