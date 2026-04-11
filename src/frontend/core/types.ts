export interface Person {
  id: string;
  name: string;
  email: string;
  department: string;
  notes: string;
  status: string;
  created_at: string;
  updated_at: string;
  account_count?: number;
  service_count?: number;
}

export interface Account {
  id: string;
  name: string;
  type: string;
  login_url: string;
  login_email: string;
  login_password?: string;
  totp_secret?: string;
  notes: string;
  created_at: string;
  updated_at: string;
  people_count?: number;
}

export interface Service {
  id: string;
  name: string;
  description: string;
  environment: string;
  owner_id: string;
  owner_name?: string;
  created_at: string;
  updated_at: string;
  credential_count?: number;
  expiring_count?: number;
  expired_count?: number;
}

export interface Credential {
  id: string;
  service_id: string;
  name: string;
  type: string;
  provider: string;
  key_value?: string;
  secret_value?: string;
  expires_at: string | null;
  last_rotated_at: string | null;
  where_used: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface Assignment {
  id: string;
  person_id: string;
  person_name?: string;
  account_id: string;
  account_name?: string;
  assigned_by: string;
  assigned_at: string;
  revoked_at?: string;
  revoked_reason?: string;
}

export interface AuditEntry {
  id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  person_id?: string;
  performed_by: string;
  details: string;
  timestamp: string;
}
