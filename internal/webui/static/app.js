/* cardimportd management UI */

'use strict';

async function apiFetch(path, opts) {
  opts = opts || {};
  const res = await fetch(path, Object.assign({ headers: { 'Content-Type': 'application/json' } }, opts));
  const body = await res.json().catch(function() { return {}; });
  if (!res.ok) throw new Error(body.error || 'HTTP ' + res.status);
  return body;
}

let toastTimer = null;
function showToast(msg, type) {
  type = type || 'success';
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast-visible toast-' + type;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(function() { el.className = ''; }, 3000);
}

function initTabs() {
  document.querySelectorAll('.tab-btn').forEach(function(btn) {
    btn.addEventListener('click', function() {
      document.querySelectorAll('.tab-btn').forEach(function(b) { b.classList.remove('active'); });
      document.querySelectorAll('.panel').forEach(function(p) { p.classList.remove('active'); });
      btn.classList.add('active');
      document.getElementById('panel-' + btn.dataset.tab).classList.add('active');
    });
  });
}

function badgeHTML(status) {
  return '<span class="badge badge-' + status + '">' + status + '</span>';
}

function renderCards(cards) {
  const panel = document.getElementById('panel-cards');
  if (!cards || Object.keys(cards).length === 0) {
    panel.innerHTML = '<div class="empty-state"><span class="empty-state-icon">&#128190;</span><div class="empty-state-title">No cards registered</div><div class="empty-state-body">Insert a card &mdash; the daemon will create a pending entry. Refresh to see it.</div></div>';
    return;
  }
  let rows = '';
  Object.entries(cards).forEach(function(pair) {
    const uuid = pair[0], entry = pair[1];
    const firstSeen = entry.first_seen ? new Date(entry.first_seen).toLocaleString() : '&mdash;';
    const ownerDisplay = entry.owner ? entry.owner : '<em style="color:#aaa">unset</em>';
    rows += '<tr data-uuid="' + uuid + '">' +
      '<td class="uuid-cell">' + uuid + '</td>' +
      '<td class="owner-cell">' + ownerDisplay + '</td>' +
      '<td>' + badgeHTML(entry.status) + '</td>' +
      '<td>' + firstSeen + '</td>' +
      '<td class="actions-cell">' +
        '<button class="btn btn-secondary btn-sm" onclick="startEditCard('' + uuid + '')">Edit</button> ' +
        '<button class="btn btn-danger btn-sm" onclick="deleteCard('' + uuid + '')">Remove</button>' +
      '</td></tr>';
  });
  panel.innerHTML = '<div class="surface"><table class="card-table">' +
    '<thead><tr><th>UUID</th><th>Owner</th><th>Status</th><th>First Seen</th><th>Actions</th></tr></thead>' +
    '<tbody>' + rows + '</tbody></table></div>';
}

function startEditCard(uuid) {
  const row = document.querySelector('tr[data-uuid="' + uuid + '"]');
  const ownerCell = row.querySelector('.owner-cell');
  const cur = ownerCell.textContent.trim() === 'unset' ? '' : ownerCell.textContent.trim();
  ownerCell.innerHTML = '<input class="inline-edit-input" id="edit-owner-' + uuid + '" value="' + cur + '" placeholder="Owner name">';
  row.querySelector('.actions-cell').innerHTML =
    '<button class="btn btn-success btn-sm" onclick="saveCard('' + uuid + '', 'active')">Activate</button> ' +
    '<button class="btn btn-secondary btn-sm" onclick="saveCard('' + uuid + '', 'pending')">Keep Pending</button> ' +
    '<button class="btn btn-secondary btn-sm" onclick="loadCards()">Cancel</button>';
  document.getElementById('edit-owner-' + uuid).focus();
}

async function saveCard(uuid, status) {
  const el = document.getElementById('edit-owner-' + uuid);
  const owner = el ? el.value.trim() : '';
  try {
    await apiFetch('/api/cards/' + uuid, { method: 'POST', body: JSON.stringify({ owner: owner, status: status }) });
    showToast('Card updated');
    await loadCards();
  } catch(err) { showToast(err.message, 'error'); }
}

async function deleteCard(uuid) {
  if (!confirm('Remove card ' + uuid + '? This only removes the registration.')) return;
  try {
    await apiFetch('/api/cards/' + uuid, { method: 'DELETE' });
    showToast('Card removed');
    await loadCards();
  } catch(err) { showToast(err.message, 'error'); }
}

