const refreshIntervalMs = 5000;

const views = [...document.querySelectorAll('.view')];
const navButtons = [...document.querySelectorAll('[data-view]')];
const statusDot = document.querySelector('#status-dot');
const statusText = document.querySelector('#status-text');
const lastUpdated = document.querySelector('#last-updated');

let activeView = 'overview';
let refreshTimer;

function formatInteger(value) {
  if (value === null || value === undefined || value === '') return '0';
  const text = String(value);
  if (/^-?\d+$/.test(text)) {
    try {
      return BigInt(text).toLocaleString();
    } catch (_) {
      return text;
    }
  }
  const number = Number(value);
  return Number.isFinite(number) ? number.toLocaleString() : text;
}

function formatBytes(value) {
  let bytes;
  try {
    bytes = BigInt(String(value || 0));
  } catch (_) {
    return String(value || 0);
  }
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  let unit = 0;
  let scaled = Number(bytes);
  while (scaled >= 1024 && unit < units.length - 1) {
    scaled /= 1024;
    unit++;
  }
  return `${scaled.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function formatTime(value) {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit'
  }).format(date);
}

function protocolName(version) {
  switch (Number(version)) {
    case 3: return 'MQTT 3.1';
    case 4: return 'MQTT 3.1.1';
    case 5: return 'MQTT 5.0';
    default: return `v${version}`;
  }
}

function escapeHTML(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}

async function fetchJSON(url) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 4000);
  try {
    const response = await fetch(url, {
      headers: { Accept: 'application/json' },
      cache: 'no-store',
      signal: controller.signal
    });
    if (!response.ok) {
      const error = new Error(`${response.status} ${response.statusText}`);
      error.status = response.status;
      throw error;
    }
    return await response.json();
  } finally {
    clearTimeout(timeout);
  }
}

function setBrokerStatus(ok, message) {
  statusDot.classList.toggle('online', ok);
  statusDot.classList.toggle('offline', !ok);
  statusText.textContent = message;
}

function setText(id, value) {
  const element = document.querySelector(`#${id}`);
  if (element) element.textContent = value;
}

async function refreshOverview() {
  const data = await fetchJSON('/admin/api/overview');
  setText('metric-active-sessions', formatInteger(data.active_sessions));
  setText('metric-subscriptions', formatInteger(data.subscriptions_current));
  setText('metric-received', formatInteger(data.messages_received));
  setText('metric-sent', formatInteger(data.messages_sent));
  setText('metric-inflight', formatInteger(data.inflight_current));
  setText('metric-queued', formatInteger(data.queued_current));
  setText('metric-dropped', formatInteger(data.messages_dropped));
  setText('connections-total', formatInteger(data.connections_total));
  setText('disconnections-total', formatInteger(data.disconnections_total));
  setText('sessions-inactive', formatInteger(data.inactive_sessions));
  setText('subscriptions-total', formatInteger(data.subscriptions_total));
  setText('packets-received', formatInteger(data.packets_received));
  setText('packets-sent', formatInteger(data.packets_sent));
  setText('bytes-received', formatBytes(data.bytes_received));
  setText('bytes-sent', formatBytes(data.bytes_sent));

  const qosBody = document.querySelector('#qos-body');
  qosBody.innerHTML = (data.qos || []).map((item) => `
    <tr>
      <td><span class="qos-badge">QoS ${escapeHTML(item.level)}</span></td>
      <td>${formatInteger(item.received)}</td>
      <td>${formatInteger(item.sent)}</td>
      <td>${formatInteger(item.dropped)}</td>
    </tr>`).join('');
}

async function refreshClients() {
  const data = await fetchJSON('/v1/clients?page=1&page_size=100');
  setText('clients-count', `${formatInteger(data.total_count)} sessions`);
  const tbody = document.querySelector('#clients-body');
  const clients = data.clients || [];
  tbody.innerHTML = clients.length ? clients.map((client) => {
    const online = !client.disconnected_at;
    return `
      <tr>
        <td><strong>${escapeHTML(client.client_id || '—')}</strong></td>
        <td><span class="state ${online ? 'state-online' : 'state-offline'}">${online ? 'online' : 'offline'}</span></td>
        <td>${escapeHTML(client.remote_addr || '—')}</td>
        <td>${escapeHTML(protocolName(client.version))}</td>
        <td>${formatInteger(client.inflight_len)}</td>
        <td>${formatInteger(client.queue_len)}</td>
        <td>${formatInteger(client.subscriptions_current)}</td>
        <td>${formatTime(client.connected_at)}</td>
      </tr>`;
  }).join('') : '<tr><td colspan="8" class="empty">No client sessions</td></tr>';
}

