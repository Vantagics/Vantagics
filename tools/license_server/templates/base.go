package templates

// LoginHTML is the login page template with math captcha
const LoginHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>VantageData License Server - 登录</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gradient-to-br from-blue-50 to-indigo-100 min-h-screen flex items-center justify-center">
    <div class="bg-white p-8 rounded-xl shadow-lg w-96">
        <div class="text-center mb-6">
            <h1 class="text-2xl font-bold text-slate-800">🔐 License Server</h1>
            <p class="text-sm text-slate-500 mt-1">VantageData 授权管理系统</p>
        </div>
        {{if .Error}}
        <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-2 rounded-lg mb-4 text-sm">
            {{.Error}}
        </div>
        {{end}}
        <form method="POST" action="/login">
            <input type="hidden" name="captcha_id" value="{{.CaptchaID}}">
            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1">用户名</label>
                <input type="text" name="username" required
                    class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                    placeholder="请输入用户名">
            </div>
            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1">密码</label>
                <input type="password" name="password" required
                    class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                    placeholder="请输入密码">
            </div>
            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1">验证码</label>
                <div class="flex items-center gap-3 mb-2">
                    <img src="data:image/png;base64,{{.CaptchaImage}}" alt="验证码" class="h-10 rounded border border-slate-200">
                    <button type="button" onclick="location.reload()" class="text-sm text-blue-600 hover:text-blue-800 whitespace-nowrap">换一题</button>
                </div>
                <input type="text" name="captcha" required
                    class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                    placeholder="请输入计算结果">
            </div>
            <button type="submit"
                class="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700 transition-colors font-medium">
                登录
            </button>
        </form>
        <p class="text-xs text-slate-400 text-center mt-4">管理端口: {{.ManagePort}} | 认证端口: {{.AuthPort}}</p>
    </div>
