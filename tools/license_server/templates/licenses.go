package templates

// LicensesHTML contains the licenses management panel HTML
const LicensesHTML = `
<div id="panel-licenses" class="tab-panel active">
    <div class="bg-white rounded-xl shadow-sm p-6">
        <div class="flex justify-between items-center mb-4">
            <h2 class="text-lg font-bold text-slate-800">序列号列表</h2>
            <div class="flex items-center gap-2">
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
`

// LicensesScripts contains the licenses management JavaScript
const LicensesScripts = `
function loadLicenses(page, search) {
    page = page || 1;
    search = search || '';
    licenseCurrentPage = page;
    licenseSearchTerm = search;
    
    var licenseGroupFilter = document.getElementById('license-group-filter').value;
    var llmGroupFilter = document.getElementById('llm-group-filter').value;
    var searchGroupFilter = document.getElementById('search-group-filter').value;
    
    var params = new URLSearchParams({
        page: page.toString(), 
        pageSize: '20', 
        search: search,
        license_group: licenseGroupFilter,
        llm_group: llmGroupFilter,
        search_group: searchGroupFilter
    });
    
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
            html += '<div class="flex items-center gap-2">';
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
            html += '<div class="flex gap-2">';
            html += '<button onclick="setLicenseGroups(\'' + l.sn + '\', \'' + (l.license_group_id || '') + '\', \'' + (l.llm_group_id || '') + '\', \'' + (l.search_group_id || '') + '\')" class="px-2 py-1 bg-indigo-100 text-indigo-700 rounded text-xs">分组</button>';
            html += '<button onclick="extendLicense(\'' + l.sn + '\', \'' + l.expires_at + '\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">展期</button>';
            html += '<button onclick="setDailyAnalysis(\'' + l.sn + '\', ' + l.daily_analysis + ')" class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs">分析次数</button>';
            html += '<button onclick="toggleLicense(\'' + l.sn + '\')" class="px-2 py-1 ' + (l.is_active ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700') + ' rounded text-xs">' + (l.is_active ? '禁用' : '启用') + '</button>';
            html += '</div>';
            html += '</div>';
        });
        list.innerHTML = html;
        
        // Pagination
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

function searchLicenses() { 
    loadLicenses(1, document.getElementById('license-search').value); 
}

function showBatchCreate() {
    var licenseGroupOpts = '<option value="">无分组</option>';
    licenseGroups.forEach(function(g) { licenseGroupOpts += '<option value="' + g.id + '">' + g.name + '</option>'; });
    var llmGroupOpts = '<option value="">无分组</option>';
    llmGroups.forEach(function(g) { llmGroupOpts += '<option value="' + g.id + '">' + g.name + '</option>'; });
    var searchGroupOpts = '<option value="">无分组</option>';
    searchGroups.forEach(function(g) { searchGroupOpts += '<option value="' + g.id + '">' + g.name + '</option>'; });
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">批量生成序列号</h3><div class="space-y-3">' +
        '<div><label class="text-sm text-slate-600">描述</label><input type="text" id="batch-desc" placeholder="可选" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="grid grid-cols-2 gap-3">' +
        '<div><label class="text-sm text-slate-600">有效天数</label><input type="number" id="batch-days" value="365" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">生成数量</label><input type="number" id="batch-count" value="100" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '</div>' +
        '<div><label class="text-sm text-slate-600">每日分析次数 (0=无限)</label><input type="number" id="batch-daily" value="20" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">序列号分组</label><select id="batch-license-group" class="w-full px-3 py-2 border rounded-lg">' + licenseGroupOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">LLM分组</label><select id="batch-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmGroupOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">搜索分组</label><select id="batch-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchGroupOpts + '</select></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doBatchCreate()" class="flex-1 py-2 bg-green-600 text-white rounded-lg">生成</button></div>' +
        '</div></div>');
}

function doBatchCreate() {
    var data = {
        description: document.getElementById('batch-desc').value,
        days: parseInt(document.getElementById('batch-days').value) || 365,
        count: parseInt(document.getElementById('batch-count').value) || 100,
        daily_analysis: parseInt(document.getElementById('batch-daily').value) || 0,
        license_group_id: document.getElementById('batch-license-group').value,
        llm_group_id: document.getElementById('batch-llm-group').value,
        search_group_id: document.getElementById('batch-search-group').value
    };
    fetch('/api/licenses/batch-create', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { alert('成功生成 ' + result.count + ' 个序列号'); loadLicenses(); } else { alert('生成失败: ' + result.error); } });
}

function toggleLicense(sn) {
    fetch('/api/licenses/toggle', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function() { loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function extendLicense(sn, currentExpiry) {
    var expiryDate = currentExpiry ? new Date(currentExpiry).toISOString().split('T')[0] : '';
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">展期序列号</h3><div class="space-y-3">' +
        '<p class="text-sm text-slate-600">序列号: <code class="font-mono text-blue-600">' + sn + '</code></p>' +
        '<p class="text-sm text-slate-600">当前到期: <span class="text-orange-600">' + (expiryDate || '未知') + '</span></p>' +
        '<div><label class="text-sm text-slate-600">延长天数</label><input type="number" id="extend-days" value="365" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doExtendLicense(\'' + sn + '\')" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">确认</button></div>' +
        '</div></div>');
}

function doExtendLicense(sn) {
    var days = parseInt(document.getElementById('extend-days').value) || 365;
    fetch('/api/licenses/extend', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, days: days})})
        .then(function() { hideModal(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function setDailyAnalysis(sn, current) {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">设置每日分析次数</h3><div class="space-y-3">' +
        '<p class="text-sm text-slate-600">序列号: <code class="font-mono text-blue-600">' + sn + '</code></p>' +
        '<div><label class="text-sm text-slate-600">每日分析次数 (0=无限)</label><input type="number" id="daily-count" value="' + current + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doSetDailyAnalysis(\'' + sn + '\')" class="flex-1 py-2 bg-purple-600 text-white rounded-lg">确认</button></div>' +
        '</div></div>');
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
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">设置分组</h3><div class="space-y-3">' +
        '<p class="text-sm text-slate-600">序列号: <code class="font-mono text-blue-600">' + sn + '</code></p>' +
        '<div><label class="text-sm text-slate-600">序列号分组</label><select id="set-license-group" class="w-full px-3 py-2 border rounded-lg">' + licenseGroupOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">LLM分组</label><select id="set-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmGroupOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">搜索分组</label><select id="set-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchGroupOpts + '</select></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doSetLicenseGroups(\'' + sn + '\')" class="flex-1 py-2 bg-indigo-600 text-white rounded-lg">确认</button></div>' +
        '</div></div>');
}

function doSetLicenseGroups(sn) {
    var data = {
        sn: sn,
        license_group_id: document.getElementById('set-license-group').value,
        llm_group_id: document.getElementById('set-llm-group').value,
        search_group_id: document.getElementById('set-search-group').value
    };
    fetch('/api/licenses/set-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)})
        .then(function() { hideModal(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function deleteUnusedByGroup() {
    var licenseGroupFilter = document.getElementById('license-group-filter').value;
    var llmGroupFilter = document.getElementById('llm-group-filter').value;
    var searchGroupFilter = document.getElementById('search-group-filter').value;
    
    if (!licenseGroupFilter && !llmGroupFilter && !searchGroupFilter) {
        alert('请先选择至少一个分组过滤条件');
        return;
    }
    
    var filterDesc = [];
    if (licenseGroupFilter) filterDesc.push('序列号分组: ' + (licenseGroupFilter === 'none' ? '默认(无组)' : getLicenseGroupName(licenseGroupFilter)));
    if (llmGroupFilter) filterDesc.push('LLM分组: ' + (llmGroupFilter === 'none' ? '默认(无组)' : getLLMGroupName(llmGroupFilter)));
    if (searchGroupFilter) filterDesc.push('搜索分组: ' + (searchGroupFilter === 'none' ? '默认(无组)' : getSearchGroupName(searchGroupFilter)));
    
    if (!confirm('确定要删除以下条件的所有未使用序列号吗？\\n\\n' + filterDesc.join('\\n') + '\\n\\n⚠️ 此操作不可恢复！')) return;
    
    var data = {};
    if (licenseGroupFilter) data.license_group_id = licenseGroupFilter;
    if (llmGroupFilter) data.llm_group_id = llmGroupFilter;
    if (searchGroupFilter) data.search_group_id = searchGroupFilter;
    
    fetch('/api/licenses/delete-unused-by-group', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { if (result.success) { alert(result.message); loadLicenses(); } else { alert('删除失败: ' + result.error); } });
}

function purgeDisabledLicenses() {
    if (!confirm('确定要永久删除所有已禁用且未绑定邮箱的序列号吗？\\n\\n⚠️ 此操作不可恢复！\\n✅ 已绑定邮箱的序列号会被保留')) return;
    
    fetch('/api/licenses/purge-disabled', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { if (result.success) { alert(result.message); loadLicenses(); } else { alert('清除失败: ' + result.error); } });
}
`
