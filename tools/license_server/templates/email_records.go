package templates

// EmailRecordsHTML contains the email records panel HTML
const EmailRecordsHTML = `
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
`

// EmailRecordsScripts contains the email records JavaScript
const EmailRecordsScripts = `
// Store email records data for button handlers
var emailRecordsData = {};

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
        
        // Fetch license info for all SNs
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
            
            // Clear and rebuild data store
            emailRecordsData = {};
            
            var html = '<div class="space-y-3">';
            data.records.forEach(function(r, idx) {
                var license = licenseMap[r.sn] || {};
                var isActive = license.is_active === true || license.is_active === 1;
                var expiresAt = license.expires_at ? new Date(license.expires_at) : null;
                var isExpired = expiresAt && expiresAt < new Date();
                var llmGroupName = getLLMGroupName(license.llm_group_id || '');
                var searchGroupName = getSearchGroupName(license.search_group_id || '');
                var licenseGroupName = getLicenseGroupName(license.license_group_id || '');
                var dailyAnalysis = license.daily_analysis !== undefined ? license.daily_analysis : 20;
                var opacityClass = !isActive ? 'opacity-50' : '';
                
                // Store data for this record
                var dataKey = 'rec_' + idx;
                emailRecordsData[dataKey] = {
                    id: r.id,
                    email: r.email,
                    sn: r.sn,
                    licenseGroupId: license.license_group_id || '',
                    llmGroupId: license.llm_group_id || '',
                    searchGroupId: license.search_group_id || '',
                    expiresAt: license.expires_at || '',
                    dailyAnalysis: dailyAnalysis,
                    isActive: isActive
                };
                
                html += '<div class="p-3 bg-slate-50 rounded-lg ' + opacityClass + '">';
                html += '<div class="flex items-start justify-between">';
                html += '<div class="flex-1">';
                html += '<div class="flex items-center gap-3 mb-1">';
                html += '<span class="text-sm text-slate-600">' + escapeHtml(r.email) + '</span>';
                html += '<code class="font-mono text-blue-600 font-bold">' + escapeHtml(r.sn) + '</code>';
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
                html += '<button data-action="edit" data-key="' + dataKey + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs hover:bg-blue-200">修改</button>';
                html += '<button data-action="groups" data-key="' + dataKey + '" class="px-2 py-1 bg-indigo-100 text-indigo-700 rounded text-xs hover:bg-indigo-200">分组</button>';
                html += '<button data-action="extend" data-key="' + dataKey + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs hover:bg-blue-200">展期</button>';
                html += '<button data-action="analysis" data-key="' + dataKey + '" class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs hover:bg-purple-200">分析次数</button>';
                html += '<button data-action="toggle" data-key="' + dataKey + '" class="px-2 py-1 ' + (isActive ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700') + ' rounded text-xs hover:opacity-80">' + (isActive ? '禁用' : '启用') + '</button>';
                html += '</div>';
                html += '</div>';
                html += '</div>';
            });
            html += '</div>';
            list.innerHTML = html;
        });
        
        // Pagination
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

// Event delegation for email records buttons
document.getElementById('email-records-list').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action]');
    if (!btn) return;
    
    var action = btn.getAttribute('data-action');
    var key = btn.getAttribute('data-key');
    var data = emailRecordsData[key];
    if (!data) return;
    
    switch(action) {
        case 'edit':
            editEmailRecord(data.id, data.email, data.sn);
            break;
        case 'groups':
            setLicenseGroups(data.sn, data.licenseGroupId, data.llmGroupId, data.searchGroupId);
            break;
        case 'extend':
            extendLicense(data.sn, data.expiresAt);
            break;
        case 'analysis':
            setDailyAnalysis(data.sn, data.dailyAnalysis);
            break;
        case 'toggle':
            toggleLicenseFromEmail(data.sn);
            break;
    }
});

function searchEmails() { 
    loadEmailRecords(1, document.getElementById('email-search').value); 
}

function toggleLicenseFromEmail(sn) {
    fetch('/api/licenses/toggle', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function() { loadEmailRecords(emailCurrentPage, emailSearchTerm); });
}

function editEmailRecord(id, email, sn) {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">修改申请记录</h3><div class="space-y-3">' +
        '<div><label class="text-sm text-slate-600">邮箱</label><input type="email" id="edit-email" value="' + escapeHtml(email) + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">序列号</label><input type="text" id="edit-sn" value="' + escapeHtml(sn) + '" class="w-full px-3 py-2 border rounded-lg font-mono"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doEditEmailRecord(' + id + ')" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
}

function doEditEmailRecord(id) {
    var email = document.getElementById('edit-email').value.trim();
    var sn = document.getElementById('edit-sn').value.trim().toUpperCase();
    if (!email || !sn) { alert('邮箱和序列号不能为空'); return; }
    
    fetch('/api/email-records/update', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id, email: email, sn: sn})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { hideModal(); if (result.success) { loadEmailRecords(emailCurrentPage, emailSearchTerm); } else { alert('修改失败: ' + result.error); } });
}
`
