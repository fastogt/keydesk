import { apiGet, apiPost, apiPut, apiDelete } from './core/api';
import { initSidebar } from './core/sidebar';
import { $, esc, makeToast, accountTypeBadge, statusDot } from './core/utils';
import { getAuthToken } from './core/storage';
import type { Account } from './core/types';

if (!getAuthToken()) { window.location.href = '/setup'; }

initSidebar('accounts');
const toast = makeToast('toast', 'toast');

let currentAccounts: Account[] = [];

async function loadList() {
  const search = ($('searchInput') as HTMLInputElement).value;
  const type = ($('typeFilter') as HTMLSelectElement).value;
  const params = new URLSearchParams();
  if (search) params.set('search', search);
  if (type) params.set('type', type);

  try {
    currentAccounts = await apiGet('/api/accounts?' + params.toString());
    renderList();
  } catch (err: any) {
    toast(err.message, 'error');
  }
}

function renderList() {
  const tbody = $('accountsBody');
  if (currentAccounts.length === 0) {
    tbody.innerHTML = '<tr><td colspan="4" class="text-muted" style="text-align:center;padding:24px">No accounts yet. Click "+ Add" to get started.</td></tr>';
    return;
  }

  tbody.innerHTML = currentAccounts.map(a => `
    <tr onclick="window.location='/accounts#${a.id}'" style="cursor:pointer">
      <td><strong>${esc(a.name)}</strong></td>
      <td>${accountTypeBadge(a.type)}</td>
      <td>${statusDot(a.people_count || 0)} ${a.people_count || 0} people</td>
      <td class="text-muted">${esc(a.login_email)}</td>
    </tr>
  `).join('');
}

$('addBtn')?.addEventListener('click', () => {
  ($('addModal') as HTMLElement).classList.add('active');
});

$('addModalClose')?.addEventListener('click', () => {
  ($('addModal') as HTMLElement).classList.remove('active');
});

$('addForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const data = {
    name: ($('addName') as HTMLInputElement).value,
    type: ($('addType') as HTMLSelectElement).value,
    login_url: ($('addURL') as HTMLInputElement).value,
    login_email: ($('addEmail') as HTMLInputElement).value,
    login_password: ($('addPassword') as HTMLInputElement).value,
    totp_secret: ($('addTOTP') as HTMLInputElement).value,
    notes: ($('addNotes') as HTMLTextAreaElement).value,
  };

  try {
    await apiPost('/api/accounts', data);
    ($('addModal') as HTMLElement).classList.remove('active');
    ($('addForm') as HTMLFormElement).reset();
    toast('Account added');
    loadList();
  } catch (err: any) {
    toast(err.message, 'error');
  }
});

async function loadDetail(id: string) {
  try {
    const data = await apiGet('/api/accounts/' + id);
    renderDetail(data);
    ($('listView') as HTMLElement).style.display = 'none';
    ($('detailView') as HTMLElement).style.display = 'block';
  } catch (err: any) {
    toast(err.message, 'error');
  }
}

