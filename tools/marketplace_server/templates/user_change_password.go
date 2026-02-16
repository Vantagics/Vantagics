package templates

import "html/template"

// UserChangePasswordTmpl is the parsed change-password page template.
var UserChangePasswordTmpl = template.Must(template.New("user_change_password").Parse(userChangePasswordHTML))

const userChangePasswordHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>修改密码 - 快捷分析包市场</title>
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
        .success-msg {
            background: #f0fdf4;
            color: #16a34a;
            padding: 10px 14px;
            border-radius: 8px;
            font-size: 13px;
            margin-bottom: 16px;
            border: 1px solid #bbf7d0;
        }
        .client-error {
            color: #dc2626;
            font-size: 12px;
            margin-top: 4px;
            display: none;
        }
        .back-link {
            display: block;
            text-align: center;
            margin-top: 16px;
            font-size: 13px;
            color: #6366f1;
            text-decoration: none;
        }
        .back-link:hover { text-decoration: underline; }
    </style>
</head>
<body>
<div class="auth-card">
    <div class="logo">📦</div>
    <h1>修改密码</h1>
    <p class="subtitle">请输入当前密码和新密码</p>
    {{if .Error}}<div class="error-msg">{{.Error}}</div>{{end}}
    {{if .Success}}<div class="success-msg">{{.Success}}</div>{{end}}
    <form method="POST" action="/user/change-password" onsubmit="return validateForm()">
        <div class="form-group">
            <label for="current_password">当前密码</label>
            <input type="password" id="current_password" name="current_password" required autocomplete="current-password" placeholder="请输入当前密码" />
            <div class="client-error" id="current-password-error"></div>
        </div>
        <div class="form-group">
            <label for="new_password">新密码</label>
            <input type="password" id="new_password" name="new_password" required autocomplete="new-password" placeholder="至少6个字符" />
            <div class="client-error" id="new-password-error"></div>
        </div>
        <div class="form-group">
            <label for="confirm_password">确认新密码</label>
            <input type="password" id="confirm_password" name="confirm_password" required autocomplete="new-password" placeholder="再次输入新密码" />
            <div class="client-error" id="confirm-password-error"></div>
        </div>
        <button type="submit" class="btn-submit">确认修改</button>
    </form>
    <a href="/user/dashboard" class="back-link">← 返回个人中心</a>
</div>
<script>
function validateForm() {
    var curPw = document.getElementById('current_password').value;
    var newPw = document.getElementById('new_password').value;
    var confirmPw = document.getElementById('confirm_password').value;
    var curErr = document.getElementById('current-password-error');
    var newErr = document.getElementById('new-password-error');
    var confirmErr = document.getElementById('confirm-password-error');
    curErr.style.display = 'none';
    newErr.style.display = 'none';
    confirmErr.style.display = 'none';
    if (curPw.length === 0) {
        curErr.textContent = '请输入当前密码';
        curErr.style.display = 'block';
        return false;
    }
    if (newPw.length < 6) {
        newErr.textContent = '密码至少6个字符';
        newErr.style.display = 'block';
        return false;
    }
    if (newPw !== confirmPw) {
        confirmErr.textContent = '两次密码不一致';
        confirmErr.style.display = 'block';
        return false;
    }
    return true;
}
</script>
</body>
</html>`