export const AUTH_TOKEN_KEY = 'auth_token';
export const AUTH_ADMIN_KEY = 'auth_admin';

export function getAuthToken(): string {
  return localStorage.getItem(AUTH_TOKEN_KEY) || '';
}

export function setAuthToken(token: string): void {
  localStorage.setItem(AUTH_TOKEN_KEY, token);
}

export function getAuthAdmin(): any {
  const stored = localStorage.getItem(AUTH_ADMIN_KEY);
  if (!stored) return null;
  try {
    return JSON.parse(stored);
  } catch {
    return null;
  }
}

export function setAuthAdmin(admin: any): void {
  localStorage.setItem(AUTH_ADMIN_KEY, JSON.stringify(admin));
}

export function clearAuth(): void {
  localStorage.removeItem(AUTH_TOKEN_KEY);
  localStorage.removeItem(AUTH_ADMIN_KEY);
}
