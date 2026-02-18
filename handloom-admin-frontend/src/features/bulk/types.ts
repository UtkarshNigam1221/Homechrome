export type BulkOperationType = 'IMPORT' | 'UPDATE' | 'EXPORT';
export type BulkEntityType = 'PRODUCT' | 'INVENTORY' | 'PRICE' | 'CUSTOMER';
export type BulkOperationStatus = 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED' | 'CANCELLED';

export interface BulkOperation {
  id: string;
  type: BulkOperationType;
  entity_type: BulkEntityType;
  status: BulkOperationStatus;
  total_records: number;
  total_count: number;
  success_count: number;
  failure_count: number;
  error_count: number;
  file_url?: string;
  output_file_url: string;
  error_file_url: string;
  created_by: string;
  created_at: string;
  completed_at?: string;
}