</body>
</html>`

// BaseHTML contains the common HTML structure with sidebar navigation layout
const BaseHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>序列号管理系统</title>
    <style>
        /* ===== Reset & Base ===== */
        *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; background: #f1f5f9; color: #334155; min-height: 100vh; }
        a { color: inherit; text-decoration: none; }

        /* ===== Layout ===== */
        .layout { display: flex; min-height: 100vh; }

        /* ===== Sidebar ===== */
        .sidebar {
            width: 240px;
            background: #1e293b;
            color: #cbd5e1;
            display: flex;
            flex-direction: column;
            position: fixed;
            top: 0;
            left: 0;
            bottom: 0;
            z-index: 40;
            overflow-y: auto;
        }
        .sidebar-brand {
            padding: 24px 20px 20px;
            border-bottom: 1px solid #334155;
        }
        .sidebar-brand h1 {
            font-size: 18px;
            font-weight: 700;
            color: #f1f5f9;
            margin-bottom: 4px;
        }
        .sidebar-brand p {
            font-size: 12px;
            color: #94a3b8;
        }
        .sidebar-nav {
            flex: 1;
            padding: 12px 0;
        }
        .sidebar-nav a {
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 10px 20px;
            font-size: 14px;
            color: #cbd5e1;
            transition: background 0.15s, color 0.15s;
            cursor: pointer;
        }
        .sidebar-nav a:hover {
            background: #334155;
            color: #f1f5f9;
        }
        .sidebar-nav a.active {
            background: #3b82f6;
            color: #ffffff;
        }
        .sidebar-nav a .nav-icon {
            width: 20px;
            text-align: center;
            flex-shrink: 0;
        }
        .sidebar-footer {
            padding: 16px 20px;
            border-top: 1px solid #334155;
        }
        .sidebar-footer a {
            display: flex;
            align-items: center;
            gap: 10px;
            font-size: 14px;
            color: #f87171;
            transition: color 0.15s;
        }
        .sidebar-footer a:hover {
            color: #fca5a5;
        }

        /* ===== Main Area ===== */
        .main-area {
            flex: 1;
            margin-left: 240px;
            display: flex;
            flex-direction: column;
            min-height: 100vh;
        }

        /* ===== Topbar ===== */
        .topbar {
            background: #ffffff;
            height: 56px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 28px;
            border-bottom: 1px solid #e2e8f0;
            position: sticky;
            top: 0;
            z-index: 30;
        }
        .topbar-title {
            font-size: 17px;
            font-weight: 600;
            color: #1e293b;
        }
        .topbar-user {
            font-size: 13px;
            color: #64748b;
        }
        .topbar-user strong {
            color: #334155;
            font-weight: 600;
        }

        /* ===== Content ===== */
        .content-area {
            flex: 1;
            padding: 24px 28px;
        }

        /* ===== Section (replaces tab-panel) ===== */
        .section { display: none; }
        .section.active { display: block; }

        /* ===== Card ===== */
        .card {
            background: #ffffff;
            border-radius: 10px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04);
            padding: 24px;
            margin-bottom: 20px;
        }
        .card-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
        }
        .card-title {
            font-size: 16px;
            font-weight: 600;
            color: #1e293b;
        }

        /* ===== Table ===== */
        table {
            width: 100%;
            border-collapse: collapse;
        }
        thead th {
            background: #f8fafc;
            color: #475569;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            padding: 10px 14px;
            text-align: left;
            border-bottom: 2px solid #e2e8f0;
        }
        tbody td {
            padding: 10px 14px;
            font-size: 14px;
            border-bottom: 1px solid #f1f5f9;
            color: #334155;
        }
        tbody tr:hover {
            background: #f8fafc;
        }

        /* ===== Buttons ===== */
        .btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            gap: 6px;
            padding: 7px 16px;
            font-size: 13px;
            font-weight: 500;
            border-radius: 6px;
            border: none;
            cursor: pointer;
            transition: background 0.15s, box-shadow 0.15s;
            line-height: 1.4;
            white-space: nowrap;
        }
        .btn-primary { background: #3b82f6; color: #fff; }
        .btn-primary:hover { background: #2563eb; }
        .btn-success { background: #22c55e; color: #fff; }
        .btn-success:hover { background: #16a34a; }
        .btn-danger { background: #ef4444; color: #fff; }
        .btn-danger:hover { background: #dc2626; }
        .btn-warning { background: #f97316; color: #fff; }
        .btn-warning:hover { background: #ea580c; }
        .btn-secondary { background: #e2e8f0; color: #475569; }
        .btn-secondary:hover { background: #cbd5e1; }
        .btn-sm { padding: 4px 10px; font-size: 12px; }

        /* ===== Forms ===== */
        .form-label {
            display: block;
            font-size: 13px;
            font-weight: 500;
            color: #475569;
            margin-bottom: 4px;
        }
        .form-input, .form-select, .form-textarea {
            width: 100%;
            padding: 8px 12px;
            font-size: 14px;
            border: 1px solid #cbd5e1;
            border-radius: 6px;
            background: #fff;
            color: #334155;
            outline: none;
            transition: border-color 0.15s, box-shadow 0.15s;
        }
        .form-input:focus, .form-select:focus, .form-textarea:focus {
            border-color: #3b82f6;
            box-shadow: 0 0 0 3px rgba(59,130,246,0.15);
        }
        .form-textarea { resize: vertical; min-height: 80px; }

        /* ===== Badges ===== */
        .badge {
            display: inline-block;
            padding: 2px 8px;
            font-size: 11px;
            font-weight: 600;
            border-radius: 9999px;
            line-height: 1.6;
        }
        .badge-success { background: #dcfce7; color: #166534; }
        .badge-danger { background: #fee2e2; color: #991b1b; }
        .badge-warning { background: #fef3c7; color: #92400e; }
        .badge-info { background: #dbeafe; color: #1e40af; }
        .badge-secondary { background: #f1f5f9; color: #475569; }

        /* ===== Modal ===== */
        .modal-overlay {
            position: fixed;
            inset: 0;
            background: rgba(0,0,0,0.45);
            display: none;
            align-items: center;
            justify-content: center;
            z-index: 50;
        }
        .modal-overlay.show {
            display: flex;
        }
        .modal-card {
            background: #ffffff;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.15);
            width: 500px;
            max-height: 80vh;
            overflow: auto;
        }
        .modal-header {
            padding: 18px 24px;
            border-bottom: 1px solid #e2e8f0;
            font-size: 16px;
            font-weight: 600;
            color: #1e293b;
        }
        .modal-body {
            padding: 20px 24px;
        }
        .modal-footer {
            padding: 14px 24px;
            border-top: 1px solid #e2e8f0;
            display: flex;
            justify-content: flex-end;
            gap: 8px;
        }

        /* ===== Pagination ===== */
        .pagination {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 4px;
            margin-top: 16px;
        }
        .pagination button {
            padding: 6px 12px;
            font-size: 13px;
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            background: #fff;
            color: #475569;
            cursor: pointer;
        }
        .pagination button:hover { background: #f1f5f9; }
        .pagination button.active { background: #3b82f6; color: #fff; border-color: #3b82f6; }
        .pagination button:disabled { opacity: 0.5; cursor: not-allowed; }

        /* ===== Utility ===== */
        .flex { display: flex; }
        .flex-wrap { flex-wrap: wrap; }
        .items-center { align-items: center; }
        .justify-between { justify-content: space-between; }
        .gap-2 { gap: 8px; }
        .gap-3 { gap: 12px; }
        .mb-2 { margin-bottom: 8px; }
        .mb-4 { margin-bottom: 16px; }
        .mt-2 { margin-top: 8px; }
        .mt-4 { margin-top: 16px; }
        .text-sm { font-size: 13px; }
        .text-xs { font-size: 11px; }
        .text-right { text-align: right; }
        .text-center { text-align: center; }
        .text-muted { color: #94a3b8; }
        .text-danger { color: #ef4444; }
        .text-success { color: #22c55e; }
        .font-bold { font-weight: 700; }
        .font-medium { font-weight: 500; }
        .w-full { width: 100%; }
        .hidden { display: none; }
    </style>
</head>
<body>
    <div class="layout">
        <!-- Sidebar -->
        <aside class="sidebar">
            <div class="sidebar-brand">
                <h1>🔐 License Server</h1>
                <p>VantageData 授权管理</p>
            </div>
            <nav class="sidebar-nav">
                <a onclick="showSection('licenses')" id="nav-licenses" class="active">
                    <span class="nav-icon">📋</span> 序列号管理
                </a>
                <a onclick="showSection('email-records')" id="nav-email-records">
                    <span class="nav-icon">📧</span> 邮箱申请记录
                </a>
                <a onclick="showSection('email-filter')" id="nav-email-filter">
                    <span class="nav-icon">🔍</span> 邮箱过滤
                </a>
                <a onclick="showSection('product-types')" id="nav-product-types">
                    <span class="nav-icon">📦</span> 产品类型
                </a>
                <a onclick="showSection('license-groups')" id="nav-license-groups">
                    <span class="nav-icon">📁</span> 序列号分组
                </a>
                <a onclick="showSection('api-keys')" id="nav-api-keys">
                    <span class="nav-icon">🔑</span> API Key
                </a>
                <a onclick="showSection('llm')" id="nav-llm">
                    <span class="nav-icon">🤖</span> LLM配置
                </a>
                <a onclick="showSection('search')" id="nav-search">
                    <span class="nav-icon">🔎</span> 搜索引擎配置
                </a>
                <a onclick="showSection('email-notify')" id="nav-email-notify">
                    <span class="nav-icon">📬</span> 邮件发送
                </a>
                <a onclick="showSection('backup')" id="nav-backup">
                    <span class="nav-icon">💾</span> 备份恢复
                </a>
                <a onclick="showSection('settings')" id="nav-settings">
                    <span class="nav-icon">⚙️</span> 系统设置
                </a>
            </nav>
            <div class="sidebar-footer">
                <a href="/logout">
                    <span class="nav-icon">🚪</span> 退出登录
                </a>
            </div>
        </aside>

        <!-- Main Area -->
        <div class="main-area">
            <!-- Topbar -->
            <header class="topbar">
                <span class="topbar-title" id="page-title">序列号管理</span>
                <span class="topbar-user">欢迎, <strong id="username-display">{{.Username}}</strong></span>
            </header>

            <!-- Content -->
            <main class="content-area">
                {{.Content}}
            </main>
        </div>
    </div>

    <!-- Modal -->
    <div id="modal" class="modal-overlay">
        <div class="modal-card">
            <div id="modal-content"></div>
        </div>
    </div>

    <script>
    {{.Scripts}}
    </script>
</body>
</html>`
