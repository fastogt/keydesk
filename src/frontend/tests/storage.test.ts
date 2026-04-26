import { describe, it, expect, beforeEach } from 'vitest';
import {
  AUTH_TOKEN_KEY,
  AUTH_ADMIN_KEY,
  getAuthToken,
  setAuthToken,
  getAuthAdmin,
  setAuthAdmin,
  clearAuth,
} from '../core/storage';

describe('storage', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('returns empty string when no token is stored', () => {
    expect(getAuthToken()).toBe('');
  });

  it('round-trips token via localStorage', () => {
    setAuthToken('abc.def.ghi');
    expect(getAuthToken()).toBe('abc.def.ghi');
    expect(localStorage.getItem(AUTH_TOKEN_KEY)).toBe('abc.def.ghi');
  });

  it('round-trips admin via JSON', () => {
    setAuthAdmin({ id: 'a-1', email: 'admin@x.com' });
    expect(getAuthAdmin()).toEqual({ id: 'a-1', email: 'admin@x.com' });
  });

  it('returns null admin when nothing is stored', () => {
    expect(getAuthAdmin()).toBeNull();
  });

  it('returns null when stored admin is malformed JSON', () => {
    localStorage.setItem(AUTH_ADMIN_KEY, '{not json');
    expect(getAuthAdmin()).toBeNull();
  });

  it('clearAuth removes both keys', () => {
    setAuthToken('t');
    setAuthAdmin({ id: 'a' });
    clearAuth();
    expect(localStorage.getItem(AUTH_TOKEN_KEY)).toBeNull();
    expect(localStorage.getItem(AUTH_ADMIN_KEY)).toBeNull();
  });
});
