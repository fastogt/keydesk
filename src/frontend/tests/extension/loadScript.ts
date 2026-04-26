import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import vm from 'node:vm';
import { vi } from 'vitest';

export const EXT_DIR = resolve(__dirname, '../../../../extension');

export type MessageListener = (
  msg: any,
  sender: any,
  sendResponse: (r: any) => void,
) => boolean | void;

export interface ChromeStub {
  runtime: {
    onMessage: { listeners: MessageListener[]; addListener: (l: MessageListener) => void };
    sendMessage: ReturnType<typeof vi.fn>;
    lastError: any;
  };
  storage: {
    local: {
      _data: Record<string, any>;
      get: (keys: string[] | string | null) => Promise<Record<string, any>>;
      set: (kv: Record<string, any>) => Promise<void>;
      remove: (keys: string[] | string) => Promise<void>;
    };
  };
  scripting: { executeScript: ReturnType<typeof vi.fn> };
}

export function makeChromeStub(initial: Record<string, any> = {}): ChromeStub {
  const data = { ...initial };
  return {
    runtime: {
      onMessage: {
        listeners: [],
        addListener(l: MessageListener) {
          this.listeners.push(l);
        },
      },
      sendMessage: vi.fn(),
      lastError: null,
    },
    storage: {
      local: {
        _data: data,
        get(keys) {
          if (!keys) return Promise.resolve({ ...data });
          const arr = Array.isArray(keys) ? keys : [keys];
          const out: Record<string, any> = {};
          for (const k of arr) out[k] = data[k];
          return Promise.resolve(out);
        },
        set(kv) {
          Object.assign(data, kv);
          return Promise.resolve();
        },
        remove(keys) {
          const arr = Array.isArray(keys) ? keys : [keys];
          for (const k of arr) delete data[k];
          return Promise.resolve();
        },
      },
    },
    scripting: { executeScript: vi.fn(() => Promise.resolve()) },
  };
}

export function loadScriptInVm(
  relPath: string,
  context: Record<string, any>,
): vm.Context {
  const code = readFileSync(resolve(EXT_DIR, relPath), 'utf8');
  const ctx = vm.createContext(context);
  vm.runInContext(code, ctx, { filename: relPath });
  return ctx;
}
