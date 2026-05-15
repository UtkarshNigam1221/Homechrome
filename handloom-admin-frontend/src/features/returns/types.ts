// Return types mirror handloom-admin/internal/domain/return_request.go.
// Prices are stored in paise.

export type ReturnStatus =
  | 'REQUESTED'
  | 'PICKED_UP'
  | 'IN_TRANSIT'
  | 'RECEIVED'
  | 'REFUNDED'
  | 'CANCELLED';

export const RETURN_STATUSES: ReturnStatus[] = [
  'REQUESTED',
  'PICKED_UP',
  'IN_TRANSIT',
  'RECEIVED',
  'REFUNDED',
  'CANCELLED',
];

export interface ReturnItem {
  product_id: string;
  sku: string;
  quantity: number;
  unit_paise: number;
}

export interface ReturnRequest {
  id: string;
  order_id: string;
  shipment_id: string;
  reverse_awb: string;
  reverse_shipment_id: string;
  reason: string;
  items: ReturnItem[];
  status: ReturnStatus;
  refund_amount_paise: number;
  refunded_at?: string;
  created_by: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreateReturnRequest {
  items: ReturnItem[];
  reason: string;
}
