export function $(id: string): HTMLElement {
  return document.getElementById(id)!;
}

export function escapeHtml(s: string): string {
  const el = document.createElement('div');
  el.textContent = s;
  return el.innerHTML;
}

export const esc = escapeHtml;

export function makeToast(
  elementId: string,
  cssClass: string,
): (msg: string, type?: 'success' | 'error') => void {
  return (msg: string, type: 'success' | 'error' = 'success') => {
    const el = document.getElementById(elementId);
    if (!el) return;
    el.textContent = msg;
    el.className = `${cssClass} ${type} show`;
    setTimeout(() => el.classList.remove('show'), 3000);
  };
}

export function formatDate(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export function formatDateTime(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

export function timeAgo(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  if (diffMs < 60000) return 'just now';
  if (diffMs < 3600000) return `${Math.floor(diffMs / 60000)}m ago`;
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

export function daysUntil(iso: string): number {
  if (!iso) return Infinity;
  const d = new Date(iso);
  const now = new Date();
  return Math.ceil((d.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
}

export function expiryBadge(expiresAt: string | null): string {
  if (!expiresAt) return '<span class="badge badge-ok">No expiry</span>';
  const days = daysUntil(expiresAt);
  if (days < 0) return `<span class="badge badge-expired">Expired ${Math.abs(days)}d ago</span>`;
  if (days <= 7) return `<span class="badge badge-danger">Expires in ${days}d</span>`;
  if (days <= 30) return `<span class="badge badge-warn">Expires in ${days}d</span>`;
  return `<span class="badge badge-ok">Expires in ${days}d</span>`;
}

export function statusDot(count: number): string {
  if (count === 0) return '<span class="dot dot-red" title="Nobody assigned"></span>';
  if (count === 1) return '<span class="dot dot-yellow" title="Only 1 person"></span>';
  return '<span class="dot dot-green" title="' + count + ' people"></span>';
}

export function accountTypeBadge(type: string): string {
  const colors: Record<string, string> = {
    email: '#3b82f6', social: '#8b5cf6', cloud: '#f59e0b', payment: '#10b981',
    platform: '#6366f1', dns: '#ec4899', analytics: '#14b8a6', ads: '#f97316', other: '#6b7280'
  };
  const color = colors[type] || colors.other;
  return `<span class="badge" style="background:${color}15;color:${color}">${esc(type)}</span>`;
}
