const STORAGE_KEYS = {
  SERVER_URL: 'keydesk_server_url',
  TOKEN: 'keydesk_token',
  PERSON: 'keydesk_person',
};

async function init() {
  const data = await chrome.storage.local.get([STORAGE_KEYS.TOKEN, STORAGE_KEYS.PERSON, STORAGE_KEYS.SERVER_URL]);

  if (data[STORAGE_KEYS.TOKEN]) {
    showAccountsView(data);
  } else {
    showLoginView();
  }
}

function showLoginView() {
  document.getElementById('loginView').style.display = 'block';
  document.getElementById('accountsView').style.display = 'none';
  document.getElementById('userName').textContent = '';

  document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const errorEl = document.getElementById('loginError');
    errorEl.style.display = 'none';

    const serverUrl = document.getElementById('serverUrl').value.replace(/\/$/, '');
    const personId = document.getElementById('personId').value;
    const pin = document.getElementById('pin').value;

    chrome.runtime.sendMessage(
      { action: 'login', serverUrl, personId, pin },
      (response) => {
        if (response?.success) {
          init();
        } else {
          errorEl.textContent = response?.error || 'Login failed';
          errorEl.style.display = 'block';
        }
      }
    );
  });
}

function showAccountsView(data) {
  document.getElementById('loginView').style.display = 'none';
  document.getElementById('accountsView').style.display = 'block';

  try {
    const person = JSON.parse(data[STORAGE_KEYS.PERSON] || '{}');
    document.getElementById('userName').textContent = person.name || person.email || '';
  } catch (_) {}

  loadAccounts();

  document.getElementById('logoutBtn').addEventListener('click', (e) => {
    e.preventDefault();
    chrome.runtime.sendMessage({ action: 'logout' }, () => init());
  });

  document.getElementById('openDashboard').addEventListener('click', (e) => {
    e.preventDefault();
    chrome.storage.local.get(STORAGE_KEYS.SERVER_URL, (d) => {
      if (d[STORAGE_KEYS.SERVER_URL]) {
        chrome.tabs.create({ url: d[STORAGE_KEYS.SERVER_URL] + '/dashboard' });
      }
    });
  });
}

function loadAccounts() {
  const list = document.getElementById('accountsList');
  list.innerHTML = '<div class="empty">Loading...</div>';

  chrome.runtime.sendMessage({ action: 'getAccounts' }, (response) => {
    if (!response?.success) {
      list.innerHTML = '<div class="empty">Failed to load accounts</div>';
      return;
    }

    const accounts = response.accounts || [];
    if (accounts.length === 0) {
      list.innerHTML = '<div class="empty">No accounts assigned to you.<br>Ask your admin.</div>';
      return;
    }

    list.innerHTML = accounts.map(a => `
      <div class="account-item" data-id="${esc(a.id)}" data-url="${esc(a.login_url)}">
        <div>
          <div class="account-name">${esc(a.name)}</div>
          <div class="account-type">${esc(a.type)}</div>
        </div>
        <button class="open-btn" data-id="${esc(a.id)}" data-url="${esc(a.login_url)}">Open</button>
      </div>
    `).join('');

    document.querySelectorAll('.open-btn').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        openAccount(btn.dataset.id, btn.dataset.url);
      });
    });
  });
}

async function openAccount(accountId, loginUrl) {
  if (loginUrl) {
    const url = loginUrl.startsWith('http') ? loginUrl : 'https://' + loginUrl;
    const tab = await chrome.tabs.create({ url });

    chrome.runtime.sendMessage({
      action: 'autofill',
      tabId: tab.id,
      accountId,
    });
  }
}

function esc(s) {
  if (!s) return '';
  const el = document.createElement('div');
  el.textContent = s;
  return el.innerHTML;
}

init();
