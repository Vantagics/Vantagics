package templates

// SearchConfigHTML contains the search configuration panel HTML
const SearchConfigHTML = `
<div id="panel-search" class="tab-panel">
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Search Groups -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-lg font-bold text-slate-800">搜索分组</h2>
                <button onclick="showSearchGroupForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加</button>
            </div>
            <div id="search-groups-list" class="space-y-2"></div>
        </div>
        
        <!-- Search Configs -->
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
`

// SearchConfigScripts contains the search configuration JavaScript
const SearchConfigScripts = `
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
                html += '<button onclick="editSearchGroup(\\'' + g.id + '\\')" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
                html += '<button onclick="deleteSearchGroup(\\'' + g.id + '\\')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
                html += '</div></div>'; 
            });
            list.innerHTML = html;
        }
        
        // Update license filter dropdown
        var filterSelect = document.getElementById('search-group-filter');
        if (filterSelect) {
            var currentValue = filterSelect.value;
            var opts = '<option value="">全部搜索组</option><option value="none">默认(无组)</option>';
            searchGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + g.name + '</option>'; });
            filterSelect.innerHTML = opts;
            filterSelect.value = currentValue;
        }
        
        // Update search config filter dropdown
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
            html += '<button onclick="editSearch(\\'' + c.id + '\\')" class="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm">编辑</button>';
            html += '<button onclick="deleteSearch(\\'' + c.id + '\\')" class="px-3 py-1 bg-red-100 text-red-700 rounded text-sm">删除</button>';
            html += '</div></div>';
        });
        list.innerHTML = html;
    });
}

function showSearchGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + ' 搜索分组</h3><div class="space-y-3">' +
        '<input type="hidden" id="search-group-id" value="' + g.id + '">' +
        '<div><label class="text-sm text-slate-600">分组名称</label>' +
        '<input type="text" id="search-group-name" value="' + g.name + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">描述</label>' +
        '<input type="text" id="search-group-desc" value="' + (g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="saveSearchGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
}

function editSearchGroup(id) {
    var group = searchGroups.find(function(g) { return g.id === id; });
    if (group) showSearchGroupForm(group);
}

function saveSearchGroup() {
    var group = {
        id: document.getElementById('search-group-id').value,
        name: document.getElementById('search-group-name').value,
        description: document.getElementById('search-group-desc').value
    };
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
    var typeOpts = '<option value="tavily"' + (c.type === 'tavily' ? ' selected' : '') + '>Tavily</option>' +
        '<option value="serper"' + (c.type === 'serper' ? ' selected' : '') + '>Serper</option>' +
        '<option value="bing"' + (c.type === 'bing' ? ' selected' : '') + '>Bing</option>';
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (c.id ? '编辑' : '添加') + ' 搜索引擎配置</h3><div class="space-y-3">' +
        '<input type="hidden" id="search-id" value="' + c.id + '">' +
        '<div><label class="text-sm text-slate-600">名称</label><input type="text" id="search-name" value="' + c.name + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">分组</label><select id="search-group" class="w-full px-3 py-2 border rounded-lg"><option value=""' + (!c.group_id ? ' selected' : '') + '>无分组</option>' + groupOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">类型</label><select id="search-type" class="w-full px-3 py-2 border rounded-lg">' + typeOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">API Key</label><input type="password" id="search-key" value="' + c.api_key + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="grid grid-cols-2 gap-3"><div><label class="text-sm text-slate-600">生效日期</label><input type="date" id="search-start-date" value="' + (c.start_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">截止日期</label><input type="date" id="search-end-date" value="' + (c.end_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div></div>' +
        '<div class="flex items-center gap-2"><input type="checkbox" id="search-active"' + (c.is_active ? ' checked' : '') + '><label class="text-sm">设为当前使用</label></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveSearch()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
}

function editSearch(id) {
    fetch('/api/search').then(function(resp) { return resp.json(); }).then(function(configs) {
        var config = configs.find(function(c) { return c.id === id; });
        if (config) showSearchForm(config);
    });
}

function saveSearch() {
    var config = {
        id: document.getElementById('search-id').value,
        name: document.getElementById('search-name').value,
        type: document.getElementById('search-type').value,
        api_key: document.getElementById('search-key').value,
        start_date: document.getElementById('search-start-date').value,
        end_date: document.getElementById('search-end-date').value,
        is_active: document.getElementById('search-active').checked,
        group_id: document.getElementById('search-group').value
    };
    fetch('/api/search', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(config)})
        .then(function() { hideModal(); loadSearchConfigs(); });
}

function deleteSearch(id) {
    if (!confirm('确定要删除此配置吗？')) return;
    fetch('/api/search', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function() { loadSearchConfigs(); });
}
`
