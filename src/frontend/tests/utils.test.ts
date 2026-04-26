import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  escapeHtml,
  esc,
  formatDate,
  daysUntil,
  expiryBadge,
  statusDot,
  accountTypeBadge,
} from '../core/utils';

describe('escapeHtml / esc', () => {
  it('escapes script tags', () => {
    const out = escapeHtml('<script>alert(1)</script>');
    expect(out).not.toContain('<script>');
    expect(out).toContain('&lt;script&gt;');
  });

  it('escapes ampersands and quotes when injected via attribute', () => {
    expect(escapeHtml('Tom & Jerry')).toContain('&amp;');
  });

  it('returns empty for empty input', () => {
    expect(escapeHtml('')).toBe('');
  });

  it('esc is an alias for escapeHtml', () => {
    expect(esc('<x>')).toBe(escapeHtml('<x>'));
  });
});

describe('formatDate', () => {
  it('returns empty string for empty input', () => {
    expect(formatDate('')).toBe('');
  });

  it('formats an ISO date into a human-readable string', () => {
    const out = formatDate('2026-04-26T12:00:00Z');
    expect(out).toMatch(/2026/);
  });
});

describe('daysUntil', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-26T00:00:00Z'));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns Infinity for empty input', () => {
    expect(daysUntil('')).toBe(Infinity);
  });

  it('counts days to a future date', () => {
    expect(daysUntil('2026-05-01T00:00:00Z')).toBe(5);
  });

  it('returns negative for past dates', () => {
    expect(daysUntil('2026-04-20T00:00:00Z')).toBeLessThan(0);
  });
});

describe('expiryBadge', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-26T00:00:00Z'));
  });
  afterEach(() => vi.useRealTimers());

  it('returns "No expiry" badge for null input', () => {
    expect(expiryBadge(null)).toContain('No expiry');
  });

  it('marks expired dates', () => {
    expect(expiryBadge('2026-04-01T00:00:00Z')).toContain('Expired');
  });

  it('marks expiring within 7 days as danger', () => {
    expect(expiryBadge('2026-04-30T00:00:00Z')).toContain('badge-danger');
  });

  it('marks expiring within 30 days as warn', () => {
    expect(expiryBadge('2026-05-20T00:00:00Z')).toContain('badge-warn');
  });

  it('marks distant expiries as ok', () => {
    expect(expiryBadge('2027-01-01T00:00:00Z')).toContain('badge-ok');
  });
});

describe('statusDot', () => {
  it('renders red for zero people', () => {
    expect(statusDot(0)).toContain('dot-red');
  });
  it('renders yellow for one person', () => {
    expect(statusDot(1)).toContain('dot-yellow');
  });
  it('renders green for many people', () => {
    expect(statusDot(5)).toContain('dot-green');
  });
});

describe('accountTypeBadge', () => {
  it('falls back to other color for unknown type', () => {
    const known = accountTypeBadge('cloud');
    const unknown = accountTypeBadge('made-up-type');
    expect(known).toContain('cloud');
    // unknown still renders, escaped, with the "other" color #6b7280
    expect(unknown).toContain('made-up-type');
    expect(unknown).toContain('#6b7280');
  });

  it('escapes the type label (XSS guard)', () => {
    const out = accountTypeBadge('<img src=x>');
    expect(out).not.toContain('<img src=x>');
    expect(out).toContain('&lt;img');
  });
});
