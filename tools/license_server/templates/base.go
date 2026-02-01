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

// BaseHTML contains the common HTML structure
const BaseHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>序列号管理系统</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        .tab-panel { display: none; }
        .tab-panel.active { display: block; }
    </style>
</head>
<body class="bg-slate-100 min-h-screen">
    <div class="max-w-7xl mx-auto p-6">
        <div class="flex justify-between items-center mb-6">
            <h1 class="text-2xl font-bold text-slate-800">序列号管理系统</h1>
            <div class="flex items-center gap-4">
                <span class="text-sm text-slate-600">欢迎, <span id="username-display">{{.Username}}</span></span>
                <a href="/logout" class="text-sm text-red-600 hover:underline">退出</a>
            </div>
        </div>
        
        <div class="flex gap-2 mb-6 flex-wrap">
            <button onclick="showTab('licenses')" id="tab-licenses" class="tab-btn px-4 py-2 rounded-lg bg-blue-600 text-white">序列号管理</button>
            <button onclick="showTab('email-records')" id="tab-email-records" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">邮箱申请记录</button>
            <button onclick="showTab('email-filter')" id="tab-email-filter" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">邮箱过滤</button>
            <button onclick="showTab('license-groups')" id="tab-license-groups" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">序列号分组</button>
            <button onclick="showTab('llm')" id="tab-llm" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">LLM配置</button>
            <button onclick="showTab('search')" id="tab-search" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">搜索引擎配置</button>
            <button onclick="showTab('settings')" id="tab-settings" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">系统设置</button>
        </div>
        
        {{.Content}}
    </div>
    
    <div id="modal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white rounded-xl shadow-xl w-[500px] max-h-[80vh] overflow-auto">
            <div id="modal-content"></div>
        </div>
    </div>
    
    <script>
    {{.Scripts}}
    </script>
</body>
</html>`
