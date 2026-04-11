import { apiGet, apiPost, apiPut, apiDelete } from './core/api';
import { initSidebar } from './core/sidebar';
import { $, esc, makeToast, expiryBadge } from './core/utils';
import { getAuthToken } from './core/storage';
import type { Service, Credential } from './core/types';

if (!getAuthToken()) { window.location.href = '/setup'; }

initSidebar('services');
const toast = makeToast('toast', 'toast');

let currentServices: Service[] = [];

async function loadList() {
  const search = ($('searchInput') as HTMLInputElement).value;
  const params = new URLSearchParams();
  if (search) params.set('search', search);

  try {
    currentServices = await apiGet('/api/services?' + params.toString());
    renderList();
  } catch (err: any) {
    toast(err.message, 'error');
  }
}

function serviceStatus(s: Service): string {
  if ((s.expired_count || 0) > 0) return '<span class="badge badge-expired">Expired</span>';
  if ((s.expiring_count || 0) > 0) return '<span class="badge badge-warn">Expiring</span>';
  return '<span class="badge badge-ok">OK</span>';
}

function renderList() {
  const tbody = $('servicesBody');
  if (currentServices.length === 0) {
    tbody.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align:center;padding:24px">No services yet. Click "+ Add" to get started.</td></tr>';
    return;
  }

  tbody.innerHTML = currentServices.map(s => `
    <tr onclick="window.location='/services#${s.id}'" style="cursor:pointer">
      <td><strong>${esc(s.name)}</strong></td>
      <td class="text-muted">${esc(s.owner_name || '-')}</td>
      <td>${s.credential_count || 0} keys</td>
      <td class="text-muted">${esc(s.environment)}</td>
      <td>${serviceStatus(s)}</td>
    </tr>
  `).join('');
}

$('addBtn')?.addEventListener('click', () => {
  ($('addModal') as HTMLElement).classList.add('active');
  loadPeopleForOwner();
});

$('addModalClose')?.addEventListener('click', () => ($('addModal') as HTMLElement).classList.remove('active'));

async function loadPeopleForOwner() {
  try {
    const people = await apiGet('/api/people?status=active');
    const select = $('addOwner') as HTMLSelectElement;
    select.innerHTML = '<option value="">No owner</option>' + people.map((p: any) =>
      `<option value="${p.id}">${esc(p.name)}</option>`
    ).join('');
  } catch (_) {}
}

$('addForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const data = {
    name: ($('addName') as HTMLInputElement).value,
    description: ($('addDesc') as HTMLTextAreaElement).value,
    environment: ($('addEnv') as HTMLSelectElement).value,
    owner_id: ($('addOwner') as HTMLSelectElement).value,
  };

  try {
    await apiPost('/api/services', data);
    ($('addModal') as HTMLElement).classList.remove('active');
    ($('addForm') as HTMLFormElement).reset();
    toast('Service added');
    loadList();
  } catch (err: any) { toast(err.message, 'error'); }
});

async function loadDetail(id: string) {
  try {
    const data = await apiGet('/api/services/' + id);
    renderDetail(data);
    ($('listView') as HTMLElement).style.display = 'none';
    ($('detailView') as HTMLElement).style.display = 'block';
  } catch (err: any) { toast(err.message, 'error'); }
}

