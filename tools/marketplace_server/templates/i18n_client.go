package templates

// I18nJS is the client-side i18n JavaScript that should be included in all pages.
// It fetches translations from the server and replaces text content based on data-i18n attributes.
// It also provides a language switcher UI.
const I18nJS = `
<script>
(function(){
  var _t = {};
  var _lang = (document.cookie.match(/(?:^|;\s*)lang=([^;]*)/)||[])[1] || '';
  var _defaultLang = (document.querySelector('meta[name="default-lang"]')||{}).content || '';
  if (!_lang) {
    _lang = _defaultLang || 'zh-CN';
  }

  window._i18nLang = _lang;
  window._i18n = function(key, fallback) {
    return _t[key] || fallback || key;
  };

  function applyTranslations() {
    // Translate elements with data-i18n attribute
    var els = document.querySelectorAll('[data-i18n]');
    for (var i = 0; i < els.length; i++) {
      var key = els[i].getAttribute('data-i18n');
      if (_t[key]) {
        if (els[i].tagName === 'INPUT' || els[i].tagName === 'TEXTAREA') {
          if (els[i].getAttribute('data-i18n-attr') === 'placeholder') {
            els[i].placeholder = _t[key];
          } else {
            els[i].value = _t[key];
          }
        } else if (els[i].tagName === 'OPTION') {
          els[i].textContent = _t[key];
        } else {
          els[i].textContent = _t[key];
        }
      }
    }
    // Translate elements with data-i18n-placeholder
    var phs = document.querySelectorAll('[data-i18n-placeholder]');
    for (var i = 0; i < phs.length; i++) {
      var key = phs[i].getAttribute('data-i18n-placeholder');
      if (_t[key]) phs[i].placeholder = _t[key];
    }
    // Translate elements with data-i18n-title
    var tls = document.querySelectorAll('[data-i18n-title]');
    for (var i = 0; i < tls.length; i++) {
      var key = tls[i].getAttribute('data-i18n-title');
      if (_t[key]) tls[i].title = _t[key];
    }
    // Update html lang attribute
    document.documentElement.lang = _lang === 'en-US' ? 'en' : 'zh-CN';
  }

  // Observe DOM for dynamically added elements and translate them
  var _observer = null;
  var _debounceTimer = null;
  function startObserver() {
    if (_observer || typeof MutationObserver === 'undefined') return;
    _observer = new MutationObserver(function(mutations) {
      var needsTranslation = false;
      for (var m = 0; m < mutations.length; m++) {
        if (mutations[m].addedNodes.length > 0) { needsTranslation = true; break; }
      }
      if (needsTranslation) {
        if (_debounceTimer) clearTimeout(_debounceTimer);
        _debounceTimer = setTimeout(applyTranslations, 50);
      }
    });
    _observer.observe(document.body, { childList: true, subtree: true });
  }

  function addLangSwitcher() {
    var switcher = document.createElement('div');
    switcher.id = 'lang-switcher';
    switcher.style.cssText = 'display:inline-flex;gap:2px;background:rgba(255,255,255,0.8);border:1px solid #e2e8f0;border-radius:8px;padding:2px;font-size:12px;margin-left:10px;vertical-align:middle;backdrop-filter:blur(4px);';
    var currentPath = encodeURIComponent(window.location.pathname + window.location.search);
    var zhBtn = document.createElement('a');
    zhBtn.href = '/set-lang?lang=zh-CN&redirect=' + currentPath;
    zhBtn.textContent = '中文';
    zhBtn.style.cssText = 'padding:3px 10px;border-radius:6px;text-decoration:none;font-weight:500;transition:all 0.2s;' + (_lang === 'zh-CN' ? 'background:#6366f1;color:#fff;' : 'color:#64748b;');
    var enBtn = document.createElement('a');
    enBtn.href = '/set-lang?lang=en-US&redirect=' + currentPath;
    enBtn.textContent = 'EN';
    enBtn.style.cssText = 'padding:3px 10px;border-radius:6px;text-decoration:none;font-weight:500;transition:all 0.2s;' + (_lang === 'en-US' ? 'background:#6366f1;color:#fff;' : 'color:#64748b;');
    switcher.appendChild(zhBtn);
    switcher.appendChild(enBtn);
    // Insert into nav alongside download buttons (not inside logo-brand which has vertical layout)
    // 1. Homepage: append into .nav-center (after .hero-buttons)
    var navCenter = document.querySelector('.nav-center');
    if (navCenter) {
      navCenter.appendChild(switcher);
    } else {
      // 2. Storefront pages: insert after #sfDlBtn inside .nav-actions
      var sfDl = document.getElementById('sfDlBtn');
      if (sfDl) {
        sfDl.parentNode.insertBefore(switcher, sfDl.nextSibling);
      } else {
        // 3. Admin panel: insert into topbar next to user info
        var topbarUser = document.querySelector('.topbar-user');
        if (topbarUser) {
          switcher.style.marginLeft = '0';
          switcher.style.marginRight = '12px';
          topbarUser.insertBefore(switcher, topbarUser.firstChild);
        } else {
          // 4. Fallback: prepend to nav-actions
          var navActions = document.querySelector('.nav-actions');
          if (navActions) {
            navActions.insertBefore(switcher, navActions.firstChild);
          } else {
            document.body.appendChild(switcher);
          }
        }
      }
    }
  }

  // Fetch translations and apply
  fetch('/api/translations')
    .then(function(r) { return r.json(); })
    .then(function(data) {
      _t = data.translations || {};
      _lang = data.lang || _lang;
      window._i18nLang = _lang;
      applyTranslations();
      addLangSwitcher();
      startObserver();
    })
    .catch(function(err) {
      console.warn('i18n: failed to load translations', err);
      // Fallback: add the switcher even if translations fail
      addLangSwitcher();
    });
})();
</script>
`
