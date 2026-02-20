package templates

// APIKeysHTML contains the API keys management panel HTML
const APIKeysHTML = `
<div id="section-api-keys" class="section">
    <div class="card">
        <div class="card-header">
            <h2 class="card-title">API Key 管理</h2>
            <button onclick="showAPIKeyForm()" class="btn btn-success btn-sm">+ 创建 API Key</button>
        </div>
        <p class="text-xs text-muted mb-4">* API Key 用于第三方系统通过 API 接口创建正式授权。每个 API Key 绑定特定产品。</p>
        <div class="mb-4" style="padding:12px;background:#eff6ff;border-radius:8px;font-size:12px;color:#1d4ed8;">
            <strong>API 接口说明：</strong>
            <br>• <code>POST /api/bind-license</code> - 通过 API Key 创建正式授权
            <br>• 请求参数：<code>{"api_key": "sk-xxx", "email": "user@example.com", "days": 365}</code>
            <br>• 返回：<code>{"success": true, "sn": "XXXX-XXXX-XXXX-XXXX"}</code>
        </div>
        <div id="api-keys-list"></div>
    </div>
</div>
`

// APIKeysScripts contains the API keys management JavaScript
const APIKeysScripts = `
var apiKeys = [];

function loadAPIKeys() {
    fetch('/api/api-keys').then(function(resp) { return resp.json(); }).then(function(data) {
        apiKeys = data || [];
        var list = document.getElementById('api-keys-list');
        
        if (!apiKeys || apiKeys.length === 0) { 
            list.innerHTML = '<p class="text-slate-500 text-center py-4 text-sm">暂无 API Key</p>'; 
            return;
        }
        
        var html = '';
        apiKeys.forEach(function(k, idx) { 
            var isExpired = k.expires_at && new Date(k.expires_at) < new Date();
            var statusClass = !k.is_active ? 'opacity-50' : (isExpired ? 'bg-orange-50' : '');
            var productName = getProductTypeName(k.product_id) || 'Vantagics';
            
            html += '<div class="p-4 bg-slate-50 rounded-lg ' + statusClass + '">';
            html += '<div class="flex items-start justify-between">';
            html += '<div class="flex-1">';
            html += '<div class="flex items-center gap-2 mb-2">';
            html += '<code class="font-mono text-sm text-blue-600 bg-blue-50 px-2 py-1 rounded">' + escapeHtml(k.api_key) + '</code>';
            html += '<button onclick="copyToClipboard(\'' + escapeHtml(k.api_key) + '\')" class="px-2 py-1 bg-slate-200 text-slate-600 rounded text-xs">复制</button>';
            if (!k.is_active) html += '<span class="px-2 py-0.5 bg-red-100 text-red-700 text-xs rounded">已禁用</span>';
            if (isExpired) html += '<span class="px-2 py-0.5 bg-orange-100 text-orange-700 text-xs rounded">已过期</span>';
            html += '</div>';
            html += '<div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-600">';
            html += '<p>📦 产品: <span class="font-medium text-amber-700">' + productName + '</span></p>';
            html += '<p>🏢 单位: <span class="font-medium">' + escapeHtml(k.organization || '-') + '</span></p>';
            html += '<p>👤 姓名: <span class="font-medium">' + escapeHtml(k.contact_name || '-') + '</span></p>';
            html += '<p>📅 有效期: <span class="font-medium ' + (isExpired ? 'text-red-600' : '') + '">' + (k.expires_at ? new Date(k.expires_at).toLocaleDateString() : '永久') + '</span></p>';
            html += '<p>📊 使用次数: <span class="font-medium text-blue-600">' + (k.usage_count || 0) + '</span></p>';
            html += '<p>🕐 创建时间: <span class="font-medium">' + new Date(k.created_at).toLocaleDateString() + '</span></p>';
            html += '</div>';
            if (k.description) html += '<p class="text-xs text-slate-400 mt-1">备注: ' + escapeHtml(k.description) + '</p>';
            html += '</div>';
            html += '<div class="flex flex-col gap-1 flex-shrink-0">';
            html += '<div class="flex gap-1">';
            html += '<button data-action="edit-api-key" data-idx="' + idx + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
            html += '<button data-action="toggle-api-key" data-idx="' + idx + '" class="px-2 py-1 ' + (k.is_active ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700') + ' rounded text-xs">' + (k.is_active ? '禁用' : '启用') + '</button>';
            if (k.usage_count === 0) {
                html += '<button data-action="delete-api-key" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
            }
            html += '</div>';
            if (k.usage_count > 0) {
                html += '<div class="flex gap-1">';
                html += '<button data-action="view-bindings" data-idx="' + idx + '" class="px-2 py-1 bg-indigo-100 text-indigo-700 rounded text-xs">查看绑定</button>';
                html += '<button data-action="clear-bindings" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">清除绑定</button>';
                html += '</div>';
            }
            html += '</div>';
            html += '</div>';
            html += '</div>';
        });
        list.innerHTML = html;
    });
}

// Event delegation for API keys
document.getElementById('api-keys-list').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action]');
    if (!btn) return;
    var action = btn.getAttribute('data-action');
    var idx = parseInt(btn.getAttribute('data-idx'));
    var key = apiKeys[idx];
    if (!key) return;
    
    if (action === 'edit-api-key') {
        showAPIKeyForm(key);
    } else if (action === 'toggle-api-key') {
        toggleAPIKey(key.id);
    } else if (action === 'delete-api-key') {
        deleteAPIKey(key.id);
    } else if (action === 'view-bindings') {
        viewAPIKeyBindings(key.id, key.api_key);
    } else if (action === 'clear-bindings') {
        clearAPIKeyBindings(key.id, key.api_key, key.usage_count);
    }
});

function viewAPIKeyBindings(keyId, apiKey) {
    fetch('/api/api-keys/bindings?id=' + encodeURIComponent(keyId))
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            var records = data.records || [];
            var html = '<div class="p-6"><h3 class="text-lg font-bold mb-4">API Key 绑定记录</h3>';
            html += '<p class="text-xs text-slate-500 mb-3">API Key: <code class="bg-slate-100 px-1 rounded">' + escapeHtml(apiKey.substring(0, 20)) + '...</code></p>';
            
            if (records.length === 0) {
                html += '<p class="text-slate-500 text-center py-4">暂无绑定记录</p>';
            } else {
                html += '<div class="max-h-80 overflow-auto space-y-2">';
                records.forEach(function(r) {
                    html += '<div class="p-2 bg-slate-50 rounded text-xs">';
                    html += '<div class="flex justify-between items-center">';
                    html += '<span class="text-slate-600">' + escapeHtml(r.email) + '</span>';
                    html += '<code class="text-blue-600 font-mono">' + escapeHtml(r.sn) + '</code>';
                    html += '</div>';
                    html += '<p class="text-slate-400 mt-1">创建时间: ' + new Date(r.created_at).toLocaleString() + '</p>';
                    html += '</div>';
                });
                html += '</div>';
            }
            
            html += '<div class="mt-4"><button onclick="hideModal()" class="w-full py-2 bg-slate-200 rounded-lg">关闭</button></div>';
            html += '</div>';
            showModal(html);
        });
}

function clearAPIKeyBindings(keyId, apiKey, count) {
    if (!confirm('确定要清除此 API Key 创建的所有 ' + count + ' 个绑定吗？\\n\\n⚠️ 这将删除所有相关的邮箱记录和序列号！\\n此操作不可恢复！')) return;
    
    fetch('/api/api-keys/clear-bindings', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: keyId})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) {
            if (result.success) {
                alert('已清除 ' + result.deleted + ' 个绑定记录');
                loadAPIKeys();
                loadEmailRecords();
                loadLicenses();
            } else {
                alert('清除失败: ' + result.error);
            }
        });
}

function showAPIKeyForm(key) {
    var k = key || {id: '', product_id: 0, organization: '', contact_name: '', description: '', expires_at: '', is_active: true};
    var isEdit = !!k.id;
    
    var productOpts = '<option value="0">Vantagics (ID: 0)</option>';
    productTypes.forEach(function(p) { productOpts += '<option value="' + p.id + '"' + (p.id === k.product_id ? ' selected' : '') + '>' + escapeHtml(p.name) + ' (ID: ' + p.id + ')</option>'; });
    
    var expiresDate = k.expires_at ? new Date(k.expires_at).toISOString().split('T')[0] : '';
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (isEdit ? '编辑' : '创建') + ' API Key</h3><div class="space-y-3">' +
        '<input type="hidden" id="api-key-id" value="' + escapeHtml(k.id) + '">' +
        (isEdit ? '<div><label class="text-sm text-slate-600">API Key</label><input type="text" value="' + escapeHtml(k.api_key || '') + '" class="w-full px-3 py-2 border rounded-lg bg-slate-100 font-mono text-sm" disabled></div>' : '') +
        '<div><label class="text-sm text-slate-600">绑定产品 *</label><select id="api-key-product" class="w-full px-3 py-2 border rounded-lg"' + (isEdit ? ' disabled' : '') + '>' + productOpts + '</select></div>' +
        '<div class="grid grid-cols-2 gap-3">' +
        '<div><label class="text-sm text-slate-600">使用者单位</label><input type="text" id="api-key-org" value="' + escapeHtml(k.organization || '') + '" placeholder="公司/组织名称" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">联系人姓名</label><input type="text" id="api-key-contact" value="' + escapeHtml(k.contact_name || '') + '" placeholder="姓名" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '</div>' +
        '<div><label class="text-sm text-slate-600">有效期（留空为永久）</label><input type="date" id="api-key-expires" value="' + expiresDate + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">备注</label><input type="text" id="api-key-desc" value="' + escapeHtml(k.description || '') + '" placeholder="可选" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="saveAPIKey(' + (isEdit ? 'true' : 'false') + ')" class="flex-1 py-2 bg-green-600 text-white rounded-lg">' + (isEdit ? '保存' : '创建') + '</button></div>' +
        '</div></div>');
}

function saveAPIKey(isEdit) {
    var data = {
        id: document.getElementById('api-key-id').value,
        product_id: parseInt(document.getElementById('api-key-product').value) || 0,
        organization: document.getElementById('api-key-org').value.trim(),
        contact_name: document.getElementById('api-key-contact').value.trim(),
        expires_at: document.getElementById('api-key-expires').value,
        description: document.getElementById('api-key-desc').value.trim()
    };
    
    fetch('/api/api-keys', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            hideModal(); 
            if (result.success) {
                if (!isEdit && result.api_key) {
                    alert('API Key 创建成功！\\n\\n' + result.api_key + '\\n\\n请妥善保管此密钥。');
                }
                loadAPIKeys(); 
            } else {
                alert('操作失败: ' + result.error);
            }
        });
}

function toggleAPIKey(id) {
    fetch('/api/api-keys/toggle', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) {
                loadAPIKeys(); 
            } else {
                alert('操作失败: ' + result.error);
            }
        });
}

function deleteAPIKey(id) {
    if (!confirm('确定要删除此 API Key 吗？此操作不可恢复。')) return;
    
    fetch('/api/api-keys', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) {
                loadAPIKeys(); 
            } else {
                alert('删除失败: ' + result.error);
            }
        });
}

function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(function() {
        alert('已复制到剪贴板');
    }).catch(function() {
        prompt('请手动复制:', text);
    });
}
`
