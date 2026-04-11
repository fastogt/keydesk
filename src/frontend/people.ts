import { apiGet, apiPost, apiPut, apiDelete } from './core/api';
import { initSidebar } from './core/sidebar';
import { $, esc, makeToast } from './core/utils';
import { getAuthToken } from './core/storage';
import type { Person } from './core/types';

if (!getAuthToken()) { window.location.href = '/setup'; }

initSidebar('people');
const toast = makeToast('toast', 'toast');

let currentPeople: Person[] = [];

async function loadList() {
  const search = ($('searchInput') as HTMLInputElement).value;
  const status = ($('statusFilter') as HTMLSelectElement).value;
  const params = new URLSearchParams();
  if (search) params.set('search', search);
  if (status) params.set('status', status);

  try {
    currentPeople = await apiGet('/api/people?' + params.toString());
    renderList();
  } catch (err: any) {
    toast(err.message, 'error');
  }
}

function renderList() {
  const tbody = $('peopleBody');
  if (currentPeople.length === 0) {
    tbody.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align:center;padding:24px">No people yet. Click "+ Add" to get started.</td></tr>';
    return;
  }

  tbody.innerHTML = currentPeople.map(p => `
    <tr onclick="window.location='/people#${p.id}'" style="cursor:pointer">
      <td><strong>${esc(p.name)}</strong></td>
      <td class="text-muted">${esc(p.department)}</td>
      <td>${p.account_count || 0} accounts</td>
      <td>${p.service_count || 0} services</td>
      <td>${p.status === 'active'
        ? '<span class="badge badge-ok">Active</span>'
        : '<span class="badge badge-expired">Offboarded</span>'
      }</td>
    </tr>
  `).join('');
}

// Add person modal
$('addBtn')?.addEventListener('click', () => {
  ($('addModal') as HTMLElement).classList.add('active');
});

$('addModalClose')?.addEventListener('click', () => {
  ($('addModal') as HTMLElement).classList.remove('active');
});

$('addForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const name = ($('addName') as HTMLInputElement).value;
  const email = ($('addEmail') as HTMLInputElement).value;
  const department = ($('addDept') as HTMLInputElement).value;
  const notes = ($('addNotes') as HTMLTextAreaElement).value;

  try {
    await apiPost('/api/people', { name, email, department, notes });
    ($('addModal') as HTMLElement).classList.remove('active');
    ($('addForm') as HTMLFormElement).reset();
    toast('Person added');
    loadList();
  } catch (err: any) {
    toast(err.message, 'error');
  }
});

// Detail view
async function loadDetail(id: string) {
  try {
    const data = await apiGet('/api/people/' + id);
    renderDetail(data);
    ($('listView') as HTMLElement).style.display = 'none';
    ($('detailView') as HTMLElement).style.display = 'block';
  } catch (err: any) {
    toast(err.message, 'error');
  }
}

function renderDetail(data: any) {
  const p = data.person;
  $('detailContent').innerHTML = `
    <div class="detail-header">
      <div>
        <h2>${esc(p.name)}</h2>
        <div class="text-muted">${esc(p.department)} ${p.email ? '&middot; ' + esc(p.email) : ''}</div>
      </div>
      <div class="flex-center">
        <button class="btn btn-danger" id="offboardBtn">Offboard</button>
      </div>
    </div>

    <h3 style="margin-top:24px">Assigned Accounts (${data.assignments.length})</h3>
    ${data.assignments.length > 0 ? `
    <table class="data-table">
      <thead><tr><th>Account</th><th>Given</th><th>By</th><th></th></tr></thead>
      <tbody>
        ${data.assignments.map((a: any) => `
          <tr>
            <td><strong>${esc(a.account_name || a.account_id)}</strong></td>
            <td class="text-muted">${esc(a.assigned_at?.slice(0,10) || '')}</td>
            <td class="text-muted">${esc(a.assigned_by)}</td>
            <td><button class="btn btn-sm btn-danger" data-revoke="${a.id}">Take</button></td>
          </tr>
        `).join('')}
      </tbody>
    </table>` : '<p class="text-muted">No accounts assigned</p>'}

    <h3 style="margin-top:24px">Services Owned (${data.services.length})</h3>
    ${data.services.length > 0 ? `<ul>${data.services.map((s: any) =>
      `<li><a href="/services#${s.id}">${esc(s.name)}</a> (${esc(s.environment)})</li>`
    ).join('')}</ul>` : '<p class="text-muted">No services owned</p>'}

    <h3 style="margin-top:24px">History</h3>
    ${data.history.length > 0 ? data.history.map((e: any) =>
      `<div class="activity-item"><span class="text-muted">${esc(e.timestamp?.slice(0,10) || '')}</span> ${esc(e.details)}</div>`
    ).join('') : '<p class="text-muted">No history</p>'}
  `;

  document.querySelectorAll('[data-revoke]').forEach(btn => {
    btn.addEventListener('click', async (e) => {
      e.stopPropagation();
      const assignmentId = (btn as HTMLElement).dataset.revoke!;
      if (!confirm('Take back this account?')) return;
      try {
        await apiDelete('/api/assignments/' + assignmentId);
        toast('Account taken back');
        loadDetail(p.id);
      } catch (err: any) {
        toast(err.message, 'error');
      }
    });
  });

  document.getElementById('offboardBtn')?.addEventListener('click', async () => {
    if (!confirm(`Offboard ${p.name}? This will revoke all account access.`)) return;
    try {
      await apiPost('/api/people/' + p.id + '/offboard', {
        service_owners: {}
      });
      toast('Person offboarded');
      goBack();
      loadList();
    } catch (err: any) {
      toast(err.message, 'error');
    }
  });
}

function goBack() {
  window.location.hash = '';
  ($('listView') as HTMLElement).style.display = 'block';
  ($('detailView') as HTMLElement).style.display = 'none';
}

$('backBtn')?.addEventListener('click', goBack);

$('searchInput')?.addEventListener('input', loadList);
$('statusFilter')?.addEventListener('change', loadList);

window.addEventListener('hashchange', () => {
  const id = window.location.hash.slice(1);
  if (id) loadDetail(id);
  else goBack();
});

loadList();
if (window.location.hash.length > 1) {
  loadDetail(window.location.hash.slice(1));
}
