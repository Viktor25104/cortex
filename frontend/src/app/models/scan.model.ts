export type ScanMode = 'CONNECT' | 'SYN' | 'UDP';

export type ScanStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface ScanRequest {
  targets: string[];
  ports: string;
  mode: ScanMode;
  timeout?: number;
  max_concurrent_targets?: number;
}

export interface ScanTask {
  id: string;
  targets: string[];
  ports: string;
  mode: ScanMode;
  status: ScanStatus;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  total_hosts?: number;
  scanned_hosts?: number;
  total_ports?: number;
  open_ports?: number;
}

export interface ScanResult {
  host: string;
  port: number;
  state: string;
  service?: string;
  banner?: string;
  version?: string;
}

export interface ScanDetail extends ScanTask {
  results: ScanResult[];
}

