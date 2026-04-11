import { apiGet, apiPut } from './core/api';
import { initSidebar } from './core/sidebar';
import { $, makeToast } from './core/utils';
import { getAuthToken } from './core/storage';

if (!getAuthToken()) { window.location.href = '/setup'; }

initSidebar('settings');
const toast = makeToast('toast', 'toast');

async function loadProfile() {
  try {
    const admin = await apiGet('/api/settings/profile');
    ($('profileName') as HTMLInputElement).value = admin.name || '';
    ($('profileEmail') as HTMLInputElement).value = admin.email || '';
  } catch (err: any) {
    toast(err.message, 'error');
  }
}

$('profileForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  try {
    await apiPut('/api/settings/profile', {
      name: ($('profileName') as HTMLInputElement).value,
      email: ($('profileEmail') as HTMLInputElement).value,
    });
    toast('Profile updated');
  } catch (err: any) { toast(err.message, 'error'); }
});

$('passwordForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const newPwd = ($('newPassword') as HTMLInputElement).value;
  const confirmPwd = ($('confirmPassword') as HTMLInputElement).value;
  if (newPwd !== confirmPwd) {
    toast('Passwords do not match', 'error');
    return;
  }
  try {
    await apiPut('/api/settings/password', {
      current_password: ($('currentPassword') as HTMLInputElement).value,
      new_password: newPwd,
    });
    ($('passwordForm') as HTMLFormElement).reset();
    toast('Password changed');
  } catch (err: any) { toast(err.message, 'error'); }
});

loadProfile();