function renderDetail(data: any) {
  const s = data.service;
  const creds: Credential[] = data.credentials || [];
  $('detailContent').innerHTML = `
    <div class="detail-header">
      <div>
        <h2>${esc(s.name)}</h2>
        <div class="text-muted">${esc(s.environment)} ${s.owner_name ? '&middot; Owner: ' + esc(s.owner_name) : ''}</div>
        ${s.description ? `<div class="text-muted mt-1">${esc(s.description)}</div>` : ''}
      </div>
      <div class="flex-center">
        <button class="btn btn-danger" id="deleteServiceBtn">Delete</button>
      </div>
    </div>

    <h3 style="margin-top:24px">Credentials (${creds.length})</h3>
    <button class="btn btn-primary btn-sm mb-3" id="addCredBtn">+ Add Credential</button>
    ${creds.length > 0 ? creds.map(c => `
      <div class="cred-card">
        <div class="cred-header">
          <strong>${esc(c.name)}</strong>
          <span class="text-muted">${esc(c.type)} &middot; ${esc(c.provider)}</span>
          ${expiryBadge(c.expires_at)}
        </div>
        <div class="cred-body">
          <div class="field-row">
            <label>Value:</label>
            <span id="credVal-${c.id}">••••••••••</span>
            <button class="btn btn-sm" data-show-cred="${c.id}">Show</button>
            <button class="btn btn-sm" data-copy-cred="${c.id}">Copy</button>
          </div>
          ${c.where_used ? `<div class="field-row"><label>Where:</label> <span class="text-muted">${esc(c.where_used)}</span></div>` : ''}
          ${c.last_rotated_at ? `<div class="field-row"><label>Rotated:</label> <span class="text-muted">${esc(c.last_rotated_at.slice(0,10))}</span></div>` : ''}
        </div>
      </div>
    `).join('') : '<p class="text-muted">No credentials</p>'}

    <h3 style="margin-top:24px">History</h3>
    ${data.history?.length > 0 ? data.history.map((e: any) =>
      `<div class="activity-item"><span class="text-muted">${esc(e.timestamp?.slice(0,10) || '')}</span> ${esc(e.details)}</div>`
    ).join('') : '<p class="text-muted">No history</p>'}
  `;

  document.querySelectorAll('[data-show-cred]').forEach(btn => {
    btn.addEventListener('click', async () => {
      const credId = (btn as HTMLElement).dataset.showCred!;
      try {
        const result = await apiPost('/api/credentials/' + credId + '/reveal');
        const el = document.getElementById('credVal-' + credId);
        if (el) el.textContent = result.key_value || result.secret_value || '(empty)';
      } catch (err: any) { toast(err.message, 'error'); }
    });
  });

  document.querySelectorAll('[data-copy-cred]').forEach(btn => {
    btn.addEventListener('click', async () => {
      const credId = (btn as HTMLElement).dataset.copyCred!;
      try {
        const result = await apiPost('/api/credentials/' + credId + '/reveal');
        await navigator.clipboard.writeText(result.key_value || result.secret_value || '');
        toast('Copied');
      } catch (err: any) { toast(err.message, 'error'); }
    });
  });

  document.getElementById('deleteServiceBtn')?.addEventListener('click', async () => {
    if (!confirm('Delete this service and all its credentials?')) return;
    try {
      await apiDelete('/api/services/' + s.id);
      toast('Service deleted');
      goBack();
      loadList();
    } catch (err: any) { toast(err.message, 'error'); }
  });

  document.getElementById('addCredBtn')?.addEventListener('click', () => {
    ($('credModal') as HTMLElement).classList.add('active');
    ($('credServiceId') as HTMLInputElement).value = s.id;
  });
}

$('credModalClose')?.addEventListener('click', () => ($('credModal') as HTMLElement).classList.remove('active'));

$('credForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const data = {
    service_id: ($('credServiceId') as HTMLInputElement).value,
    name: ($('credName') as HTMLInputElement).value,
    type: ($('credType') as HTMLSelectElement).value,
    provider: ($('credProvider') as HTMLInputElement).value,
    key_value: ($('credKey') as HTMLInputElement).value,
    secret_value: ($('credSecret') as HTMLInputElement).value,
    expires_at: ($('credExpires') as HTMLInputElement).value || null,
    where_used: ($('credWhere') as HTMLInputElement).value,
    notes: ($('credNotes') as HTMLTextAreaElement).value,
  };

  try {
    await apiPost('/api/credentials', data);
    ($('credModal') as HTMLElement).classList.remove('active');
    ($('credForm') as HTMLFormElement).reset();
    toast('Credential added');
    loadDetail(data.service_id);
  } catch (err: any) { toast(err.message, 'error'); }
});

function goBack() {
  window.location.hash = '';
  ($('listView') as HTMLElement).style.display = 'block';
  ($('detailView') as HTMLElement).style.display = 'none';
}

$('backBtn')?.addEventListener('click', goBack);
$('searchInput')?.addEventListener('input', loadList);

window.addEventListener('hashchange', () => {
  const id = window.location.hash.slice(1);
  if (id) loadDetail(id);
  else goBack();
});

loadList();
if (window.location.hash.length > 1) {
  loadDetail(window.location.hash.slice(1));
}
