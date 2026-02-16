package templates

import "html/template"

// UserRegisterTmpl is the parsed user registration page template.
var UserRegisterTmpl = template.Must(template.New("user_register").Parse(userRegisterHTML))

const userRegisterHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>绑定注册 - 快捷分析包市场</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #0f172a;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .auth-card {
            background: #1e293b;
            border-radius: 12px;
            padding: 40px;
            width: 420px;
            max-width: 90%;
            box-shadow: 0 4px 24px rgba(0,0,0,0.3);
            border: 1px solid rgba(255,255,255,0.06);
        }
        .logo { text-align: center; margin-bottom: 20px; font-size: 36px; }
        .auth-card h1 {
            font-size: 22px;
            color: #f1f5f9;
            margin-bottom: 8px;
            text-align: center;
            font-weight: 700;
        }
        .auth-card .subtitle {
            font-size: 14px;
            color: #94a3b8;
            text-align: center;
            margin-bottom: 28px;
        }
        .form-group { margin-bottom: 18px; }
        .form-group label {
            display: block;
            font-size: 13px;
            color: #cbd5e1;
            margin-bottom: 6px;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 10px 12px;
            border: 1px solid #334155;
            border-radius: 6px;
            font-size: 14px;
            color: #f1f5f9;
            background: #0f172a;
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #3b82f6;
            box-shadow: 0 0 0 3px rgba(59,130,246,0.15);
        }
        .form-group input::placeholder { color: #475569; }
        .captcha-row { display: flex; gap: 10px; align-items: flex-end; }
        .captcha-row input { flex: 1; }
        .captcha-img {
            height: 42px;
            border-radius: 6px;
            cursor: pointer;
            border: 1px solid #334155;
            background: #fff;
        }
        .captcha-refresh {
            background: none;
            border: 1px solid #334155;
            border-radius: 6px;
            color: #94a3b8;
            cursor: pointer;
            padding: 0 10px;
            height: 42px;
            font-size: 18px;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .captcha-refresh:hover { border-color: #3b82f6; color: #3b82f6; }
        .btn-submit {
            width: 100%;
            padding: 11px;
            background: #3b82f6;
            color: #fff;
            border: none;
            border-radius: 6px;
            font-size: 15px;
            font-weight: 500;
            cursor: pointer;
            margin-top: 8px;
            transition: background 0.2s;
        }
        .btn-submit:hover { background: #2563eb; }
        .error-msg {
            background: rgba(239,68,68,0.1);
            color: #fca5a5;
            padding: 10px 14px;
            border-radius: 6px;
            font-size: 13px;
            margin-bottom: 16px;
            border: 1px solid rgba(239,68,68,0.2);
        }
        .client-error {
            color: #fca5a5;
            font-size: 12px;
            margin-top: 4px;
            display: none;
        }
        .auth-footer {
            text-align: center;
            margin-top: 20px;
            padding-top: 16px;
            border-top: 1px solid rgba(255,255,255,0.06);
        }
        .auth-footer a {
            color: #3b82f6;
            text-decoration: none;
            font-size: 14px;
            transition: color 0.2s;
        }
        .auth-footer a:hover { color: #60a5fa; }
    </style>
</head>
<body>
<div class="auth-card">
    <div class="logo">📦</div>
    <h1>绑定注册</h1>
    <p class="subtitle">通过邮箱和产品序列号创建账号</p>
    {{if .Error}}<div class="error-msg">{{.Error}}</div>{{end}}
    <form method="POST" action="/user/register" onsubmit="return validateForm()">
        <input type="hidden" name="captcha_id" id="captcha_id" value="{{.CaptchaID}}" />
        <div class="form-group">
            <label for="email">邮箱</label>
            <input type="email" id="email" name="email" required autocomplete="email" placeholder="请输入邮箱地址" />
        </div>
        <div class="form-group">
            <label for="sn">产品序列号 (SN)</label>
            <input type="text" id="sn" name="sn" required autocomplete="off" placeholder="请输入产品序列号" />
        </div>
        <div class="form-group">
            <label for="password">新密码</label>
            <input type="password" id="password" name="password" required autocomplete="new-password" placeholder="至少6个字符" />
            <div class="client-error" id="password-error"></div>
        </div>
        <div class="form-group">
            <label for="password2">确认密码</label>
            <input type="password" id="password2" name="password2" required autocomplete="new-password" placeholder="再次输入密码" />
            <div class="client-error" id="password2-error"></div>
        </div>
        <div class="form-group">
            <label for="captcha_answer">验证码</label>
            <div class="captcha-row">
                <input type="text" id="captcha_answer" name="captcha_answer" required placeholder="输入计算结果" autocomplete="off" />
                <img class="captcha-img" id="captcha-img" src="/user/captcha?id={{.CaptchaID}}" alt="验证码" title="点击刷新" onclick="refreshCaptcha()" />
                <button type="button" class="captcha-refresh" onclick="refreshCaptcha()" title="刷新验证码">↻</button>
            </div>
        </div>
        <button type="submit" class="btn-submit">注 册</button>
    </form>
    <div class="auth-footer">
        <a href="/user/login">已有账号？去登录</a>
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
function validateForm() {
    var pw = document.getElementById('password').value;
    var pw2 = document.getElementById('password2').value;
    var pwErr = document.getElementById('password-error');
    var pw2Err = document.getElementById('password2-error');
    pwErr.style.display = 'none';
    pw2Err.style.display = 'none';
    if (pw.length < 6) {
        pwErr.textContent = '密码至少6个字符';
        pwErr.style.display = 'block';
        return false;
    }
    if (pw !== pw2) {
        pw2Err.textContent = '两次密码不一致';
        pw2Err.style.display = 'block';
        return false;
    }
    return true;
}
</script>
</body>
</html>`
