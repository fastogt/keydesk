const STORAGE_KEYS = {
  SERVER_URL: 'keydesk_server_url',
  TOKEN: 'keydesk_token',
  PERSON: 'keydesk_person',
};

async function getConfig() {
  const data = await chrome.storage.local.get([STORAGE_KEYS.SERVER_URL, STORAGE_KEYS.TOKEN]);
  return {
    serverUrl: data[STORAGE_KEYS.SERVER_URL] || '',
    token: data[STORAGE_KEYS.TOKEN] || '',
  };
}

async function apiCall(path, options = {}) {
  const config = await getConfig();
  if (!config.serverUrl || !config.token) {
    throw new Error('Not configured');
  }

  const res = await fetch(config.serverUrl + path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + config.token,
      ...(options.headers || {}),
    },
  });

  const json = await res.json();
  if (!res.ok) {
    throw new Error(json.error?.message || 'Request failed');
  }
  return json.data || json;
}

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.action === 'getAccounts') {
    apiCall('/api/ext/accounts')
      .then(accounts => sendResponse({ success: true, accounts }))
      .catch(err => sendResponse({ success: false, error: err.message }));
    return true;
  }

  if (msg.action === 'getCredentials') {
    apiCall('/api/ext/credentials/' + msg.accountId, { method: 'POST' })
      .then(creds => sendResponse({ success: true, credentials: creds }))
      .catch(err => sendResponse({ success: false, error: err.message }));
    return true;
  }

  if (msg.action === 'getTOTP') {
    apiCall('/api/ext/totp/' + msg.accountId, { method: 'POST' })
      .then(data => sendResponse({ success: true, code: data.code }))
      .catch(err => sendResponse({ success: false, error: err.message }));
    return true;
  }

  if (msg.action === 'matchURL') {
    apiCall('/api/ext/match?url=' + encodeURIComponent(msg.url))
      .then(data => sendResponse({ success: true, match: data }))
      .catch(err => sendResponse({ success: false, error: err.message }));
    return true;
  }

  if (msg.action === 'logAudit') {
    apiCall('/api/ext/audit', {
      method: 'POST',
      body: JSON.stringify({
        action: msg.auditAction,
        account_id: msg.accountId,
        details: msg.details,
      }),
    })
      .then(() => sendResponse({ success: true }))
      .catch(() => sendResponse({ success: false }));
    return true;
  }

  if (msg.action === 'login') {
    login(msg.serverUrl, msg.personId, msg.pin)
      .then(data => sendResponse({ success: true, data }))
      .catch(err => sendResponse({ success: false, error: err.message }));
    return true;
  }

  if (msg.action === 'autofill') {
    handleAutofill(msg.tabId, msg.accountId);
    sendResponse({ success: true });
    return false;
  }

  if (msg.action === 'getConfig') {
    getConfig().then(config => sendResponse({ success: true, config }));
    return true;
  }

  if (msg.action === 'logout') {
    chrome.storage.local.remove([STORAGE_KEYS.TOKEN, STORAGE_KEYS.PERSON]);
    sendResponse({ success: true });
    return false;
  }
});

async function login(serverUrl, personId, pin) {
  const res = await fetch(serverUrl + '/api/ext/auth', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ person_id: personId, pin: pin }),
  });

  const json = await res.json();
  if (!res.ok) {
    throw new Error(json.error?.message || 'Login failed');
  }

  const data = json.data || json;
  await chrome.storage.local.set({
    [STORAGE_KEYS.SERVER_URL]: serverUrl,
    [STORAGE_KEYS.TOKEN]: data.token,
    [STORAGE_KEYS.PERSON]: JSON.stringify(data.person),
  });

  return data;
}

async function handleAutofill(tabId, accountId) {
  try {
    const creds = await apiCall('/api/ext/credentials/' + accountId, { method: 'POST' });

    await chrome.scripting.executeScript({
      target: { tabId },
      func: fillCredentials,
      args: [creds.login_email, creds.password],
    });

    await apiCall('/api/ext/audit', {
      method: 'POST',
      body: JSON.stringify({
        action: 'autofill',
        account_id: accountId,
        details: 'Auto-filled credentials via extension',
      }),
    });
  } catch (err) {
    console.error('Autofill failed:', err);
  }
}

function fillCredentials(email, password) {
  function setNativeValue(el, value) {
    const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
      window.HTMLInputElement.prototype, 'value'
    ).set;
    nativeInputValueSetter.call(el, value);
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }

  const emailSelectors = [
    'input[type="email"]',
    'input[name="email"]',
    'input[name="username"]',
    'input[name="login"]',
    'input[name="login_field"]',
    'input[name="user[login]"]',
    'input[autocomplete="username"]',
    'input[autocomplete="email"]',
    'input[id*="email"]',
    'input[id*="user"]',
    'input[id*="login"]',
  ];

  const passwordSelectors = [
    'input[type="password"]',
    'input[name="password"]',
    'input[autocomplete="current-password"]',
  ];

  let emailField = null;
  for (const sel of emailSelectors) {
    emailField = document.querySelector(sel);
    if (emailField) break;
  }

  let passwordField = null;
  for (const sel of passwordSelectors) {
    passwordField = document.querySelector(sel);
    if (passwordField) break;
  }

  if (emailField) {
    setNativeValue(emailField, email);
  }
  if (passwordField) {
    setNativeValue(passwordField, password);
  }

  if (emailField && passwordField) {
    const form = passwordField.closest('form');
    if (form) {
      const submitBtn = form.querySelector(
        'button[type="submit"], input[type="submit"], button:not([type])'
      );
      if (submitBtn) {
        setTimeout(() => submitBtn.click(), 200);
      }
    }
  }
}