async function refreshSubscriptions() {
  const data = await fetchJSON('/v1/subscriptions?page=1&page_size=100');
  setText('subscriptions-count', `${formatInteger(data.total_count)} subscriptions`);
  const tbody = document.querySelector('#subscriptions-body');
  const subscriptions = data.subscriptions || [];
  tbody.innerHTML = subscriptions.length ? subscriptions.map((sub) => `
    <tr>
      <td><code>${escapeHTML(sub.topic_name || '—')}</code></td>
      <td>${escapeHTML(sub.client_id || '—')}</td>
      <td><span class="qos-badge">QoS ${formatInteger(sub.qos)}</span></td>
      <td>${sub.no_local ? 'yes' : 'no'}</td>
      <td>${sub.retain_as_published ? 'yes' : 'no'}</td>
    </tr>`).join('') : '<tr><td colspan="5" class="empty">No subscriptions</td></tr>';
}

async function refreshFederation() {
  const panel = document.querySelector('#federation-panel');
  try {
    const data = await fetchJSON('/v1/federation/members');
    const members = data.members || [];
    panel.innerHTML = members.length ? `<div class="member-grid">${members.map((member) => `
      <article class="member-card">
        <div class="member-title">
          <strong>${escapeHTML(member.name || 'unnamed')}</strong>
          <span class="state ${String(member.status).includes('ALIVE') ? 'state-online' : 'state-offline'}">${escapeHTML(String(member.status || 'unknown').replace('STATUS_', '').toLowerCase())}</span>
        </div>
        <dl>
          <div><dt>Address</dt><dd>${escapeHTML(member.addr || '—')}</dd></div>
          <div><dt>Federation</dt><dd>${escapeHTML(member.tags?.fed_addr || '—')}</dd></div>
        </dl>
      </article>`).join('')}</div>` : '<div class="empty-card">Federation is enabled but no members were returned.</div>';
  } catch (error) {
    if (error.status === 404) {
      panel.innerHTML = '<div class="empty-card">Federation plugin is not enabled on this broker.</div>';
      return;
    }
    throw error;
  }
}

async function refreshActiveView() {
  switch (activeView) {
    case 'clients': await refreshClients(); break;
    case 'subscriptions': await refreshSubscriptions(); break;
    case 'federation': await refreshFederation(); break;
    default: break;
  }
}

async function refresh() {
  try {
    await refreshOverview();
    setBrokerStatus(true, 'Broker reachable');
    lastUpdated.textContent = `Updated ${new Date().toLocaleTimeString()}`;
  } catch (error) {
    setBrokerStatus(false, error.name === 'AbortError' ? 'Broker timeout' : 'Broker unavailable');
  }

  if (activeView !== 'overview') {
    try {
      await refreshActiveView();
    } catch (error) {
      const target = document.querySelector(`#${activeView}-error`);
      if (target) target.textContent = `Unable to refresh: ${error.message}`;
    }
  }
}

function selectView(name) {
  activeView = name;
  views.forEach((view) => view.classList.toggle('active', view.dataset.viewName === name));
  navButtons.forEach((button) => button.classList.toggle('active', button.dataset.view === name));
  document.title = `GMQTT Admin · ${name[0].toUpperCase()}${name.slice(1)}`;
  refresh();
}

navButtons.forEach((button) => {
  button.addEventListener('click', () => selectView(button.dataset.view));
});

document.querySelector('#refresh-now').addEventListener('click', refresh);

refresh();
refreshTimer = setInterval(refresh, refreshIntervalMs);
window.addEventListener('beforeunload', () => clearInterval(refreshTimer));
