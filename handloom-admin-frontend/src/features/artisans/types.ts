export type ArtisanStatus = 'ACTIVE' | 'INACTIVE' | 'SUSPENDED';

export interface BankDetails {
  account_name: string;
  account_number: string;
  bank_name: string;
  ifsc_code: string;
  upi_id?: string;
}

export interface Location {
  city: string;
  state: string;
  country: string;
}

export interface Artisan {
  id: string;
  name: string;
  email?: string;
  phone: string;
  craft_type?: string;
  skills?: string[];
  location: Location;
  bio?: string;
  profile_image?: string;
  bank_details?: BankDetails;
  status: ArtisanStatus;
  product_count: number;
  total_earnings: number;
  created_at: string;
  updated_at: string;
}

export interface CreateArtisanRequest {
  name: string;
  email?: string;
  phone: string;
  craft_type?: string;
  skills?: string[];
  location: Location;
  bio?: string;
  bank_details?: BankDetails;
}

export interface UpdateArtisanRequest extends Partial<CreateArtisanRequest> {
  status?: ArtisanStatus;
}
