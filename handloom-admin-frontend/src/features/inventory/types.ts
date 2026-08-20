export type TransactionType =
  | 'ADD'
  | 'REMOVE'
  | 'RESERVE'
  | 'RELEASE'
  | 'ADJUST'
  | 'COMMIT'
  | 'RETURN'
  | 'WRITE_OFF';

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
  reference_type?: string;
  reference_id?: string;
  // What inside the order caused the movement — a refund id — when it was not the
  // order itself. Two write-offs on one order are told apart by this and nothing else.
  source_id?: string;
  created_by: string;
  // Resolved server-side; created_by alone is an opaque user id.
  created_by_name?: string;
  created_at: string;
}