async function loadCards() {
  try {
    renderCards(await apiFetch('/api/cards'));
  } catch(err) {
    document.getElementById('panel-cards').innerHTML = '<div class="note">Failed to load cards: ' + err.message + '</div>';
  }
}

let _cfg = null;

function renderSettings(cfg) {
  _cfg = cfg;
  document.getElementById('panel-settings').innerHTML = '<div class="surface">' +
    '<div class="panel-heading">General</div>' +
    '<div class="form-group"><label>Import root</label>' +
      '<input type="text" id="s-import-root" value="' + (cfg.import_root || '') + '">' +
      '<span class="form-hint">Base directory, e.g. /volume1/photos</span></div>' +
    '<div class="form-group"><label>Watch paths (one per line)</label>' +
      '<textarea id="s-watch-paths" rows="3">' + (cfg.watch_paths || []).join('\n') + '</textarea>' +
      '<span class="form-hint">Mount-point prefixes, e.g. /volumeUSB1/usbshare</span></div>' +
    '<div class="form-group"><label>File extensions (one per line)</label>' +
      '<textarea id="s-extensions" rows="4">' + (cfg.file_extensions || []).join('\n') + '</textarea>' +
      '<span class="form-hint">e.g. .jpg .raf .arw .mp4 .mov</span></div>' +
    '<div class="form-group"><label>Log path</label>' +
      '<input type="text" id="s-log-path" value="' + (cfg.log_path || '') + '"></div>' +
    '<div class="form-actions"><button class="btn btn-primary" onclick="saveSettings()">Save</button></div>' +
    '</div>';
}

async function saveSettings() {
  const updated = Object.assign({}, _cfg, {
    import_root: document.getElementById('s-import-root').value.trim(),
    watch_paths: document.getElementById('s-watch-paths').value.split('\n').map(function(s){return s.trim();}).filter(Boolean),
    file_extensions: document.getElementById('s-extensions').value.split('\n').map(function(s){return s.trim();}).filter(Boolean),
    log_path: document.getElementById('s-log-path').value.trim(),
  });
  try {
    await apiFetch('/api/config', { method: 'POST', body: JSON.stringify(updated) });
    _cfg = updated;
    showToast('Settings saved');
  } catch(err) { showToast(err.message, 'error'); }
}

function renderNotifications(cfg) {
  const po = (cfg.notifications && cfg.notifications.pushover) ? cfg.notifications.pushover : { app_token: '', user_key: '' };
  document.getElementById('panel-notifications').innerHTML = '<div class="surface">' +
    '<div class="panel-heading">Pushover</div>' +
    '<div class="note">Pushover sends push notifications when imports complete or fail. ' +
      'Create an application at <strong>pushover.net</strong> to get an app token.</div>' +
    '<div class="form-group"><label>App token</label>' +
      '<input type="text" id="n-app-token" value="' + (po.app_token || '') + '" placeholder="aXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"></div>' +
    '<div class="form-group"><label>User key</label>' +
      '<input type="text" id="n-user-key" value="' + (po.user_key || '') + '" placeholder="uXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"></div>' +
    '<div class="form-actions">' +
      '<button class="btn btn-primary" onclick="saveNotifications()">Save</button> ' +
      '<button class="btn btn-secondary" onclick="testNotification()">Send test</button>' +
    '</div></div>';
}

async function saveNotifications() {
  const updated = Object.assign({}, _cfg, {
    notifications: { pushover: {
      app_token: document.getElementById('n-app-token').value.trim(),
      user_key: document.getElementById('n-user-key').value.trim(),
    }},
  });
  try {
    await apiFetch('/api/config', { method: 'POST', body: JSON.stringify(updated) });
    _cfg = updated;
    showToast('Notifications saved');
  } catch(err) { showToast(err.message, 'error'); }
}

async function testNotification() {
  try {
    await apiFetch('/api/notify/test', { method: 'POST' });
    showToast('Test notification sent');
  } catch(err) { showToast(err.message, 'error'); }
}

async function boot() {
  initTabs();
  try {
    const cfg = await apiFetch('/api/config');
    _cfg = cfg;
    renderSettings(cfg);
    renderNotifications(cfg);
  } catch(err) {
    showToast('Failed to load config: ' + err.message, 'error');
  }
  await loadCards();
}

document.addEventListener('DOMContentLoaded', boot);
