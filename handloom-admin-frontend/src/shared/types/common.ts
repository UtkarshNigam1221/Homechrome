// Common types shared across all features

export interface PaginationParams {
  limit?: number;
  cursor?: string;
  sort_by?: string;
  sort_order?: 'ASC' | 'DESC';
}

export interface PaginationResponse {
  limit: number;
  next_cursor?: string;
  has_more: boolean;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
  error?: {
    code: string;
    message: string;
  };
}

export interface ListResponse<T> {
  items: T[];
  pagination: PaginationResponse;
}

// Shared across orders + customers
// Mirrors domain.Address on the backend.
export interface Address {
  id?: string;
  first_name: string;
  last_name: string;
  phone?: string;
  address_line1: string;
  address_line2?: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
  is_default?: boolean;
}

// Shared across products + orders
export interface Dimensions {
  length: number;
  width: number;
  height?: number;
  unit: string;
}

// Asset types (used by ImageUpload + bulk)
export type AssetType = 'IMAGE' | 'VIDEO' | 'DOCUMENT';

export interface UploadURLResponse {
  upload_url: string;
  tmp_key: string;
  tmp_url: string;
  expires_at: string;
}

// Audit types (cross-feature)
export interface AuditLog {
  id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  user_id: string;
  user_email: string;
  changes?: {
    field: string;
    old_value: unknown;
    new_value: unknown;
  }[];
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}
