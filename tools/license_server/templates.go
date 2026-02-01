package main

// loginHTML is the login page template with captcha
const loginHTML = `<!DOCTYPE html>
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
                <input type="text" name="username" required class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" placeholder="请输入用户名">
            </div>
            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1">密码</label>
                <input type="password" name="password" required class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" placeholder="请输入密码">
            </div>
            <div class="mb-4">
                <label class="block text-sm font-medium text-slate-700 mb-1">验证码</label>
                <div class="flex items-center gap-3 mb-2">
                    <img src="data:image/png;base64,{{.CaptchaImage}}" alt="验证码" class="h-10 rounded border">
                    <button type="button" onclick="location.reload()" class="text-sm text-blue-600 hover:text-blue-800">换一张</button>
                </div>
                <input type="text" name="captcha" required class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" placeholder="请输入图中数字">
            </div>
            <button type="submit" class="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700 transition-colors font-medium">登录</button>
        </form>
    </div>
</body>
</html>`

// dashboardHTML is the complete dashboard template
const dashboardHTML = `<!DOCTYPE html>
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

        <!-- Panel: Licenses -->
        <div id="panel-licenses" class="tab-panel active">
            <div class="bg-white rounded-xl shadow-sm p-6">
                <div class="flex justify-between items-center mb-4">
                    <h2 class="text-lg font-bold text-slate-800">序列号列表</h2>
                    <div class="flex items-center gap-2 flex-wrap">
                        <select id="license-group-filter" onchange="loadLicenses(1, licenseSearchTerm)" class="px-3 py-1.5 border rounded-lg text-sm">
                            <option value="">全部序列号组</option>
                            <option value="none">默认(无组)</option>
                        </select>
                        <select id="llm-group-filter" onchange="loadLicenses(1, licenseSearchTerm)" class="px-3 py-1.5 border rounded-lg text-sm">
                            <option value="">全部LLM组</option>
                            <option value="none">默认(无组)</option>
                        </select>
                        <select id="search-group-filter" onchange="loadLicenses(1, licenseSearchTerm)" class="px-3 py-1.5 border rounded-lg text-sm">
                            <option value="">全部搜索组</option>
                            <option value="none">默认(无组)</option>
                        </select>
                        <input type="text" id="license-search" placeholder="搜索序列号..." class="px-3 py-1.5 border rounded-lg text-sm w-48">
                        <button onclick="searchLicenses()" class="px-3 py-1.5 bg-blue-600 text-white rounded-lg text-sm">搜索</button>
                        <button onclick="showBatchCreate()" class="px-4 py-2 bg-green-600 text-white rounded-lg text-sm">批量生成</button>
                        <button onclick="deleteUnusedByGroup()" class="px-3 py-1.5 bg-orange-600 text-white rounded-lg text-sm">🗑️ 删除未使用</button>
                        <button onclick="purgeDisabledLicenses()" class="px-3 py-1.5 bg-red-600 text-white rounded-lg text-sm">🧹 清除已禁用</button>
                    </div>
                </div>
                <div id="license-list" class="space-y-2"></div>
                <div id="license-pagination" class="flex justify-center items-center gap-2 mt-4"></div>
            </div>
        </div>

        <!-- Panel: Email Records -->
        <div id="panel-email-records" class="tab-panel">
            <div class="bg-white rounded-xl shadow-sm p-6">
                <div class="flex justify-between items-center mb-4">
                    <h2 class="text-lg font-bold text-slate-800">邮箱申请记录</h2>
                    <div class="flex items-center gap-2">
                        <input type="text" id="email-search" placeholder="搜索邮箱或序列号..." class="px-3 py-1.5 border rounded-lg text-sm w-64">
                        <button onclick="searchEmails()" class="px-3 py-1.5 bg-blue-600 text-white rounded-lg text-sm">搜索</button>
                    </div>
                </div>
                <div id="email-records-list" class="space-y-3"></div>
                <div id="email-pagination" class="flex justify-center items-center gap-2 mt-4"></div>
            </div>
        </div>

        <!-- Panel: Email Filter -->
        <div id="panel-email-filter" class="tab-panel">
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div class="bg-white rounded-xl shadow-sm p-6 lg:col-span-3">
                    <h2 class="text-lg font-bold text-slate-800 mb-4">过滤模式设置</h2>
                    <div class="flex items-center gap-6 mb-4">
                        <label class="flex items-center gap-2"><input type="checkbox" id="blacklist-enabled" class="w-4 h-4" onchange="saveFilterSettings()"><span class="text-sm">启用黑名单</span></label>
                        <label class="flex items-center gap-2"><input type="checkbox" id="whitelist-enabled" class="w-4 h-4" onchange="saveFilterSettings()"><span class="text-sm">启用白名单</span></label>
                        <label class="flex items-center gap-2"><input type="checkbox" id="conditions-enabled" class="w-4 h-4" onchange="saveFilterSettings()"><span class="text-sm">启用条件名单</span></label>
                    </div>
                    <p class="text-xs text-slate-500">* 黑名单优先检查。启用白名单时，邮箱必须在白名单中且不在黑名单中</p>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">⚫ 黑名单</h2>
                        <button onclick="showAddBlacklist()" class="px-3 py-1.5 bg-red-600 text-white rounded-lg text-sm">+ 添加</button>
                    </div>
                    <p class="text-xs text-slate-500 mb-3">* 匹配的邮箱/域名将被拒绝</p>
                    <div id="blacklist-items" class="space-y-2 max-h-80 overflow-y-auto"></div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">⚪ 白名单</h2>
                        <button onclick="showAddWhitelist()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加</button>
                    </div>
                    <p class="text-xs text-slate-500 mb-3">* 启用时，只有匹配的邮箱/域名才能申请</p>
                    <div id="whitelist-items" class="space-y-2 max-h-80 overflow-y-auto"></div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">📋 条件名单</h2>
                        <button onclick="showAddCondition()" class="px-3 py-1.5 bg-amber-600 text-white rounded-lg text-sm">+ 添加</button>
                    </div>
                    <p class="text-xs text-slate-500 mb-3">* 匹配的邮箱/域名将分配指定分组的序列号</p>
                    <div id="condition-items" class="space-y-2 max-h-80 overflow-y-auto"></div>
                </div>
            </div>
        </div>

        <!-- Panel: License Groups -->
        <div id="panel-license-groups" class="tab-panel">
            <div class="bg-white rounded-xl shadow-sm p-6">
                <div class="flex justify-between items-center mb-4">
                    <h2 class="text-lg font-bold text-slate-800">序列号分组管理</h2>
                    <button onclick="showLicenseGroupForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加分组</button>
                </div>
                <div id="license-groups-list" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"></div>
            </div>
        </div>

        <!-- Panel: LLM Config -->
        <div id="panel-llm" class="tab-panel">
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">LLM 分组</h2>
                        <button onclick="showLLMGroupForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加</button>
                    </div>
                    <div id="llm-groups-list" class="space-y-2"></div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6 lg:col-span-2">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">LLM API 配置</h2>
                        <div class="flex items-center gap-2">
                            <select id="llm-config-group-filter" onchange="loadLLMConfigs()" class="px-3 py-1.5 border rounded-lg text-sm">
                                <option value="">全部分组</option>
                                <option value="none">默认(无组)</option>
                            </select>
                            <button onclick="showLLMForm()" class="px-4 py-2 bg-green-600 text-white rounded-lg text-sm">+ 添加配置</button>
                        </div>
                    </div>
                    <div id="llm-list" class="space-y-2"></div>
                </div>
            </div>
        </div>

        <!-- Panel: Search Config -->
        <div id="panel-search" class="tab-panel">
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">搜索分组</h2>
                        <button onclick="showSearchGroupForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加</button>
                    </div>
                    <div id="search-groups-list" class="space-y-2"></div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6 lg:col-span-2">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">搜索引擎配置</h2>
                        <div class="flex items-center gap-2">
                            <select id="search-config-group-filter" onchange="loadSearchConfigs()" class="px-3 py-1.5 border rounded-lg text-sm">
                                <option value="">全部分组</option>
                                <option value="none">默认(无组)</option>
                            </select>
                            <button onclick="showSearchForm()" class="px-4 py-2 bg-green-600 text-white rounded-lg text-sm">+ 添加配置</button>
                        </div>
                    </div>
                    <div id="search-list" class="space-y-2"></div>
                </div>
            </div>
        </div>

        <!-- Panel: Settings -->
        <div id="panel-settings" class="tab-panel">
            <div class="grid grid-cols-2 gap-6">
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <h2 class="text-lg font-bold text-slate-800 mb-4">修改密码</h2>
                    <div class="space-y-3">
                        <input type="password" id="old-password" placeholder="当前密码" class="w-full px-3 py-2 border rounded-lg">
                        <input type="password" id="new-password" placeholder="新密码" class="w-full px-3 py-2 border rounded-lg">
                        <button onclick="changePassword()" class="w-full bg-blue-600 text-white py-2 rounded-lg">修改密码</button>
                    </div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <h2 class="text-lg font-bold text-slate-800 mb-4">端口配置</h2>
                    <div class="space-y-3">
                        <div><label class="text-sm text-slate-600">管理端口</label><input type="number" id="manage-port" value="{{.ManagePort}}" class="w-full px-3 py-2 border rounded-lg"></div>
                        <div><label class="text-sm text-slate-600">授权端口</label><input type="number" id="auth-port" value="{{.AuthPort}}" class="w-full px-3 py-2 border rounded-lg"></div>
                        <button onclick="changePorts()" class="w-full bg-blue-600 text-white py-2 rounded-lg">保存端口配置</button>
                        <p class="text-xs text-slate-500">* 修改端口后需要重启服务生效</p>
                    </div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6 col-span-2">
                    <h2 class="text-lg font-bold text-slate-800 mb-4">请求限制设置</h2>
                    <div class="space-y-4">
                        <div class="flex items-center gap-6">
                            <div class="flex items-center gap-2">
                                <label class="text-sm text-slate-600">每日请求次数限制:</label>
                                <input type="number" id="daily-request-limit" min="1" max="100" value="5" class="w-20 px-3 py-2 border rounded-lg">
                            </div>
                            <div class="flex items-center gap-2">
                                <label class="text-sm text-slate-600">每日邮箱数限制:</label>
                                <input type="number" id="daily-email-limit" min="1" max="100" value="5" class="w-20 px-3 py-2 border rounded-lg">
                            </div>
                        </div>
                        <button onclick="saveRequestLimits()" class="bg-blue-600 text-white px-4 py-2 rounded-lg">保存限制设置</button>
                        <p class="text-xs text-slate-500">* 同一IP每日最多申请指定次数，使用不同邮箱最多指定个数</p>
                    </div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6 col-span-2">
                    <h2 class="text-lg font-bold text-slate-800 mb-4">SSL/HTTPS 配置</h2>
                    <div class="space-y-3">
                        <div class="flex items-center gap-3"><input type="checkbox" id="use-ssl" class="w-4 h-4"><label class="text-sm text-slate-700">启用 HTTPS</label></div>
                        <div id="ssl-fields" class="space-y-3 hidden">
                            <div><label class="text-sm text-slate-600">SSL 证书文件路径</label><input type="text" id="ssl-cert" placeholder="/path/to/cert.pem" class="w-full px-3 py-2 border rounded-lg"></div>
                            <div><label class="text-sm text-slate-600">SSL 密钥文件路径</label><input type="text" id="ssl-key" placeholder="/path/to/key.pem" class="w-full px-3 py-2 border rounded-lg"></div>
                        </div>
                        <button onclick="saveSSLConfig()" class="w-full bg-blue-600 text-white py-2 rounded-lg">保存 SSL 配置</button>
                        <p class="text-xs text-slate-500">* 修改 SSL 配置后需要重启服务生效</p>
                    </div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6 col-span-2">
                    <h2 class="text-lg font-bold text-red-600 mb-4">⚠️ 危险操作</h2>
                    <div class="space-y-3">
                        <button onclick="showForceDeleteLicense()" class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700">🗑️ 强制删除序列号</button>
                        <p class="text-xs text-slate-500">* 强制删除指定序列号及其所有相关记录（邮箱申请记录等），此操作不可恢复</p>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <div id="modal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white rounded-xl shadow-xl w-[500px] max-h-[80vh] overflow-auto">
            <div id="modal-content"></div>
        </div>
    </div>

<script>
// ============ Global Variables ============
var llmGroups = [];
var searchGroups = [];
var licenseGroups = [];
var licenseCurrentPage = 1;
var licenseSearchTerm = '';
var emailCurrentPage = 1;
var emailSearchTerm = '';

// ============ Tab Switching ============
function showTab(name) {
    document.querySelectorAll('.tab-panel').forEach(function(p) { p.classList.remove('active'); });
    document.querySelectorAll('.tab-btn').forEach(function(b) {
        b.classList.remove('bg-blue-600', 'text-white');
        b.classList.add('bg-slate-200', 'text-slate-700');
    });
    var panel = document.getElementById('panel-' + name);
    if (panel) panel.classList.add('active');
    var btn = document.getElementById('tab-' + name);
    if (btn) {
        btn.classList.remove('bg-slate-200', 'text-slate-700');
        btn.classList.add('bg-blue-600', 'text-white');
    }
}

// ============ Modal Functions ============
function showModal(content) {
    document.getElementById('modal-content').innerHTML = content;
    document.getElementById('modal').classList.remove('hidden');
    document.getElementById('modal').classList.add('flex');
    setTimeout(function() {
        var firstInput = document.querySelector('#modal-content input:not([type=hidden]), #modal-content select');
        if (firstInput) firstInput.focus();
    }, 100);
}

function hideModal() {
    document.getElementById('modal').classList.add('hidden');
    document.getElementById('modal').classList.remove('flex');
}

// ============ Helper Functions ============
function getLLMGroupName(id) {
    if (!id) return '';
    var g = llmGroups.find(function(g) { return g.id === id; });
    return g ? g.name : id;
}

function getSearchGroupName(id) {
    if (!id) return '';
    var g = searchGroups.find(function(g) { return g.id === id; });
    return g ? g.name : id;
}

function getLicenseGroupName(id) {
    if (!id) return '';
    var g = licenseGroups.find(function(g) { return g.id === id; });
    return g ? g.name : id;
}

// ============ Licenses Functions ============
function loadLicenses(page, search) {
    page = page || 1;
    search = search || '';
    licenseCurrentPage = page;
    licenseSearchTerm = search;
    
    var licenseGroupFilter = document.getElementById('license-group-filter').value;
    var llmGroupFilter = document.getElementById('llm-group-filter').value;
    var searchGroupFilter = document.getElementById('search-group-filter').value;
    
    var params = new URLSearchParams({page: page.toString(), pageSize: '20', search: search, license_group: licenseGroupFilter, llm_group: llmGroupFilter, search_group: searchGroupFilter});
    
    fetch('/api/licenses/search?' + params).then(function(resp) { return resp.json(); }).then(function(data) {
        var list = document.getElementById('license-list');
        if (!data.licenses || data.licenses.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无序列号</p>'; 
            document.getElementById('license-pagination').innerHTML = '';
            return; 
        }
        
        var html = '';
        data.licenses.forEach(function(l) {
            var isExpired = new Date(l.expires_at) < new Date();
            var statusClass = !l.is_active ? 'opacity-50' : (isExpired ? 'bg-orange-50' : '');
            var llmGroupName = getLLMGroupName(l.llm_group_id);
            var searchGroupName = getSearchGroupName(l.search_group_id);
            var licenseGroupName = getLicenseGroupName(l.license_group_id);
            
            html += '<div class="flex items-center justify-between p-3 bg-slate-50 rounded-lg ' + statusClass + '">';
            html += '<div class="flex-1">';
            html += '<div class="flex items-center gap-2 flex-wrap">';
            html += '<code class="font-mono font-bold text-blue-600">' + l.sn + '</code>';
            if (!l.is_active) html += '<span class="px-2 py-0.5 bg-red-100 text-red-700 text-xs rounded">已禁用</span>';
            if (isExpired) html += '<span class="px-2 py-0.5 bg-orange-100 text-orange-700 text-xs rounded">已过期</span>';
            if (licenseGroupName) html += '<span class="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs rounded">' + licenseGroupName + '</span>';
            if (llmGroupName) html += '<span class="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded">' + llmGroupName + '</span>';
            if (searchGroupName) html += '<span class="px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">' + searchGroupName + '</span>';
            html += '</div>';
            html += '<p class="text-xs text-slate-500 mt-1">' + (l.description || '无描述') + '</p>';
            html += '<p class="text-xs text-slate-400">过期: ' + new Date(l.expires_at).toLocaleDateString() + ' | 使用: ' + l.usage_count + '次 | 每日分析: ' + (l.daily_analysis === 0 ? '无限' : l.daily_analysis + '次') + '</p>';
            html += '</div>';
            html += '<div class="flex gap-2 flex-wrap">';
            html += '<button onclick="setLicenseGroups(\'' + l.sn + '\', \'' + (l.license_group_id || '') + '\', \'' + (l.llm_group_id || '') + '\', \'' + (l.search_group_id || '') + '\')" class="px-2 py-1 bg-indigo-100 text-indigo-700 rounded text-xs">分组</button>';
            html += '<button onclick="extendLicense(\'' + l.sn + '\', \'' + l.expires_at + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">展期</button>';
            html += '<button onclick="setDailyAnalysis(\'' + l.sn + '\', ' + l.daily_analysis + ')" class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs">分析次数</button>';
            html += '<button onclick="toggleLicense(\'' + l.sn + '\')" class="px-2 py-1 ' + (l.is_active ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700') + ' rounded text-xs">' + (l.is_active ? '禁用' : '启用') + '</button>';
            html += '</div>';
            html += '</div>';
        });
        list.innerHTML = html;
        
        var pagination = document.getElementById('license-pagination');
        var paginationHTML = '<span class="text-sm text-slate-500">共 ' + data.total + ' 条</span>';
        if (data.totalPages > 1) {
            paginationHTML += '<button onclick="loadLicenses(1, licenseSearchTerm)" class="px-2 py-1 rounded ' + (data.page === 1 ? 'text-slate-300' : 'hover:bg-slate-100') + '"' + (data.page === 1 ? ' disabled' : '') + '>首页</button>';
            paginationHTML += '<button onclick="loadLicenses(' + (data.page - 1) + ', licenseSearchTerm)" class="px-2 py-1 rounded ' + (data.page === 1 ? 'text-slate-300' : 'hover:bg-slate-100') + '"' + (data.page === 1 ? ' disabled' : '') + '>上一页</button>';
            paginationHTML += '<span class="px-2 text-sm">' + data.page + ' / ' + data.totalPages + '</span>';
            paginationHTML += '<button onclick="loadLicenses(' + (data.page + 1) + ', licenseSearchTerm)" class="px-2 py-1 rounded ' + (data.page === data.totalPages ? 'text-slate-300' : 'hover:bg-slate-100') + '"' + (data.page === data.totalPages ? ' disabled' : '') + '>下一页</button>';
            paginationHTML += '<button onclick="loadLicenses(' + data.totalPages + ', licenseSearchTerm)" class="px-2 py-1 rounded ' + (data.page === data.totalPages ? 'text-slate-300' : 'hover:bg-slate-100') + '"' + (data.page === data.totalPages ? ' disabled' : '') + '>末页</button>';
        }
        pagination.innerHTML = paginationHTML;
    });
}

function searchLicenses() { loadLicenses(1, document.getElementById('license-search').value); }

function toggleLicense(sn) {
    fetch('/api/licenses/toggle', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                loadLicenses(licenseCurrentPage, licenseSearchTerm); 
            } else { 
                alert('操作失败: ' + (result.error || '未知错误')); 
            } 
        })
        .catch(function(err) { alert('请求失败: ' + err.message); });
}

function showBatchCreate() {
    var licenseGroupOpts = '<option value="">无分组</option>';
    licenseGroups.forEach(function(g) { licenseGroupOpts += '<option value="' + g.id + '">' + g.name + '</option>'; });
    var llmGroupOpts = '<option value="">无分组</option>';
    llmGroups.forEach(function(g) { llmGroupOpts += '<option value="' + g.id + '">' + g.name + '</option>'; });
    var searchGroupOpts = '<option value="">无分组</option>';
    searchGroups.forEach(function(g) { searchGroupOpts += '<option value="' + g.id + '">' + g.name + '</option>'; });
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">批量生成序列号</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">描述</label><input type="text" id="batch-desc" placeholder="可选" class="w-full px-3 py-2 border rounded-lg"></div><div class="grid grid-cols-2 gap-3"><div><label class="text-sm text-slate-600">有效天数</label><input type="number" id="batch-days" value="365" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">生成数量</label><input type="number" id="batch-count" value="100" class="w-full px-3 py-2 border rounded-lg"></div></div><div><label class="text-sm text-slate-600">每日分析次数 (0=无限)</label><input type="number" id="batch-daily" value="20" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">序列号分组</label><select id="batch-license-group" class="w-full px-3 py-2 border rounded-lg">' + licenseGroupOpts + '</select></div><div><label class="text-sm text-slate-600">LLM分组</label><select id="batch-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmGroupOpts + '</select></div><div><label class="text-sm text-slate-600">搜索分组</label><select id="batch-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchGroupOpts + '</select></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doBatchCreate()" class="flex-1 py-2 bg-green-600 text-white rounded-lg">生成</button></div></div></div>');
}

function doBatchCreate() {
    var data = {description: document.getElementById('batch-desc').value, days: parseInt(document.getElementById('batch-days').value) || 365, count: parseInt(document.getElementById('batch-count').value) || 100, daily_analysis: parseInt(document.getElementById('batch-daily').value) || 0, license_group_id: document.getElementById('batch-license-group').value, llm_group_id: document.getElementById('batch-llm-group').value, search_group_id: document.getElementById('batch-search-group').value};
    fetch('/api/licenses/batch-create', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { alert('成功生成 ' + result.count + ' 个序列号'); loadLicenses(); } else { alert('生成失败: ' + result.error); } });
}

function extendLicense(sn, currentExpiry) {
    var expiryDate = currentExpiry ? new Date(currentExpiry).toISOString().split('T')[0] : '';
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">展期序列号</h3><div class="space-y-3"><p class="text-sm text-slate-600">序列号: <code class="font-mono text-blue-600">' + sn + '</code></p><p class="text-sm text-slate-600">当前到期: <span class="text-orange-600">' + (expiryDate || '未知') + '</span></p><div><label class="text-sm text-slate-600">延长天数</label><input type="number" id="extend-days" value="365" class="w-full px-3 py-2 border rounded-lg"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doExtendLicense(\'' + sn + '\')" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">确认</button></div></div></div>');
}

function doExtendLicense(sn) {
    var days = parseInt(document.getElementById('extend-days').value) || 365;
    fetch('/api/licenses/extend', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, days: days})})
        .then(function() { hideModal(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function setDailyAnalysis(sn, current) {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">设置每日分析次数</h3><div class="space-y-3"><p class="text-sm text-slate-600">序列号: <code class="font-mono text-blue-600">' + sn + '</code></p><div><label class="text-sm text-slate-600">每日分析次数 (0=无限)</label><input type="number" id="daily-count" value="' + current + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doSetDailyAnalysis(\'' + sn + '\')" class="flex-1 py-2 bg-purple-600 text-white rounded-lg">确认</button></div></div></div>');
}

function doSetDailyAnalysis(sn) {
    var count = parseInt(document.getElementById('daily-count').value) || 0;
    fetch('/api/licenses/set-daily', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, daily_analysis: count})})
        .then(function() { hideModal(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function setLicenseGroups(sn, licenseGroupId, llmGroupId, searchGroupId) {
    var licenseGroupOpts = '<option value="">无分组</option>';
    licenseGroups.forEach(function(g) { licenseGroupOpts += '<option value="' + g.id + '"' + (g.id === licenseGroupId ? ' selected' : '') + '>' + g.name + '</option>'; });
    var llmGroupOpts = '<option value="">无分组</option>';
    llmGroups.forEach(function(g) { llmGroupOpts += '<option value="' + g.id + '"' + (g.id === llmGroupId ? ' selected' : '') + '>' + g.name + '</option>'; });
    var searchGroupOpts = '<option value="">无分组</option>';
    searchGroups.forEach(function(g) { searchGroupOpts += '<option value="' + g.id + '"' + (g.id === searchGroupId ? ' selected' : '') + '>' + g.name + '</option>'; });
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">设置分组</h3><div class="space-y-3"><p class="text-sm text-slate-600">序列号: <code class="font-mono text-blue-600">' + sn + '</code></p><div><label class="text-sm text-slate-600">序列号分组</label><select id="set-license-group" class="w-full px-3 py-2 border rounded-lg">' + licenseGroupOpts + '</select></div><div><label class="text-sm text-slate-600">LLM分组</label><select id="set-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmGroupOpts + '</select></div><div><label class="text-sm text-slate-600">搜索分组</label><select id="set-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchGroupOpts + '</select></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doSetLicenseGroups(\'' + sn + '\')" class="flex-1 py-2 bg-indigo-600 text-white rounded-lg">保存</button></div></div></div>');
}

function doSetLicenseGroups(sn) {
    var data = {sn: sn, license_group_id: document.getElementById('set-license-group').value, llm_group_id: document.getElementById('set-llm-group').value, search_group_id: document.getElementById('set-search-group').value};
    fetch('/api/licenses/set-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)})
        .then(function() { hideModal(); loadLicenses(licenseCurrentPage, licenseSearchTerm); loadEmailRecords(emailCurrentPage, emailSearchTerm); });
}

function deleteUnusedByGroup() {
    var licenseGroupFilter = document.getElementById('license-group-filter').value;
    var groupName = licenseGroupFilter === 'none' ? '默认(无组)' : (licenseGroupFilter ? getLicenseGroupName(licenseGroupFilter) : '全部');
    
    var warningMsg = '确定要删除 [' + groupName + '] 中所有未使用的序列号吗？\n\n此操作不可恢复！';
    if (!licenseGroupFilter) {
        warningMsg = '⚠️ 警告：您将删除【所有分组】中未使用的序列号！\n\n此操作不可恢复！确定继续吗？';
    }
    if (!confirm(warningMsg)) return;
    
    fetch('/api/licenses/delete-unused-by-group', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({license_group_id: licenseGroupFilter, delete_all: !licenseGroupFilter})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { if (result.success) { alert('已删除 ' + result.count + ' 个未使用的序列号'); loadLicenses(); } else { alert('删除失败: ' + result.error); } });
}

function purgeDisabledLicenses() {
    if (!confirm('确定要清除所有已禁用且未绑定邮箱的序列号吗？\n\n此操作不可恢复！')) return;
    
    fetch('/api/licenses/purge-disabled', {method: 'POST', headers: {'Content-Type': 'application/json'}})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { if (result.success) { alert('已清除 ' + result.deleted + ' 个已禁用的序列号'); loadLicenses(); } else { alert('清除失败: ' + result.error); } });
}

// ============ Email Records Functions ============
function loadEmailRecords(page, search) {
    page = page || 1;
    search = search || '';
    emailCurrentPage = page;
    emailSearchTerm = search;
    
    var params = new URLSearchParams({page: page.toString(), pageSize: '15', search: search});
    fetch('/api/email-records?' + params).then(function(resp) { return resp.json(); }).then(function(data) {
        var list = document.getElementById('email-records-list');
        if (!data.records || data.records.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无申请记录</p>'; 
            document.getElementById('email-pagination').innerHTML = ''; 
            return; 
        }
        
        var sns = data.records.map(function(r) { return r.sn; });
        Promise.all(sns.map(function(sn) {
            return fetch('/api/licenses/search?search=' + encodeURIComponent(sn) + '&pageSize=1&hide_used=false').then(function(r) { return r.json(); });
        })).then(function(licenseResults) {
            var licenseMap = {};
            licenseResults.forEach(function(result, idx) {
                if (result.licenses && result.licenses.length > 0) {
                    licenseMap[sns[idx]] = result.licenses[0];
                }
            });
            
            var html = '<div class="space-y-3">';
            data.records.forEach(function(r) {
                var license = licenseMap[r.sn] || {};
                var isActive = license.is_active === true || license.is_active === 1;
                var expiresAt = license.expires_at ? new Date(license.expires_at) : null;
                var isExpired = expiresAt && expiresAt < new Date();
                var llmGroupName = getLLMGroupName(license.llm_group_id || '');
                var searchGroupName = getSearchGroupName(license.search_group_id || '');
                var licenseGroupName = getLicenseGroupName(license.license_group_id || '');
                var dailyAnalysis = license.daily_analysis !== undefined ? license.daily_analysis : 20;
                var opacityClass = !isActive ? 'opacity-50' : '';
                
                html += '<div class="p-3 bg-slate-50 rounded-lg ' + opacityClass + '">';
                html += '<div class="flex items-start justify-between">';
                html += '<div class="flex-1">';
                html += '<div class="flex items-center gap-3 mb-1">';
                html += '<span class="text-sm text-slate-600">' + r.email + '</span>';
                html += '<code class="font-mono text-blue-600 font-bold">' + r.sn + '</code>';
                if (!isActive) html += '<span class="px-2 py-0.5 bg-red-100 text-red-700 text-xs rounded">已禁用</span>';
                if (isExpired) html += '<span class="px-2 py-0.5 bg-orange-100 text-orange-700 text-xs rounded">已过期</span>';
                html += '</div>';
                html += '<p class="text-xs text-slate-400">申请时间: ' + new Date(r.created_at).toLocaleString() + ' | IP: ' + r.ip + '</p>';
                html += '<p class="text-xs text-slate-400">';
                if (expiresAt) html += '过期: <span class="' + (isExpired ? 'text-red-600' : '') + '">' + expiresAt.toLocaleDateString() + '</span> | ';
                html += '每日分析: ' + (dailyAnalysis === 0 ? '无限' : dailyAnalysis + '次') + ' | ';
                html += '序列号分组: <span class="text-purple-600">' + (licenseGroupName || '默认') + '</span> | ';
                html += 'LLM分组: <span class="text-blue-600">' + (llmGroupName || '默认') + '</span> | ';
                html += '搜索分组: <span class="text-green-600">' + (searchGroupName || '默认') + '</span>';
                html += '</p>';
                html += '</div>';
                html += '<div class="flex gap-2 flex-shrink-0">';
                html += '<button onclick="editEmailRecord(' + r.id + ', \'' + r.email + '\', \'' + r.sn + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs hover:bg-blue-200">修改</button>';
                html += '<button onclick="setLicenseGroups(\'' + r.sn + '\', \'' + (license.license_group_id || '') + '\', \'' + (license.llm_group_id || '') + '\', \'' + (license.search_group_id || '') + '\')" class="px-2 py-1 bg-indigo-100 text-indigo-700 rounded text-xs hover:bg-indigo-200">分组</button>';
                html += '<button onclick="extendLicense(\'' + r.sn + '\', \'' + (license.expires_at || '') + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs hover:bg-blue-200">展期</button>';
                html += '<button onclick="setDailyAnalysis(\'' + r.sn + '\', ' + dailyAnalysis + ')" class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs hover:bg-purple-200">分析次数</button>';
                html += '<button onclick="toggleLicenseFromEmail(\'' + r.sn + '\')" class="px-2 py-1 ' + (isActive ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700') + ' rounded text-xs hover:opacity-80">' + (isActive ? '禁用' : '启用') + '</button>';
                html += '</div>';
                html += '</div>';
                html += '</div>';
            });
            html += '</div>';
            list.innerHTML = html;
        });
        
        var pagination = document.getElementById('email-pagination');
        var paginationHTML = '<span class="text-sm text-slate-500">共 ' + data.total + ' 条记录</span>';
        if (data.totalPages > 1) {
            paginationHTML += '<button onclick="loadEmailRecords(1, emailSearchTerm)" class="px-2 py-1 rounded ' + (data.page === 1 ? 'text-slate-300' : 'hover:bg-slate-100') + '"' + (data.page === 1 ? ' disabled' : '') + '>首页</button>';
            paginationHTML += '<button onclick="loadEmailRecords(' + (data.page - 1) + ', emailSearchTerm)" class="px-2 py-1 rounded ' + (data.page === 1 ? 'text-slate-300' : 'hover:bg-slate-100') + '"' + (data.page === 1 ? ' disabled' : '') + '>上一页</button>';
            paginationHTML += '<span class="px-2 text-sm">' + data.page + ' / ' + data.totalPages + '</span>';
            paginationHTML += '<button onclick="loadEmailRecords(' + (data.page + 1) + ', emailSearchTerm)" class="px-2 py-1 rounded ' + (data.page === data.totalPages ? 'text-slate-300' : 'hover:bg-slate-100') + '"' + (data.page === data.totalPages ? ' disabled' : '') + '>下一页</button>';
            paginationHTML += '<button onclick="loadEmailRecords(' + data.totalPages + ', emailSearchTerm)" class="px-2 py-1 rounded ' + (data.page === data.totalPages ? 'text-slate-300' : 'hover:bg-slate-100') + '"' + (data.page === data.totalPages ? ' disabled' : '') + '>末页</button>';
        }
        pagination.innerHTML = paginationHTML;
    });
}

function searchEmails() { loadEmailRecords(1, document.getElementById('email-search').value); }

function toggleLicenseFromEmail(sn) {
    fetch('/api/licenses/toggle', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                loadEmailRecords(emailCurrentPage, emailSearchTerm); 
            } else { 
                alert('操作失败: ' + (result.error || '未知错误')); 
            } 
        })
        .catch(function(err) { alert('请求失败: ' + err.message); });
}

function editEmailRecord(id, email, sn) {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">修改申请记录</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">邮箱</label><input type="email" id="edit-email" value="' + email + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">序列号</label><input type="text" id="edit-sn" value="' + sn + '" class="w-full px-3 py-2 border rounded-lg font-mono"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doEditEmailRecord(' + id + ')" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function doEditEmailRecord(id) {
    var email = document.getElementById('edit-email').value.trim();
    var sn = document.getElementById('edit-sn').value.trim().toUpperCase();
    if (!email || !sn) { alert('邮箱和序列号不能为空'); return; }
    
    fetch('/api/email-records/update', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id, email: email, sn: sn})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadEmailRecords(emailCurrentPage, emailSearchTerm); } else { alert('修改失败: ' + result.error); } });
}

// ============ Email Filter Functions ============
function loadFilterSettings() {
    fetch('/api/email-filter').then(function(resp) { return resp.json(); }).then(function(data) {
        document.getElementById('whitelist-enabled').checked = data.whitelist_enabled;
        document.getElementById('conditions-enabled').checked = data.conditions_enabled;
        document.getElementById('blacklist-enabled').checked = data.blacklist_enabled;
        document.getElementById('daily-request-limit').value = data.daily_request_limit || '5';
        document.getElementById('daily-email-limit').value = data.daily_email_limit || '5';
    });
}

function saveFilterSettings() {
    var data = {whitelist_enabled: document.getElementById('whitelist-enabled').checked, blacklist_enabled: document.getElementById('blacklist-enabled').checked, conditions_enabled: document.getElementById('conditions-enabled').checked, daily_request_limit: document.getElementById('daily-request-limit').value, daily_email_limit: document.getElementById('daily-email-limit').value};
    fetch('/api/email-filter', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)});
}

function loadBlacklist() {
    fetch('/api/blacklist').then(function(resp) { return resp.json(); }).then(function(items) {
        var list = document.getElementById('blacklist-items');
        if (!items || items.length === 0) { list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无黑名单</p>'; return; }
        var html = '';
        items.forEach(function(item) { 
            html += '<div class="flex items-center justify-between p-2 bg-red-50 rounded-lg">';
            html += '<div><code class="text-sm font-mono text-red-700">' + item.pattern + '</code>';
            html += '<p class="text-xs text-slate-400">' + new Date(item.created_at).toLocaleString() + '</p></div>';
            html += '<button onclick="deleteBlacklist(\'' + item.pattern + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
            html += '</div>'; 
        });
        list.innerHTML = html;
    });
}

function loadWhitelist() {
    fetch('/api/whitelist').then(function(resp) { return resp.json(); }).then(function(items) {
        var list = document.getElementById('whitelist-items');
        if (!items || items.length === 0) { list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无白名单</p>'; return; }
        var html = '';
        items.forEach(function(item) { 
            html += '<div class="flex items-center justify-between p-2 bg-green-50 rounded-lg">';
            html += '<div><code class="text-sm font-mono text-green-700">' + item.pattern + '</code>';
            html += '<p class="text-xs text-slate-400">' + new Date(item.created_at).toLocaleString() + '</p></div>';
            html += '<button onclick="deleteWhitelist(\'' + item.pattern + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
            html += '</div>'; 
        });
        list.innerHTML = html;
    });
}

function loadConditions() {
    Promise.all([fetch('/api/conditions').then(function(r){return r.json();}), fetch('/api/llm-groups').then(function(r){return r.json();}), fetch('/api/search-groups').then(function(r){return r.json();})]).then(function(results) {
        var items = results[0] || [];
        var llmGroupsList = results[1] || [];
        var searchGroupsList = results[2] || [];
        var list = document.getElementById('condition-items');
        
        if (!items || items.length === 0) { list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无条件名单</p>'; return; }
        
        var html = '';
        items.forEach(function(item) {
            var llmGroupName = item.llm_group_id ? (llmGroupsList.find(function(g){return g.id===item.llm_group_id;}) || {}).name || item.llm_group_id : '无限制';
            var searchGroupName = item.search_group_id ? (searchGroupsList.find(function(g){return g.id===item.search_group_id;}) || {}).name || item.search_group_id : '无限制';
            
            html += '<div class="p-3 bg-amber-50 rounded-lg">';
            html += '<div class="flex items-center justify-between">';
            html += '<code class="text-sm font-mono text-amber-700">' + item.pattern + '</code>';
            html += '<button onclick="deleteCondition(\'' + item.pattern + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
            html += '</div>';
            html += '<div class="mt-2 text-xs text-slate-500">';
            html += '<span class="mr-3">LLM组: <span class="text-blue-600">' + llmGroupName + '</span></span>';
            html += '<span>搜索组: <span class="text-purple-600">' + searchGroupName + '</span></span>';
            html += '</div>';
            html += '<p class="text-xs text-slate-400 mt-1">' + new Date(item.created_at).toLocaleString() + '</p>';
            html += '</div>';
        });
        list.innerHTML = html;
    });
}

function showAddBlacklist() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">添加黑名单</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">邮箱或域名</label><input type="text" id="blacklist-pattern" placeholder="例如: @spam.com" class="w-full px-3 py-2 border rounded-lg"></div><p class="text-xs text-slate-500">* 以 @ 开头表示域名匹配</p><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doAddBlacklist()" class="flex-1 py-2 bg-red-600 text-white rounded-lg">添加</button></div></div></div>');
}

function showAddWhitelist() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">添加白名单</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">邮箱或域名</label><input type="text" id="whitelist-pattern" placeholder="例如: @company.com" class="w-full px-3 py-2 border rounded-lg"></div><p class="text-xs text-slate-500">* 以 @ 开头表示域名匹配</p><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doAddWhitelist()" class="flex-1 py-2 bg-green-600 text-white rounded-lg">添加</button></div></div></div>');
}

function showAddCondition() {
    Promise.all([fetch('/api/llm-groups').then(function(r){return r.json();}), fetch('/api/search-groups').then(function(r){return r.json();})]).then(function(results) {
        var llmGroupsList = results[0] || [];
        var searchGroupsList = results[1] || [];
        
        var llmOptions = '<option value="">无限制（随机）</option>';
        llmGroupsList.forEach(function(g) { llmOptions += '<option value="' + g.id + '">' + g.name + '</option>'; });
        var searchOptions = '<option value="">无限制（随机）</option>';
        searchGroupsList.forEach(function(g) { searchOptions += '<option value="' + g.id + '">' + g.name + '</option>'; });
        
        showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">添加条件名单</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">邮箱或域名</label><input type="text" id="condition-pattern" placeholder="例如: @company.com" class="w-full px-3 py-2 border rounded-lg"></div><p class="text-xs text-slate-500">* 以 @ 开头表示域名匹配</p><div><label class="text-sm text-slate-600">绑定LLM组</label><select id="condition-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmOptions + '</select></div><div><label class="text-sm text-slate-600">绑定搜索引擎组</label><select id="condition-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchOptions + '</select></div><p class="text-xs text-slate-500">* 匹配的邮箱申请时将分配指定分组的序列号</p><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doAddCondition()" class="flex-1 py-2 bg-amber-600 text-white rounded-lg">添加</button></div></div></div>');
    });
}

function doAddBlacklist() {
    var pattern = document.getElementById('blacklist-pattern').value.trim();
    if (!pattern) { alert('请输入邮箱或域名'); return; }
    fetch('/api/blacklist', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadBlacklist(); } else { alert('添加失败: ' + result.error); } });
}

function doAddWhitelist() {
    var pattern = document.getElementById('whitelist-pattern').value.trim();
    if (!pattern) { alert('请输入邮箱或域名'); return; }
    fetch('/api/whitelist', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadWhitelist(); } else { alert('添加失败: ' + result.error); } });
}

function doAddCondition() {
    var pattern = document.getElementById('condition-pattern').value.trim();
    var llmGroupId = document.getElementById('condition-llm-group').value;
    var searchGroupId = document.getElementById('condition-search-group').value;
    if (!pattern) { alert('请输入邮箱或域名'); return; }
    fetch('/api/conditions', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern, llm_group_id: llmGroupId, search_group_id: searchGroupId})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadConditions(); } else { alert('添加失败: ' + result.error); } });
}

function deleteBlacklist(pattern) {
    if (!confirm('确定要删除此黑名单项吗？')) return;
    fetch('/api/blacklist', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})}).then(function() { loadBlacklist(); });
}

function deleteWhitelist(pattern) {
    if (!confirm('确定要删除此白名单项吗？')) return;
    fetch('/api/whitelist', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})}).then(function() { loadWhitelist(); });
}

function deleteCondition(pattern) {
    if (!confirm('确定要删除此条件名单项吗？')) return;
    fetch('/api/conditions', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})}).then(function() { loadConditions(); });
}

// ============ License Groups Functions ============
function loadLicenseGroups() {
    fetch('/api/license-groups').then(function(resp) { return resp.json(); }).then(function(data) {
        licenseGroups = data || [];
        var list = document.getElementById('license-groups-list');
        
        if (!licenseGroups || licenseGroups.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm col-span-3">暂无分组</p>'; 
        } else {
            var html = '';
            licenseGroups.forEach(function(g) { 
                html += '<div class="flex items-center justify-between p-3 bg-purple-50 rounded-lg">';
                html += '<div><span class="font-bold text-sm">' + g.name + '</span>';
                html += '<p class="text-xs text-slate-400">' + (g.description || '无描述') + '</p></div>';
                html += '<div class="flex gap-1">';
                html += '<button onclick="editLicenseGroup(\'' + g.id + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
                html += '<button onclick="deleteLicenseGroup(\'' + g.id + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
                html += '</div></div>'; 
            });
            list.innerHTML = html;
        }
        
        var filterSelect = document.getElementById('license-group-filter');
        if (filterSelect) {
            var currentValue = filterSelect.value;
            var opts = '<option value="">全部序列号组</option><option value="none">默认(无组)</option>';
            licenseGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + g.name + '</option>'; });
            filterSelect.innerHTML = opts;
            filterSelect.value = currentValue;
        }
    });
}

function showLicenseGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + '序列号分组</h3><div class="space-y-3"><input type="hidden" id="license-group-id" value="' + g.id + '"><div><label class="text-sm text-slate-600">分组名称</label><input type="text" id="license-group-name" value="' + g.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">描述</label><input type="text" id="license-group-desc" value="' + (g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveLicenseGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function editLicenseGroup(id) {
    var group = licenseGroups.find(function(g) { return g.id === id; });
    if (group) showLicenseGroupForm(group);
}

function saveLicenseGroup() {
    var group = {id: document.getElementById('license-group-id').value, name: document.getElementById('license-group-name').value, description: document.getElementById('license-group-desc').value};
    if (!group.name) { alert('分组名称不能为空'); return; }
    
    fetch('/api/license-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(group)})
        .then(function() { hideModal(); loadLicenseGroups(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function deleteLicenseGroup(id) {
    if (!confirm('确定要删除此分组吗？')) return;
    fetch('/api/license-groups', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function() { loadLicenseGroups(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

// ============ LLM Config Functions ============
function loadLLMGroups() {
    fetch('/api/llm-groups').then(function(resp) { return resp.json(); }).then(function(data) {
        llmGroups = data || [];
        var list = document.getElementById('llm-groups-list');
        
        if (!llmGroups || llmGroups.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无分组</p>'; 
        } else {
            var html = '';
            llmGroups.forEach(function(g) { 
                html += '<div class="flex items-center justify-between p-2 bg-blue-50 rounded-lg">';
                html += '<div><span class="font-bold text-sm">' + g.name + '</span>';
                html += '<p class="text-xs text-slate-400">' + (g.description || '') + '</p></div>';
                html += '<div class="flex gap-1">';
                html += '<button onclick="editLLMGroup(\'' + g.id + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
                html += '<button onclick="deleteLLMGroup(\'' + g.id + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
                html += '</div></div>'; 
            });
            list.innerHTML = html;
        }
        
        var filterSelect = document.getElementById('llm-group-filter');
        if (filterSelect) {
            var currentValue = filterSelect.value;
            var opts = '<option value="">全部LLM组</option><option value="none">默认(无组)</option>';
            llmGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + g.name + '</option>'; });
            filterSelect.innerHTML = opts;
            filterSelect.value = currentValue;
        }
        
        var configFilterSelect = document.getElementById('llm-config-group-filter');
        if (configFilterSelect) {
            var currentValue = configFilterSelect.value;
            var opts = '<option value="">全部分组</option><option value="none">默认(无组)</option>';
            llmGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + g.name + '</option>'; });
            configFilterSelect.innerHTML = opts;
            configFilterSelect.value = currentValue;
        }
    });
}

function loadLLMConfigs() {
    var groupFilter = document.getElementById('llm-config-group-filter') ? document.getElementById('llm-config-group-filter').value : '';
    
    fetch('/api/llm').then(function(resp) { return resp.json(); }).then(function(configs) {
        var list = document.getElementById('llm-list');
        var filteredConfigs = configs || [];
        
        if (groupFilter === 'none') {
            filteredConfigs = filteredConfigs.filter(function(c) { return !c.group_id || c.group_id === ''; });
        } else if (groupFilter) {
            filteredConfigs = filteredConfigs.filter(function(c) { return c.group_id === groupFilter; });
        }
        
        if (!filteredConfigs || filteredConfigs.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无配置</p>'; 
            return; 
        }
        
        var html = '';
        filteredConfigs.forEach(function(c) {
            var groupName = getLLMGroupName(c.group_id);
            var ringClass = c.is_active ? 'ring-2 ring-green-500' : '';
            
            html += '<div class="flex items-center justify-between p-3 bg-slate-50 rounded-lg ' + ringClass + '">';
            html += '<div><span class="font-bold">' + c.name + '</span>';
            if (c.is_active) html += '<span class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">当前使用</span>';
            if (groupName) html += '<span class="ml-2 px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded">' + groupName + '</span>';
            html += '<p class="text-sm text-slate-500">类型: ' + c.type + ' | 模型: ' + c.model + '</p>';
            html += '<p class="text-xs text-slate-400">URL: ' + (c.base_url || '默认') + '</p>';
            html += '<p class="text-xs text-slate-400">有效期: ' + (c.start_date || '无限制') + ' ~ ' + (c.end_date || '永久') + '</p></div>';
            html += '<div class="flex gap-2">';
            html += '<button onclick="editLLM(\'' + c.id + '\')" class="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm">编辑</button>';
            html += '<button onclick="deleteLLM(\'' + c.id + '\')" class="px-3 py-1 bg-red-100 text-red-700 rounded text-sm">删除</button>';
            html += '</div></div>';
        });
        list.innerHTML = html;
    });
}

function showLLMGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + ' LLM 分组</h3><div class="space-y-3"><input type="hidden" id="llm-group-id" value="' + g.id + '"><div><label class="text-sm text-slate-600">分组名称</label><input type="text" id="llm-group-name" value="' + g.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">描述</label><input type="text" id="llm-group-desc" value="' + (g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveLLMGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function editLLMGroup(id) {
    var group = llmGroups.find(function(g) { return g.id === id; });
    if (group) showLLMGroupForm(group);
}

function saveLLMGroup() {
    var group = {id: document.getElementById('llm-group-id').value, name: document.getElementById('llm-group-name').value, description: document.getElementById('llm-group-desc').value};
    if (!group.name) { alert('分组名称不能为空'); return; }
    
    fetch('/api/llm-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(group)})
        .then(function() { hideModal(); loadLLMGroups(); loadLLMConfigs(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function deleteLLMGroup(id) {
    if (!confirm('确定要删除此分组吗？')) return;
    fetch('/api/llm-groups', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function() { loadLLMGroups(); loadLLMConfigs(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function showLLMForm(config) {
    var c = config || {id: '', name: '', type: 'openai', base_url: '', api_key: '', model: '', is_active: false, start_date: '', end_date: '', group_id: ''};
    var groupOpts = llmGroups.map(function(g) { return '<option value="' + g.id + '"' + (g.id === c.group_id ? ' selected' : '') + '>' + g.name + '</option>'; }).join('');
    var typeOpts = '<option value="openai"' + (c.type === 'openai' ? ' selected' : '') + '>OpenAI</option><option value="anthropic"' + (c.type === 'anthropic' ? ' selected' : '') + '>Anthropic</option><option value="gemini"' + (c.type === 'gemini' ? ' selected' : '') + '>Gemini</option><option value="deepseek"' + (c.type === 'deepseek' ? ' selected' : '') + '>DeepSeek</option>';
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (c.id ? '编辑' : '添加') + ' LLM 配置</h3><div class="space-y-3"><input type="hidden" id="llm-id" value="' + c.id + '"><div><label class="text-sm text-slate-600">名称</label><input type="text" id="llm-name" value="' + c.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">分组</label><select id="llm-group" class="w-full px-3 py-2 border rounded-lg"><option value=""' + (!c.group_id ? ' selected' : '') + '>无分组</option>' + groupOpts + '</select></div><div><label class="text-sm text-slate-600">类型</label><select id="llm-type" class="w-full px-3 py-2 border rounded-lg">' + typeOpts + '</select></div><div><label class="text-sm text-slate-600">Base URL（可选）</label><input type="text" id="llm-url" value="' + c.base_url + '" placeholder="留空使用默认" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">API Key</label><input type="password" id="llm-key" value="' + c.api_key + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">模型</label><input type="text" id="llm-model" value="' + c.model + '" placeholder="例如: gpt-4o" class="w-full px-3 py-2 border rounded-lg"></div><div class="grid grid-cols-2 gap-3"><div><label class="text-sm text-slate-600">生效日期</label><input type="date" id="llm-start-date" value="' + (c.start_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">截止日期</label><input type="date" id="llm-end-date" value="' + (c.end_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div></div><div class="flex items-center gap-2"><input type="checkbox" id="llm-active"' + (c.is_active ? ' checked' : '') + '><label class="text-sm">设为当前使用</label></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveLLM()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function editLLM(id) {
    fetch('/api/llm').then(function(resp) { return resp.json(); }).then(function(configs) {
        var config = configs.find(function(c) { return c.id === id; });
        if (config) showLLMForm(config);
    });
}

function saveLLM() {
    var config = {id: document.getElementById('llm-id').value, name: document.getElementById('llm-name').value, type: document.getElementById('llm-type').value, base_url: document.getElementById('llm-url').value, api_key: document.getElementById('llm-key').value, model: document.getElementById('llm-model').value, start_date: document.getElementById('llm-start-date').value, end_date: document.getElementById('llm-end-date').value, is_active: document.getElementById('llm-active').checked, group_id: document.getElementById('llm-group').value};
    fetch('/api/llm', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(config)})
        .then(function() { hideModal(); loadLLMConfigs(); });
}

function deleteLLM(id) {
    if (!confirm('确定要删除此配置吗？')) return;
    fetch('/api/llm', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function() { loadLLMConfigs(); });
}

// ============ Search Config Functions ============
function loadSearchGroups() {
    fetch('/api/search-groups').then(function(resp) { return resp.json(); }).then(function(data) {
        searchGroups = data || [];
        var list = document.getElementById('search-groups-list');
        
        if (!searchGroups || searchGroups.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无分组</p>'; 
        } else {
            var html = '';
            searchGroups.forEach(function(g) { 
                html += '<div class="flex items-center justify-between p-2 bg-green-50 rounded-lg">';
                html += '<div><span class="font-bold text-sm">' + g.name + '</span>';
                html += '<p class="text-xs text-slate-400">' + (g.description || '') + '</p></div>';
                html += '<div class="flex gap-1">';
                html += '<button onclick="editSearchGroup(\'' + g.id + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
                html += '<button onclick="deleteSearchGroup(\'' + g.id + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
                html += '</div></div>'; 
            });
            list.innerHTML = html;
        }
        
        var filterSelect = document.getElementById('search-group-filter');
        if (filterSelect) {
            var currentValue = filterSelect.value;
            var opts = '<option value="">全部搜索组</option><option value="none">默认(无组)</option>';
            searchGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + g.name + '</option>'; });
            filterSelect.innerHTML = opts;
            filterSelect.value = currentValue;
        }
        
        var configFilterSelect = document.getElementById('search-config-group-filter');
        if (configFilterSelect) {
            var currentValue = configFilterSelect.value;
            var opts = '<option value="">全部分组</option><option value="none">默认(无组)</option>';
            searchGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + g.name + '</option>'; });
            configFilterSelect.innerHTML = opts;
            configFilterSelect.value = currentValue;
        }
    });
}

function loadSearchConfigs() {
    var groupFilter = document.getElementById('search-config-group-filter') ? document.getElementById('search-config-group-filter').value : '';
    
    fetch('/api/search').then(function(resp) { return resp.json(); }).then(function(configs) {
        var list = document.getElementById('search-list');
        var filteredConfigs = configs || [];
        
        if (groupFilter === 'none') {
            filteredConfigs = filteredConfigs.filter(function(c) { return !c.group_id || c.group_id === ''; });
        } else if (groupFilter) {
            filteredConfigs = filteredConfigs.filter(function(c) { return c.group_id === groupFilter; });
        }
        
        if (!filteredConfigs || filteredConfigs.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无配置</p>'; 
            return; 
        }
        
        var html = '';
        filteredConfigs.forEach(function(c) {
            var groupName = getSearchGroupName(c.group_id);
            var ringClass = c.is_active ? 'ring-2 ring-green-500' : '';
            
            html += '<div class="flex items-center justify-between p-3 bg-slate-50 rounded-lg ' + ringClass + '">';
            html += '<div><span class="font-bold">' + c.name + '</span>';
            if (c.is_active) html += '<span class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">当前使用</span>';
            if (groupName) html += '<span class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">' + groupName + '</span>';
            html += '<p class="text-sm text-slate-500">类型: ' + c.type + '</p>';
            html += '<p class="text-xs text-slate-400">有效期: ' + (c.start_date || '无限制') + ' ~ ' + (c.end_date || '永久') + '</p></div>';
            html += '<div class="flex gap-2">';
            html += '<button onclick="editSearch(\'' + c.id + '\')" class="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm">编辑</button>';
            html += '<button onclick="deleteSearch(\'' + c.id + '\')" class="px-3 py-1 bg-red-100 text-red-700 rounded text-sm">删除</button>';
            html += '</div></div>';
        });
        list.innerHTML = html;
    });
}

function showSearchGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + ' 搜索分组</h3><div class="space-y-3"><input type="hidden" id="search-group-id" value="' + g.id + '"><div><label class="text-sm text-slate-600">分组名称</label><input type="text" id="search-group-name" value="' + g.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">描述</label><input type="text" id="search-group-desc" value="' + (g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveSearchGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function editSearchGroup(id) {
    var group = searchGroups.find(function(g) { return g.id === id; });
    if (group) showSearchGroupForm(group);
}

function saveSearchGroup() {
    var group = {id: document.getElementById('search-group-id').value, name: document.getElementById('search-group-name').value, description: document.getElementById('search-group-desc').value};
    if (!group.name) { alert('分组名称不能为空'); return; }
    
    fetch('/api/search-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(group)})
        .then(function() { hideModal(); loadSearchGroups(); loadSearchConfigs(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function deleteSearchGroup(id) {
    if (!confirm('确定要删除此分组吗？')) return;
    fetch('/api/search-groups', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function() { loadSearchGroups(); loadSearchConfigs(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function showSearchForm(config) {
    var c = config || {id: '', name: '', type: 'tavily', api_key: '', is_active: false, start_date: '', end_date: '', group_id: ''};
    var groupOpts = searchGroups.map(function(g) { return '<option value="' + g.id + '"' + (g.id === c.group_id ? ' selected' : '') + '>' + g.name + '</option>'; }).join('');
    var typeOpts = '<option value="tavily"' + (c.type === 'tavily' ? ' selected' : '') + '>Tavily</option><option value="serper"' + (c.type === 'serper' ? ' selected' : '') + '>Serper</option><option value="bing"' + (c.type === 'bing' ? ' selected' : '') + '>Bing</option>';
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (c.id ? '编辑' : '添加') + ' 搜索引擎配置</h3><div class="space-y-3"><input type="hidden" id="search-id" value="' + c.id + '"><div><label class="text-sm text-slate-600">名称</label><input type="text" id="search-name" value="' + c.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">分组</label><select id="search-group" class="w-full px-3 py-2 border rounded-lg"><option value=""' + (!c.group_id ? ' selected' : '') + '>无分组</option>' + groupOpts + '</select></div><div><label class="text-sm text-slate-600">类型</label><select id="search-type" class="w-full px-3 py-2 border rounded-lg">' + typeOpts + '</select></div><div><label class="text-sm text-slate-600">API Key</label><input type="password" id="search-key" value="' + c.api_key + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="grid grid-cols-2 gap-3"><div><label class="text-sm text-slate-600">生效日期</label><input type="date" id="search-start-date" value="' + (c.start_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">截止日期</label><input type="date" id="search-end-date" value="' + (c.end_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div></div><div class="flex items-center gap-2"><input type="checkbox" id="search-active"' + (c.is_active ? ' checked' : '') + '><label class="text-sm">设为当前使用</label></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveSearch()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function editSearch(id) {
    fetch('/api/search').then(function(resp) { return resp.json(); }).then(function(configs) {
        var config = configs.find(function(c) { return c.id === id; });
        if (config) showSearchForm(config);
    });
}

function saveSearch() {
    var config = {id: document.getElementById('search-id').value, name: document.getElementById('search-name').value, type: document.getElementById('search-type').value, api_key: document.getElementById('search-key').value, start_date: document.getElementById('search-start-date').value, end_date: document.getElementById('search-end-date').value, is_active: document.getElementById('search-active').checked, group_id: document.getElementById('search-group').value};
    fetch('/api/search', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(config)})
        .then(function() { hideModal(); loadSearchConfigs(); });
}

function deleteSearch(id) {
    if (!confirm('确定要删除此配置吗？')) return;
    fetch('/api/search', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function() { loadSearchConfigs(); });
}

// ============ Settings Functions ============
function changePassword() {
    var oldPwd = document.getElementById('old-password').value;
    var newPwd = document.getElementById('new-password').value;
    if (!oldPwd || !newPwd) { alert('请输入当前密码和新密码'); return; }
    
    fetch('/api/password', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({old_password: oldPwd, new_password: newPwd})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                alert('密码修改成功'); 
                document.getElementById('old-password').value = ''; 
                document.getElementById('new-password').value = ''; 
            } else { 
                alert('修改失败: ' + result.error); 
            } 
        });
}

function changePorts() {
    var managePort = parseInt(document.getElementById('manage-port').value);
    var authPort = parseInt(document.getElementById('auth-port').value);
    if (!managePort || !authPort) { alert('请输入有效的端口号'); return; }
    
    fetch('/api/ports', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({manage_port: managePort, auth_port: authPort})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                alert('端口配置已保存，请重启服务生效'); 
            } else { 
                alert('保存失败: ' + result.error); 
            } 
        });
}

function loadRequestLimits() {
    fetch('/api/settings/request-limits').then(function(resp) { return resp.json(); }).then(function(data) {
        document.getElementById('daily-request-limit').value = data.daily_request_limit || 5;
        document.getElementById('daily-email-limit').value = data.daily_email_limit || 5;
    }).catch(function() {
        document.getElementById('daily-request-limit').value = 5;
        document.getElementById('daily-email-limit').value = 5;
    });
}

function saveRequestLimits() {
    var dailyRequestLimit = parseInt(document.getElementById('daily-request-limit').value) || 5;
    var dailyEmailLimit = parseInt(document.getElementById('daily-email-limit').value) || 5;
    
    fetch('/api/settings/request-limits', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({daily_request_limit: dailyRequestLimit, daily_email_limit: dailyEmailLimit})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                alert('请求限制设置已保存'); 
            } else { 
                alert('保存失败: ' + result.error); 
            } 
        });
}

function loadSSLConfig() {
    fetch('/api/ssl').then(function(resp) { return resp.json(); }).then(function(config) {
        document.getElementById('use-ssl').checked = config.use_ssl;
        document.getElementById('ssl-cert').value = config.ssl_cert || '';
        document.getElementById('ssl-key').value = config.ssl_key || '';
        toggleSSLFields();
    });
}

function toggleSSLFields() {
    var useSSL = document.getElementById('use-ssl').checked;
    document.getElementById('ssl-fields').classList.toggle('hidden', !useSSL);
}

function saveSSLConfig() {
    var useSSL = document.getElementById('use-ssl').checked;
    var sslCert = document.getElementById('ssl-cert').value;
    var sslKey = document.getElementById('ssl-key').value;
    
    fetch('/api/ssl', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({use_ssl: useSSL, ssl_cert: sslCert, ssl_key: sslKey})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                alert(result.message); 
            } else { 
                alert('保存失败: ' + result.error); 
            } 
        });
}

function showForceDeleteLicense() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold text-red-600 mb-4">⚠️ 强制删除序列号</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">输入要删除的序列号</label><input type="text" id="force-delete-sn" placeholder="XXXX-XXXX-XXXX-XXXX" class="w-full px-3 py-2 border rounded-lg font-mono"></div><p class="text-xs text-red-500">警告：此操作将永久删除该序列号及其所有相关记录（包括邮箱申请记录），不可恢复！</p><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doForceDeleteLicense()" class="flex-1 py-2 bg-red-600 text-white rounded-lg">确认删除</button></div></div></div>');
}

function doForceDeleteLicense() {
    var sn = document.getElementById('force-delete-sn').value.trim().toUpperCase();
    if (!sn) { alert('请输入序列号'); return; }
    if (!confirm('确定要强制删除序列号 ' + sn + ' 吗？\n\n此操作将删除：\n- 序列号本身\n- 相关的邮箱申请记录\n\n此操作不可恢复！')) return;
    
    fetch('/api/licenses/force-delete', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            hideModal(); 
            if (result.success) { 
                alert('序列号 ' + sn + ' 已被强制删除\n\n' + result.message); 
                loadLicenses(); 
                loadEmailRecords(); 
            } else { 
                alert('删除失败: ' + result.error); 
            } 
        });
}

// ============ Initialization ============
document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('use-ssl').addEventListener('change', toggleSSLFields);
    
    Promise.all([
        fetch('/api/llm-groups').then(function(r) { return r.json(); }),
        fetch('/api/search-groups').then(function(r) { return r.json(); }),
        fetch('/api/license-groups').then(function(r) { return r.json(); })
    ]).then(function(results) {
        llmGroups = results[0] || [];
        searchGroups = results[1] || [];
        licenseGroups = results[2] || [];
        
        loadLLMGroups();
        loadSearchGroups();
        loadLicenseGroups();
        loadLicenses();
        loadLLMConfigs();
        loadSearchConfigs();
        loadEmailRecords();
        loadSSLConfig();
        loadFilterSettings();
        loadBlacklist();
        loadWhitelist();
        loadConditions();
        loadRequestLimits();
    });
});
</script>
</body>
</html>`
