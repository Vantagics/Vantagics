package templates

// AdminHTML contains the marketplace admin panel HTML template.
const AdminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>市场管理后台</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f7fa; color: #333; }
        .container { max-width: 960px; margin: 0 auto; padding: 20px; }
        h1 { font-size: 24px; margin-bottom: 20px; color: #1a1a2e; }
        h2 { font-size: 18px; margin-bottom: 12px; color: #1a1a2e; }
        .card { background: #fff; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .nav { display: flex; gap: 12px; margin-bottom: 24px; }
        .nav a { padding: 8px 16px; background: #e8eaf0; border-radius: 6px; text-decoration: none; color: #333; font-size: 14px; }
        .nav a:hover, .nav a.active { background: #4361ee; color: #fff; }
        table { width: 100%; border-collapse: collapse; }
        th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid #eee; font-size: 14px; }
        th { background: #f8f9fb; font-weight: 600; color: #555; }
        .btn { display: inline-block; padding: 6px 14px; border: none; border-radius: 5px; cursor: pointer; font-size: 13px; text-decoration: none; }
        .btn-primary { background: #4361ee; color: #fff; }
        .btn-danger { background: #e63946; color: #fff; }
        .btn-secondary { background: #6c757d; color: #fff; }
        .btn:hover { opacity: 0.85; }
        input[type="text"], input[type="number"], textarea { width: 100%; padding: 8px 10px; border: 1px solid #ddd; border-radius: 5px; font-size: 14px; margin-bottom: 8px; }
        textarea { resize: vertical; min-height: 60px; }
        .form-group { margin-bottom: 12px; }
        .form-group label { display: block; font-size: 13px; color: #555; margin-bottom: 4px; font-weight: 500; }
        .msg { padding: 10px 14px; border-radius: 5px; margin-bottom: 16px; font-size: 14px; }
        .msg-success { background: #d4edda; color: #155724; }
        .msg-error { background: #f8d7da; color: #721c24; }
        .actions { display: flex; gap: 6px; }
        .badge { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 12px; }
        .badge-preset { background: #e0e7ff; color: #3730a3; }
        .modal-overlay { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.4); z-index: 100; justify-content: center; align-items: center; }
        .modal-overlay.show { display: flex; }
        .modal { background: #fff; border-radius: 8px; padding: 24px; width: 400px; max-width: 90%; }
        .modal h3 { margin-bottom: 16px; font-size: 16px; }
        .modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px; }
    </style>
</head>
<body>
<div class="container">
    <h1>📦 市场管理后台</h1>
    <div class="nav">
        {{if eq .Role "super"}}
        <a href="#categories" class="active" onclick="showSection('categories')">分类管理</a>
        <a href="#settings" onclick="showSection('settings')">系统设置</a>
        <a href="#admins" onclick="showSection('admins')">管理员管理</a>
        {{end}}
        <a href="#review" {{if ne .Role "super"}}class="active"{{end}} onclick="showSection('review')">审核管理</a>
        <a href="#profile" onclick="showSection('profile')">修改资料</a>
        <a href="/admin/logout" style="margin-left:auto; background:#e63946; color:#fff;">退出登录</a>
    </div>

    <div id="msg-area"></div>

    <!-- Categories Section -->
    <div id="section-categories">
        <div class="card">
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px;">
                <h2>分类管理</h2>
                <button class="btn btn-primary" onclick="showCreateCategory()">+ 新建分类</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>描述</th><th>分析包数</th><th>类型</th><th>操作</th></tr>
                </thead>
                <tbody id="category-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Settings Section -->
    <div id="section-settings" style="display:none;">
        <div class="card">
            <h2>初始 Credits 余额</h2>
            <p style="font-size:13px; color:#666; margin-bottom:12px;">新用户注册时自动获得的 Credits 数量</p>
            <form id="credits-form" onsubmit="saveInitialCredits(event)">
                <div class="form-group">
                    <label for="initial-credits">初始余额</label>
                    <input type="number" id="initial-credits" min="0" step="1" value="{{.InitialCredits}}" />
                </div>
                <button type="submit" class="btn btn-primary">保存设置</button>
            </form>
        </div>
    </div>

    <!-- Review Section (all admins) -->
    <div id="section-review" style="display:none;">
        <div class="card">
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px;">
                <h2>审核管理</h2>
                <button class="btn btn-secondary" onclick="loadPendingPacks()">刷新</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>分类</th><th>作者</th><th>模式</th><th>价格</th><th>上传时间</th><th>操作</th></tr>
                </thead>
                <tbody id="pending-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Admin Management Section (super only) -->
    {{if eq .Role "super"}}
    <div id="section-admins" style="display:none;">
        <div class="card">
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px;">
                <h2>管理员管理</h2>
                <button class="btn btn-primary" onclick="showAddAdminModal()">+ 添加管理员</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>用户名</th><th>角色</th><th>创建时间</th></tr>
                </thead>
                <tbody id="admin-list"></tbody>
            </table>
        </div>
    </div>
    {{end}}

    <!-- Profile Section (all admins) -->
    <div id="section-profile" style="display:none;">
        <div class="card">
            <h2>修改资料</h2>
            <div class="form-group">
                <label for="profile-username">用户名</label>
                <input type="text" id="profile-username" placeholder="新用户名（留空不修改）" />
            </div>
            <hr style="margin: 16px 0; border: none; border-top: 1px solid #eee;" />
            <h2 style="margin-top: 8px;">修改密码</h2>
            <div class="form-group">
                <label for="profile-old-password">当前密码</label>
                <input type="password" id="profile-old-password" placeholder="输入当前密码" />
            </div>
            <div class="form-group">
                <label for="profile-new-password">新密码（至少6个字符）</label>
                <input type="password" id="profile-new-password" placeholder="输入新密码" />
            </div>
            <button class="btn btn-primary" onclick="saveProfile()">保存修改</button>
        </div>
    </div>
</div>

<!-- Reject Reason Modal -->
<div id="reject-modal" class="modal-overlay">
    <div class="modal">
        <h3>拒绝审核</h3>
        <input type="hidden" id="reject-pack-id" value="" />
        <div class="form-group">
            <label for="reject-reason">拒绝原因（必填）</label>
            <textarea id="reject-reason" placeholder="请输入拒绝原因"></textarea>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideRejectModal()">取消</button>
            <button class="btn btn-danger" onclick="submitReject()">确认拒绝</button>
        </div>
    </div>
</div>

<!-- Add Admin Modal -->
<div id="add-admin-modal" class="modal-overlay">
    <div class="modal">
        <h3>添加管理员</h3>
        <div class="form-group">
            <label for="new-admin-username">用户名（至少3个字符）</label>
            <input type="text" id="new-admin-username" placeholder="输入用户名" />
        </div>
        <div class="form-group">
            <label for="new-admin-password">密码（至少6个字符）</label>
            <input type="text" id="new-admin-password" placeholder="输入密码" />
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideAddAdminModal()">取消</button>
            <button class="btn btn-primary" onclick="submitAddAdmin()">添加</button>
        </div>
    </div>
</div>

<!-- Create/Edit Category Modal -->
<div id="category-modal" class="modal-overlay">
    <div class="modal">
        <h3 id="modal-title">新建分类</h3>
        <input type="hidden" id="edit-category-id" value="" />
        <div class="form-group">
            <label for="cat-name">分类名称</label>
            <input type="text" id="cat-name" placeholder="输入分类名称" />
        </div>
        <div class="form-group">
            <label for="cat-desc">描述（可选）</label>
            <textarea id="cat-desc" placeholder="输入分类描述"></textarea>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideModal()">取消</button>
            <button class="btn btn-primary" onclick="saveCategory()">保存</button>
        </div>
    </div>
</div>

<script>
function showSection(name) {
    var sections = ['categories', 'settings', 'admins', 'review', 'profile'];
    for (var i = 0; i < sections.length; i++) {
        var el = document.getElementById('section-' + sections[i]);
        if (el) el.style.display = sections[i] === name ? '' : 'none';
    }
    var links = document.querySelectorAll('.nav a');
    for (var i = 0; i < links.length; i++) {
        links[i].className = links[i].getAttribute('href') === '#' + name ? 'active' : '';
    }
}

function showMsg(text, isError) {
    var area = document.getElementById('msg-area');
    area.innerHTML = '<div class="msg ' + (isError ? 'msg-error' : 'msg-success') + '">' + text + '</div>';
    setTimeout(function() { area.innerHTML = ''; }, 4000);
}

function apiFetch(url, opts) {
    return fetch(url, opts).then(function(r) {
        if (r.status === 401) {
            showMsg('会话已过期，正在跳转到登录页...', true);
            setTimeout(function() { window.location.href = '/admin/login'; }, 1500);
            return Promise.reject(new Error('session_expired'));
        }
        return r;
    });
}

// --- Category Management ---
function loadCategories() {
    apiFetch('/api/categories').then(function(r) { return r.json(); }).then(function(data) {
        var cats = Array.isArray(data) ? data : (data.categories || []);
        var tbody = document.getElementById('category-list');
        if (cats.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#999;">暂无分类</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < cats.length; i++) {
            var c = cats[i];
            html += '<tr>';
            html += '<td>' + c.id + '</td>';
            html += '<td>' + escHtml(c.name) + '</td>';
            html += '<td>' + escHtml(c.description || '-') + '</td>';
            html += '<td>' + c.pack_count + '</td>';
            html += '<td>' + (c.is_preset ? '<span class="badge badge-preset">预设</span>' : '自定义') + '</td>';
            html += '<td class="actions">';
            html += '<button class="btn btn-primary" onclick="showEditCategory(' + c.id + ',\'' + escAttr(c.name) + '\',\'' + escAttr(c.description || '') + '\')">编辑</button> ';
            if (!c.is_preset) {
                html += '<button class="btn btn-danger" onclick="deleteCategory(' + c.id + ',\'' + escAttr(c.name) + '\',' + c.pack_count + ')">删除</button>';
            }
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载分类失败: ' + err, true); });
}

function showCreateCategory() {
    document.getElementById('modal-title').textContent = '新建分类';
    document.getElementById('edit-category-id').value = '';
    document.getElementById('cat-name').value = '';
    document.getElementById('cat-desc').value = '';
    document.getElementById('category-modal').className = 'modal-overlay show';
}

function showEditCategory(id, name, desc) {
    document.getElementById('modal-title').textContent = '编辑分类';
    document.getElementById('edit-category-id').value = id;
    document.getElementById('cat-name').value = name;
    document.getElementById('cat-desc').value = desc;
    document.getElementById('category-modal').className = 'modal-overlay show';
}

function hideModal() {
    document.getElementById('category-modal').className = 'modal-overlay';
}

function saveCategory() {
    var id = document.getElementById('edit-category-id').value;
    var name = document.getElementById('cat-name').value.trim();
    var desc = document.getElementById('cat-desc').value.trim();
    if (!name) { alert('请输入分类名称'); return; }

    var url, method;
    if (id) {
        url = '/api/admin/categories/' + id;
        method = 'PUT';
    } else {
        url = '/api/admin/categories';
        method = 'POST';
    }
    apiFetch(url, {
        method: method,
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({name: name, description: desc})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) {
            hideModal();
            showMsg(id ? '分类已更新' : '分类已创建', false);
            loadCategories();
        } else {
            showMsg(res.data.error || '操作失败', true);
        }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function deleteCategory(id, name, packCount) {
    if (packCount > 0) {
        alert('分类 "' + name + '" 下有 ' + packCount + ' 个分析包，请先迁移后再删除。');
        return;
    }
    if (!confirm('确定要删除分类 "' + name + '" 吗？')) return;
    apiFetch('/api/admin/categories/' + id, { method: 'DELETE' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('分类已删除', false); loadCategories(); }
            else { showMsg(res.data.error || '删除失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Settings ---
function saveInitialCredits(e) {
    e.preventDefault();
    var val = document.getElementById('initial-credits').value;
    apiFetch('/admin/settings/initial-credits', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: 'value=' + encodeURIComponent(val)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg('初始余额已更新为 ' + val, false); }
        else { showMsg(res.data.error || '保存失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Helpers ---
function escHtml(s) { var d = document.createElement('div'); d.textContent = s; return d.innerHTML; }
function escAttr(s) { return s.replace(/\\/g,'\\\\').replace(/'/g,"\\'").replace(/"/g,'\\"'); }

// --- Admin Management ---
function loadAdmins() {
    apiFetch('/api/admin/admins').then(function(r) { return r.json(); }).then(function(data) {
        var admins = data.admins || [];
        var tbody = document.getElementById('admin-list');
        if (admins.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;color:#999;">暂无管理员</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < admins.length; i++) {
            var a = admins[i];
            var roleBadge = a.role === 'super' ? '<span class="badge badge-preset">超级管理员</span>' : '普通管理员';
            html += '<tr>';
            html += '<td>' + a.id + '</td>';
            html += '<td>' + escHtml(a.username) + '</td>';
            html += '<td>' + roleBadge + '</td>';
            html += '<td>' + a.created_at + '</td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载管理员列表失败: ' + err, true); });
}

function showAddAdminModal() {
    document.getElementById('new-admin-username').value = '';
    document.getElementById('new-admin-password').value = '';
    document.getElementById('add-admin-modal').className = 'modal-overlay show';
}

function hideAddAdminModal() {
    document.getElementById('add-admin-modal').className = 'modal-overlay';
}

function submitAddAdmin() {
    var username = document.getElementById('new-admin-username').value.trim();
    var password = document.getElementById('new-admin-password').value;
    if (username.length < 3) { alert('用户名至少3个字符'); return; }
    if (password.length < 6) { alert('密码至少6个字符'); return; }
    apiFetch('/api/admin/admins', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({username: username, password: password})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { hideAddAdminModal(); showMsg('管理员已添加', false); loadAdmins(); }
        else { showMsg(res.data.error || '添加失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Review Management ---
function loadPendingPacks() {
    apiFetch('/api/admin/review/pending').then(function(r) { return r.json(); }).then(function(data) {
        var packs = data || [];
        var tbody = document.getElementById('pending-list');
        if (packs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#999;">暂无待审核分析包</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < packs.length; i++) {
            var p = packs[i];
            html += '<tr>';
            html += '<td>' + p.id + '</td>';
            html += '<td>' + escHtml(p.pack_name) + '</td>';
            html += '<td>' + escHtml(p.category_name) + '</td>';
            html += '<td>' + escHtml(p.author_name || '-') + '</td>';
            html += '<td>' + p.share_mode + '</td>';
            html += '<td>' + (p.share_mode === 'paid' ? p.credits_price + ' Credits' : '免费') + '</td>';
            html += '<td>' + p.created_at + '</td>';
            html += '<td class="actions">';
            html += '<button class="btn btn-primary" onclick="approvePack(' + p.id + ')">通过</button> ';
            html += '<button class="btn btn-danger" onclick="showRejectModal(' + p.id + ')">拒绝</button>';
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载待审核列表失败: ' + err, true); });
}

function approvePack(id) {
    if (!confirm('确定通过审核？')) return;
    apiFetch('/api/admin/review/' + id + '/approve', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('审核已通过', false); loadPendingPacks(); }
            else { showMsg(res.data.error || '操作失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function showRejectModal(id) {
    document.getElementById('reject-pack-id').value = id;
    document.getElementById('reject-reason').value = '';
    document.getElementById('reject-modal').className = 'modal-overlay show';
}

function hideRejectModal() {
    document.getElementById('reject-modal').className = 'modal-overlay';
}

function submitReject() {
    var id = document.getElementById('reject-pack-id').value;
    var reason = document.getElementById('reject-reason').value.trim();
    if (!reason) { alert('请输入拒绝原因'); return; }
    apiFetch('/api/admin/review/' + id + '/reject', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({reason: reason})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { hideRejectModal(); showMsg('已拒绝', false); loadPendingPacks(); }
        else { showMsg(res.data.error || '操作失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Profile ---
function saveProfile() {
    var username = document.getElementById('profile-username').value.trim();
    var oldPassword = document.getElementById('profile-old-password').value;
    var newPassword = document.getElementById('profile-new-password').value;
    if (!username && !newPassword) { alert('请输入要修改的内容'); return; }
    if (newPassword && !oldPassword) { alert('修改密码需要输入当前密码'); return; }
    if (newPassword && newPassword.length < 6) { alert('新密码至少6个字符'); return; }
    var body = {};
    if (username) body.username = username;
    if (newPassword) { body.old_password = oldPassword; body.new_password = newPassword; }
    apiFetch('/api/admin/profile', {
        method: 'PUT',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(body)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) {
            showMsg('资料已更新', false);
            document.getElementById('profile-username').value = '';
            document.getElementById('profile-old-password').value = '';
            document.getElementById('profile-new-password').value = '';
        } else {
            var errMsg = res.data.error;
            if (errMsg === 'invalid_old_password') errMsg = '当前密码错误';
            else if (errMsg === 'username_already_exists') errMsg = '用户名已被使用';
            showMsg(errMsg || '修改失败', true);
        }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// Init
{{if eq .Role "super"}}
loadCategories();
loadAdmins();
showSection('categories');
{{else}}
loadPendingPacks();
showSection('review');
{{end}}
</script>
</body>
</html>`
