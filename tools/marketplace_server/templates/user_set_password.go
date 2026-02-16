package templates

import "html/template"

// UserSetPasswordTmpl is the parsed set-password page template.
var UserSetPasswordTmpl = template.Must(template.New("user_set_password").Parse(userSetPasswordHTML))

const userSetPasswordHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>设置密码 - 快捷分析包市场</title>
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
        .client-error {
            color: #dc2626;
            font-size: 12px;
            margin-top: 4px;
            display: none;
        }
        .info-box {
            background: #eff6ff;
            color: #1e40af;
            padding: 10px 14px;
            border-radius: 8px;
            font-size: 13px;
            margin-bottom: 16px;
            border: 1px solid #bfdbfe;
        }
    </style>
</head>
<body>
<div class="auth-card">
    <div class="logo">📦</div>
    <h1>设置密码</h1>
    <p class="subtitle">首次登录，请为您的账户设置密码</p>
    <div class="info-box">账户邮箱：{{.Email}}</div>
    {{if .Error}}<div class="error-msg">{{.Error}}</div>{{end}}
    <form method="POST" action="/user/set-password" onsubmit="return validateForm()">
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
        <button type="submit" class="btn-submit">确认设置</button>
    </form>
</div>
<script>
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
