import { setAuthToken, setAuthAdmin, getAuthToken } from './core/storage';

if (getAuthToken()) {
  window.location.href = '/dashboard';
}

const form = document.getElementById('login-form') as HTMLFormElement;
const errorEl = document.getElementById('login-error') as HTMLElement;

form?.addEventListener('submit', async (e) => {
  e.preventDefault();
  errorEl.style.display = 'none';

  const email = (document.getElementById('email') as HTMLInputElement).value;
  const password = (document.getElementById('password') as HTMLInputElement).value;

  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });
    const json = await res.json();

    if (!res.ok) {
      errorEl.textContent = json.error?.message || 'Login failed';
      errorEl.style.display = 'block';
      return;
    }

    setAuthToken(json.data.token);
    setAuthAdmin(json.data.admin);
    window.location.href = '/dashboard';
  } catch (err: any) {
    errorEl.textContent = err.message || 'Connection failed';
    errorEl.style.display = 'block';
  }
});
