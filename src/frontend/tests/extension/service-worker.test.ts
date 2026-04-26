/**
 * @vitest-environment node
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { loadScriptInVm, makeChromeStub, ChromeStub, MessageListener } from './loadScript';

function setupSW(initialStorage: Record<string, any> = {}) {
  const chromeStub = makeChromeStub(initialStorage);
  const fetchMock = vi.fn();
  const ctx = {
    chrome: chromeStub,
    fetch: fetchMock,
    console: { error: vi.fn(), log: vi.fn(), warn: vi.fn() },
    setTimeout,
    clearTimeout,
    encodeURIComponent,
    URL,
    Promise,
    Error,
    JSON,
    Object,
    globalThis: undefined as any,
  };
  ctx.globalThis = ctx;
  loadScriptInVm('background/service-worker.js', ctx);
  const listener = chromeStub.runtime.onMessage.listeners[0] as MessageListener;
  return { chrome: chromeStub, fetch: fetchMock, listener };
}

function send(listener: MessageListener, msg: any): Promise<any> {
  return new Promise((resolve) => {
    const ret = listener(msg, {}, resolve);
    // Listener returns true when it will call sendResponse asynchronously.
    // If it returns false/undefined, it must have called sendResponse synchronously.
    if (ret !== true) {
      // Guard: nothing else to do; resolve will be called synchronously above.
    }
  });
}

function jsonResponse(status: number, body: any): any {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

describe('service-worker login', () => {
  it('stores token + person on successful login', async () => {
    const { chrome, fetch, listener } = setupSW();
    fetch.mockResolvedValue(
      jsonResponse(200, { data: { token: 'tok-1', person: { id: 'p-1', name: 'Alice' } } }),
    );

    const resp = await send(listener, {
      action: 'login',
      serverUrl: 'https://kd.example.com',
      personId: 'p-1',
      pin: '',
    });

    expect(resp.success).toBe(true);
    expect(fetch).toHaveBeenCalledWith(
      'https://kd.example.com/api/ext/auth',
      expect.objectContaining({ method: 'POST' }),
    );
    expect(chrome.storage.local._data.keydesk_token).toBe('tok-1');
    expect(chrome.storage.local._data.keydesk_server_url).toBe('https://kd.example.com');
  });

  it('returns success=false on login failure', async () => {
    const { fetch, listener } = setupSW();
    fetch.mockResolvedValue(jsonResponse(401, { error: { message: 'Invalid credentials' } }));

    const resp = await send(listener, {
      action: 'login',
      serverUrl: 'https://x',
      personId: 'p-1',
      pin: '',
    });

    expect(resp.success).toBe(false);
    expect(resp.error).toBe('Invalid credentials');
  });
});

describe('service-worker authed API calls', () => {
  it('attaches Bearer auth + serverUrl when fetching getAccounts', async () => {
    const { fetch, listener } = setupSW({
      keydesk_server_url: 'https://kd.example.com',
      keydesk_token: 'tok-9',
    });
    fetch.mockResolvedValue(jsonResponse(200, { data: [{ id: 'a-1', name: 'GitHub' }] }));

    const resp = await send(listener, { action: 'getAccounts' });

    expect(resp.success).toBe(true);
    expect(resp.accounts).toEqual([{ id: 'a-1', name: 'GitHub' }]);
    expect(fetch).toHaveBeenCalledWith(
      'https://kd.example.com/api/ext/accounts',
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: 'Bearer tok-9' }),
      }),
    );
  });

  it('returns the error message when the server responds with non-2xx', async () => {
    const { fetch, listener } = setupSW({
      keydesk_server_url: 'https://x',
      keydesk_token: 't',
    });
    fetch.mockResolvedValue(jsonResponse(403, { error: { message: 'Access denied' } }));

    const resp = await send(listener, { action: 'getCredentials', accountId: 'a-1' });

    expect(resp.success).toBe(false);
    expect(resp.error).toBe('Access denied');
  });

  it('refuses to call API when not configured', async () => {
    const { listener } = setupSW(); // empty storage
    const resp = await send(listener, { action: 'getAccounts' });
    expect(resp.success).toBe(false);
    expect(resp.error).toBe('Not configured');
  });

  it('logout clears stored token and person', async () => {
    const { chrome, listener } = setupSW({
      keydesk_server_url: 'https://x',
      keydesk_token: 't',
      keydesk_person: '{}',
    });
    await send(listener, { action: 'logout' });
    expect(chrome.storage.local._data.keydesk_token).toBeUndefined();
    expect(chrome.storage.local._data.keydesk_person).toBeUndefined();
    // server URL is intentionally preserved (so re-login uses the same server).
    expect(chrome.storage.local._data.keydesk_server_url).toBe('https://x');
  });

  it('matchURL forwards URL via query string', async () => {
    const { fetch, listener } = setupSW({
      keydesk_server_url: 'https://x',
      keydesk_token: 't',
    });
    fetch.mockResolvedValue(jsonResponse(200, { data: { match: true, account_id: 'a-1' } }));

    const resp = await send(listener, { action: 'matchURL', url: 'https://github.com/login' });

    expect(resp.success).toBe(true);
    expect(resp.match).toEqual({ match: true, account_id: 'a-1' });
    const calledUrl = (fetch.mock.calls[0] as any[])[0] as string;
    expect(calledUrl).toContain('/api/ext/match?url=');
    expect(calledUrl).toContain(encodeURIComponent('https://github.com/login'));
  });
});
