package templates

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
