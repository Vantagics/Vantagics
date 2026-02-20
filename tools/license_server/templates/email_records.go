package templates

// EmailRecordsHTML contains the email records panel HTML
const EmailRecordsHTML = `
<div id="section-email-records" class="section">
    <div class="card">
        <div class="card-header">
            <h2 class="card-title">邮箱申请记录</h2>
            <div class="flex items-center gap-2 flex-wrap">
                <select id="email-product-filter" onchange="filterEmailsByProduct()" class="form-select" style="width:auto">
                    <option value="-1">全部产品</option>
                    <option value="0">Vantagics (ID: 0)</option>
                </select>
                <select id="email-license-group-filter" onchange="filterEmailsByLicenseGroup()" class="form-select" style="width:auto">
                    <option value="">全部序列号组</option>
                    <option value="none">默认(无组)</option>
                </select>
                <input type="text" id="email-search" placeholder="搜索邮箱或序列号..." class="form-input" style="width:12rem" onkeypress="if(event.key==='Enter')searchEmails()">
                <button onclick="searchEmails()" class="btn btn-primary btn-sm">搜索</button>
                <button onclick="showManualRequest()" class="btn btn-success btn-sm">+ 手工绑定</button>
            </div>
        </div>
        <div id="email-records-list"></div>
        <div id="email-pagination" class="pagination"></div>
    </div>
</div>
`

