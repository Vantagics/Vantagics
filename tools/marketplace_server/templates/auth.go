package templates

import "html/template"

// LoginTmpl is the parsed login page template.
var LoginTmpl = template.Must(template.New("login").Funcs(BaseFuncMap).Parse(loginHTML))

// SetupTmpl is the parsed first-time setup page template.
var SetupTmpl = template.Must(template.New("setup").Funcs(BaseFuncMap).Parse(setupHTML))

const authCSS = `
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f0f2f5; min-height: 100vh; display: flex; align-items: center; justify-content: center; }
.auth-card { background: #fff; border-radius: 12px; padding: 40px; width: 400px; max-width: 90%; box-shadow: 0 2px 12px rgba(0,0,0,0.1); }
.auth-card h1 { font-size: 22px; color: #1a1a2e; margin-bottom: 8px; text-align: center; }
.auth-card .subtitle { font-size: 14px; color: #666; text-align: center; margin-bottom: 24px; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; color: #555; margin-bottom: 6px; font-weight: 500; }
.form-group input { width: 100%; padding: 10px 12px; border: 1px solid #ddd; border-radius: 6px; font-size: 14px; transition: border-color 0.2s; }
.form-group input:focus { outline: none; border-color: #4361ee; }
.captcha-row { display: flex; gap: 10px; align-items: flex-end; flex-wrap: wrap; }
.captcha-row input { flex: 1; min-width: 0; }
@media (max-width: 480px) { .captcha-row input { flex: 1 1 100%; } }
.captcha-img { height: 42px; border-radius: 6px; cursor: pointer; border: 1px solid #ddd; }
.btn-submit { width: 100%; padding: 11px; background: #4361ee; color: #fff; border: none; border-radius: 6px; font-size: 15px; cursor: pointer; margin-top: 8px; transition: background 0.2s; }
.btn-submit:hover { background: #3451d1; }
.error-msg { background: #fef2f2; color: #dc2626; padding: 10px 14px; border-radius: 6px; font-size: 13px; margin-bottom: 16px; border: 1px solid #fecaca; }
.logo { text-align: center; margin-bottom: 20px; }
.logo img { width: 48px; height: 48px; border-radius: 12px; }
`

const loginHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title data-i18n="admin_login_title">管理员登录 - 市场管理后台</title>
    <style>` + authCSS + `</style>
</head>
<body>
<div class="auth-card">
    <div class="logo"><img src="{{logoURL}}" alt=""></div>
    <h1 data-i18n="admin_panel">市场管理后台</h1>
    <p class="subtitle" data-i18n="enter_admin_credentials">请输入管理员凭据登录</p>
    {{if .Error}}<div class="error-msg">{{.Error}}</div>{{end}}
    <form method="POST" action="/admin/login">
        <input type="hidden" name="captcha_id" id="captcha_id" value="{{.CaptchaID}}" />
        <div class="form-group">
            <label for="username" data-i18n="username">用户名</label>
            <input type="text" id="username" name="username" value="{{.Username}}" required autocomplete="username" />
        </div>
        <div class="form-group">
            <label for="password" data-i18n="password">密码</label>
            <input type="password" id="password" name="password" required autocomplete="current-password" />
        </div>
        <div class="form-group">
            <label for="captcha" data-i18n="captcha">验证码</label>
            <div class="captcha-row">
                <input type="text" id="captcha" name="captcha" required maxlength="4" data-i18n-placeholder="enter_captcha" placeholder="输入验证码" autocomplete="off" />
                <img class="captcha-img" id="captcha-img" src="/admin/captcha?id={{.CaptchaID}}" alt="验证码" title="点击刷新" onclick="refreshCaptcha()" />
            </div>
        </div>
        <button type="submit" class="btn-submit" data-i18n="login">登 录</button>
    </form>
</div>
<script>
function refreshCaptcha() {
    fetch('/admin/captcha/refresh').then(function(r){return r.json();}).then(function(d){
        document.getElementById('captcha_id').value = d.captcha_id;
        document.getElementById('captcha-img').src = '/admin/captcha?id=' + d.captcha_id;
        document.getElementById('captcha').value = '';
    });
}
</script>
` + I18nJS + `
</body>
</html>`

const setupHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title data-i18n="initial_setup">初始设置 - 市场管理后台</title>
    <style>` + authCSS + `
    .setup-note { background: #eff6ff; color: #1e40af; padding: 10px 14px; border-radius: 6px; font-size: 13px; margin-bottom: 16px; border: 1px solid #bfdbfe; }
    </style>
</head>
<body>
<div class="auth-card">
    <div class="logo"><img src="{{logoURL}}" alt=""></div>
    <h1 data-i18n="initial_setup">初始设置</h1>
    <p class="subtitle" data-i18n="first_time_setup">首次使用，请设置管理员账号</p>
    <div class="setup-note" data-i18n="setup_note">请牢记您设置的用户名和密码，这将是管理后台的唯一登录凭据。</div>
    {{if .Error}}<div class="error-msg">{{.Error}}</div>{{end}}
    <form method="POST" action="/admin/setup">
        <input type="hidden" name="captcha_id" id="captcha_id" value="{{.CaptchaID}}" />
        <div class="form-group">
            <label for="username" data-i18n="admin_username_min3">管理员用户名（至少3个字符）</label>
            <input type="text" id="username" name="username" value="{{.Username}}" required minlength="3" autocomplete="username" />
        </div>
        <div class="form-group">
            <label for="password" data-i18n="password_min6">密码（至少6个字符）</label>
            <input type="password" id="password" name="password" required minlength="6" autocomplete="new-password" />
        </div>
        <div class="form-group">
            <label for="password2" data-i18n="confirm_password">确认密码</label>
            <input type="password" id="password2" name="password2" required minlength="6" autocomplete="new-password" />
        </div>
        <div class="form-group">
            <label for="captcha" data-i18n="captcha">验证码</label>
            <div class="captcha-row">
                <input type="text" id="captcha" name="captcha" required maxlength="4" data-i18n-placeholder="enter_captcha" placeholder="输入验证码" autocomplete="off" />
                <img class="captcha-img" id="captcha-img" src="/admin/captcha?id={{.CaptchaID}}" alt="验证码" title="点击刷新" onclick="refreshCaptcha()" />
            </div>
        </div>
        <button type="submit" class="btn-submit" data-i18n="create_admin">创建管理员账号</button>
    </form>
</div>
<script>
function refreshCaptcha() {
    fetch('/admin/captcha/refresh').then(function(r){return r.json();}).then(function(d){
        document.getElementById('captcha_id').value = d.captcha_id;
        document.getElementById('captcha-img').src = '/admin/captcha?id=' + d.captcha_id;
        document.getElementById('captcha').value = '';
    });
}
</script>
` + I18nJS + `
</body>
</html>`
