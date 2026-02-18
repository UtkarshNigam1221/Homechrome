export type CategoryStatus = 'ACTIVE' | 'INACTIVE';
export type AttributeType =
  | 'SELECT'
  | 'MULTI_SELECT'
  | 'TEXT'
  | 'NUMBER'
  | 'BOOLEAN'
  | 'DIMENSION'
  | 'DIMENSION_RANGE';

export interface AttributeOption {
  value: string;
  label: string;
  surcharge?: number;
}

export interface CategoryAttribute {
  name: string;
  label: string;
  type: AttributeType;
  required: boolean;
  searchable: boolean; // When true, indexed for filtering and shown in filter UI
  display_order: number;
  options?: AttributeOption[];
}

export interface Category {
  id: string;
  name: string;
  slug: string;
  description?: string;
  image_url?: string;
  own_attributes?: CategoryAttribute[];
  status: CategoryStatus;
  product_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateCategoryRequest {
  name: string;
  description?: string;
  image_url?: string;
  own_attributes?: CategoryAttribute[];
  status?: CategoryStatus;
}
