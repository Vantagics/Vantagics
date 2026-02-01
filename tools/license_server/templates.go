package main

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
                    <img src="data:image/png;base64,{{.CaptchaImage}}" alt="验证码" class="h-10 rounded border">
                    <button type="button" onclick="location.reload()" class="text-sm text-blue-600 hover:text-blue-800">换一张</button>
                </div>
                <input type="text" name="captcha" required
                    class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                    placeholder="请输入图中数字">
            </div>
            <button type="submit"
                class="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700 transition-colors font-medium">
                登录
            </button>
        </form>
    </div>
</body>
</html>`

var dashboardHTML = getDashboardHTML()

func getDashboardHTML() string {
    return htmlHead + htmlBody + htmlScript + htmlEnd
}

const htmlHead = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>VantageData License Server - 管理面板</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
`

const htmlBody = `<body class="bg-slate-100 min-h-screen">
    <nav class="bg-white shadow-sm border-b border-slate-200">
        <div class="max-w-7xl mx-auto px-4 py-3 flex justify-between items-center">
            <h1 class="text-xl font-bold text-slate-800">🔐 VantageData License Server</h1>
            <div class="flex items-center gap-4">
                <span class="text-sm text-slate-500">管理端口: {{.ManagePort}} | 授权端口: {{.AuthPort}}</span>
                <a href="/logout" class="text-sm text-red-600 hover:text-red-700">退出登录</a>
            </div>
        </div>
    </nav>
    <div class="max-w-7xl mx-auto px-4 py-6">
        <div class="flex gap-2 mb-6">
            <button onclick="showTab('licenses')" id="tab-licenses" class="tab-btn px-4 py-2 rounded-lg bg-blue-600 text-white">序列号管理</button>
            <button onclick="showTab('emails')" id="tab-emails" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">邮箱申请记录</button>
            <button onclick="showTab('email-filter')" id="tab-email-filter" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">邮箱过滤</button>
            <button onclick="showTab('license-groups')" id="tab-license-groups" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">序列号分组</button>
            <button onclick="showTab('llm')" id="tab-llm" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">LLM 配置</button>
            <button onclick="showTab('search')" id="tab-search" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">搜索引擎配置</button>
            <button onclick="showTab('settings')" id="tab-settings" class="tab-btn px-4 py-2 rounded-lg bg-slate-200 text-slate-700">系统设置</button>
        </div>
        <div id="panel-licenses" class="tab-panel">
            <div class="bg-white rounded-xl shadow-sm p-6">
                <div class="flex justify-between items-center mb-4">
                    <h2 class="text-lg font-bold text-slate-800">序列号列表</h2>
                    <div class="flex items-center gap-2">
                    <select id="license-group-filter" onchange="searchLicenses()" class="px-3 py-1.5 border rounded-lg text-sm">
                        <option value="">全部序列号组</option>
                        <option value="none">默认(无组)</option>
                    </select>
                    <select id="llm-group-filter" onchange="searchLicenses()" class="px-3 py-1.5 border rounded-lg text-sm">
                        <option value="">全部LLM组</option>
                        <option value="none">默认(无组)</option>
                    </select>
                    <select id="search-group-filter" onchange="searchLicenses()" class="px-3 py-1.5 border rounded-lg text-sm">
                        <option value="">全部搜索组</option>
                        <option value="none">默认(无组)</option>
                    </select>
                        <input type="text" id="license-search" placeholder="搜索序列号或描述..." class="px-3 py-1.5 border rounded-lg text-sm w-48" onkeyup="if(event.key==='Enter')searchLicenses()">
                        <button onclick="searchLicenses()" class="px-3 py-1.5 bg-blue-600 text-white rounded-lg text-sm">搜索</button>
                        <button onclick="deleteUnusedByGroup()" class="px-3 py-1.5 bg-red-600 text-white rounded-lg text-sm">🗑️ 删除未用</button>
                        <button onclick="createLicense()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 单个生成</button>
                        <button onclick="batchCreateLicense()" class="px-3 py-1.5 bg-purple-600 text-white rounded-lg text-sm">+ 批量生成</button>
                    </div>
                </div>
                <div id="licenses-list" class="space-y-2"></div>
                <div id="license-pagination" class="flex justify-center items-center gap-2 mt-4"></div>
            </div>
        </div>
        <div id="panel-emails" class="tab-panel hidden">
            <div class="bg-white rounded-xl shadow-sm p-6">
                <div class="flex justify-between items-center mb-4">
                    <h2 class="text-lg font-bold text-slate-800">邮箱申请记录</h2>
                    <div class="flex items-center gap-2">
                        <input type="text" id="email-search" placeholder="搜索邮箱或序列号..." class="px-3 py-1.5 border rounded-lg text-sm w-64" onkeyup="if(event.key==='Enter')searchEmails()">
                        <button onclick="searchEmails()" class="px-3 py-1.5 bg-blue-600 text-white rounded-lg text-sm">搜索</button>
                    </div>
                </div>
                <div id="email-records-list" class="space-y-2"></div>
                <div id="email-pagination" class="flex justify-center items-center gap-2 mt-4"></div>
            </div>
        </div>
        <div id="panel-email-filter" class="tab-panel hidden">
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div class="bg-white rounded-xl shadow-sm p-6 lg:col-span-2">
                    <h2 class="text-lg font-bold text-slate-800 mb-4">过滤模式设置</h2>
                    <div class="flex items-center gap-6 mb-4">
                        <label class="flex items-center gap-2"><input type="checkbox" id="blacklist-enabled" class="w-4 h-4" onchange="saveFilterSettings()"><span class="text-sm">启用黑名单</span></label>
                        <label class="flex items-center gap-2"><input type="checkbox" id="whitelist-enabled" class="w-4 h-4" onchange="saveFilterSettings()"><span class="text-sm">启用白名单</span></label>
                    </div>
                    <p class="text-xs text-slate-500 mb-4">* 默认启用黑名单模式。同时启用时，白名单优先</p>
                    
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">白名单</h2>
                        <button onclick="showAddWhitelist()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加</button>
                    </div>
                    <p class="text-xs text-slate-500 mb-3">* 可为白名单项指定LLM和搜索引擎组，申请的SN将绑定到指定组</p>
                    <div id="whitelist-items" class="space-y-2 max-h-96 overflow-y-auto"></div>
                </div>
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-bold text-slate-800">黑名单</h2>
                        <button onclick="showAddBlacklist()" class="px-3 py-1.5 bg-red-600 text-white rounded-lg text-sm">+ 添加</button>
                    </div>
                    <div id="blacklist-items" class="space-y-2 max-h-96 overflow-y-auto"></div>
                </div>
            </div>
        </div>
        <div id="panel-license-groups" class="tab-panel hidden">
            <div class="bg-white rounded-xl shadow-sm p-6">
                <div class="flex justify-between items-center mb-4">
                    <h2 class="text-lg font-bold text-slate-800">序列号分组</h2>
                    <button onclick="showLicenseGroupForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加</button>
                </div>
                <div id="license-groups-list" class="space-y-2"></div>
            </div>
        </div>
        
                <div id="panel-llm" class="tab-panel hidden">
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
                        <button onclick="showLLMForm()" class="px-4 py-2 bg-green-600 text-white rounded-lg text-sm">+ 添加配置</button>
                    </div>
                    <div id="llm-list" class="space-y-2"></div>
                </div>
            </div>
        </div>
        <div id="panel-search" class="tab-panel hidden">
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
                        <button onclick="showSearchForm()" class="px-4 py-2 bg-green-600 text-white rounded-lg text-sm">+ 添加配置</button>
                    </div>
                    <div id="search-list" class="space-y-2"></div>
                </div>
            </div>
        </div>
        <div id="panel-settings" class="tab-panel hidden">
            <div class="grid grid-cols-2 gap-6">
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <h2 class="text-lg font-bold text-slate-800 mb-4">修改用户名</h2>
                    <div class="space-y-3">
                        <div><label class="text-sm text-slate-600">当前用户名</label><input type="text" id="current-username" readonly class="w-full px-3 py-2 border rounded-lg bg-slate-50"></div>
                        <div><label class="text-sm text-slate-600">新用户名</label><input type="text" id="new-username" placeholder="输入新用户名" class="w-full px-3 py-2 border rounded-lg"></div>
                        <button onclick="changeUsername()" class="w-full bg-blue-600 text-white py-2 rounded-lg">修改用户名</button>
                    </div>
                </div>
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
                <div class="bg-white rounded-xl shadow-sm p-6">
                    <h2 class="text-lg font-bold text-slate-800 mb-4">请求限制</h2>
                    <div class="space-y-3">
                        <div><label class="text-sm text-slate-600">每日请求次数限制 (同一IP)</label><input type="number" id="daily-request-limit" min="1" max="100" value="5" class="w-full px-3 py-2 border rounded-lg"></div>
                        <div><label class="text-sm text-slate-600">每日邮箱数限制 (同一IP)</label><input type="number" id="daily-email-limit" min="1" max="100" value="5" class="w-full px-3 py-2 border rounded-lg"></div>
                        <button onclick="saveRateLimits()" class="w-full bg-blue-600 text-white py-2 rounded-lg">保存限制设置</button>
                        <p class="text-xs text-slate-500">* 限制同一IP每日申请SN的次数和使用不同邮箱的数量</p>
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
            </div>
        </div>
    </div>
    <div id="modal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white rounded-xl shadow-xl w-[500px] max-h-[80vh] overflow-auto">
            <div id="modal-content"></div>
        </div>
    </div>
`

