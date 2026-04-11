import { apiGet } from './core/api';
import { initSidebar } from './core/sidebar';
import { $, esc, timeAgo, expiryBadge, makeToast } from './core/utils';
import { getAuthToken } from './core/storage';

if (!getAuthToken()) { window.location.href = '/setup'; }

initSidebar('dashboard');
const toast = makeToast('toast', 'toast');

async function load() {
  try {
    const data = await apiGet('/api/dashboard');

    const stats = data.stats;
    $('statsGrid').innerHTML = `
      <div class="db-card"><div class="db-card-label">People</div><div class="db-card-value">${stats.people}</div></div>
      <div class="db-card"><div class="db-card-label">Accounts</div><div class="db-card-value">${stats.accounts}</div></div>
      <div class="db-card"><div class="db-card-label">Services</div><div class="db-card-value">${stats.services}</div></div>
      <div class="db-card"><div class="db-card-label">Credentials</div><div class="db-card-value">${stats.credentials}</div></div>
      ${stats.expired_credentials > 0 ? `<div class="db-card"><div class="db-card-label">Expired</div><div class="db-card-value" style="color:#dc2626">${stats.expired_credentials}</div></div>` : ''}
      ${stats.expiring_credentials > 0 ? `<div class="db-card"><div class="db-card-label">Expiring (30d)</div><div class="db-card-value" style="color:#d97706">${stats.expiring_credentials}</div></div>` : ''}
      ${stats.unassigned_accounts > 0 ? `<div class="db-card"><div class="db-card-label">Unassigned</div><div class="db-card-value" style="color:#6b7280">${stats.unassigned_accounts}</div></div>` : ''}
    `;

    const warnings = $('warnings');
    const warningItems: string[] = [];
    if (data.expiring_credentials?.length > 0) {
      for (const c of data.expiring_credentials) {
        warningItems.push(`<div class="warning-item">${expiryBadge(c.expires_at)} <strong>${esc(c.name)}</strong> (${esc(c.provider)})</div>`);
      }
    }
    if (stats.unassigned_accounts > 0) {
      warningItems.push(`<div class="warning-item"><span class="badge badge-warn">${stats.unassigned_accounts} accounts</span> have nobody assigned</div>`);
    }
    warnings.innerHTML = warningItems.length > 0
      ? `<h3>Warnings</h3>${warningItems.join('')}`
      : '';

    const activity = $('activity');
    if (data.recent_activity?.length > 0) {
      activity.innerHTML = '<h3>Recent Activity</h3>' + data.recent_activity.map((e: any) =>
        `<div class="activity-item"><span class="text-muted">${timeAgo(e.timestamp)}</span> ${esc(e.details)}</div>`
      ).join('');
    } else {
      activity.innerHTML = '<p class="text-muted">No activity yet</p>';
    }
  } catch (err: any) {
    toast(err.message, 'error');
  }
}

load();
