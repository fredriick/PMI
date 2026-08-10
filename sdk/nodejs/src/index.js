export class ProxyMeshPeer {
  constructor(options = {}) {
    this.baseURL = options.baseURL || 'http://localhost:8000/api/peer';
    this.token = options.token || null;
    this.nodeId = options.nodeId || null;
  }

  setAuth(token, nodeId) {
    this.token = token;
    this.nodeId = nodeId;
  }

  clearAuth() {
    this.token = null;
    this.nodeId = null;
  }

  async request(path, options = {}) {
    const headers = {
      'Content-Type': 'application/json',
      ...(this.token && { 'X-Peer-Token': this.token }),
      ...options.headers,
    };

    const res = await fetch(`${this.baseURL}${path}`, {
      ...options,
      headers,
    });

    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || `HTTP ${res.status}`);
    }
    return data;
  }

  async login(nodeId) {
    const data = await this.request('/auth', {
      method: 'POST',
      body: JSON.stringify({ node_id: nodeId }),
    });
    this.setAuth(data.token, data.node_id);
    return data;
  }

  async disconnect() {
    await this.request('/disconnect', { method: 'POST', body: '{}' });
    this.clearAuth();
  }

  async getStatus() {
    return this.request('/status');
  }

  async getBandwidth() {
    return this.request('/bandwidth');
  }

  async getEarnings() {
    return this.request('/earnings');
  }

  async getHealth() {
    const data = await this.request('/health');
    return data.score;
  }

  async setConsent(enabled) {
    return this.request('/consent', {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    });
  }

  async updateNode(updates) {
    return this.request('/node', {
      method: 'PATCH',
      body: JSON.stringify(updates),
    });
  }
}
