package templates

// LLMConfigHTML contains the LLM configuration panel HTML
const LLMConfigHTML = `
<div id="panel-llm" class="tab-panel">
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- LLM Groups -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-lg font-bold text-slate-800">LLM 分组</h2>
                <button onclick="showLLMGroupForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加</button>
            </div>
            <div id="llm-groups-list" class="space-y-2"></div>
        </div>
        
        <!-- LLM Configs -->
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
`

// LLMConfigScripts contains the LLM configuration JavaScript
const LLMConfigScripts = `
function loadLLMGroups() {
    fetch('/api/llm-groups').then(function(resp) { return resp.json(); }).then(function(data) {
        llmGroups = data || [];
        var list = document.getElementById('llm-groups-list');
        
        if (!llmGroups || llmGroups.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无分组</p>'; 
        } else {
            var html = '';
            llmGroups.forEach(function(g, idx) { 
                html += '<div class="flex items-center justify-between p-2 bg-blue-50 rounded-lg">';
                html += '<div><span class="font-bold text-sm">' + escapeHtml(g.name) + '</span>';
                html += '<p class="text-xs text-slate-400">' + escapeHtml(g.description || '') + '</p></div>';
                html += '<div class="flex gap-1">';
                html += '<button data-action="edit-llm-group" data-idx="' + idx + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
                html += '<button data-action="delete-llm-group" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
                html += '</div></div>'; 
            });
            list.innerHTML = html;
        }
        
        // Update license filter dropdown
        var filterSelect = document.getElementById('llm-group-filter');
        if (filterSelect) {
            var currentValue = filterSelect.value;
            var opts = '<option value="">全部LLM组</option><option value="none">默认(无组)</option>';
            llmGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + escapeHtml(g.name) + '</option>'; });
            filterSelect.innerHTML = opts;
            filterSelect.value = currentValue;
        }
        
        // Update LLM config filter dropdown
        var configFilterSelect = document.getElementById('llm-config-group-filter');
        if (configFilterSelect) {
            var currentValue = configFilterSelect.value;
            var opts = '<option value="">全部分组</option><option value="none">默认(无组)</option>';
            llmGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + escapeHtml(g.name) + '</option>'; });
            configFilterSelect.innerHTML = opts;
            configFilterSelect.value = currentValue;
        }
    });
}

// Store LLM configs for event delegation
var llmConfigsData = [];

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
        
        llmConfigsData = filteredConfigs;
        
        if (!filteredConfigs || filteredConfigs.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4">暂无配置</p>'; 
            return; 
        }
        
        var html = '';
        filteredConfigs.forEach(function(c, idx) {
            var groupName = getLLMGroupName(c.group_id);
            var ringClass = c.is_active ? 'ring-2 ring-green-500' : '';
            
            html += '<div class="flex items-center justify-between p-3 bg-slate-50 rounded-lg ' + ringClass + '">';
            html += '<div><span class="font-bold">' + escapeHtml(c.name) + '</span>';
            if (c.is_active) html += '<span class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">当前使用</span>';
            if (groupName) html += '<span class="ml-2 px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded">' + escapeHtml(groupName) + '</span>';
            html += '<p class="text-sm text-slate-500">类型: ' + escapeHtml(c.type) + ' | 模型: ' + escapeHtml(c.model) + '</p>';
            html += '<p class="text-xs text-slate-400">URL: ' + escapeHtml(c.base_url || '默认') + '</p>';
            html += '<p class="text-xs text-slate-400">有效期: ' + (c.start_date || '无限制') + ' ~ ' + (c.end_date || '永久') + '</p></div>';
            html += '<div class="flex gap-2">';
            html += '<button data-action="edit-llm" data-idx="' + idx + '" class="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm">编辑</button>';
            html += '<button data-action="delete-llm" data-idx="' + idx + '" class="px-3 py-1 bg-red-100 text-red-700 rounded text-sm">删除</button>';
            html += '</div></div>';
        });
        list.innerHTML = html;
    });
}

// Event delegation for LLM groups
document.getElementById('llm-groups-list').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action]');
    if (!btn) return;
    var action = btn.getAttribute('data-action');
    var idx = parseInt(btn.getAttribute('data-idx'));
    var group = llmGroups[idx];
    if (!group) return;
    
    if (action === 'edit-llm-group') {
        showLLMGroupForm(group);
    } else if (action === 'delete-llm-group') {
        deleteLLMGroup(group.id);
    }
});

// Event delegation for LLM configs
document.getElementById('llm-list').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action]');
    if (!btn) return;
    var action = btn.getAttribute('data-action');
    var idx = parseInt(btn.getAttribute('data-idx'));
    var config = llmConfigsData[idx];
    if (!config) return;
    
    if (action === 'edit-llm') {
        showLLMForm(config);
    } else if (action === 'delete-llm') {
        deleteLLM(config.id);
    }
});

function showLLMGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + ' LLM 分组</h3><div class="space-y-3">' +
        '<input type="hidden" id="llm-group-id" value="' + escapeHtml(g.id) + '">' +
        '<div><label class="text-sm text-slate-600">分组名称</label>' +
        '<input type="text" id="llm-group-name" value="' + escapeHtml(g.name) + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">描述</label>' +
        '<input type="text" id="llm-group-desc" value="' + escapeHtml(g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="saveLLMGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
}

function editLLMGroup(id) {
    var group = llmGroups.find(function(g) { return g.id === id; });
    if (group) showLLMGroupForm(group);
}

function saveLLMGroup() {
    var group = {
        id: document.getElementById('llm-group-id').value,
        name: document.getElementById('llm-group-name').value,
        description: document.getElementById('llm-group-desc').value
    };
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
    var groupOpts = llmGroups.map(function(g) { return '<option value="' + g.id + '"' + (g.id === c.group_id ? ' selected' : '') + '>' + escapeHtml(g.name) + '</option>'; }).join('');
    var typeOpts = '<option value="openai"' + (c.type === 'openai' ? ' selected' : '') + '>OpenAI</option>' +
        '<option value="anthropic"' + (c.type === 'anthropic' ? ' selected' : '') + '>Anthropic</option>' +
        '<option value="gemini"' + (c.type === 'gemini' ? ' selected' : '') + '>Gemini</option>' +
        '<option value="deepseek"' + (c.type === 'deepseek' ? ' selected' : '') + '>DeepSeek</option>';
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (c.id ? '编辑' : '添加') + ' LLM 配置</h3><div class="space-y-3">' +
        '<input type="hidden" id="llm-id" value="' + escapeHtml(c.id) + '">' +
        '<div><label class="text-sm text-slate-600">名称</label><input type="text" id="llm-name" value="' + escapeHtml(c.name) + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">分组</label><select id="llm-group" class="w-full px-3 py-2 border rounded-lg"><option value=""' + (!c.group_id ? ' selected' : '') + '>无分组</option>' + groupOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">类型</label><select id="llm-type" class="w-full px-3 py-2 border rounded-lg">' + typeOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">Base URL（可选）</label><input type="text" id="llm-url" value="' + escapeHtml(c.base_url) + '" placeholder="留空使用默认" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">API Key</label><input type="password" id="llm-key" value="' + escapeHtml(c.api_key) + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">模型</label><input type="text" id="llm-model" value="' + escapeHtml(c.model) + '" placeholder="例如: gpt-4o" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="grid grid-cols-2 gap-3"><div><label class="text-sm text-slate-600">生效日期</label><input type="date" id="llm-start-date" value="' + (c.start_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">截止日期</label><input type="date" id="llm-end-date" value="' + (c.end_date || '') + '" class="w-full px-3 py-2 border rounded-lg"></div></div>' +
        '<div class="flex items-center gap-2"><input type="checkbox" id="llm-active"' + (c.is_active ? ' checked' : '') + '><label class="text-sm">设为当前使用</label></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="saveLLM()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
}

function editLLM(id) {
    fetch('/api/llm').then(function(resp) { return resp.json(); }).then(function(configs) {
        var config = configs.find(function(c) { return c.id === id; });
        if (config) showLLMForm(config);
    });
}

function saveLLM() {
    var config = {
        id: document.getElementById('llm-id').value,
        name: document.getElementById('llm-name').value,
        type: document.getElementById('llm-type').value,
        base_url: document.getElementById('llm-url').value,
        api_key: document.getElementById('llm-key').value,
        model: document.getElementById('llm-model').value,
        start_date: document.getElementById('llm-start-date').value,
        end_date: document.getElementById('llm-end-date').value,
        is_active: document.getElementById('llm-active').checked,
        group_id: document.getElementById('llm-group').value
    };
    fetch('/api/llm', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(config)})
        .then(function() { hideModal(); loadLLMConfigs(); });
}

function deleteLLM(id) {
    if (!confirm('确定要删除此配置吗？')) return;
    fetch('/api/llm', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function() { loadLLMConfigs(); });
}
`
