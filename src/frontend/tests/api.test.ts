import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { unwrapResponse, getHeaders, apiCall, apiPost } from '../core/api';
import { setAuthToken, clearAuth } from '../core/storage';

function jsonResponse(status: number, body: any, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  });
}

describe('unwrapResponse', () => {
  it('unwraps the data envelope', () => {
    expect(unwrapResponse({ data: { id: 'x' } })).toEqual({ id: 'x' });
  });

  it('falls back to the raw object if there is no data field', () => {
    expect(unwrapResponse({ id: 'x' })).toEqual({ id: 'x' });
  });

  it('throws when an error field is present', () => {
    expect(() =>
      unwrapResponse({ error: { code: 400, message: 'Bad request' } }),
    ).toThrow('Bad request');
  });
});

describe('getHeaders', () => {
  beforeEach(() => clearAuth());

  it('returns Content-Type only when no token is stored', () => {
    const h = getHeaders();
    expect(h['Content-Type']).toBe('application/json');
    expect(h['Authorization']).toBeUndefined();
  });

  it('attaches Bearer token when one is stored', () => {
    setAuthToken('tok-123');
    expect(getHeaders().Authorization).toBe('Bearer tok-123');
  });
});

describe('apiCall', () => {
  let originalLocation: Location;

  beforeEach(() => {
    clearAuth();
    originalLocation = window.location;
    // Override location.href setter so 401 redirect doesn't actually navigate.
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { href: '' } as Location,
    });
  });

  afterEach(() => {
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation });
    vi.unstubAllGlobals();
  });

  it('unwraps the data envelope on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(jsonResponse(200, { data: { hello: 'world' } }))),
    );
    const result = await apiCall('/api/x');
    expect(result).toEqual({ hello: 'world' });
  });

  it('throws an Error with the server message on non-2xx', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(jsonResponse(400, { error: { code: 400, message: 'Bad input' } })),
      ),
    );
    await expect(apiCall('/api/x')).rejects.toThrow('Bad input');
  });

  it('redirects to /setup and clears auth on 401', async () => {
    setAuthToken('expired');
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(jsonResponse(401, { error: { code: 401, message: 'Expired' } })),
      ),
    );
    await expect(apiCall('/api/x')).rejects.toThrow('Session expired');
    expect(window.location.href).toBe('/setup');
    expect(localStorage.getItem('auth_token')).toBeNull();
  });

  it('returns null instead of throwing when silent=true', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(jsonResponse(404, { error: { message: 'nope' } }))),
    );
    const r = await apiCall('/api/x', {}, { silent: true });
    expect(r).toBeNull();
  });

  it('apiPost stringifies the body and uses POST', async () => {
    const fetchMock = vi.fn(
      (_url: string, _init?: RequestInit) =>
        Promise.resolve(jsonResponse(200, { data: 'ok' })),
    );
    vi.stubGlobal('fetch', fetchMock);
    await apiPost('/api/x', { name: 'Alice' });
    const init = fetchMock.mock.calls[0]?.[1];
    expect(init?.method).toBe('POST');
    expect(init?.body).toBe(JSON.stringify({ name: 'Alice' }));
  });

  it('persists X-Refresh-Token from response headers', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(jsonResponse(200, { data: 'ok' }, { 'X-Refresh-Token': 'new-token' })),
      ),
    );
    await apiCall('/api/x');
    expect(localStorage.getItem('auth_token')).toBe('new-token');
  });
});
