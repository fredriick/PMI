const API_BASE = __DEV__ ? 'http://localhost:8000/api/peer' : 'https://api.proxymesh.io/api/peer';

export interface LoginRequest {
  node_id: string;
}

export interface LoginResponse {
  token: string;
  node_id: string;
}

export interface NodeStatus {
  node: {
    id: string;
    online?: boolean;
    battery?: number;
    cpu_usage?: number;
    country?: string;
    city?: string;
    ip?: string;
    os?: string;
    node_type?: string;
    last_seen?: string;
    reputation?: number;
    isp?: string;
  };
  load?: number;
}

export interface BandwidthData {
  current: {
    bytes_sent: number;
    bytes_received: number;
    duration_seconds: number;
  };
  history: Record<string, { bytes_sent: number; bytes_received: number }>;
}

export interface PayoutData {
  payout: {
    amount: number;
    period: string;
    tier: string;
    gb_sent: number;
    gb_received: number;
  };
  rates: Record<string, number>;
  tiers: Array<{ name: string; rate_per_gb_sent: number; rate_per_gb_recv: number }>;
  payout_history?: Array<{ period: string; amount: number; tier: string }>;
}

export interface HealthScore {
  overall_score: number;
  latency_score?: number;
  bandwidth_score?: number;
  reliability_score?: number;
  reputation_score?: number;
}

export interface TelemetryEvent {
  score?: { overall_score: number };
  bandwidth?: BandwidthData;
  earnings?: PayoutData;
  node?: NodeStatus['node'];
  load?: number;
}

class ApiService {
  private token: string | null = null;
  private nodeId: string | null = null;

  setAuth(token: string, nodeId: string) {
    this.token = token;
    this.nodeId = nodeId;
  }

  clearAuth() {
    this.token = null;
    this.nodeId = null;
  }

  async login(nodeId: string): Promise<LoginResponse> {
    const response = await fetch(`${API_BASE}/auth`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node_id: nodeId }),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Login failed');
    return data;
  }

  async disconnect(): Promise<void> {
    if (!this.token) return;
    await fetch(`${API_BASE}/disconnect`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Peer-Token': this.token },
      body: JSON.stringify({}),
    });
  }

  async getStatus(): Promise<NodeStatus> {
    if (!this.token) throw new Error('Not authenticated');
    const response = await fetch(`${API_BASE}/status`, {
      headers: { 'X-Peer-Token': this.token },
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Failed to load status');
    return data;
  }

  async getBandwidth(): Promise<BandwidthData> {
    if (!this.token) throw new Error('Not authenticated');
    const response = await fetch(`${API_BASE}/bandwidth`, {
      headers: { 'X-Peer-Token': this.token },
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Failed to load bandwidth');
    return data;
  }

  async getEarnings(): Promise<PayoutData> {
    if (!this.token) throw new Error('Not authenticated');
    const response = await fetch(`${API_BASE}/earnings`, {
      headers: { 'X-Peer-Token': this.token },
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Failed to load earnings');
    return data;
  }

  async getHealth(): Promise<HealthScore> {
    if (!this.token) throw new Error('Not authenticated');
    const response = await fetch(`${API_BASE}/health`, {
      headers: { 'X-Peer-Token': this.token },
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Failed to load health');
    return data.score;
  }

  async setConsent(enabled: boolean): Promise<void> {
    if (!this.token) throw new Error('Not authenticated');
    const response = await fetch(`${API_BASE}/consent`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Peer-Token': this.token },
      body: JSON.stringify({ enabled }),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Failed to update consent');
  }
}

export const api = new ApiService();