const htmlScript = `<script>
function showTab(name) {
    document.querySelectorAll('.tab-panel').forEach(function(p) { p.classList.add('hidden'); });
    document.querySelectorAll('.tab-btn').forEach(function(b) {
        b.classList.remove('bg-blue-600', 'text-white');
        b.classList.add('bg-slate-200', 'text-slate-700');
    });
    document.getElementById('panel-' + name).classList.remove('hidden');
    var btn = document.getElementById('tab-' + name);
    btn.classList.remove('bg-slate-200', 'text-slate-700');
    btn.classList.add('bg-blue-600', 'text-white');
}

function showModal(content) {
    document.getElementById('modal-content').innerHTML = content;
    document.getElementById('modal').classList.remove('hidden');
    document.getElementById('modal').classList.add('flex');
}

function hideModal() {
    document.getElementById('modal').classList.add('hidden');
    document.getElementById('modal').classList.remove('flex');
}

var licenseCurrentPage = 1;
var licenseSearchTerm = '';
var llmGroups = [];
var searchGroups = [];
var licenseGroups = [];
var emailCurrentPage = 1;
var emailSearchTerm = '';

function getLLMGroupName(groupId) {
    if (!groupId) return '';
    var group = llmGroups.find(function(g) { return g.id === groupId; });
    return group ? group.name : '';
}

function getSearchGroupName(groupId) {
    if (!groupId) return '';
    var group = searchGroups.find(function(g) { return g.id === groupId; });
    return group ? group.name : '';
}

function getLicenseGroupName(groupId) {
    if (!groupId) return '';
    var group = licenseGroups.find(function(g) { return g.id === groupId; });
    return group ? group.name : '';
}

function loadLicenses(page, search, licenseGroupFilter, llmGroupFilter, searchGroupFilter) {
    page = page || 1;
    search = search || '';
    licenseGroupFilter = licenseGroupFilter !== undefined ? licenseGroupFilter : '';
    llmGroupFilter = llmGroupFilter !== undefined ? llmGroupFilter : '';
    searchGroupFilter = searchGroupFilter !== undefined ? searchGroupFilter : '';
    licenseCurrentPage = page;
    licenseSearchTerm = search;
    var params = new URLSearchParams({page: page.toString(), pageSize: '15', search: search});
    if (licenseGroupFilter) params.append('license_group', licenseGroupFilter);
    if (llmGroupFilter) params.append('llm_group', llmGroupFilter);
    if (searchGroupFilter) params.append('search_group', searchGroupFilter);
    fetch('/api/licenses/search?' + params)
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            var list = document.getElementById('licenses-list');
            if (!data.licenses || data.licenses.length === 0) {
                list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无序列号</p>';
                document.getElementById('license-pagination').innerHTML = '';
                return;
            }
            var now = new Date();
            var html = '';
            data.licenses.forEach(function(l) {
                var expiresAt = new Date(l.expires_at);
                var isExpired = expiresAt < now;
                var expireClass = isExpired ? 'text-red-600' : '';
                var llmGroupName = getLLMGroupName(l.llm_group_id);
                var searchGroupName = getSearchGroupName(l.search_group_id);
                var licenseGroupName = getLicenseGroupName(l.license_group_id);
                var licenseType = l.daily_analysis === 0 ? '正式版' : '试用版';
                var licenseTypeClass = l.daily_analysis === 0 ? 'bg-green-100 text-green-700' : 'bg-orange-100 text-orange-700';
                var opacityClass = !l.is_active ? 'opacity-50' : '';
                html += '<div class="flex items-center justify-between p-3 bg-slate-50 rounded-lg ' + opacityClass + '">';
                html += '<div><div class="flex items-center gap-2"><code class="text-lg font-mono font-bold text-blue-600">' + l.sn + '</code>';
                html += '<span class="px-2 py-0.5 text-xs rounded ' + licenseTypeClass + '">' + licenseType + '</span></div>';
                html += '<p class="text-sm text-slate-500">' + (l.description || '无描述') + '</p>';
                html += '<p class="text-xs text-slate-400">创建: ' + new Date(l.created_at).toLocaleDateString() + ' | ';
                html += '<span class="' + expireClass + '">过期: ' + expiresAt.toLocaleDateString() + (isExpired ? ' (已过期)' : '') + '</span> | ';
                html += '使用: ' + l.usage_count + '次 | 每日分析: ' + (l.daily_analysis === 0 ? '无限' : l.daily_analysis + '次') + '</p>';
                html += '<p class="text-xs text-slate-400">';
                html += '序列号分组: <span class="text-purple-600">' + (licenseGroupName || '默认') + '</span> | ';
                html += 'LLM分组: <span class="text-blue-600">' + (llmGroupName || '默认') + '</span> | ';
                html += '搜索分组: <span class="text-green-600">' + (searchGroupName || '默认') + '</span></p></div>';
                html += '<div class="flex gap-2">';
                html += '<button onclick="setLicenseGroups(\'' + l.sn + '\', \'' + (l.license_group_id || '') + '\', \'' + (l.llm_group_id || '') + '\', \'' + (l.search_group_id || '') + '\')" class="px-3 py-1 bg-indigo-100 text-indigo-700 rounded text-sm">分组</button>';
                html += '<button onclick="extendLicense(\'' + l.sn + '\', \'' + (l.expires_at || '') + '\')" class="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm">展期</button>';
                html += '<button onclick="setDailyAnalysis(\'' + l.sn + '\', ' + l.daily_analysis + ')" class="px-3 py-1 bg-purple-100 text-purple-700 rounded text-sm">分析次数</button>';
                html += '<button onclick="toggleLicense(\'' + l.sn + '\')" class="px-3 py-1 ' + (l.is_active ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700') + ' rounded text-sm">' + (l.is_active ? '禁用' : '启用') + '</button>';
                html += '<button onclick="deleteLicense(\'' + l.sn + '\')" class="px-3 py-1 bg-red-100 text-red-700 rounded text-sm">删除</button>';
                html += '</div></div>';
            });
            list.innerHTML = html;
            var pagination = document.getElementById('license-pagination');
            var paginationHTML = '<span class="text-sm text-slate-500">共 ' + data.total + ' 条记录</span>';
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

function searchLicenses(page) { 
    page = page || 1;
    var search = document.getElementById('license-search').value;
    var licenseGroupFilter = document.getElementById('license-group-filter').value;
    var llmGroupFilter = document.getElementById('llm-group-filter').value;
    var searchGroupFilter = document.getElementById('search-group-filter').value;
    loadLicenses(page, search, licenseGroupFilter, llmGroupFilter, searchGroupFilter); 
}

function createLicense() {
    var llmOpts = llmGroups.map(function(g) { return '<option value="' + g.id + '">' + g.name + '</option>'; }).join('');
    var searchOpts = searchGroups.map(function(g) { return '<option value="' + g.id + '">' + g.name + '</option>'; }).join('');
    var licenseGroupOpts = licenseGroups.map(function(g) { return '<option value="' + g.id + '">' + g.name + '</option>'; }).join('');
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">生成新序列号</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">授权类型</label><select id="license-type" onchange="updateDailyAnalysisDefault()" class="w-full px-3 py-2 border rounded-lg"><option value="trial">试用版 (有分析次数限制)</option><option value="official">正式版 (无限制)</option></select></div><div><label class="text-sm text-slate-600">序列号分组</label><select id="license-group-create" class="w-full px-3 py-2 border rounded-lg"><option value="">默认</option>' + licenseGroupOpts + '</select></div><div><label class="text-sm text-slate-600">描述</label><input type="text" id="license-desc" placeholder="例如: 客户A" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">有效期（天）</label><input type="number" id="license-days" value="365" class="w-full px-3 py-2 border rounded-lg"></div><div id="daily-analysis-row"><label class="text-sm text-slate-600">每日分析次数</label><input type="number" id="license-daily" value="20" min="0" class="w-full px-3 py-2 border rounded-lg"><p class="text-xs text-slate-400 mt-1">0 表示无限制</p></div><div><label class="text-sm text-slate-600">LLM 分组</label><select id="license-llm-group-create" class="w-full px-3 py-2 border rounded-lg"><option value="">默认</option>' + llmOpts + '</select></div><div><label class="text-sm text-slate-600">搜索 分组</label><select id="license-search-group-create" class="w-full px-3 py-2 border rounded-lg"><option value="">默认</option>' + searchOpts + '</select></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doCreateLicense()" class="flex-1 py-2 bg-green-600 text-white rounded-lg">生成</button></div></div></div>');
}

function batchCreateLicense() {
    var llmOpts = llmGroups.map(function(g) { return '<option value="' + g.id + '">' + g.name + '</option>'; }).join('');
    var searchOpts = searchGroups.map(function(g) { return '<option value="' + g.id + '">' + g.name + '</option>'; }).join('');
    var licenseGroupOpts = licenseGroups.map(function(g) { return '<option value="' + g.id + '">' + g.name + '</option>'; }).join('');
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">批量生成序列号</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">描述</label><input type="text" id="batch-license-desc" placeholder="例如: 批次2026-02" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">有效期（天）</label><input type="number" id="batch-license-days" value="365" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">生成数量</label><input type="number" id="batch-license-count" value="100" min="1" max="1000" class="w-full px-3 py-2 border rounded-lg"><p class="text-xs text-slate-400 mt-1">最多1000个</p></div><div><label class="text-sm text-slate-600">每日分析次数</label><input type="number" id="batch-license-daily" value="20" min="0" class="w-full px-3 py-2 border rounded-lg"><p class="text-xs text-slate-400 mt-1">0 表示无限制</p></div><div><label class="text-sm text-slate-600">序列号分组</label><select id="batch-license-group" class="w-full px-3 py-2 border rounded-lg"><option value="">默认</option>' + licenseGroupOpts + '</select></div><div><label class="text-sm text-slate-600">LLM 分组</label><select id="batch-license-llm-group" class="w-full px-3 py-2 border rounded-lg"><option value="">默认</option>' + llmOpts + '</select></div><div><label class="text-sm text-slate-600">搜索 分组</label><select id="batch-license-search-group" class="w-full px-3 py-2 border rounded-lg"><option value="">默认</option>' + searchOpts + '</select></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doBatchCreateLicense()" class="flex-1 py-2 bg-purple-600 text-white rounded-lg">批量生成</button></div></div></div>');
}


function updateDailyAnalysisDefault() {
    var licenseType = document.getElementById('license-type').value;
    var dailyInput = document.getElementById('license-daily');
    if (licenseType === 'official') {
        dailyInput.value = '0';
    } else {
        dailyInput.value = '20';
    }
}

function doCreateLicense() {
    var desc = document.getElementById('license-desc').value;
    var days = parseInt(document.getElementById('license-days').value) || 365;
    var dailyAnalysis = parseInt(document.getElementById('license-daily').value);
    var licenseGroupId = document.getElementById('license-group-create').value;
    var llmGroupId = document.getElementById('license-llm-group-create').value;
    var searchGroupId = document.getElementById('license-search-group-create').value;
    fetch('/api/licenses/create', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({description: desc, days: days, daily_analysis: isNaN(dailyAnalysis) ? 20 : dailyAnalysis, license_group_id: licenseGroupId, llm_group_id: llmGroupId, search_group_id: searchGroupId})})
        .then(function(resp) { return resp.json(); })
        .then(function(license) { hideModal(); loadLicenses(licenseCurrentPage, licenseSearchTerm); alert('序列号已生成: ' + license.sn); });
}

function doBatchCreateLicense() {
    var desc = document.getElementById('batch-license-desc').value;
    var days = parseInt(document.getElementById('batch-license-days').value) || 365;
    var count = parseInt(document.getElementById('batch-license-count').value) || 100;
    var dailyAnalysis = parseInt(document.getElementById('batch-license-daily').value);
    var licenseGroupId = document.getElementById('batch-license-group').value;
    var llmGroupId = document.getElementById('batch-license-llm-group').value;
    var searchGroupId = document.getElementById('batch-license-search-group').value;
    fetch('/api/licenses/batch-create', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({description: desc, days: days, count: count, daily_analysis: isNaN(dailyAnalysis) ? 20 : dailyAnalysis, license_group_id: licenseGroupId, llm_group_id: llmGroupId, search_group_id: searchGroupId})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); loadLicenses(1, licenseSearchTerm); alert('成功生成 ' + result.count + ' 个序列号'); });
}

function setLicenseGroups(sn, currentLicenseGroup, currentLLMGroup, currentSearchGroup) {
    var licenseGroupOpts = licenseGroups.map(function(g) { return '<option value=\"' + g.id + '\"' + (g.id === currentLicenseGroup ? ' selected' : '') + '>' + g.name + '</option>'; }).join('');
    var llmOpts = llmGroups.map(function(g) { return '<option value="' + g.id + '"' + (g.id === currentLLMGroup ? ' selected' : '') + '>' + g.name + '</option>'; }).join('');
    var searchOpts = searchGroups.map(function(g) { return '<option value="' + g.id + '"' + (g.id === currentSearchGroup ? ' selected' : '') + '>' + g.name + '</option>'; }).join('');
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">设置分组</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">序列号</label><input type="text" value="' + sn + '" readonly class="w-full px-3 py-2 border rounded-lg bg-slate-100 font-mono"></div><div><label class="text-sm text-slate-600">序列号 分组</label><select id="license-group" class="w-full px-3 py-2 border rounded-lg"><option value=""' + (!currentLicenseGroup ? ' selected' : '') + '>默认</option>' + licenseGroupOpts + '</select></div><div><label class="text-sm text-slate-600">LLM 分组</label><select id="license-llm-group" class="w-full px-3 py-2 border rounded-lg"><option value=""' + (!currentLLMGroup ? ' selected' : '') + '>默认</option>' + llmOpts + '</select></div><div><label class="text-sm text-slate-600">搜索 分组</label><select id="license-search-group" class="w-full px-3 py-2 border rounded-lg"><option value=""' + (!currentSearchGroup ? ' selected' : '') + '>默认</option>' + searchOpts + '</select></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doSetLicenseGroups(\'' + sn + '\')" class="flex-1 py-2 bg-indigo-600 text-white rounded-lg">保存</button></div></div></div>');
}

function doSetLicenseGroups(sn) {
    var licenseGroupId = document.getElementById('license-group').value;
    var llmGroupId = document.getElementById('license-llm-group').value;
    var searchGroupId = document.getElementById('license-search-group').value;
    fetch('/api/licenses/set-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, license_group_id: licenseGroupId, llm_group_id: llmGroupId, search_group_id: searchGroupId})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadLicenses(licenseCurrentPage, licenseSearchTerm); alert('分组设置成功'); } else { alert('设置失败: ' + result.error); } });
}

function setDailyAnalysis(sn, currentValue) {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">设置每日分析次数</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">序列号</label><input type="text" value="' + sn + '" readonly class="w-full px-3 py-2 border rounded-lg bg-slate-100 font-mono"></div><div><label class="text-sm text-slate-600">每日分析次数</label><input type="number" id="daily-analysis-value" value="' + currentValue + '" min="0" class="w-full px-3 py-2 border rounded-lg"><p class="text-xs text-slate-400 mt-1">0 表示无限制</p></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doSetDailyAnalysis(\'' + sn + '\')" class="flex-1 py-2 bg-purple-600 text-white rounded-lg">保存</button></div></div></div>');
}

function doSetDailyAnalysis(sn) {
    var dailyAnalysis = parseInt(document.getElementById('daily-analysis-value').value) || 0;
    fetch('/api/licenses/set-daily', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, daily_analysis: dailyAnalysis})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadLicenses(licenseCurrentPage, licenseSearchTerm); alert('设置成功'); } else { alert('设置失败: ' + result.error); } });
}

function extendLicense(sn, expiresAt) {
    var expireDate = expiresAt ? new Date(expiresAt).toLocaleDateString() : '未设置';
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">展期序列号</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">序列号</label><input type="text" value="' + sn + '" readonly class="w-full px-3 py-2 border rounded-lg bg-slate-100 font-mono"></div><div><label class="text-sm text-slate-600">当前到期日期</label><input type="text" value="' + expireDate + '" readonly class="w-full px-3 py-2 border rounded-lg bg-slate-100"></div><div><label class="text-sm text-slate-600">延长天数</label><input type="number" id="extend-days" value="30" min="1" class="w-full px-3 py-2 border rounded-lg"></div><p class="text-xs text-slate-500">从当前到期日期开始计算</p><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doExtendLicense(\'' + sn + '\')" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">确认展期</button></div></div></div>');
}

function doExtendLicense(sn) {
    var days = parseInt(document.getElementById('extend-days').value) || 30;
    fetch('/api/licenses/extend', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, days: days})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadLicenses(licenseCurrentPage, licenseSearchTerm); alert('展期成功，新到期日期: ' + result.new_expiry); } else { alert('展期失败: ' + result.error); } });
}

function toggleLicense(sn) {
    fetch('/api/licenses/toggle', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function() { loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function deleteLicense(sn) {
    if (!confirm('确定要删除此序列号吗？')) return;
    fetch('/api/licenses/delete', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function() { loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function deleteUnusedByGroup() {
    var licenseGroupFilter = document.getElementById('license-group-filter').value;
    var llmGroupFilter = document.getElementById('llm-group-filter').value;
    var searchGroupFilter = document.getElementById('search-group-filter').value;
    
    if (!licenseGroupFilter && !llmGroupFilter && !searchGroupFilter) {
        alert('请至少选择一个分组条件来删除未使用的序列号');
        return;
    }
    
    var filterDesc = [];
    if (licenseGroupFilter) {
        var groupName = licenseGroupFilter === 'none' ? '默认(无组)' : getLicenseGroupName(licenseGroupFilter);
        filterDesc.push('序列号分组: ' + groupName);
    }
    if (llmGroupFilter) {
        var groupName = llmGroupFilter === 'none' ? '默认(无组)' : getLLMGroupName(llmGroupFilter);
        filterDesc.push('LLM分组: ' + groupName);
    }
    if (searchGroupFilter) {
        var groupName = searchGroupFilter === 'none' ? '默认(无组)' : getSearchGroupName(searchGroupFilter);
        filterDesc.push('搜索分组: ' + groupName);
    }
    
    var confirmMsg = '确定要删除以下条件的所有未使用序列号吗？\\n\\n' + filterDesc.join('\\n') + '\\n\\n⚠️ 此操作不可恢复！只会删除使用次数为0的序列号。';
    
    if (!confirm(confirmMsg)) return;
    
    var requestData = {};
    if (licenseGroupFilter) requestData.license_group_id = licenseGroupFilter;
    if (llmGroupFilter) requestData.llm_group_id = llmGroupFilter;
    if (searchGroupFilter) requestData.search_group_id = searchGroupFilter;
    
    fetch('/api/licenses/delete-unused-by-group', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(requestData)
    })
    .then(function(resp) { return resp.json(); })
    .then(function(result) {
        if (result.success) {
            alert(result.message || '删除成功');
            loadLicenses(licenseCurrentPage, licenseSearchTerm);
        } else {
            alert('删除失败: ' + (result.error || '未知错误'));
        }
    })
    .catch(function(err) {
        alert('删除失败: ' + err.message);
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
        if (!filteredConfigs || filteredConfigs.length === 0) { list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无配置</p>'; return; }
        var html = '';
        filteredConfigs.forEach(function(c) {
            var groupName = getLLMGroupName(c.group_id);
            var ringClass = c.is_active ? 'ring-2 ring-green-500' : '';
            html += '<div class="flex items-center justify-between p-3 bg-slate-50 rounded-lg ' + ringClass + '"><div><span class="font-bold">' + c.name + '</span>';
            if (c.is_active) html += '<span class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">当前使用</span>';
            if (groupName) html += '<span class="ml-2 px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded">' + groupName + '</span>';
            html += '<p class="text-sm text-slate-500">类型: ' + c.type + ' | 模型: ' + c.model + '</p>';
            html += '<p class="text-xs text-slate-400">URL: ' + (c.base_url || '默认') + '</p>';
            html += '<p class="text-xs text-slate-400">有效期: ' + (c.start_date || '无限制') + ' ~ ' + (c.end_date || '永久') + '</p></div>';
            html += '<div class="flex gap-2"><button onclick="editLLM(\'' + c.id + '\')" class="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm">编辑</button>';
            html += '<button onclick="deleteLLM(\'' + c.id + '\')" class="px-3 py-1 bg-red-100 text-red-700 rounded text-sm">删除</button></div></div>';
        });
        list.innerHTML = html;
    });
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
    fetch('/api/llm', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(config)}).then(function() { hideModal(); loadLLMConfigs(); });
}

function deleteLLM(id) {
    if (!confirm('确定要删除此配置吗？')) return;
    fetch('/api/llm', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})}).then(function() { loadLLMConfigs(); });
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
        if (!filteredConfigs || filteredConfigs.length === 0) { list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无配置</p>'; return; }
        var html = '';
        filteredConfigs.forEach(function(c) {
            var groupName = getSearchGroupName(c.group_id);
            var ringClass = c.is_active ? 'ring-2 ring-green-500' : '';
            html += '<div class="flex items-center justify-between p-3 bg-slate-50 rounded-lg ' + ringClass + '"><div><span class="font-bold">' + c.name + '</span>';
            if (c.is_active) html += '<span class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">当前使用</span>';
            if (groupName) html += '<span class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">' + groupName + '</span>';
            html += '<p class="text-sm text-slate-500">类型: ' + c.type + '</p>';
            html += '<p class="text-xs text-slate-400">有效期: ' + (c.start_date || '无限制') + ' ~ ' + (c.end_date || '永久') + '</p></div>';
            html += '<div class="flex gap-2"><button onclick="editSearch(\'' + c.id + '\')" class="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm">编辑</button>';
            html += '<button onclick="deleteSearch(\'' + c.id + '\')" class="px-3 py-1 bg-red-100 text-red-700 rounded text-sm">删除</button></div></div>';
        });
        list.innerHTML = html;
    });
}

function showSearchForm(config) {
    var c = config || {id: '', name: '', type: 'tavily', api_key: '', is_active: false, start_date: '', end_date: '', group_id: ''};
    var groupOpts = searchGroups.map(function(g) { return '<option value="' + g.id + '"' + (g.id === c.group_id ? ' selected' : '') + '>' + g.name + '</option>'; }).join('');
    var typeOpts = '<option value="tavily"' + (c.type === 'tavily' ? ' selected' : '') + '>Tavily</option><option value="serper"' + (c.type === 'serper' ? ' selected' : '') + '>Serper</option><option value="bing"' + (c.type === 'bing' ? ' selected' : '') + '>Bing</option>';
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (c.id ? '编辑' : '添加') + '搜索引擎配置</h3><div class="space-y-3"><input type="hidden" id="search-id" value="' + c.id + '"><div><label class="text-sm text-slate-600">名称</label><input type="text" id="search-name" value="' + c.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">分组</label><select id="search-group" class="w-full px-3 py-2 border rounded-lg"><option value=""' + (!c.group_id ? ' selected' : '') + '>无分组</option>' + groupOpts + '</select></div><div><label class="text-sm text-slate-600">类型</label><select id="search-type" class="w-full px-3 py-2 border rounded-lg">' + typeOpts + '</select></div><div><label class="text-sm text-slate-600">API Key</label><input type="password" id="search-key" value="' + c.api_key + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="grid grid-cols-2 gap-3"><div><label class="text-sm text-slate-600">生效日期</label><input type="date" id="search-start-date" value="' + (c.start_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">截止日期</label><input type="date" id="search-end-date" value="' + (c.end_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div></div><div class="flex items-center gap-2"><input type="checkbox" id="search-active"' + (c.is_active ? ' checked' : '') + '><label class="text-sm">设为当前使用</label></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveSearch()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function editSearch(id) {
    fetch('/api/search').then(function(resp) { return resp.json(); }).then(function(configs) {
        var config = configs.find(function(c) { return c.id === id; });
        if (config) showSearchForm(config);
    });
}

function saveSearch() {
    var config = {id: document.getElementById('search-id').value, name: document.getElementById('search-name').value, type: document.getElementById('search-type').value, api_key: document.getElementById('search-key').value, start_date: document.getElementById('search-start-date').value, end_date: document.getElementById('search-end-date').value, is_active: document.getElementById('search-active').checked, group_id: document.getElementById('search-group').value};
    fetch('/api/search', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(config)}).then(function() { hideModal(); loadSearchConfigs(); });
}

function deleteSearch(id) {
    if (!confirm('确定要删除此配置吗？')) return;
    fetch('/api/search', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})}).then(function() { loadSearchConfigs(); });
}

function loadUsername() {
    fetch('/api/username').then(function(resp) { return resp.json(); }).then(function(data) {
        document.getElementById('current-username').value = data.username || 'admin';
    });
}

function changeUsername() {
    var newUsername = document.getElementById('new-username').value.trim();
    if (!newUsername) { alert('请输入新用户名'); return; }
    fetch('/api/username', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({username: newUsername})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { if (result.success) { alert('用户名修改成功'); document.getElementById('new-username').value = ''; loadUsername(); } else { alert('修改失败: ' + result.error); } });
}

function changePassword() {
    var oldPwd = document.getElementById('old-password').value;
    var newPwd = document.getElementById('new-password').value;
    fetch('/api/password', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({old_password: oldPwd, new_password: newPwd})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { if (result.success) { alert('密码修改成功'); document.getElementById('old-password').value = ''; document.getElementById('new-password').value = ''; } else { alert('修改失败: ' + result.error); } });
}

function changePorts() {
    var managePort = parseInt(document.getElementById('manage-port').value);
    var authPort = parseInt(document.getElementById('auth-port').value);
    fetch('/api/ports', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({manage_port: managePort, auth_port: authPort})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { alert(result.message); });
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
        .then(function(result) { if (result.success) { alert(result.message); } else { alert('保存失败: ' + result.error); } });
}

function loadEmailRecords(page, search) {
    page = page || 1;
    search = search || '';
    emailCurrentPage = page;
    emailSearchTerm = search;
    var params = new URLSearchParams({page: page.toString(), pageSize: '15', search: search});
    fetch('/api/email-records?' + params).then(function(resp) { return resp.json(); }).then(function(data) {
        var list = document.getElementById('email-records-list');
        if (!data.records || data.records.length === 0) { list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无申请记录</p>'; document.getElementById('email-pagination').innerHTML = ''; return; }
        
        var sns = data.records.map(function(r) { return r.sn; });
        Promise.all(sns.map(function(sn) {
            return fetch('/api/licenses/search?search=' + encodeURIComponent(sn) + '&pageSize=1').then(function(r) { return r.json(); });
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
                var isActive = license.is_active !== false;
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
                html += '<button onclick="setLicenseGroups(\x27' + r.sn + '\x27, \x27' + (license.license_group_id || '') + '\x27, \x27' + (license.llm_group_id || '') + '\x27, \x27' + (license.search_group_id || '') + '\x27)" class="px-2 py-1 bg-indigo-100 text-indigo-700 rounded text-xs hover:bg-indigo-200">分组</button>';
                html += '<button onclick="extendLicense(\x27' + r.sn + '\x27, \x27' + (license.expires_at || '') + '\x27)" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs hover:bg-blue-200">展期</button>';
                html += '<button onclick="setDailyAnalysis(\x27' + r.sn + '\x27, ' + dailyAnalysis + ')" class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs hover:bg-purple-200">分析次数</button>';
                html += '<button onclick="toggleLicenseFromEmail(\x27' + r.sn + '\x27)" class="px-2 py-1 ' + (isActive ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700') + ' rounded text-xs hover:opacity-80">' + (isActive ? '禁用' : '启用') + '</button>';
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

function toggleLicenseFromEmail(sn) {
    fetch('/api/licenses/toggle', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function() { loadEmailRecords(emailCurrentPage, emailSearchTerm); });
}

function searchEmails() { loadEmailRecords(1, document.getElementById('email-search').value); }

function loadFilterSettings() {
    fetch('/api/email-filter').then(function(resp) { return resp.json(); }).then(function(data) {
        document.getElementById('whitelist-enabled').checked = data.whitelist_enabled;
        document.getElementById('blacklist-enabled').checked = data.blacklist_enabled;
    });
}

function saveFilterSettings() {
    var whitelistEnabled = document.getElementById('whitelist-enabled').checked;
    var blacklistEnabled = document.getElementById('blacklist-enabled').checked;
    fetch('/api/email-filter', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({whitelist_enabled: whitelistEnabled, blacklist_enabled: blacklistEnabled})});
}

function loadWhitelist() {
    Promise.all([fetch('/api/whitelist').then(function(r){return r.json();}), fetch('/api/llm-groups').then(function(r){return r.json();}), fetch('/api/search-groups').then(function(r){return r.json();})]).then(function(results) {
        var items = results[0];
        var llmGroups = results[1] || [];
        var searchGroups = results[2] || [];
        var list = document.getElementById('whitelist-items');
        if (!items || items.length === 0) { list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无白名单</p>'; return; }
        var html = '';
        items.forEach(function(item) {
            var llmGroupName = item.llm_group_id ? (llmGroups.find(function(g){return g.id===item.llm_group_id;}) || {}).name || item.llm_group_id : '无限制';
            var searchGroupName = item.search_group_id ? (searchGroups.find(function(g){return g.id===item.search_group_id;}) || {}).name || item.search_group_id : '无限制';
            html += '<div class="p-3 bg-green-50 rounded-lg"><div class="flex items-center justify-between"><code class="text-sm font-mono text-green-700">' + item.pattern + '</code><button onclick="deleteWhitelist(\'' + item.pattern + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button></div><div class="mt-2 text-xs text-slate-500"><span class="mr-3">LLM组: <span class="text-blue-600">' + llmGroupName + '</span></span><span>搜索组: <span class="text-purple-600">' + searchGroupName + '</span></span></div><p class="text-xs text-slate-400 mt-1">' + new Date(item.created_at).toLocaleString() + '</p></div>';
        });
        list.innerHTML = html;
    });
}

function loadBlacklist() {
    fetch('/api/blacklist').then(function(resp) { return resp.json(); }).then(function(items) {
        var list = document.getElementById('blacklist-items');
        if (!items || items.length === 0) { list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无黑名单</p>'; return; }
        var html = '';
        items.forEach(function(item) { html += '<div class="flex items-center justify-between p-2 bg-red-50 rounded-lg"><div><code class="text-sm font-mono text-red-700">' + item.pattern + '</code><p class="text-xs text-slate-400">' + new Date(item.created_at).toLocaleString() + '</p></div><button onclick="deleteBlacklist(\'' + item.pattern + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button></div>'; });
        list.innerHTML = html;
    });
}

function showAddWhitelist() {
    Promise.all([fetch('/api/llm-groups').then(function(r){return r.json();}), fetch('/api/search-groups').then(function(r){return r.json();})]).then(function(results) {
        var llmGroups = results[0] || [];
        var searchGroups = results[1] || [];
        var llmOptions = '<option value="">无限制（随机）</option>';
        llmGroups.forEach(function(g) { llmOptions += '<option value="' + g.id + '">' + g.name + '</option>'; });
        var searchOptions = '<option value="">无限制（随机）</option>';
        searchGroups.forEach(function(g) { searchOptions += '<option value="' + g.id + '">' + g.name + '</option>'; });
        showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">添加白名单</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">邮箱或域名</label><input type="text" id="whitelist-pattern" placeholder="例如: @company.com" class="w-full px-3 py-2 border rounded-lg"></div><p class="text-xs text-slate-500">* 以 @ 开头表示域名匹配</p><div><label class="text-sm text-slate-600">绑定LLM组</label><select id="whitelist-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmOptions + '</select></div><div><label class="text-sm text-slate-600">绑定搜索引擎组</label><select id="whitelist-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchOptions + '</select></div><p class="text-xs text-slate-500">* 不选择表示无限制，申请的SN将随机分配</p><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doAddWhitelist()" class="flex-1 py-2 bg-green-600 text-white rounded-lg">添加</button></div></div></div>');
    });
}

function showAddBlacklist() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">添加黑名单</h3><div class="space-y-3"><div><label class="text-sm text-slate-600">邮箱或域名</label><input type="text" id="blacklist-pattern" placeholder="例如: @spam.com" class="w-full px-3 py-2 border rounded-lg"></div><p class="text-xs text-slate-500">* 以 @ 开头表示域名匹配</p><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doAddBlacklist()" class="flex-1 py-2 bg-red-600 text-white rounded-lg">添加</button></div></div></div>');
}

function doAddWhitelist() {
    var pattern = document.getElementById('whitelist-pattern').value.trim();
    var llmGroupId = document.getElementById('whitelist-llm-group').value;
    var searchGroupId = document.getElementById('whitelist-search-group').value;
    if (!pattern) { alert('请输入邮箱或域名'); return; }
    fetch('/api/whitelist', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern, llm_group_id: llmGroupId, search_group_id: searchGroupId})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadWhitelist(); } else { alert('添加失败: ' + result.error); } });
}

function doAddBlacklist() {
    var pattern = document.getElementById('blacklist-pattern').value.trim();
    if (!pattern) { alert('请输入邮箱或域名'); return; }
    fetch('/api/blacklist', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadBlacklist(); } else { alert('添加失败: ' + result.error); } });
}

function deleteWhitelist(pattern) {
    if (!confirm('确定要删除此白名单项吗？')) return;
    fetch('/api/whitelist', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})}).then(function() { loadWhitelist(); });
}

function deleteBlacklist(pattern) {
    if (!confirm('确定要删除此黑名单项吗？')) return;
    fetch('/api/blacklist', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})}).then(function() { loadBlacklist(); });
}

function loadRateLimits() {
    fetch('/api/email-filter').then(function(resp) { return resp.json(); }).then(function(data) {
        document.getElementById('daily-request-limit').value = data.daily_request_limit || '5';
        document.getElementById('daily-email-limit').value = data.daily_email_limit || '5';
    });
}

function saveRateLimits() {
    var dailyRequestLimit = document.getElementById('daily-request-limit').value;
    var dailyEmailLimit = document.getElementById('daily-email-limit').value;
    fetch('/api/email-filter', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({daily_request_limit: dailyRequestLimit, daily_email_limit: dailyEmailLimit})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { if (result.success) { alert('保存成功'); } });
}

function loadLLMGroups() {
    fetch('/api/llm-groups').then(function(resp) { return resp.json(); }).then(function(data) {
        llmGroups = data || [];
        var list = document.getElementById('llm-groups-list');
        if (!llmGroups || llmGroups.length === 0) {
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无分组</p>';
        } else {
            var html = '';
            llmGroups.forEach(function(g) { html += '<div class="flex items-center justify-between p-2 bg-blue-50 rounded-lg"><div><span class="font-bold text-sm">' + g.name + '</span><p class="text-xs text-slate-400">' + (g.description || '') + '</p></div><div class="flex gap-1"><button onclick="editLLMGroup(\'' + g.id + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button><button onclick="deleteLLMGroup(\'' + g.id + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button></div></div>'; });
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

function loadSearchGroups() {
    fetch('/api/search-groups').then(function(resp) { return resp.json(); }).then(function(data) {
        searchGroups = data || [];
        var list = document.getElementById('search-groups-list');
        if (!searchGroups || searchGroups.length === 0) {
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无分组</p>';
        } else {
            var html = '';
            searchGroups.forEach(function(g) { html += '<div class="flex items-center justify-between p-2 bg-green-50 rounded-lg"><div><span class="font-bold text-sm">' + g.name + '</span><p class="text-xs text-slate-400">' + (g.description || '') + '</p></div><div class="flex gap-1"><button onclick="editSearchGroup(\'' + g.id + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button><button onclick="deleteSearchGroup(\'' + g.id + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button></div></div>'; });
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

function showLLMGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + ' LLM 分组</h3><div class="space-y-3"><input type="hidden" id="llm-group-id" value="' + g.id + '"><div><label class="text-sm text-slate-600">分组名称</label><input type="text" id="llm-group-name" value="' + g.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">描述</label><input type="text" id="llm-group-desc" value="' + (g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveLLMGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function showSearchGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + ' 搜索分组</h3><div class="space-y-3"><input type="hidden" id="search-group-id" value="' + g.id + '"><div><label class="text-sm text-slate-600">分组名称</label><input type="text" id="search-group-name" value="' + g.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">描述</label><input type="text" id="search-group-desc" value="' + (g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveSearchGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}


function loadLicenseGroups() {
    fetch('/api/license-groups').then(function(resp) { return resp.json(); }).then(function(data) {
        licenseGroups = data || [];
        var list = document.getElementById('license-groups-list');
        if (!licenseGroups || licenseGroups.length === 0) {
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无分组</p>';
        } else {
            var html = '';
            licenseGroups.forEach(function(g) { html += '<div class="flex items-center justify-between p-2 bg-purple-50 rounded-lg"><div><span class="font-bold text-sm">' + g.name + '</span><p class="text-xs text-slate-400">' + (g.description || '') + '</p></div><div class="flex gap-1"><button onclick="editLicenseGroup(\'' + g.id + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button><button onclick="deleteLicenseGroup(\'' + g.id + '\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button></div></div>'; });
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
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + ' 序列号分组</h3><div class="space-y-3"><input type="hidden" id="license-group-id" value="' + g.id + '"><div><label class="text-sm text-slate-600">分组名称</label><input type="text" id="license-group-name" value="' + g.name + '" class="w-full px-3 py-2 border rounded-lg"></div><div><label class="text-sm text-slate-600">描述</label><input type="text" id="license-group-desc" value="' + (g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div><div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveLicenseGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div></div></div>');
}

function saveLicenseGroup() {
    var group = {id: document.getElementById('license-group-id').value, name: document.getElementById('license-group-name').value, description: document.getElementById('license-group-desc').value};
    fetch('/api/license-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(group)}).then(function() { hideModal(); loadLicenseGroups(); });
}

function editLicenseGroup(id) { var group = licenseGroups.find(function(g) { return g.id === id; }); if (group) showLicenseGroupForm(group); }

function deleteLicenseGroup(id) {
    if (!confirm('确定要删除此分组吗？')) return;
    fetch('/api/license-groups', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) {
            if (result.success) {
                loadLicenseGroups();
                loadLicenses(licenseCurrentPage, licenseSearchTerm);
            } else {
                alert('删除失败: ' + (result.error || '未知错误'));
            }
        })
        .catch(function(err) {
            alert('删除失败: ' + err.message);
        });
}


function saveLLMGroup() {
    var group = {id: document.getElementById('llm-group-id').value, name: document.getElementById('llm-group-name').value, description: document.getElementById('llm-group-desc').value};
    fetch('/api/llm-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(group)}).then(function() { hideModal(); loadLLMGroups(); });
}

function saveSearchGroup() {
    var group = {id: document.getElementById('search-group-id').value, name: document.getElementById('search-group-name').value, description: document.getElementById('search-group-desc').value};
    fetch('/api/search-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(group)}).then(function() { hideModal(); loadSearchGroups(); });
}

function editLLMGroup(id) { var group = llmGroups.find(function(g) { return g.id === id; }); if (group) showLLMGroupForm(group); }
function editSearchGroup(id) { var group = searchGroups.find(function(g) { return g.id === id; }); if (group) showSearchGroupForm(group); }

function deleteLLMGroup(id) {
    if (!confirm('确定要删除此分组吗？')) return;
    fetch('/api/llm-groups', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})}).then(function() { loadLLMGroups(); loadLLMConfigs(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function deleteSearchGroup(id) {
    if (!confirm('确定要删除此分组吗？')) return;
    fetch('/api/search-groups', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})}).then(function() { loadSearchGroups(); loadSearchConfigs(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

document.addEventListener('DOMContentLoaded', function() {
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
        loadRateLimits();
        loadUsername();
        loadFilterSettings();
        loadWhitelist();
        loadBlacklist();
    });
    document.getElementById('use-ssl').addEventListener('change', toggleSSLFields);
});

document.getElementById('modal').addEventListener('click', function(e) { if (e.target.id === 'modal') hideModal(); });
</script>
`

const htmlEnd = `</body>
</html>`
