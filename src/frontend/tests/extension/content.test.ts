/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { makeChromeStub, ChromeStub, MessageListener, EXT_DIR } from './loadScript';

const CONTENT_JS = readFileSync(resolve(EXT_DIR, 'content/content.js'), 'utf8');

let chromeStub: ChromeStub;
let messageListener: MessageListener;

function loadContentScript() {
  // The IIFE registers a message listener via chrome.runtime.onMessage.addListener
  // and calls checkCurrentURL on load. Run it in current jsdom scope.
  // eslint-disable-next-line @typescript-eslint/no-implied-eval
  new Function(CONTENT_JS)();
  messageListener = chromeStub.runtime.onMessage.listeners[0]!;
}

beforeEach(() => {
  document.body.innerHTML = '';
  chromeStub = makeChromeStub();
  (globalThis as any).chrome = chromeStub;
});

afterEach(() => {
  delete (globalThis as any).chrome;
  vi.useRealTimers();
});

describe('content fillForm', () => {
  it('fills email + password into a standard login form', () => {
    document.body.innerHTML = `
      <form>
        <input type="email" name="email" id="email-field" />
        <input type="password" name="password" id="pw-field" />
        <button type="submit">Sign in</button>
      </form>
    `;
    loadContentScript();

    const responses: any[] = [];
    messageListener(
      { action: 'fill', login: 'alice@x.com', password: 'hunter2' },
      {},
      (r) => responses.push(r),
    );

    const email = document.getElementById('email-field') as HTMLInputElement;
    const pw = document.getElementById('pw-field') as HTMLInputElement;
    expect(email.value).toBe('alice@x.com');
    expect(pw.value).toBe('hunter2');
    expect(responses[0]).toEqual({ success: true });
  });

  it('dispatches input + change events when filling fields', () => {
    document.body.innerHTML = `
      <form>
        <input type="email" name="email" id="e" />
        <input type="password" name="password" id="p" />
      </form>
    `;
    loadContentScript();

    const e = document.getElementById('e') as HTMLInputElement;
    const events: string[] = [];
    e.addEventListener('input', () => events.push('input'));
    e.addEventListener('change', () => events.push('change'));

    messageListener({ action: 'fill', login: 'a@x', password: 'p' }, {}, () => {});

    expect(events).toEqual(['input', 'change']);
  });

  it('matches input by autocomplete attribute when name is missing', () => {
    document.body.innerHTML = `
      <form>
        <input id="user" autocomplete="username" />
        <input type="password" id="pw" autocomplete="current-password" />
      </form>
    `;
    loadContentScript();

    messageListener({ action: 'fill', login: 'bob', password: 'sekret' }, {}, () => {});

    expect((document.getElementById('user') as HTMLInputElement).value).toBe('bob');
    expect((document.getElementById('pw') as HTMLInputElement).value).toBe('sekret');
  });

  it('does nothing when no inputs are present', () => {
    document.body.innerHTML = `<div>no form here</div>`;
    loadContentScript();

    const responses: any[] = [];
    messageListener({ action: 'fill', login: 'x', password: 'y' }, {}, (r) => responses.push(r));
    // sendResponse is still called (success: true) — fillForm just no-ops.
    expect(responses[0]).toEqual({ success: true });
  });

  it('also fills a TOTP field when totp is provided', async () => {
    vi.useFakeTimers();
    document.body.innerHTML = `
      <form>
        <input type="email" name="email" />
        <input type="password" name="password" />
        <input name="totp" id="totp-field" />
      </form>
    `;
    loadContentScript();

    messageListener(
      { action: 'fill', login: 'a@x', password: 'p', totp: '123456' },
      {},
      () => {},
    );

    // TOTP fill is scheduled via setTimeout(2000).
    await vi.advanceTimersByTimeAsync(2100);
    expect((document.getElementById('totp-field') as HTMLInputElement).value).toBe('123456');
  });
});

describe('content matchURL banner', () => {
  it('asks the background script to match the current URL on load', () => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, hostname: 'github.com' },
    });
    // Document readyState is 'complete' under jsdom, so checkCurrentURL fires immediately.
    loadContentScript();
    expect(chromeStub.runtime.sendMessage).toHaveBeenCalledWith(
      expect.objectContaining({ action: 'matchURL', url: 'github.com' }),
      expect.any(Function),
    );
  });
});
