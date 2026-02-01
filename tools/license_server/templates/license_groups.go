package templates

// LicenseGroupsHTML contains the license groups panel HTML
const LicenseGroupsHTML = `
<div id="panel-license-groups" class="tab-panel">
    <div class="bg-white rounded-xl shadow-sm p-6">
        <div class="flex justify-between items-center mb-4">
            <h2 class="text-lg font-bold text-slate-800">序列号分组管理</h2>
            <button onclick="showLicenseGroupForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加分组</button>
        </div>
        <p class="text-xs text-slate-500 mb-4">* 序列号分组用于组织和管理序列号，方便批量操作</p>
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
                html += '<div class="flex items-center justify-between p-3 bg-purple-50 rounded-lg">';
                html += '<div><span class="font-bold text-sm">' + escapeHtml(g.name) + '</span>';
                html += '<p class="text-xs text-slate-400">' + escapeHtml(g.description || '无描述') + '</p></div>';
                html += '<div class="flex gap-1">';
                html += '<button data-action="edit-license-group" data-idx="' + idx + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
                html += '<button data-action="delete-license-group" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
                html += '</div></div>'; 
            });
            list.innerHTML = html;
        }
        
        // Update filter dropdown
        var filterSelect = document.getElementById('license-group-filter');
        if (filterSelect) {
            var currentValue = filterSelect.value;
            var opts = '<option value="">全部序列号组</option><option value="none">默认(无组)</option>';
            licenseGroups.forEach(function(g) { opts += '<option value="' + g.id + '">' + escapeHtml(g.name) + '</option>'; });
            filterSelect.innerHTML = opts;
            filterSelect.value = currentValue;
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
    } else if (action === 'delete-license-group') {
        deleteLicenseGroup(group.id);
    }
});

function showLicenseGroupForm(group) {
    var g = group || {id: '', name: '', description: ''};
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (g.id ? '编辑' : '添加') + '序列号分组</h3><div class="space-y-3">' +
        '<input type="hidden" id="license-group-id" value="' + escapeHtml(g.id) + '">' +
        '<div><label class="text-sm text-slate-600">分组名称</label>' +
        '<input type="text" id="license-group-name" value="' + escapeHtml(g.name) + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">描述</label>' +
        '<input type="text" id="license-group-desc" value="' + escapeHtml(g.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="saveLicenseGroup()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
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
        .then(function() { loadLicenseGroups(); loadLicenses(licenseCurrentPage, licenseSearchTerm); });
}
`
