package templates

import "html/template"

// UserLoginTmpl is the parsed user login page template.
var UserLoginTmpl = template.Must(template.New("user_login").Parse(userLoginHTML))

const userLoginHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>用户登录 - 快捷分析包市场</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #f0f4ff 0%, #e8f5e9 50%, #f3e8ff 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .auth-card {
            background: #fff;
            border-radius: 16px;
            padding: 40px;
            width: 420px;
            max-width: 90%;
            box-shadow: 0 4px 24px rgba(0,0,0,0.08);
            border: 1px solid #e2e8f0;
        }
        .logo { text-align: center; margin-bottom: 20px; font-size: 36px; }
        .auth-card h1 {
            font-size: 22px;
            color: #1e293b;
            margin-bottom: 8px;
            text-align: center;
            font-weight: 700;
        }
        .auth-card .subtitle {
            font-size: 14px;
            color: #64748b;
            text-align: center;
            margin-bottom: 28px;
        }
        .form-group { margin-bottom: 18px; }
        .form-group label {
            display: block;
            font-size: 13px;
            color: #475569;
            margin-bottom: 6px;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 10px 12px;
            border: 1px solid #cbd5e1;
            border-radius: 8px;
            font-size: 14px;
            color: #1e293b;
            background: #f8fafc;
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #6366f1;
            box-shadow: 0 0 0 3px rgba(99,102,241,0.1);
            background: #fff;
        }
        .form-group input::placeholder { color: #94a3b8; }
        .captcha-row { display: flex; gap: 10px; align-items: flex-end; }
        .captcha-row input { flex: 1; }
        .captcha-img {
            height: 42px;
            border-radius: 8px;
            cursor: pointer;
            border: 1px solid #cbd5e1;
            background: #fff;
        }
        .captcha-refresh {
            background: none;
            border: 1px solid #cbd5e1;
            border-radius: 8px;
            color: #64748b;
            cursor: pointer;
            padding: 0 10px;
            height: 42px;
            font-size: 18px;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .captcha-refresh:hover { border-color: #6366f1; color: #6366f1; }
        .btn-submit {
            width: 100%;
            padding: 11px;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            color: #fff;
            border: none;
            border-radius: 8px;
            font-size: 15px;
            font-weight: 500;
            cursor: pointer;
            margin-top: 8px;
            transition: opacity 0.2s;
        }
        .btn-submit:hover { opacity: 0.9; }
        .error-msg {
            background: #fef2f2;
            color: #dc2626;
            padding: 10px 14px;
            border-radius: 8px;
            font-size: 13px;
            margin-bottom: 16px;
            border: 1px solid #fecaca;
        }
        .auth-footer {
            text-align: center;
            margin-top: 20px;
            padding-top: 16px;
            border-top: 1px solid #e2e8f0;
        }
        .auth-footer a {
            color: #6366f1;
            text-decoration: none;
            font-size: 14px;
            transition: color 0.2s;
        }
        .auth-footer a:hover { color: #4f46e5; }
    </style>
</head>
<body>
<div class="auth-card">
    <div class="logo">📦</div>
    <h1>快捷分析包市场</h1>
    <p class="subtitle">请输入用户名和密码登录</p>
    {{if .Error}}<div class="error-msg">{{.Error}}</div>{{end}}
    <form method="POST" action="/user/login">
        <input type="hidden" name="captcha_id" id="captcha_id" value="{{.CaptchaID}}" />
        <input type="hidden" name="redirect" value="{{.Redirect}}" />
        <div class="form-group">
            <label for="username">用户名</label>
            <input type="text" id="username" name="username" required autocomplete="username" placeholder="请输入用户名" />
        </div>
        <div class="form-group">
            <label for="password">密码</label>
            <input type="password" id="password" name="password" required autocomplete="current-password" placeholder="请输入密码" />
        </div>
        <div class="form-group">
            <label for="captcha_answer">验证码</label>
            <div class="captcha-row">
                <input type="text" id="captcha_answer" name="captcha_answer" required placeholder="输入计算结果" autocomplete="off" />
                <img class="captcha-img" id="captcha-img" src="/user/captcha?id={{.CaptchaID}}" alt="验证码" title="点击刷新" onclick="refreshCaptcha()" />
                <button type="button" class="captcha-refresh" onclick="refreshCaptcha()" title="刷新验证码">↻</button>
            </div>
        </div>
        <button type="submit" class="btn-submit">登 录</button>
    </form>
    <div class="auth-footer">
        <a href="/user/register{{if .Redirect}}?redirect={{.Redirect}}{{end}}">没有账号？绑定用户</a>
    </div>
</div>
<script>
function refreshCaptcha() {
    fetch('/user/captcha/refresh').then(function(r){return r.json();}).then(function(d){
        document.getElementById('captcha_id').value = d.captcha_id;
        document.getElementById('captcha-img').src = '/user/captcha?id=' + d.captcha_id;
        document.getElementById('captcha_answer').value = '';
    });
}
</script>
</body>
</html>`