function renderDetail(data: any) {
  const a = data.account;
  $('detailContent').innerHTML = `
    <div class="detail-header">
      <div>
        <h2>${esc(a.name)}</h2>
        <div class="text-muted">${accountTypeBadge(a.type)} ${a.login_url ? '&middot; ' + esc(a.login_url) : ''}</div>
      </div>
      <div class="flex-center">
        <button class="btn btn-danger" id="deleteAccountBtn">Delete</button>
      </div>
    </div>

    <div class="detail-fields" style="margin-top:16px">
      <div class="field-row"><label>Login:</label> <span>${esc(a.login_email)}</span></div>
      <div class="field-row">
        <label>Password:</label>
        <span id="pwdValue">••••••••••</span>
        <button class="btn btn-sm" id="showPwdBtn">Show</button>
        <button class="btn btn-sm" id="copyPwdBtn">Copy</button>
        <button class="btn btn-sm btn-danger" id="rotatePwdBtn">Rotate</button>
      </div>
      ${a.totp_secret ? `
      <div class="field-row">
        <label>2FA Code:</label>
        <span id="totpValue">------</span>
        <button class="btn btn-sm" id="showTotpBtn">Get Code</button>
      </div>` : ''}
      ${a.notes ? `<div class="field-row"><label>Notes:</label> <span>${esc(a.notes)}</span></div>` : ''}
    </div>

    <h3 style="margin-top:24px">People with access (${data.assignments.length})</h3>
    ${data.assignments.length > 0 ? `
    <table class="data-table">
      <thead><tr><th>Person</th><th>Since</th><th>Given by</th><th></th></tr></thead>
      <tbody>
        ${data.assignments.map((as: any) => `
          <tr>
            <td><strong>${esc(as.person_name || as.person_id)}</strong></td>
            <td class="text-muted">${esc(as.assigned_at?.slice(0,10) || '')}</td>
            <td class="text-muted">${esc(as.assigned_by)}</td>
            <td><button class="btn btn-sm btn-danger" data-revoke="${as.id}">Take</button></td>
          </tr>
        `).join('')}
      </tbody>
    </table>` : '<p class="text-muted">Nobody assigned</p>'}
    <button class="btn btn-primary btn-sm" id="assignBtn" style="margin-top:8px">+ Give to someone</button>

    <h3 style="margin-top:24px">History</h3>
    ${data.history.length > 0 ? data.history.map((e: any) =>
      `<div class="activity-item"><span class="text-muted">${esc(e.timestamp?.slice(0,10) || '')}</span> ${esc(e.details)}</div>`
    ).join('') : '<p class="text-muted">No history</p>'}
  `;

  document.getElementById('showPwdBtn')?.addEventListener('click', async () => {
    try {
      const result = await apiPost('/api/accounts/' + a.id + '/reveal');
      $('pwdValue').textContent = result.password;
    } catch (err: any) { toast(err.message, 'error'); }
  });

  document.getElementById('copyPwdBtn')?.addEventListener('click', async () => {
    try {
      const result = await apiPost('/api/accounts/' + a.id + '/reveal');
      await navigator.clipboard.writeText(result.password);
      toast('Password copied');
    } catch (err: any) { toast(err.message, 'error'); }
  });

  document.getElementById('rotatePwdBtn')?.addEventListener('click', async () => {
    if (!confirm('Generate a new password? The old one will be lost.')) return;
    try {
      const result = await apiPost('/api/accounts/' + a.id + '/rotate');
      $('pwdValue').textContent = result.password;
      toast('Password rotated');
    } catch (err: any) { toast(err.message, 'error'); }
  });

  document.getElementById('showTotpBtn')?.addEventListener('click', async () => {
    try {
      const result = await apiPost('/api/accounts/' + a.id + '/totp');
      $('totpValue').textContent = result.code;
    } catch (err: any) { toast(err.message, 'error'); }
  });

  document.getElementById('deleteAccountBtn')?.addEventListener('click', async () => {
    if (!confirm('Delete this account? This cannot be undone.')) return;
    try {
      await apiDelete('/api/accounts/' + a.id);
      toast('Account deleted');
      goBack();
      loadList();
    } catch (err: any) { toast(err.message, 'error'); }
  });

  document.querySelectorAll('[data-revoke]').forEach(btn => {
    btn.addEventListener('click', async (e) => {
      e.stopPropagation();
      if (!confirm('Take back this account access?')) return;
      try {
        await apiDelete('/api/assignments/' + (btn as HTMLElement).dataset.revoke);
        toast('Access revoked');
        loadDetail(a.id);
      } catch (err: any) { toast(err.message, 'error'); }
    });
  });

  document.getElementById('assignBtn')?.addEventListener('click', () => {
    ($('assignModal') as HTMLElement).classList.add('active');
    loadPeopleForAssign(a.id);
  });
}

async function loadPeopleForAssign(accountId: string) {
  try {
    const people = await apiGet('/api/people?status=active');
    const select = $('assignPerson') as HTMLSelectElement;
    select.innerHTML = people.map((p: any) =>
      `<option value="${p.id}">${esc(p.name)} (${esc(p.department)})</option>`
    ).join('');

    $('assignForm')!.onsubmit = async (e) => {
      e.preventDefault();
      try {
        await apiPost('/api/assignments', { person_id: select.value, account_id: accountId });
        ($('assignModal') as HTMLElement).classList.remove('active');
        toast('Access granted');
        loadDetail(accountId);
      } catch (err: any) { toast(err.message, 'error'); }
    };
  } catch (err: any) { toast(err.message, 'error'); }
}

function goBack() {
  window.location.hash = '';
  ($('listView') as HTMLElement).style.display = 'block';
  ($('detailView') as HTMLElement).style.display = 'none';
}

$('backBtn')?.addEventListener('click', goBack);
$('assignModalClose')?.addEventListener('click', () => ($('assignModal') as HTMLElement).classList.remove('active'));

$('searchInput')?.addEventListener('input', loadList);
$('typeFilter')?.addEventListener('change', loadList);

window.addEventListener('hashchange', () => {
  const id = window.location.hash.slice(1);
  if (id) loadDetail(id);
  else goBack();
});

loadList();
if (window.location.hash.length > 1) {
  loadDetail(window.location.hash.slice(1));
}
