package templates

// LicenseGroupsHTML contains the license groups panel HTML
const LicenseGroupsHTML = `
<div id="panel-license-groups" class="tab-panel">
    <div class="bg-white rounded-xl shadow-sm p-6">
        <div class="flex justify-between items-center mb-4">
            <h2 class="text-lg font-bold text-slate-800">序列号分组管理</h2>
            <button onclick="showLicenseGroupForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加分组</button>
        </div>
        <p class="text-xs text-slate-500 mb-4">* 序列号分组用于组织和管理序列号，可设置可信度级别（高可信=正式版，低可信=试用版）</p>
        <div class="mb-4 p-3 bg-blue-50 rounded-lg text-xs text-blue-700">
            <strong>可信度说明：</strong>
            <br>• <span class="font-semibold text-green-600">高可信（正式）</span>：每月刷新一次序列号验证
            <br>• <span class="font-semibold text-orange-600">低可信（试用）</span>：每天强制刷新序列号验证
        </div>
        <div id="license-groups-list" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"></div>
    </div>
</div>
`

// LicenseGroupsScripts contains the license groups JavaScript
const LicenseGroupsScripts = `
function loadLicenseGroups() {
    fetch('/api/license-groups').then(function(resp) { return resp.json(); }).then(function(data) {
        licenseGroups = data || [];
        var list = document.getElementById('license-groups-list');
        
        if (!licenseGroups || licenseGroups.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm col-span-3">暂无分组</p>'; 
        } else {
            var html = '';
            licenseGroups.forEach(function(g, idx) { 
                var isBuiltIn = g.id.startsWith('official_') || g.id.startsWith('trial_');
                var isOfficial = g.id.startsWith('official_');
                var trustBadge = g.trust_level === 'high' ? 
                    '<span class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">🔒 高可信(正式)</span>' :
                    '<span class="ml-2 px-2 py-0.5 bg-orange-100 text-orange-700 rounded text-xs">⚠️ 低可信(试用)</span>';
                var builtInBadge = isBuiltIn ? '<span class="ml-2 px-2 py-0.5 bg-slate-100 text-slate-600 rounded text-xs">内置</span>' : '';
                var llmGroupName = getLLMGroupName(g.llm_group_id || '');
                var searchGroupName = getSearchGroupName(g.search_group_id || '');
                html += '<div class="flex items-center justify-between p-3 bg-purple-50 rounded-lg">';
                html += '<div class="flex-1"><span class="font-bold text-sm">' + escapeHtml(g.name) + '</span>' + trustBadge + builtInBadge;
                html += '<p class="text-xs text-slate-400 mt-1">' + escapeHtml(g.description || '无描述') + '</p>';
                if (isOfficial) {
                    html += '<p class="text-xs text-slate-400">LLM: <span class="text-blue-600">' + (llmGroupName || '默认') + '</span> | 搜索: <span class="text-green-600">' + (searchGroupName || '默认') + '</span></p>';
                }
                html += '</div>';
                html += '<div class="flex gap-1">';
                if (isOfficial) {
                    // Built-in official groups can be edited (LLM/Search groups) but not deleted
                    html += '<button data-action="edit-official-group" data-idx="' + idx + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">配置</button>';
                } else if (!isBuiltIn) {
                    // User-created groups can be edited and deleted
                    html += '<button data-action="edit-license-group" data-idx="' + idx + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
                    html += '<button data-action="delete-license-group" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
                }
                html += '</div></div>'; 
            });
            list.innerHTML = html;
        }
        
        // Update filter dropdown in licenses page
        var filterSelect = document.getElementById('license-group-filter');
        if (filterSelect) {
            var currentValue = filterSelect.value;
            var opts = '<option value="">全部序列号组</option><option value="none">默认(无组)</option>';
            licenseGroups.forEach(function(g) { 
                var label = escapeHtml(g.name);
                if (g.trust_level === 'high') label += ' (正式)';
                else label += ' (试用)';
                opts += '<option value="' + g.id + '">' + label + '</option>'; 
            });
            filterSelect.innerHTML = opts;
            filterSelect.value = currentValue;
        }
        
        // Update filter dropdown in email records page
        var emailFilterSelect = document.getElementById('email-license-group-filter');
        if (emailFilterSelect) {
            var currentValue = emailFilterSelect.value;
            var opts = '<option value="">全部序列号组</option><option value="none">默认(无组)</option>';
            licenseGroups.forEach(function(g) { 
                var label = escapeHtml(g.name);
                if (g.trust_level === 'high') label += ' (正式)';
                else label += ' (试用)';
                opts += '<option value="' + g.id + '">' + label + '</option>'; 
            });
            emailFilterSelect.innerHTML = opts;
            emailFilterSelect.value = currentValue;
        }
    });
}

// Event delegation for license groups
document.getElementById('license-groups-list').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action]');
    if (!btn) return;
    var action = btn.getAttribute('data-action');
    var idx = parseInt(btn.getAttribute('data-idx'));
    var group = licenseGroups[idx];
    if (!group) return;
    
    if (action === 'edit-license-group') {
        showLicenseGroupForm(group);
    } else if (action === 'edit-official-group') {
        showOfficialGroupForm(group);
    } else if (action === 'delete-license-group') {
        deleteLicenseGroup(group.id);
    }
});

function showLicenseGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + '序列号分组</h3><div class="space-y-3">' +
        '<div class="p-2 bg-orange-50 rounded text-xs text-orange-700">' +
        '<strong>⚠️ 注意</strong>：用户创建的序列号分组均为低可信（试用）级别，每天刷新一次。高可信（正式）授权组由系统在手工邮件绑定时自动创建。' +
        '</div>' +
        '<input type="hidden" id="license-group-id" value="' + escapeHtml(g.id) + '">' +
        '<div><label class="text-sm text-slate-600">分组名称</label>' +
        '<input type="text" id="license-group-name" value="' + escapeHtml(g.name) + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">描述</label>' +
        '<input type="text" id="license-group-desc" value="' + escapeHtml(g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="saveLicenseGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
}

function showOfficialGroupForm(group) {
    var llmGroupOpts = '<option value="">默认</option>';
    llmGroups.forEach(function(g) { llmGroupOpts += '<option value="' + g.id + '"' + (g.id === group.llm_group_id ? ' selected' : '') + '>' + escapeHtml(g.name) + '</option>'; });
    var searchGroupOpts = '<option value="">默认</option>';
    searchGroups.forEach(function(g) { searchGroupOpts += '<option value="' + g.id + '"' + (g.id === group.search_group_id ? ' selected' : '') + '>' + escapeHtml(g.name) + '</option>'; });
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">配置正式授权组</h3><div class="space-y-3">' +
        '<div class="p-2 bg-green-50 rounded text-xs text-green-700">' +
        '<strong>🔒 内置正式授权组</strong>：此组由系统自动创建，用于手工邮件绑定的高可信授权。您可以配置此组使用的 LLM 和搜索引擎分组。' +
        '</div>' +
        '<input type="hidden" id="official-group-id" value="' + escapeHtml(group.id) + '">' +
        '<div><label class="text-sm text-slate-600">分组名称</label>' +
        '<input type="text" value="' + escapeHtml(group.name) + '" class="w-full px-3 py-2 border rounded-lg bg-slate-100" disabled></div>' +
        '<div><label class="text-sm text-slate-600">LLM 分组</label>' +
        '<select id="official-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmGroupOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">搜索引擎分组</label>' +
        '<select id="official-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchGroupOpts + '</select></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="saveOfficialGroup()" class="flex-1 py-2 bg-green-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
}

function saveOfficialGroup() {
    var data = {
        id: document.getElementById('official-group-id').value,
        llm_group_id: document.getElementById('official-llm-group').value,
        search_group_id: document.getElementById('official-search-group').value
    };
    
    fetch('/api/license-groups/config', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            hideModal(); 
            if (result.success) {
                loadLicenseGroups(); 
            } else {
                alert('保存失败: ' + result.error);
            }
        });
}

function editLicenseGroup(id) {
    var group = licenseGroups.find(function(g) { return g.id === id; });
    if (group) showLicenseGroupForm(group);
}

function saveLicenseGroup() {
    var group = {
        id: document.getElementById('license-group-id').value,
        name: document.getElementById('license-group-name').value,
        description: document.getElementById('license-group-desc').value
    };
    if (!group.name) { alert('分组名称不能为空'); return; }
    
    fetch('/api/license-groups', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(group)})
        .then(function() { hideModal(); loadLicenseGroups(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}

function deleteLicenseGroup(id) {
    if (!confirm('确定要删除此分组吗？')) return;
    fetch('/api/license-groups', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) {
                loadLicenseGroups(); 
                loadLicenses(licenseCurrentPage, licenseSearchTerm); 
            } else {
                alert('删除失败: ' + result.error);
            }
        });
}
`
