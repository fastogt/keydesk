(() => {
  let matchedAccountId = null;
  let bannerShown = false;

  async function checkCurrentURL() {
    const url = window.location.hostname;
    if (!url) return;

    chrome.runtime.sendMessage(
      { action: 'matchURL', url },
      (response) => {
        if (chrome.runtime.lastError) return;
        if (!response?.success || !response.match?.match) return;

        matchedAccountId = response.match.account_id;
        if (!bannerShown) {
          showLoginBanner();
        }
      }
    );
  }

  function showLoginBanner() {
    bannerShown = true;

    const banner = document.createElement('div');
    banner.id = 'keydesk-banner';
    banner.innerHTML = `
      <div style="
        position: fixed; bottom: 20px; right: 20px; z-index: 999999;
        background: #1e1b4b; color: white; padding: 12px 20px;
        border-radius: 10px; box-shadow: 0 4px 20px rgba(0,0,0,0.3);
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        font-size: 14px; display: flex; align-items: center; gap: 12px;
        animation: keydesk-slide-in 0.3s ease;
      ">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M15.75 5.25a3 3 0 0 1 3 3m3 0a6 6 0 0 1-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1 1 21.75 8.25Z"/>
        </svg>
        <span>KeyDesk account detected</span>
        <button id="keydesk-login-btn" style="
          background: #4f46e5; border: none; color: white;
          padding: 6px 16px; border-radius: 6px; cursor: pointer;
          font-size: 13px; font-weight: 500;
        ">Login</button>
        <button id="keydesk-dismiss-btn" style="
          background: none; border: none; color: rgba(255,255,255,0.6);
          cursor: pointer; font-size: 18px; padding: 0 4px;
        ">&times;</button>
      </div>
    `;

    const style = document.createElement('style');
    style.textContent = `
      @keyframes keydesk-slide-in {
        from { transform: translateY(20px); opacity: 0; }
        to { transform: translateY(0); opacity: 1; }
      }
    `;
    document.head.appendChild(style);
    document.body.appendChild(banner);

    document.getElementById('keydesk-login-btn')?.addEventListener('click', () => {
      performAutofill();
    });

    document.getElementById('keydesk-dismiss-btn')?.addEventListener('click', () => {
      banner.remove();
    });

    setTimeout(() => {
      if (banner.parentNode) banner.remove();
    }, 15000);
  }

  function performAutofill() {
    if (!matchedAccountId) return;

    chrome.runtime.sendMessage(
      { action: 'getCredentials', accountId: matchedAccountId },
      (response) => {
        if (chrome.runtime.lastError) return;
        if (!response?.success) return;

        const { login_email, password } = response.credentials;
        fillForm(login_email, password);

        chrome.runtime.sendMessage({
          action: 'logAudit',
          auditAction: 'autofill',
          accountId: matchedAccountId,
          details: 'Content script auto-filled on ' + window.location.hostname,
        });

        const banner = document.getElementById('keydesk-banner');
        if (banner) banner.remove();
      }
    );
  }

  function setNativeValue(el, value) {
    const setter = Object.getOwnPropertyDescriptor(
      window.HTMLInputElement.prototype, 'value'
    ).set;
    setter.call(el, value);
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }

  function fillForm(email, password) {
    const emailSelectors = [
      'input[type="email"]',
      'input[name="email"]',
      'input[name="username"]',
      'input[name="login"]',
      'input[name="login_field"]',
      'input[autocomplete="username"]',
      'input[autocomplete="email"]',
      'input[id*="email" i]',
      'input[id*="user" i]',
      'input[id*="login" i]',
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

    if (emailField) setNativeValue(emailField, email);
    if (passwordField) setNativeValue(passwordField, password);

    if (passwordField) {
      const form = passwordField.closest('form');
      if (form) {
        const submitBtn = form.querySelector(
          'button[type="submit"], input[type="submit"], button:not([type])'
        );
        if (submitBtn) {
          setTimeout(() => submitBtn.click(), 300);
        }
      }
    }
  }

  // Google multi-step login handler
  function handleGoogleLogin(email, password) {
    const identifierInput = document.querySelector('input[type="email"], input[name="identifier"]');
    if (identifierInput && !document.querySelector('input[type="password"]')) {
      setNativeValue(identifierInput, email);
      const nextBtn = document.querySelector('#identifierNext button, button[jsname="LgbsSe"]');
      if (nextBtn) {
        nextBtn.click();
        setTimeout(() => {
          const pwdField = document.querySelector('input[type="password"]');
          if (pwdField) {
            setNativeValue(pwdField, password);
            const pwdNext = document.querySelector('#passwordNext button, button[jsname="LgbsSe"]');
            if (pwdNext) setTimeout(() => pwdNext.click(), 200);
          }
        }, 1500);
      }
      return true;
    }
    return false;
  }

  chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
    if (msg.action === 'fill') {
      const isGoogle = window.location.hostname.includes('google.com') ||
                       window.location.hostname.includes('accounts.google');
      if (isGoogle) {
        handleGoogleLogin(msg.login, msg.password);
      } else {
        fillForm(msg.login, msg.password);
      }

      if (msg.totp) {
        setTimeout(() => {
          const totpField = document.querySelector(
            'input[name="totp"], input[name="otp"], input[name="code"], ' +
            'input[autocomplete="one-time-code"], input[id*="totp" i], input[id*="otp" i], input[id*="2fa" i]'
          );
          if (totpField) {
            setNativeValue(totpField, msg.totp);
            const form = totpField.closest('form');
            if (form) {
              const btn = form.querySelector('button[type="submit"], input[type="submit"]');
              if (btn) setTimeout(() => btn.click(), 200);
            }
          }
        }, 2000);
      }

      sendResponse({ success: true });
    }
  });

  if (document.readyState === 'complete') {
    checkCurrentURL();
  } else {
    window.addEventListener('load', checkCurrentURL);
  }
})();
