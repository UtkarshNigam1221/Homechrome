import type { Dimensions } from '@/shared/types/common';

export interface ProductImage {
  url: string;
  alt_text?: string;
  is_primary?: boolean;
  sort_order?: number;
}

export type ProductStatus = 'ACTIVE' | 'INACTIVE' | 'DRAFT';

export interface Product {
  id: string;
  name: string;
  sku: string;
  slug: string;
  description?: string;
  category_id: string;
  artisan_id?: string;
  base_price: number;
  selling_price: number;
  cost_price?: number;
  currency: string;
  dimensions?: Dimensions;
  weight?: number;
  allow_custom_dimensions: boolean;
  pricing_rule_id?: string;
  attributes?: Record<string, unknown>;
  material?: string;
  color?: string;
  weave_type?: string;
  origin?: string;
  craft_type?: string;
  images?: ProductImage[];
  tags?: string[];
  quantity: number;
  reserved_qty: number;
  available_qty: number;
  low_stock_threshold: number;
  sort_order: number;
  status: ProductStatus;
  created_at: string;
  updated_at: string;
}

export interface CreateProductRequest {
  name: string;
  sku: string;
  description?: string;
  category_id: string;
  artisan_id?: string;
  base_price: number;
  selling_price: number;
  cost_price?: number;
  currency?: string;
  dimensions?: Dimensions;
  weight?: number;
  allow_custom_dimensions?: boolean;
  pricing_rule_id?: string;
  attributes?: Record<string, unknown>;
  material?: string;
  color?: string;
  weave_type?: string;
  origin?: string;
  craft_type?: string;
  images?: ProductImage[];
  tags?: string[];
  initial_stock?: number;
  low_stock_threshold?: number;
  status?: ProductStatus;
}

export interface ReorderProductsRequest {
  product_ids: string[];
}
