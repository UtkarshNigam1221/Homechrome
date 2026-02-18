export type UserRole = 'ADMIN' | 'OPERATOR';
export type UserStatus = 'ACTIVE' | 'INACTIVE' | 'PENDING';

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  phone?: string;
  role: UserRole;
  permissions?: string[];
  status: UserStatus;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface CreateUserRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  phone?: string;
  role: UserRole;
  permissions?: string[];
  status?: UserStatus;
}