// EmailRecordsScripts contains the email records JavaScript
const EmailRecordsScripts = `
// Store email records data for button handlers
var emailRecordsData = {};
var emailProductFilter = -1;
var emailLicenseGroupFilter = '';

function initEmailProductFilter() {
    var select = document.getElementById('email-product-filter');
    if (!select) return;
    // Add product types from global productTypes array
    productTypes.forEach(function(p) {
        if (p.id === 0) return;
        var opt = document.createElement('option');
        opt.value = p.id;
        opt.textContent = p.name + ' (ID: ' + p.id + ')';
        select.appendChild(opt);
    });
}

function filterEmailsByProduct() {
    emailProductFilter = parseInt(document.getElementById('email-product-filter').value);
    loadEmailRecords(1, emailSearchTerm);
}

function filterEmailsByLicenseGroup() {
    emailLicenseGroupFilter = document.getElementById('email-license-group-filter').value;
    loadEmailRecords(1, emailSearchTerm);
}

function loadEmailRecords(page, search) {
    page = page || 1;
    search = search || '';
    emailCurrentPage = page;
    emailSearchTerm = search;
    
    var params = new URLSearchParams({page: page.toString(), pageSize: '15', search: search});
    if (emailProductFilter >= 0) {
        params.set('product_id', emailProductFilter.toString());
    }
    if (emailLicenseGroupFilter) {
        params.set('license_group', emailLicenseGroupFilter);
    }
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
            return fetch('/api/licenses/search?search=' + encodeURIComponent(sn) + '&pageSize=1&hide_used=false').then(function(r) { return r.json(); });
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
                var recordProductId = r.product_id || 0;
                var productName = getProductTypeName(recordProductId);
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
                    productId: license.product_id || 0,
                    expiresAt: license.expires_at || '',
                    dailyAnalysis: dailyAnalysis,
                    creditsMode: license.credits_mode || false,
                    totalCredits: license.total_credits || 0,
                    isActive: isActive
                };
                
                html += '<div class="p-3 bg-slate-50 rounded-lg ' + opacityClass + '">';
                html += '<div class="flex items-start justify-between">';
                html += '<div class="flex-1">';
                html += '<div class="flex items-center gap-3 mb-1">';
                html += '<span class="text-sm text-slate-600">' + escapeHtml(r.email) + '</span>';
                html += '<code class="font-mono text-blue-600 font-bold">' + escapeHtml(r.sn) + '</code>';
                html += '<span class="px-2 py-0.5 bg-amber-100 text-amber-700 text-xs rounded">📦 ' + (productName || 'Vantagics') + '</span>';
                // Show trust level badge
                var licenseGroup = licenseGroups.find(function(g) { return g.id === license.license_group_id; });
                var trustLevel = licenseGroup ? licenseGroup.trust_level : 'low';
                if (trustLevel === 'high') {
                    html += '<span class="px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">🔒 高可信(正式)</span>';
                } else {
                    html += '<span class="px-2 py-0.5 bg-orange-100 text-orange-700 text-xs rounded">⚠️ 低可信(试用)</span>';
                }
                if (!isActive) html += '<span class="px-2 py-0.5 bg-red-100 text-red-700 text-xs rounded">已禁用</span>';
                if (isExpired) html += '<span class="px-2 py-0.5 bg-orange-100 text-orange-700 text-xs rounded">已过期</span>';
                html += '</div>';
                html += '<p class="text-xs text-slate-400">申请时间: ' + new Date(r.created_at).toLocaleString() + ' | IP: ' + r.ip + '</p>';
                html += '<p class="text-xs text-slate-400">';
                if (expiresAt) html += '过期: <span class="' + (isExpired ? 'text-red-600' : '') + '">' + expiresAt.toLocaleDateString() + '</span> | ';
                html += (license.credits_mode ? 'Credits: ' + (license.total_credits > 0 ? '<span class="text-teal-600">已用 ' + (license.used_credits || 0) + ' / ' + license.total_credits + '</span>' : '无限制') : '每日分析: ' + (dailyAnalysis === 0 ? '无限' : dailyAnalysis + '次')) + ' | ';
                html += '序列号分组: <span class="text-purple-600">' + (licenseGroupName || '默认') + '</span> | ';
                html += 'LLM分组: <span class="text-blue-600">' + (llmGroupName || '默认') + '</span> | ';
                html += '搜索分组: <span class="text-green-600">' + (searchGroupName || '默认') + '</span>';
                html += '</p>';
                html += '</div>';
                html += '<div class="flex gap-2 flex-shrink-0">';
                html += '<button data-action="edit" data-key="' + dataKey + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs hover:bg-blue-200">修改</button>';
                html += '<button data-action="groups" data-key="' + dataKey + '" class="px-2 py-1 bg-indigo-100 text-indigo-700 rounded text-xs hover:bg-indigo-200">分组</button>';
                html += '<button data-action="extend" data-key="' + dataKey + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs hover:bg-blue-200">展期</button>';
                if (license.credits_mode) {
                    html += '<button data-action="credits" data-key="' + dataKey + '" class="px-2 py-1 bg-teal-100 text-teal-700 rounded text-xs hover:bg-teal-200">Credits</button>';
                    html += '<button data-action="usage-log" data-key="' + dataKey + '" class="px-2 py-1 bg-cyan-100 text-cyan-700 rounded text-xs hover:bg-cyan-200">使用记录</button>';
                } else {
                    html += '<button data-action="analysis" data-key="' + dataKey + '" class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs hover:bg-purple-200">分析次数</button>';
                }
                html += '<button data-action="switchmode" data-key="' + dataKey + '" class="px-2 py-1 bg-orange-100 text-orange-700 rounded text-xs hover:bg-orange-200">授权方式</button>';
                html += '<button data-action="toggle" data-key="' + dataKey + '" class="px-2 py-1 ' + (isActive ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700') + ' rounded text-xs hover:opacity-80">' + (isActive ? '禁用' : '启用') + '</button>';
                html += '<button data-action="delete" data-key="' + dataKey + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs hover:bg-red-200">删除</button>';
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
            setLicenseGroups(data.sn, data.licenseGroupId, data.llmGroupId, data.searchGroupId, data.productId);
            break;
        case 'extend':
            extendLicense(data.sn, data.expiresAt);
            break;
        case 'analysis':
            setDailyAnalysis(data.sn, data.dailyAnalysis);
            break;
        case 'credits':
            setCredits(data.sn, data.totalCredits);
            break;
        case 'switchmode':
            switchLicenseMode(data.sn, data.creditsMode, data.dailyAnalysis, data.totalCredits);
            break;
        case 'usage-log':
            showUsageLog(data.sn);
            break;
        case 'toggle':
            toggleLicenseFromEmail(data.sn);
            break;
        case 'delete':
            deleteLicenseFromEmail(data.sn, data.email);
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

function deleteLicenseFromEmail(sn, email) {
    if (!confirm('确定要删除序列号 ' + sn + ' 及其邮箱绑定记录（' + email + '）吗？\\n\\n⚠️ 此操作不可恢复！')) return;
    fetch('/api/licenses/force-delete', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) {
            if (result.success) {
                alert(result.message);
                refreshAllPanels();
            } else {
                alert('删除失败: ' + result.error);
            }
        });
}

function switchLicenseMode(sn, currentCreditsMode, dailyAnalysis, totalCredits) {
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">切换授权方式</h3><div class="space-y-3">' +
        '<p class="text-sm text-slate-600">序列号: <code class="font-mono text-blue-600">' + sn + '</code></p>' +
        '<p class="text-sm text-slate-500">当前模式: <span class="font-bold ' + (currentCreditsMode ? 'text-teal-600' : 'text-purple-600') + '">' + (currentCreditsMode ? 'Credits' : '每日限制') + '</span></p>' +
        '<div class="space-y-2">' +
        '<label class="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-slate-50' + (!currentCreditsMode ? ' border-purple-400 bg-purple-50' : '') + '">' +
        '<input type="radio" name="switch-mode" value="daily"' + (!currentCreditsMode ? ' checked' : '') + ' onchange="onSwitchModeChange()"> <span class="text-sm">📊 每日限制模式</span></label>' +
        '<label class="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-slate-50' + (currentCreditsMode ? ' border-teal-400 bg-teal-50' : '') + '">' +
        '<input type="radio" name="switch-mode" value="credits"' + (currentCreditsMode ? ' checked' : '') + ' onchange="onSwitchModeChange()"> <span class="text-sm">🪙 Credits 模式</span></label>' +
        '</div>' +
        '<div id="switch-mode-params">' +
        (currentCreditsMode ?
            '<div><label class="text-sm text-slate-600">Credits 总量 (0=无限制)</label><input type="number" id="switch-credits-value" value="' + totalCredits + '" step="0.5" class="w-full px-3 py-2 border rounded-lg"></div>' :
            '<div><label class="text-sm text-slate-600">每日分析次数 (0=无限)</label><input type="number" id="switch-daily-value" value="' + dailyAnalysis + '" class="w-full px-3 py-2 border rounded-lg"></div>'
        ) +
        '</div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doSwitchMode(\'' + sn + '\')" class="flex-1 py-2 bg-orange-600 text-white rounded-lg">确认切换</button></div>' +
        '</div></div>');
    // Store context for param switching
    window._switchModeCtx = {dailyAnalysis: dailyAnalysis, totalCredits: totalCredits};
}

function onSwitchModeChange() {
    var mode = document.querySelector('input[name="switch-mode"]:checked');
    if (!mode) return;
    var paramsDiv = document.getElementById('switch-mode-params');
    var ctx = window._switchModeCtx || {dailyAnalysis: 20, totalCredits: 1000};
    if (mode.value === 'credits') {
        paramsDiv.innerHTML = '<div><label class="text-sm text-slate-600">Credits 总量 (0=无限制)</label><input type="number" id="switch-credits-value" value="' + ctx.totalCredits + '" step="0.5" class="w-full px-3 py-2 border rounded-lg"></div>';
    } else {
        paramsDiv.innerHTML = '<div><label class="text-sm text-slate-600">每日分析次数 (0=无限)</label><input type="number" id="switch-daily-value" value="' + ctx.dailyAnalysis + '" class="w-full px-3 py-2 border rounded-lg"></div>';
    }
}

function doSwitchMode(sn) {
    var mode = document.querySelector('input[name="switch-mode"]:checked');
    if (!mode) { alert('请选择授权方式'); return; }
    if (mode.value === 'credits') {
        var credits = parseFloat(document.getElementById('switch-credits-value').value) || 0;
        fetch('/api/licenses/set-credits', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, total_credits: credits, credits_mode: true})})
            .then(function(resp) { return resp.json(); })
            .then(function(result) {
                hideModal();
                if (result.success) { refreshAllPanels(); } else { alert('切换失败: ' + (result.error || '未知错误')); }
            }).catch(function(err) { hideModal(); alert('请求失败: ' + err); });
    } else {
        var daily = parseInt(document.getElementById('switch-daily-value').value) || 0;
        // Switch to daily mode: set credits_mode=false, then set daily analysis
        fetch('/api/licenses/set-credits', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, total_credits: 0, credits_mode: false})})
            .then(function(resp) { return resp.json(); })
            .then(function(result) {
                if (!result.success) { hideModal(); alert('切换失败: ' + (result.error || '未知错误')); return; }
                return fetch('/api/licenses/set-daily', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn, daily_analysis: daily})});
            })
            .then(function(resp) { if (resp) return resp.json(); })
            .then(function() { hideModal(); refreshAllPanels(); })
            .catch(function(err) { hideModal(); alert('请求失败: ' + err); });
    }
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

function showManualRequest() {
    var productOpts = '<option value="0">Vantagics (ID: 0)</option>';
    productTypes.forEach(function(p) { if (p.id === 0) return; productOpts += '<option value="' + p.id + '">' + escapeHtml(p.name) + ' (ID: ' + p.id + ')</option>'; });
    
    var llmGroupOpts = '<option value="">默认</option>';
    llmGroups.forEach(function(g) { llmGroupOpts += '<option value="' + g.id + '">' + escapeHtml(g.name) + '</option>'; });
    
    var searchGroupOpts = '<option value="">默认</option>';
    searchGroups.forEach(function(g) { searchGroupOpts += '<option value="' + g.id + '">' + escapeHtml(g.name) + '</option>'; });
    
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">🎫 手工绑定序列号</h3><div class="space-y-3">' +
        '<p class="text-xs text-slate-500 bg-blue-50 p-2 rounded">此功能为指定邮箱创建新的高可信正式授权序列号，绑定到产品内置的正式授权组。</p>' +
        '<div><label class="text-sm text-slate-600">邮箱地址 *</label><input type="email" id="manual-email" placeholder="user@example.com" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">产品类型</label><select id="manual-product" class="w-full px-3 py-2 border rounded-lg">' + productOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">有效期（天）</label><input type="number" id="manual-days" value="365" min="1" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="grid grid-cols-2 gap-3">' +
        '<div><label class="text-sm text-slate-600">LLM 分组</label><select id="manual-llm-group" class="w-full px-3 py-2 border rounded-lg">' + llmGroupOpts + '</select></div>' +
        '<div><label class="text-sm text-slate-600">搜索引擎分组</label><select id="manual-search-group" class="w-full px-3 py-2 border rounded-lg">' + searchGroupOpts + '</select></div>' +
        '</div>' +
        '<div class="p-2 bg-green-50 rounded text-xs text-green-700">' +
        '<strong>✓ 高可信正式授权</strong>：每月刷新一次，分析次数无限制' +
        '</div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button><button onclick="doManualBind()" class="flex-1 py-2 bg-green-600 text-white rounded-lg">创建并绑定</button></div>' +
        '</div></div>');
}

function doManualBind() {
    var email = document.getElementById('manual-email').value.trim().toLowerCase();
    var productId = parseInt(document.getElementById('manual-product').value) || 0;
    var days = parseInt(document.getElementById('manual-days').value) || 365;
    var llmGroupId = document.getElementById('manual-llm-group').value;
    var searchGroupId = document.getElementById('manual-search-group').value;
    
    if (!email || !email.includes('@') || !email.includes('.')) {
        alert('请输入有效的邮箱地址');
        return;
    }
    
    if (days < 1) {
        alert('有效期必须大于0天');
        return;
    }
    
    fetch('/api/email-records/manual-bind', {
        method: 'POST', 
        headers: {'Content-Type': 'application/json'}, 
        body: JSON.stringify({
            email: email, 
            product_id: productId,
            days: days,
            llm_group_id: llmGroupId,
            search_group_id: searchGroupId
        })
    })
    .then(function(resp) { return resp.json(); })
    .then(function(result) { 
        hideModal(); 
        if (result.success) { 
            alert('绑定成功！\\n\\n序列号: ' + result.sn + '\\n有效期: ' + days + '天\\n授权类型: 高可信正式授权\\n分析次数: 无限制');
            emailProductFilter = -1;
            document.getElementById('email-product-filter').value = '-1';
            loadEmailRecords(1, email); 
        } else { 
            alert('绑定失败: ' + result.message); 
        } 
    })
    .catch(function(err) {
        hideModal();
        alert('请求失败: ' + err);
    });
}

function doManualRequest() {
    var email = document.getElementById('manual-email').value.trim().toLowerCase();
    var productId = parseInt(document.getElementById('manual-product').value) || 0;
    
    if (!email || !email.includes('@') || !email.includes('.')) {
        alert('请输入有效的邮箱地址');
        return;
    }
    
    fetch('/api/email-records/manual-request', {
        method: 'POST', 
        headers: {'Content-Type': 'application/json'}, 
        body: JSON.stringify({email: email, product_id: productId})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(result) { 
        hideModal(); 
        if (result.success) { 
            alert('申请成功！\\n\\n序列号: ' + result.sn + '\\n' + result.message);
            // Reset product filter to show all, then search by email
            emailProductFilter = -1;
            document.getElementById('email-product-filter').value = '-1';
            loadEmailRecords(1, email); 
        } else { 
            alert('申请失败: ' + result.message); 
        } 
    })
    .catch(function(err) {
        hideModal();
        alert('请求失败: ' + err);
    });
}

function showUsageLog(sn) {
    loadUsageLogPage(sn, 1);
}

function loadUsageLogPage(sn, page) {
    fetch('/api/credits-usage-log?sn=' + encodeURIComponent(sn) + '&page=' + page + '&pageSize=15')
        .then(function(resp) {
            if (!resp.ok) throw new Error('HTTP ' + resp.status);
            return resp.json();
        })
        .then(function(data) {
            var logs = data.records || [];
            var total = data.total || 0;
            var currentPage = data.page || 1;
            var totalPages = data.totalPages || 1;
            var totalCredits = data.total_credits || 0;
            var usedCredits = data.used_credits || 0;

            var usageColor = (totalCredits > 0 && usedCredits >= totalCredits) ? 'text-red-600' : 'text-teal-600';
            var usageText = usedCredits + ' / ' + (totalCredits > 0 ? totalCredits : '∞');

            var html = '<div class="p-6">';
            html += '<div class="flex justify-between items-center mb-4">';
            html += '<h3 class="text-lg font-bold">📊 Credits 使用记录</h3>';
            html += '<span class="text-sm font-mono font-bold ' + usageColor + '">已用 ' + usageText + '</span>';
            html += '</div>';
            html += '<p class="text-sm text-slate-600 mb-3">序列号: <code class="font-mono text-blue-600">' + escapeHtml(sn) + '</code></p>';

            if (logs.length === 0) {
                html += '<p class="text-slate-500 text-center py-4">暂无使用记录</p>';
            } else {
                html += '<div class="max-h-80 overflow-y-auto"><table class="w-full text-sm">';
                html += '<thead class="bg-slate-100 sticky top-0"><tr><th class="px-3 py-2 text-left">上报时间</th><th class="px-3 py-2 text-right">已用量</th><th class="px-3 py-2 text-left">客户端 IP</th></tr></thead>';
                html += '<tbody>';
                logs.forEach(function(log) {
                    html += '<tr class="border-b border-slate-100">';
                    html += '<td class="px-3 py-2 text-slate-600">' + new Date(log.reported_at).toLocaleString() + '</td>';
                    html += '<td class="px-3 py-2 text-right font-mono text-teal-600">' + log.used_credits + '</td>';
                    html += '<td class="px-3 py-2 text-slate-500">' + (log.client_ip || '-') + '</td>';
                    html += '</tr>';
                });
                html += '</tbody></table></div>';
            }

            // Pagination
            html += '<div class="flex justify-between items-center mt-3">';
            html += '<span class="text-xs text-slate-400">共 ' + total + ' 条记录</span>';
            if (totalPages > 1) {
                html += '<div class="flex items-center gap-1">';
                html += '<button onclick="loadUsageLogPage(\'' + escapeHtml(sn) + '\',' + 1 + ')" class="px-2 py-1 text-xs rounded ' + (currentPage === 1 ? 'text-slate-300 cursor-default' : 'hover:bg-slate-100 text-slate-600') + '"' + (currentPage === 1 ? ' disabled' : '') + '>首页</button>';
                html += '<button onclick="loadUsageLogPage(\'' + escapeHtml(sn) + '\',' + (currentPage - 1) + ')" class="px-2 py-1 text-xs rounded ' + (currentPage === 1 ? 'text-slate-300 cursor-default' : 'hover:bg-slate-100 text-slate-600') + '"' + (currentPage === 1 ? ' disabled' : '') + '>上一页</button>';
                html += '<span class="px-2 text-xs text-slate-500">' + currentPage + ' / ' + totalPages + '</span>';
                html += '<button onclick="loadUsageLogPage(\'' + escapeHtml(sn) + '\',' + (currentPage + 1) + ')" class="px-2 py-1 text-xs rounded ' + (currentPage === totalPages ? 'text-slate-300 cursor-default' : 'hover:bg-slate-100 text-slate-600') + '"' + (currentPage === totalPages ? ' disabled' : '') + '>下一页</button>';
                html += '<button onclick="loadUsageLogPage(\'' + escapeHtml(sn) + '\',' + totalPages + ')" class="px-2 py-1 text-xs rounded ' + (currentPage === totalPages ? 'text-slate-300 cursor-default' : 'hover:bg-slate-100 text-slate-600') + '"' + (currentPage === totalPages ? ' disabled' : '') + '>末页</button>';
                html += '</div>';
            }
            html += '</div>';

            html += '<div class="mt-4"><button onclick="hideModal()" class="w-full py-2 bg-slate-200 rounded-lg">关闭</button></div>';
            html += '</div>';
            showModal(html);
        })
        .catch(function(err) {
            alert('查询失败: ' + err);
        });
}
`
