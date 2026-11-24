export interface User {
  id: number;
  username: string;
  full_name: string;
  role: string;
  store_code: string;
  theme_pref: string;
  created_at: string;
}

export interface Ticket {
  id: number;
  ticket_no: string;
  category: string;
  title: string;
  description: string;
  urgency: string;
  status: string;
  device_name?: string;
  reported_by_id: number;
  reported_by_name: string;
  assigned_to_id?: number;
  escalated_to?: string;
  escalation_notes?: string;
  created_at: string;
  updated_at: string;
}

export interface ChecklistItem {
  id: number;
  shift_type: string;
  check_date: string;
  item_name: string;
  category: string;
  is_checked: number;
  notes?: string;
  checked_by_id?: number;
}

export interface Asset {
  id: number;
  asset_code: string;
  name: string;
  category: string;
  serial_number?: string;
  location: string;
  status: string;
  specs?: string;
  warranty_until?: string;
  last_maintenance?: string;
}

export interface Software {
  id: number;
  name: string;
  version: string;
  category: string;
  status: string;
  description: string;
}

export interface AntivirusStatus {
  id: number;
  hostname: string;
  ip_address: string;
  antivirus_name: string;
  service_active: number;
  defs_updated_at: string;
  last_scan_at: string;
  threats_found: number;
  status: string;
}

export interface NetworkDevice {
  id: number;
  name: string;
  ip_address: string;
  device_type: string;
  vlan?: string;
  mac_address?: string;
  status: string;
  last_ping_ms: number;
  last_checked: string;
}

export interface ServerHealth {
  hostname: string;
  os: string;
  cpu_usage_pct: number;
  ram_usage_pct: number;
  ram_total_gb: number;
  ram_used_gb: number;
  uptime_days: number;
  disks: {
    drive: string;
    total_gb: number;
    free_gb: number;
    free_pct: number;
    status_flag: string;
  }[];
  last_backup_at: string;
  backup_status: string;
}

export interface WindowsService {
  id: number;
  service_name: string;
  display_name: string;
  status: string;
  is_whitelisted: number;
}

export interface ActivityLog {
  id: number;
  user_id?: number;
  username: string;
  activity_type: string;
  description: string;
  ip_address?: string;
  created_at: string;
}

export interface KnowledgeBase {
  id: number;
  title: string;
  category: string;
  summary: string;
  content: string;
  created_at: string;
}

export interface VendorContact {
  id: number;
  vendor_name: string;
  service_type: string;
  contact_person?: string;
  phone: string;
  email?: string;
  sla_response?: string;
  contract_info?: string;
}
