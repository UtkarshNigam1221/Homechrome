export type ScopeType = 'GLOBAL' | 'CATEGORY' | 'SUBCATEGORY' | 'PRODUCT' | 'MATERIAL';
export type PricingType = 'AREA_BASED' | 'LENGTH_BASED' | 'FIXED' | 'TIERED' | 'FORMULA';
export type PricingUnit =
  | 'SQ_INCH'
  | 'SQ_FOOT'
  | 'SQ_CM'
  | 'SQ_METER'
  | 'INCH'
  | 'CM'
  | 'FOOT'
  | 'METER';

export interface PricingRule {
  id: string;
  name: string;
  description?: string;
  scope_type: ScopeType;
  scope_id?: string;
  category_id?: string;
  pricing_type: PricingType;
  base_price: number;
  price_per_unit?: number;
  unit?: PricingUnit;
  material_multipliers?: Record<string, number>;
  attribute_surcharges?: {
    attribute_name: string;
    attribute_value: string;
    surcharge_type: 'FIXED' | 'PERCENTAGE';
    surcharge_value: number;
  }[];
  min_area?: number;
  max_area?: number;
  min_order_value?: number;
  priority: number;
  is_active: boolean;
  valid_from?: string;
  valid_until?: string;
  created_at: string;
  updated_at: string;
}

export interface CreatePricingRuleRequest {
  name: string;
  description?: string;
  scope_type: ScopeType;
  category_id?: string;
  pricing_type: PricingType;
  base_price: number;
  price_per_unit?: number;
  unit?: string;
  min_area?: number;
  max_area?: number;
  priority: number;
  is_active: boolean;
}

export type UpdatePricingRuleRequest = Partial<CreatePricingRuleRequest>;

export interface CalculatePriceRequest {
  category_id: string;
  product_id?: string;
  dimensions: {
    length: number;
    width: number;
    height?: number;
    unit: string;
  };
  attributes?: Record<string, string>;
  quantity: number;
}

export interface PriceBreakdown {
  area: number;
  area_unit: string;
  base_cost: number;
  material_cost: number;
  surcharges_total: number;
  subtotal_per_unit: number;
  quantity: number;
  total: number;
}

export interface CalculatePriceResponse {
  price_breakdown: PriceBreakdown;
  formatted_price: {
    subtotal: string;
    total: string;
    currency: string;
  };
  pricing_rule_id: string;
}
