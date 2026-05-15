// Shipping types mirror the Go backend (handloom-admin/internal/domain).
// Prices are stored and exchanged in **paise** (1 INR = 100 paise).

export type RateSource = 'delhivery_api' | 'manual_override';

export interface ShippingRate {
  zone: string;
  weight_slab_grams: number;
  prepaid_paise: number;
  cod_paise: number;
  rto_paise: number;
  refreshed_at: string;
  source: RateSource;
}

export interface UpdateRateRequest {
  prepaid_paise: number;
  cod_paise: number;
  rto_paise: number;
}

export interface PincodeZone {
  pincode: string;
  zone: string;
  city: string;
  state: string;
  serviceable: boolean;
  cod_available: boolean;
  prepaid_available: boolean;
  refreshed_at: string;
}

export type CODRemittanceStatus = 'RECEIVED' | 'RECONCILED' | 'UNMATCHED';

export const COD_REMITTANCE_STATUSES: CODRemittanceStatus[] = [
  'RECEIVED',
  'RECONCILED',
  'UNMATCHED',
];

export interface CODEntry {
  awb: string;
  order_id: string;
  amount_paise: number;
  matched: boolean;
}

export interface CODRemittance {
  id: string;
  remittance_ref: string;
  amount_paise: number;
  remitted_at: string;
  bank_ref: string;
  status: CODRemittanceStatus;
  entries: CODEntry[];
}

export type ShipmentStatus =
  | 'CREATED'
  | 'PICKED_UP'
  | 'IN_TRANSIT'
  | 'OUT_FOR_DELIVERY'
  | 'DELIVERED'
  | 'RTO'
  | 'MANIFESTED'
  | 'NDR'
  | 'NDR_ESCALATED'
  | 'RETURNING'
  | 'RETURNED';

export type ShipmentPriority = 'NORMAL' | 'PRIORITY';

export interface Shipment {
  id: string;
  order_id: string;
  provider: string;
  provider_order_id?: string;
  provider_shipment_id?: string;
  awb_number?: string;
  courier_name?: string;
  status: ShipmentStatus;
  label_url?: string;
  estimated_delivery?: string;
  weight_grams: number;
  shipped_at?: string;
  delivered_at?: string;
  priority: ShipmentPriority;
  pickup_location: string;
  manifest_id?: string;
  ndr_count: number;
  last_ndr_reason?: string;
  last_ndr_at?: string;
  ndr_escalated: boolean;
  shipping_charge: number;
  actual_weight_grams: number;
  charged_weight_grams: number;
  is_cod: boolean;
  cod_amount: number;
  cod_remitted: boolean;
  cod_remitted_at?: string;
  cod_remittance_ref?: string;
  created_at?: string;
  updated_at?: string;
}

export interface BatchResult {
  manifest_id: string;
  shipment_count: number;
  shipment_marked_ids: string[] | null;
  failed_shipment_ids: string[] | null;
}

export type NDRAction = 'reattempt' | 'mark_contacted' | 'rto';

export interface NDRActionRequest {
  action: NDRAction;
}

export interface CreateShipmentResponse {
  shipment?: Shipment;
  status?: string;
}
