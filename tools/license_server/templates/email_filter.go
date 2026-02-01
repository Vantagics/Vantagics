package templates

// EmailFilterHTML contains the email filter panel HTML
const EmailFilterHTML = `
<div id="panel-email-filter" class="tab-panel">
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Filter Settings -->
        <div class="bg-white rounded-xl shadow-sm p-6 lg:col-span-3">
            <h2 class="text-lg font-bold text-slate-800 mb-4">过滤模式设置</h2>
            <div class="flex items-center gap-6 mb-4">
                <label class="flex items-center gap-2">
                    <input type="checkbox" id="blacklist-enabled" class="w-4 h-4" onchange="saveFilterSettings()">
                    <span class="text-sm">启用黑名单</span>
                </label>
                <label class="flex items-center gap-2">
                    <input type="checkbox" id="whitelist-enabled" class="w-4 h-4" onchange="saveFilterSettings()">
                    <span class="text-sm">启用白名单</span>
                </label>
                <label class="flex items-center gap-2">
                    <input type="checkbox" id="conditions-enabled" class="w-4 h-4" onchange="saveFilterSettings()">
                    <span class="text-sm">启用条件名单</span>
                </label>
            </div>
            <p class="text-xs text-slate-500">* 黑名单优先检查。启用白名单时，邮箱必须在白名单中且不在黑名单中</p>
        </div>
        
        <!-- Blacklist -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-lg font-bold text-slate-800">⚫ 黑名单</h2>
                <button onclick="showAddBlacklist()" class="px-3 py-1.5 bg-red-600 text-white rounded-lg text-sm">+ 添加</button>
            </div>
            <p class="text-xs text-slate-500 mb-3">* 匹配的邮箱/域名将被拒绝</p>
            <div id="blacklist-items" class="space-y-2 max-h-80 overflow-y-auto"></div>
        </div>
        
        <!-- Whitelist -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-lg font-bold text-slate-800">⚪ 白名单</h2>
                <button onclick="showAddWhitelist()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加</button>
            </div>
            <p class="text-xs text-slate-500 mb-3">* 启用时，只有匹配的邮箱/域名才能申请</p>
            <div id="whitelist-items" class="space-y-2 max-h-80 overflow-y-auto"></div>
        </div>
        
        <!-- Conditions -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-lg font-bold text-slate-800">🔶 条件名单</h2>
                <button onclick="showAddCondition()" class="px-3 py-1.5 bg-amber-600 text-white rounded-lg text-sm">+ 添加</button>
            </div>
            <p class="text-xs text-slate-500 mb-3">* 匹配的邮箱/域名将分配指定分组的序列号</p>
            <div id="condition-items" class="space-y-2 max-h-80 overflow-y-auto"></div>
        </div>
    </div>
</div>
`

