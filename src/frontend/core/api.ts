import { getAuthToken, setAuthToken, clearAuth } from './storage';

export const API_BASE = '';

export function unwrapResponse(json: any): any {
  if (json.error) {
    throw new Error(json.error.message || json.error);
  }
  return json.data || json;
}

export function getHeaders(): Record<string, string> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = getAuthToken();
  if (token) {
    headers['Authorization'] = 'Bearer ' + token;
  }
  return headers;
}

export async function fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
  const res = await fetch(url, {
    ...options,
    headers: { ...getHeaders(), ...(options.headers || {}) }
  });
  if (res.status === 401) {
    clearAuth();
    window.location.href = '/setup';
    throw new Error('Session expired');
  }
  const refreshToken = res.headers.get('X-Refresh-Token');
  if (refreshToken) {
    setAuthToken(refreshToken);
  }
  return res;
}

export async function apiCall(url: string, options: any = {}, { silent = false } = {}): Promise<any> {
  const res = await fetchWithAuth(url, options);
  const json = await res.json();

  if (!res.ok) {
    const message = json.error?.message || json.message || 'Request failed';
    const error = new Error(message) as any;
    error.status = res.status;
    error.data = json;
    if (silent) return null;
    throw error;
  }

  return unwrapResponse(json);
}

export const apiGet = (url: string, config?: any) => apiCall(url, { method: 'GET' }, config);

export const apiPost = (url: string, body?: any, config?: any) => apiCall(url, {
  method: 'POST',
  body: JSON.stringify(body)
}, config);

export const apiPut = (url: string, body?: any, config?: any) => apiCall(url, {
  method: 'PUT',
  body: JSON.stringify(body)
}, config);

export const apiDelete = (url: string, config?: any) => apiCall(url, { method: 'DELETE' }, config);
