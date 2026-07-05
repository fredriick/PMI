(function () {
  'use strict';

  const API = '/api/peer';
  let token = localStorage.getItem('peer_token');
  let nodeId = localStorage.getItem('peer_node_id');
  let refreshTimer = null;

  // DOM
  const $ = id => document.getElementById(id);
  const loginScreen = $('login-screen');
  const mainApp = $('main-app');
  const loginForm = $('login-form');
  const loginError = $('login-error');
  const nodeIdInput = $('node-id');
  const headerNodeId = $('header-node-id');
  const statusDot = $('status-dot');
  const toast = $('toast');

  // Init
  if (token && nodeId) {
    showApp();
  } else {
    showLogin();
  }

  // Service worker
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('./sw.js').catch(() => {});
  }

  // Login
  loginForm.addEventListener('submit', async e => {
    e.preventDefault();
    const id = nodeIdInput.value.trim();
    if (!id) return;

    loginError.classList.add('hidden');
    const btn = loginForm.querySelector('button');
    btn.disabled = true;
    btn.textContent = 'Connecting...';

    try {
      const res = await api('POST', '/auth', { node_id: id }, true);
      token = res.token;
      nodeId = res.node_id;
      localStorage.setItem('peer_token', token);
      localStorage.setItem('peer_node_id', nodeId);
      showApp();
    } catch (err) {
      loginError.textContent = err.message;
      loginError.classList.remove('hidden');
    } finally {
      btn.disabled = false;
      btn.textContent = 'Connect';
    }
  });

  // Navigation
  document.querySelectorAll('.nav-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
      document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
      btn.classList.add('active');
      $('page-' + btn.dataset.page).classList.add('active');
    });
  });

  // Logout
  $('logout-btn').addEventListener('click', disconnect);
  $('disconnect-btn').addEventListener('click', disconnect);

  // Consent toggle
  $('consent-toggle').addEventListener('change', async function () {
    try {
      await api('POST', '/consent', { enabled: this.checked });
      showToast(this.checked ? 'Node activated' : 'Node paused', 'success');
    } catch (err) {
      this.checked = !this.checked;
      showToast(err.message, 'error');
    }
  });

  // --- Functions ---

  function showLogin() {
    loginScreen.classList.add('active');
    mainApp.classList.remove('active');
    if (refreshTimer) clearInterval(refreshTimer);
  }

  function showApp() {
    loginScreen.classList.remove('active');
    mainApp.classList.add('active');
    headerNodeId.textContent = nodeId;
    $('s-node-id').textContent = nodeId;
    refresh();
    refreshTimer = setInterval(refresh, 15000);
  }

  async function disconnect() {
    try {
      await api('POST', '/disconnect', {});
    } catch (_) {}
    token = null;
    nodeId = null;
    localStorage.removeItem('peer_token');
    localStorage.removeItem('peer_node_id');
    showLogin();
    showToast('Disconnected', 'success');
  }

  async function refresh() {
    try {
      await Promise.all([loadStatus(), loadBandwidth(), loadEarnings()]);
    } catch (err) {
      if (err.message.includes('401') || err.message.includes('Unauthorized')) {
        disconnect();
      }
    }
  }

  async function loadStatus() {
    const res = await api('GET', '/status');
    const node = res.node;
    const load = res.load || 0;

    statusDot.className = 'status-dot online';
    $('m-status').textContent = 'Online';
    $('m-battery').textContent = node.battery ? node.battery + '%' : '--';
    $('m-cpu').textContent = node.cpu_usage ? node.cpu_usage.toFixed(1) + '%' : '--';
    $('m-load').textContent = load;
    $('m-country').textContent = node.country || '--';
    $('m-isp').textContent = node.isp || '--';

    $('i-node-id').textContent = node.id || '--';
    $('i-type').textContent = node.node_type || 'residential';
    $('i-ip').textContent = node.ip || '--';
    $('i-city').textContent = node.city || '--';
    $('i-os').textContent = node.os || '--';
    $('i-last-seen').textContent = node.last_seen ? timeAgo(node.last_seen) : '--';
    $('i-reputation').textContent = node.reputation ? node.reputation.toFixed(0) : '--';

    $('consent-toggle').checked = true;
  }

  async function loadBandwidth() {
    const res = await api('GET', '/bandwidth');
    const bw = res.current || {};
    const history = res.history || {};

    $('bw-sent').textContent = formatBytes(bw.bytes_sent || 0);
    $('bw-received').textContent = formatBytes(bw.bytes_received || 0);
    $('bw-duration').textContent = formatDuration(bw.duration_seconds || 0);

    // History
    const entries = Object.entries(history);
    const historyCard = $('history-card');
    const historyList = $('history-list');

    if (entries.length > 0) {
      historyCard.style.display = '';
      historyList.innerHTML = '';
      entries.sort((a, b) => b[0].localeCompare(a[0])).forEach(([date, data]) => {
        const div = document.createElement('div');
        div.className = 'history-item';
        div.innerHTML = `<span>${date}</span><span>${formatBytes(data.bytes_sent || 0)} / ${formatBytes(data.bytes_received || 0)}</span>`;
        historyList.appendChild(div);
      });
    } else {
      historyCard.style.display = 'none';
    }
  }

  async function loadEarnings() {
    const res = await api('GET', '/earnings');
    const p = res.payout || {};
    const rates = res.rates || {};
    const tiers = res.tiers || [];

    $('e-amount').textContent = '$' + (p.amount || 0).toFixed(2);
    $('e-period').textContent = p.period || 'Current Month';
    $('e-tier').textContent = p.tier || 'Basic';
    $('r-tier').textContent = p.tier || 'Basic';

    if (p.tier) {
      const tierInfo = tiers.find(t => t.name === p.tier);
      if (tierInfo) {
        $('r-sent').textContent = '$' + tierInfo.rate_per_gb_sent.toFixed(2);
        $('r-received').textContent = '$' + tierInfo.rate_per_gb_recv.toFixed(2);
      }
    } else {
      $('r-sent').textContent = '$' + (rates.RatePerGBSent || 0.50).toFixed(2);
      $('r-received').textContent = '$' + (rates.RatePerGBReceived || 0.30).toFixed(2);
    }
    $('r-min').textContent = '$' + (rates.MinPayoutAmount || 10.00).toFixed(2);

    const gbSent = p.gb_sent || 0;
    const gbReceived = p.gb_received || 0;
    const maxGB = Math.max(gbSent, gbReceived, 1);

    $('bar-sent').style.width = ((gbSent / maxGB) * 100) + '%';
    $('bar-received').style.width = ((gbReceived / maxGB) * 100) + '%';
    $('e-sent-gb').textContent = gbSent.toFixed(2) + ' GB';
    $('e-received-gb').textContent = gbReceived.toFixed(2) + ' GB';

    const historyCard = $('history-card');
    const historyList = $('history-list');
    const history = res.history || [];

    if (history.length > 0) {
      historyCard.style.display = '';
      historyList.innerHTML = '';
      history.forEach(h => {
        const item = document.createElement('div');
        item.className = 'history-item';
        item.innerHTML = '<span>' + (h.period || '--') + '</span><span>$' + (h.amount || 0).toFixed(2) + ' · ' + (h.tier || '--') + '</span>';
        historyList.appendChild(item);
      });
    } else {
      historyCard.style.display = 'none';
    }
  }

  async function api(method, path, body, noAuth) {
    const opts = {
      method,
      headers: { 'Content-Type': 'application/json' }
    };
    if (token && !noAuth) {
      opts.headers['X-Peer-Token'] = token;
    }
    if (body) {
      opts.body = JSON.stringify(body);
    }

    const res = await fetch(API + path, opts);
    const data = await res.json();

    if (!res.ok) {
      throw new Error(data.error || 'Request failed');
    }
    return data;
  }

  function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  function formatDuration(seconds) {
    if (seconds < 60) return seconds + 's';
    if (seconds < 3600) return Math.floor(seconds / 60) + 'm ' + (seconds % 60) + 's';
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    return h + 'h ' + m + 'm';
  }

  function timeAgo(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now - date) / 1000);
    if (seconds < 60) return 'just now';
    if (seconds < 3600) return Math.floor(seconds / 60) + 'm ago';
    if (seconds < 86400) return Math.floor(seconds / 3600) + 'h ago';
    return Math.floor(seconds / 86400) + 'd ago';
  }

  function showToast(msg, type) {
    toast.textContent = msg;
    toast.className = 'toast ' + (type || '');
    requestAnimationFrame(() => toast.classList.add('visible'));
    setTimeout(() => toast.classList.remove('visible'), 3000);
  }
})();
