export type TransactionType =
  | 'ADD'
  | 'REMOVE'
  | 'RESERVE'
  | 'RELEASE'
  | 'ADJUST'
  | 'COMMIT'
  | 'RETURN'
  | 'WRITE_OFF';

// A reservation with no dispatch and no release: stock held against an order
// that went nowhere. The drift signature the ledger movements exist to catch.
export interface OrphanReservation {
  product_id: string;
  product_name: string;
  sku: string;
  order_id: string;
  quantity: number;
  reserved_at: string;
}

export interface ReconciliationReport {
  reservations: OrphanReservation[];
  order_count: number;
  stranded_units: number;
}

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
  // What caused the movement when that is not the order itself — today, the
  // refund. Two write-offs on one order are otherwise indistinguishable.
  source_id?: string;
  created_by: string;
  // Resolved server-side; created_by alone is an opaque user id.
  created_by_name?: string;
  created_at: string;
}
