export type ReportType = 'SALES' | 'INVENTORY' | 'ORDERS' | 'CUSTOMERS' | 'PRODUCTS' | 'ARTISANS';
export type ReportFormat = 'CSV' | 'EXCEL' | 'PDF';
export type ReportStatus = 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED';

export interface Report {
  id: string;
  type: ReportType;
  status: ReportStatus;
  format: ReportFormat;
  filters?: Record<string, unknown>;
  file_url?: string;
  created_by: string;
  created_at: string;
  completed_at?: string;
}
