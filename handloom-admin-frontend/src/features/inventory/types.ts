export type TransactionType = 'ADD' | 'REMOVE' | 'RESERVE' | 'RELEASE' | 'ADJUST';

export interface Inventory {
  product_id: string;
  product_name: string;
  sku: string;
  quantity: number;
  reserved_qty: number;
  available_qty: number;
  low_stock_threshold: number;
  is_low_stock: boolean;
  last_updated_at: string;
}

export interface InventoryTransaction {
  id: string;
  product_id: string;
  type: TransactionType;
  quantity: number;
  previous_qty: number;
  new_qty: number;
  reason?: string;
  reference_id?: string;
  created_by: string;
  created_at: string;
}
