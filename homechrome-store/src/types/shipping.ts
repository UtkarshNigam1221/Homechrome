export type TrackingEvent =
  | 'MANIFESTED'
  | 'PICKED_UP'
  | 'IN_TRANSIT'
  | 'OUT_FOR_DELIVERY'
  | 'DELIVERED'
  | 'NDR'
  | 'RTO_INITIATED'
  | 'RTO_DELIVERED'
  | 'REVERSE_PICKED_UP'
  | 'REVERSE_DELIVERED'
  | 'UNKNOWN';

export interface TrackingScan {
  status: TrackingEvent | string;
  timestamp: string;
  location?: string;
  description?: string;
}
