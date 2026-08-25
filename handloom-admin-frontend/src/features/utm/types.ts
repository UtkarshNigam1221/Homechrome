export type UTMDestType = 'HOME' | 'CATEGORY' | 'PRODUCT';

export interface UTMLink {
  id: string;
  name: string;
  dest_type: UTMDestType;
  dest_slug?: string;
  utm_source: string;
  utm_medium: string;
  utm_campaign: string;
  url: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
}

export interface CreateUTMLinkRequest {
  name: string;
  dest_type: UTMDestType;
  dest_slug?: string;
  utm_source: string;
  utm_medium: string;
  utm_campaign: string;
}