// EmailFilterScripts contains the email filter JavaScript
const EmailFilterScripts = `
// Store filter data for button handlers
var blacklistData = [];
var whitelistData = [];
var conditionsData = [];

function loadFilterSettings() {
    fetch('/api/email-filter').then(function(resp) { return resp.json(); }).then(function(data) {
        document.getElementById('whitelist-enabled').checked = data.whitelist_enabled;
        document.getElementById('blacklist-enabled').checked = data.blacklist_enabled;
        document.getElementById('conditions-enabled').checked = data.conditions_enabled;
    });
}

function saveFilterSettings() {
    var data = {
        whitelist_enabled: document.getElementById('whitelist-enabled').checked,
        blacklist_enabled: document.getElementById('blacklist-enabled').checked,
        conditions_enabled: document.getElementById('conditions-enabled').checked
    };
    fetch('/api/email-filter', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)});
}

function loadBlacklist() {
    fetch('/api/blacklist').then(function(resp) { return resp.json(); }).then(function(items) {
        blacklistData = items || [];
        var list = document.getElementById('blacklist-items');
        if (!items || items.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无黑名单</p>'; 
            return; 
        }
        var html = '';
        items.forEach(function(item, idx) { 
            html += '<div class="flex items-center justify-between p-2 bg-red-50 rounded-lg">';
            html += '<div><code class="text-sm font-mono text-red-700">' + escapeHtml(item.pattern) + '</code>';
            html += '<p class="text-xs text-slate-400">' + new Date(item.created_at).toLocaleString() + '</p></div>';
            html += '<button data-action="delete-blacklist" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
            html += '</div>'; 
        });
        list.innerHTML = html;
    });
}

function loadWhitelist() {
    fetch('/api/whitelist').then(function(resp) { return resp.json(); }).then(function(items) {
        whitelistData = items || [];
        var list = document.getElementById('whitelist-items');
        if (!items || items.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无白名单</p>'; 
            return; 
        }
        var html = '';
        items.forEach(function(item, idx) { 
            html += '<div class="flex items-center justify-between p-2 bg-green-50 rounded-lg">';
            html += '<div><code class="text-sm font-mono text-green-700">' + escapeHtml(item.pattern) + '</code>';
            html += '<p class="text-xs text-slate-400">' + new Date(item.created_at).toLocaleString() + '</p></div>';
            html += '<button data-action="delete-whitelist" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
            html += '</div>'; 
        });
        list.innerHTML = html;
    });
}

function loadConditions() {
    Promise.all([
        fetch('/api/conditions').then(function(r){return r.json();}), 
        fetch('/api/llm-groups').then(function(r){return r.json();}), 
        fetch('/api/search-groups').then(function(r){return r.json();})
    ]).then(function(results) {
        var items = results[0] || [];
        conditionsData = items;
        var llmGroupsList = results[1] || [];
        var searchGroupsList = results[2] || [];
        var list = document.getElementById('condition-items');
        
        if (!items || items.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无条件名单</p>'; 
            return; 
        }
        
        var html = '';
        items.forEach(function(item, idx) {
            var llmGroupName = item.llm_group_id ? (llmGroupsList.find(function(g){return g.id===item.llm_group_id;}) || {}).name || item.llm_group_id : '无限制';
            var searchGroupName = item.search_group_id ? (searchGroupsList.find(function(g){return g.id===item.search_group_id;}) || {}).name || item.search_group_id : '无限制';
            
            html += '<div class="p-3 bg-amber-50 rounded-lg">';
            html += '<div class="flex items-center justify-between">';
            html += '<code class="text-sm font-mono text-amber-700">' + escapeHtml(item.pattern) + '</code>';
            html += '<button data-action="delete-condition" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
            html += '</div>';
            html += '<div class="mt-2 text-xs text-slate-500">';
            html += '<span class="mr-3">LLM组: <span class="text-blue-600">' + escapeHtml(llmGroupName) + '</span></span>';
            html += '<span>搜索组: <span class="text-purple-600">' + escapeHtml(searchGroupName) + '</span></span>';
            html += '</div>';
            html += '<p class="text-xs text-slate-400 mt-1">' + new Date(item.created_at).toLocaleString() + '</p>';
            html += '</div>';
        });
        list.innerHTML = html;
    });
}

// Event delegation for filter lists
document.getElementById('blacklist-items').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action="delete-blacklist"]');
    if (!btn) return;
    var idx = parseInt(btn.getAttribute('data-idx'));
    var item = blacklistData[idx];
    if (item) deleteBlacklist(item.pattern);
});

document.getElementById('whitelist-items').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action="delete-whitelist"]');
    if (!btn) return;
    var idx = parseInt(btn.getAttribute('data-idx'));
    var item = whitelistData[idx];
    if (item) deleteWhitelist(item.pattern);
});

document.getElementById('condition-items').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action="delete-condition"]');
    if (!btn) return;
    var idx = parseInt(btn.getAttribute('data-idx'));
    var item = conditionsData[idx];
    if (item) deleteCondition(item.pattern);
});

function showAddBlacklist() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">添加黑名单</h3><div class="space-y-3">' +
        '<div><label class="text-sm text-slate-600">邮箱或域名</label>' +
        '<input type="text" id="blacklist-pattern" placeholder="例如: @spam.com" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<p class="text-xs text-slate-500">* 以 @ 开头表示域名匹配</p>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="doAddBlacklist()" class="flex-1 py-2 bg-red-600 text-white rounded-lg">添加</button></div>' +
        '</div></div>');
}

function showAddWhitelist() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">添加白名单</h3><div class="space-y-3">' +
        '<div><label class="text-sm text-slate-600">邮箱或域名</label>' +
        '<input type="text" id="whitelist-pattern" placeholder="例如: @company.com" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<p class="text-xs text-slate-500">* 以 @ 开头表示域名匹配</p>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="doAddWhitelist()" class="flex-1 py-2 bg-green-600 text-white rounded-lg">添加</button></div>' +
        '</div></div>');
}

function showAddCondition() {
    Promise.all([
        fetch('/api/llm-groups').then(function(r){return r.json();}), 
        fetch('/api/search-groups').then(function(r){return r.json();})
    ]).then(function(results) {
        var llmGroupsList = results[0] || [];
        var searchGroupsList = results[1] || [];
        
        var llmOptions = '<option value="">无限制（随机）</option>';
        llmGroupsList.forEach(function(g) { llmOptions += '<option value="' + g.id + '">' + escapeHtml(g.name) + '</option>'; });
        var searchOptions = '<option value="">无限制（随机）</option>';
        searchGroupsList.forEach(function(g) { searchOptions += '<option value="' + g.id + '">' + escapeHtml(g.name) + '</option>'; });
        
        showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">添加条件名单</h3><div class="space-y-3">' +
            '<div><label class="text-sm text-slate-600">邮箱或域名</label>' +
            '<input type="text" id="condition-pattern" placeholder="例如: @company.com" class="w-full px-3 py-2 border rounded-lg"></div>' +
            '<p class="text-xs text-slate-500">* 以 @ 开头表示域名匹配</p>' +
            '<div><label class="text-sm text-slate-600">绑定LLM组</label>' +
            '<select id="condition-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmOptions + '</select></div>' +
            '<div><label class="text-sm text-slate-600">绑定搜索引擎组</label>' +
            '<select id="condition-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchOptions + '</select></div>' +
            '<p class="text-xs text-slate-500">* 匹配的邮箱申请时将分配指定分组的序列号</p>' +
            '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
            '<button onclick="doAddCondition()" class="flex-1 py-2 bg-amber-600 text-white rounded-lg">添加</button></div>' +
            '</div></div>');
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
    fetch('/api/blacklist', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})})
        .then(function() { loadBlacklist(); });
}

function deleteWhitelist(pattern) {
    if (!confirm('确定要删除此白名单项吗？')) return;
    fetch('/api/whitelist', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})})
        .then(function() { loadWhitelist(); });
}

function deleteCondition(pattern) {
    if (!confirm('确定要删除此条件名单项吗？')) return;
    fetch('/api/conditions', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({pattern: pattern})})
        .then(function() { loadConditions(); });
}
`
