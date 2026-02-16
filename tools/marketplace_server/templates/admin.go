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
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f0f2f5; color: #1f2937; display: flex; min-height: 100vh; }

        /* Sidebar */
        .sidebar { width: 220px; background: #1e293b; color: #cbd5e1; display: flex; flex-direction: column; position: fixed; top: 0; left: 0; bottom: 0; z-index: 50; }
        .sidebar-brand { padding: 20px 20px 16px; border-bottom: 1px solid rgba(255,255,255,0.08); }
        .sidebar-brand h1 { font-size: 16px; color: #f1f5f9; font-weight: 700; letter-spacing: 0.5px; }
        .sidebar-brand span { font-size: 12px; color: #64748b; margin-top: 2px; display: block; }
        .sidebar-nav { flex: 1; padding: 12px 0; }
        .sidebar-nav a { display: flex; align-items: center; gap: 10px; padding: 10px 20px; color: #94a3b8; text-decoration: none; font-size: 14px; transition: all 0.15s; border-left: 3px solid transparent; }
        .sidebar-nav a:hover { color: #e2e8f0; background: rgba(255,255,255,0.04); }
        .sidebar-nav a.active { color: #fff; background: rgba(255,255,255,0.08); border-left-color: #3b82f6; }
        .sidebar-nav .nav-icon { width: 18px; text-align: center; font-size: 15px; }
        .sidebar-nav .nav-divider { height: 1px; background: rgba(255,255,255,0.06); margin: 8px 20px; }
        .sidebar-footer { padding: 16px 20px; border-top: 1px solid rgba(255,255,255,0.08); }
        .sidebar-footer a { display: flex; align-items: center; gap: 8px; color: #ef4444; text-decoration: none; font-size: 13px; padding: 8px 12px; border-radius: 6px; transition: background 0.15s; }
        .sidebar-footer a:hover { background: rgba(239,68,68,0.1); }

        /* Main content */
        .main-wrap { margin-left: 220px; flex: 1; min-height: 100vh; display: flex; flex-direction: column; }
        .topbar { background: #fff; border-bottom: 1px solid #e5e7eb; padding: 0 32px; height: 56px; display: flex; align-items: center; justify-content: space-between; position: sticky; top: 0; z-index: 40; }
        .topbar-title { font-size: 15px; font-weight: 600; color: #374151; }
        .topbar-user { font-size: 13px; color: #6b7280; display: flex; align-items: center; gap: 8px; }
        .topbar-user .avatar { width: 30px; height: 30px; border-radius: 50%; background: #3b82f6; color: #fff; display: flex; align-items: center; justify-content: center; font-size: 13px; font-weight: 600; }
        .content { flex: 1; padding: 24px 32px; }

        /* Cards & Typography */
        h2 { font-size: 16px; margin-bottom: 16px; color: #111827; font-weight: 600; }
        .card { background: #fff; border-radius: 10px; padding: 24px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04); border: 1px solid #e5e7eb; }
        .card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
        .card-header h2 { margin-bottom: 0; }

        /* Table */
        table { width: 100%; border-collapse: collapse; }
        th, td { text-align: left; padding: 12px 16px; font-size: 13px; }
        th { background: #f9fafb; font-weight: 600; color: #6b7280; text-transform: uppercase; font-size: 11px; letter-spacing: 0.5px; border-bottom: 2px solid #e5e7eb; }
        td { border-bottom: 1px solid #f3f4f6; color: #374151; }
        tr:hover td { background: #f9fafb; }

        /* Buttons */
        .btn { display: inline-flex; align-items: center; gap: 6px; padding: 7px 16px; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: 500; text-decoration: none; transition: all 0.15s; }
        .btn-primary { background: #3b82f6; color: #fff; }
        .btn-primary:hover { background: #2563eb; }
        .btn-danger { background: #ef4444; color: #fff; }
        .btn-danger:hover { background: #dc2626; }
        .btn-secondary { background: #f3f4f6; color: #374151; border: 1px solid #d1d5db; }
        .btn-secondary:hover { background: #e5e7eb; }
        .btn-sm { padding: 5px 12px; font-size: 12px; }

        /* Forms */
        input[type="text"], input[type="number"], input[type="password"], textarea { width: 100%; padding: 9px 12px; border: 1px solid #d1d5db; border-radius: 6px; font-size: 14px; color: #1f2937; background: #fff; transition: border-color 0.15s, box-shadow 0.15s; }
        input[type="text"]:focus, input[type="number"]:focus, input[type="password"]:focus, textarea:focus { outline: none; border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,0.1); }
        textarea { resize: vertical; min-height: 60px; }
        .form-group { margin-bottom: 16px; }
        .form-group label { display: block; font-size: 13px; color: #374151; margin-bottom: 6px; font-weight: 500; }
        .form-hint { font-size: 12px; color: #9ca3af; margin-top: 4px; }

        /* Messages */
        .msg { padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 13px; display: flex; align-items: center; gap: 8px; }
        .msg-success { background: #ecfdf5; color: #065f46; border: 1px solid #a7f3d0; }
        .msg-error { background: #fef2f2; color: #991b1b; border: 1px solid #fecaca; }

        /* Badges */
        .actions { display: flex; gap: 6px; }
        .badge { display: inline-block; padding: 3px 10px; border-radius: 12px; font-size: 11px; font-weight: 600; }
        .badge-preset { background: #ede9fe; color: #5b21b6; }

        /* Modal */
        .modal-overlay { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); backdrop-filter: blur(2px); z-index: 100; justify-content: center; align-items: center; }
        .modal-overlay.show { display: flex; }
        .modal { background: #fff; border-radius: 12px; padding: 28px; width: 440px; max-width: 90%; box-shadow: 0 20px 60px rgba(0,0,0,0.15); }
        .modal h3 { margin-bottom: 20px; font-size: 16px; font-weight: 600; color: #111827; }
        .modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 20px; }

        /* Profile section layout */
        .profile-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
        @media (max-width: 768px) { .profile-grid { grid-template-columns: 1fr; } }
        .profile-card { background: #fff; border-radius: 10px; padding: 24px; box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04); border: 1px solid #e5e7eb; }
        .profile-card h3 { font-size: 15px; font-weight: 600; color: #111827; margin-bottom: 16px; padding-bottom: 12px; border-bottom: 1px solid #f3f4f6; }
        .profile-card .icon-header { display: flex; align-items: center; gap: 8px; }
        .profile-card .icon-header span { font-size: 18px; }
    </style>
</head>
<body>
<!-- Sidebar -->
<aside class="sidebar">
    <div class="sidebar-brand">
        <h1>📦 市场管理</h1>
        <span>Marketplace Admin</span>
    </div>
    <nav class="sidebar-nav">
        <a href="#categories" data-perm="categories" onclick="showSection('categories')" style="display:none;"><span class="nav-icon">📂</span>分类管理</a>
        <a href="#marketplace" data-perm="marketplace" onclick="showSection('marketplace')" style="display:none;"><span class="nav-icon">🏪</span>市场管理</a>
        <a href="#authors" data-perm="authors" onclick="showSection('authors')" style="display:none;"><span class="nav-icon">✍️</span>作者管理</a>
        <a href="#review" data-perm="review" onclick="showSection('review')" style="display:none;"><span class="nav-icon">📋</span>审核管理</a>
        <a href="#settings" data-perm="settings" onclick="showSection('settings')" style="display:none;"><span class="nav-icon">⚙️</span>系统设置</a>
        <a href="#admins" data-perm="admin_manage" onclick="showSection('admins')" style="display:none;"><span class="nav-icon">👥</span>管理员管理</a>
        <div class="nav-divider"></div>
        <a href="#profile" onclick="showSection('profile')"><span class="nav-icon">👤</span>修改资料</a>
    </nav>
    <div class="sidebar-footer">
        <a href="/admin/logout">⏻ 退出登录</a>
    </div>
</aside>

<!-- Main -->
<div class="main-wrap">
    <header class="topbar">
        <div class="topbar-title" id="topbar-title">管理面板</div>
        <div class="topbar-user">
            <div class="avatar">A</div>
            <span>管理员</span>
        </div>
    </header>
    <main class="content">
    <div id="msg-area"></div>

    <!-- Categories Section -->
    <div id="section-categories">
        <div class="card">
            <div class="card-header">
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
            <p class="form-hint" style="margin-bottom:16px;">新用户注册时自动获得的 Credits 数量</p>
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
            <div class="card-header">
                <h2>审核管理</h2>
                <button class="btn btn-secondary" onclick="loadPendingPacks()">↻ 刷新</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>分类</th><th>作者</th><th>模式</th><th>价格</th><th>上传时间</th><th>操作</th></tr>
                </thead>
                <tbody id="pending-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Marketplace Management Section -->
    <div id="section-marketplace" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>市场管理 - 在售分析包</h2>
                <button class="btn btn-secondary" onclick="loadMarketplacePacks()">↻ 刷新</button>
            </div>
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <select id="mp-status-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="published">在售</option>
                    <option value="delisted">已下架</option>
                </select>
                <select id="mp-category-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="">全部分类</option>
                </select>
                <select id="mp-mode-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="">全部付费方式</option>
                    <option value="free">免费</option>
                    <option value="per_use">按次付费</option>
                    <option value="subscription">订阅</option>
                </select>
                <select id="mp-sort" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="downloads">按下载量排序</option>
                    <option value="price">按价格排序</option>
                    <option value="name">按名称排序</option>
                </select>
                <select id="mp-order" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="desc">降序</option>
                    <option value="asc">升序</option>
                </select>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>分类</th><th>作者</th><th>付费方式</th><th>价格</th><th>下载量</th><th>上架时间</th><th>操作</th></tr>
                </thead>
                <tbody id="marketplace-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Author Management Section -->
    <div id="section-authors" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>作者管理</h2>
                <button class="btn btn-secondary" onclick="loadAuthors()">↻ 刷新</button>
            </div>
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <input type="text" id="author-email-search" placeholder="按邮箱搜索..." style="width:240px;" onkeydown="if(event.key==='Enter')loadAuthors()" />
                <button class="btn btn-primary btn-sm" onclick="loadAuthors()">搜索</button>
                <select id="author-sort" onchange="loadAuthors()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="total_downloads">按总下载量</option>
                    <option value="total_packs">按总包数</option>
                    <option value="year_downloads">按年下载量</option>
                    <option value="year_revenue">按年收入</option>
                    <option value="month_downloads">按月下载量</option>
                    <option value="month_revenue">按月收入</option>
                </select>
                <select id="author-order" onchange="loadAuthors()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="desc">降序</option>
                    <option value="asc">升序</option>
                </select>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>邮箱</th><th>包数</th><th>总下载</th><th>总收入</th><th>年下载</th><th>年收入</th><th>月下载</th><th>月收入</th><th>操作</th></tr>
                </thead>
                <tbody id="author-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Author Detail Modal -->
    <div id="author-detail-modal" class="modal-overlay">
        <div class="modal" style="width:640px;">
            <h3 id="author-detail-title">作者销售详情</h3>
            <div id="author-detail-info" style="margin-bottom:16px;font-size:13px;color:#6b7280;"></div>
            <table>
                <thead>
                    <tr><th>包名</th><th>分类</th><th>付费方式</th><th>单价</th><th>下载量</th><th>总收入</th><th>状态</th></tr>
                </thead>
                <tbody id="author-detail-packs"></tbody>
            </table>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="hideAuthorDetailModal()">关闭</button>
            </div>
        </div>
    </div>

    <!-- Admin Management Section (id=1 only) -->
    <div id="section-admins" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>管理员管理</h2>
                <button class="btn btn-primary" onclick="showAddAdminModal()">+ 添加管理员</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>用户名</th><th>权限</th><th>创建时间</th></tr>
                </thead>
                <tbody id="admin-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Profile Section (all admins) -->
    <div id="section-profile" style="display:none;">
        <div class="profile-grid">
            <div class="profile-card">
                <h3><span class="icon-header"><span>👤</span> 修改资料</span></h3>
                <div class="form-group">
                    <label for="profile-username">用户名</label>
                    <input type="text" id="profile-username" placeholder="新用户名（留空不修改）" />
                    <div class="form-hint">修改后需要重新登录</div>
                </div>
            </div>
            <div class="profile-card">
                <h3><span class="icon-header"><span>🔒</span> 修改密码</span></h3>
                <div class="form-group">
                    <label for="profile-old-password">当前密码</label>
                    <input type="password" id="profile-old-password" placeholder="输入当前密码" />
                </div>
                <div class="form-group">
                    <label for="profile-new-password">新密码</label>
                    <input type="password" id="profile-new-password" placeholder="输入新密码" />
                    <div class="form-hint">至少 6 个字符</div>
                </div>
            </div>
        </div>
        <div style="margin-top: 20px; display: flex; justify-content: flex-end;">
            <button class="btn btn-primary" onclick="saveProfile()" style="padding: 10px 28px; font-size: 14px;">保存修改</button>
        </div>
    </div>

    </main>
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
        <div class="form-group">
            <label>权限设置</label>
            <div style="display:flex;flex-wrap:wrap;gap:12px;margin-top:6px;">
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="categories" class="new-admin-perm" /> 分类管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="marketplace" class="new-admin-perm" /> 市场管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="authors" class="new-admin-perm" /> 作者管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="review" class="new-admin-perm" /> 审核管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="settings" class="new-admin-perm" /> 系统设置
                </label>
            </div>
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
// Permission system
var adminID = {{.AdminID}};
var permissions = {{.PermissionsJSON}};
var permLabels = { categories: '分类管理', marketplace: '市场管理', authors: '作者管理', review: '审核管理', settings: '系统设置' };

function hasPerm(p) { return permissions.indexOf(p) !== -1; }
function isSuperAdmin() { return adminID === 1; }

// Initialize sidebar visibility based on permissions
(function initSidebar() {
    var links = document.querySelectorAll('.sidebar-nav a[data-perm]');
    for (var i = 0; i < links.length; i++) {
        var perm = links[i].getAttribute('data-perm');
        if (perm === 'admin_manage') {
            if (isSuperAdmin()) links[i].style.display = '';
        } else {
            if (hasPerm(perm)) links[i].style.display = '';
        }
    }
})();

function showSection(name) {
    var sections = ['categories', 'marketplace', 'authors', 'settings', 'admins', 'review', 'profile'];
    var titles = { categories: '分类管理', marketplace: '市场管理', authors: '作者管理', settings: '系统设置', admins: '管理员管理', review: '审核管理', profile: '修改资料' };
    for (var i = 0; i < sections.length; i++) {
        var el = document.getElementById('section-' + sections[i]);
        if (el) el.style.display = sections[i] === name ? '' : 'none';
    }
    var links = document.querySelectorAll('.sidebar-nav a');
    for (var i = 0; i < links.length; i++) {
        var href = links[i].getAttribute('href');
        if (href) links[i].className = href === '#' + name ? 'active' : '';
    }
    document.getElementById('topbar-title').textContent = titles[name] || '管理面板';
    if (name === 'categories') loadCategories();
    if (name === 'marketplace') loadMarketplacePacks();
    if (name === 'authors') loadAuthors();
    if (name === 'admins') loadAdmins();
    if (name === 'review') loadPendingPacks();
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
            var permDisplay;
            if (a.id === 1) {
                permDisplay = '<span class="badge badge-preset">超级管理员（全部权限）</span>';
            } else {
                var perms = a.permissions || [];
                if (perms.length === 0) {
                    permDisplay = '<span style="color:#999;">无权限</span>';
                } else {
                    permDisplay = '';
                    for (var j = 0; j < perms.length; j++) {
                        permDisplay += '<span class="badge" style="background:#e0f2fe;color:#0369a1;margin:2px;">' + (permLabels[perms[j]] || perms[j]) + '</span>';
                    }
                }
            }
            html += '<tr>';
            html += '<td>' + a.id + '</td>';
            html += '<td>' + escHtml(a.username) + '</td>';
            html += '<td>' + permDisplay + '</td>';
            html += '<td>' + a.created_at + '</td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载管理员列表失败: ' + err, true); });
}

function showAddAdminModal() {
    document.getElementById('new-admin-username').value = '';
    document.getElementById('new-admin-password').value = '';
    var checks = document.querySelectorAll('.new-admin-perm');
    for (var i = 0; i < checks.length; i++) checks[i].checked = false;
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
    var permCheckboxes = document.querySelectorAll('.new-admin-perm:checked');
    var perms = [];
    for (var i = 0; i < permCheckboxes.length; i++) { perms.push(permCheckboxes[i].value); }
    apiFetch('/api/admin/admins', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({username: username, password: password, permissions: perms})
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

// --- Marketplace Management ---
var mpCategoriesLoaded = false;
function loadMarketplaceCategoryFilter() {
    if (mpCategoriesLoaded) return;
    apiFetch('/api/categories').then(function(r) { return r.json(); }).then(function(data) {
        var cats = Array.isArray(data) ? data : (data.categories || []);
        var sel = document.getElementById('mp-category-filter');
        for (var i = 0; i < cats.length; i++) {
            var opt = document.createElement('option');
            opt.value = cats[i].id;
            opt.textContent = cats[i].name;
            sel.appendChild(opt);
        }
        mpCategoriesLoaded = true;
    });
}

function shareModeLabel(mode) {
    var labels = { free: '免费', per_use: '按次', subscription: '订阅' };
    return labels[mode] || mode;
}

function loadMarketplacePacks() {
    loadMarketplaceCategoryFilter();
    var status = document.getElementById('mp-status-filter').value;
    var catId = document.getElementById('mp-category-filter').value;
    var mode = document.getElementById('mp-mode-filter').value;
    var sort = document.getElementById('mp-sort').value;
    var order = document.getElementById('mp-order').value;
    document.querySelector('#section-marketplace .card-header h2').textContent = status === 'delisted' ? '市场管理 - 已下架分析包' : '市场管理 - 在售分析包';
    var url = '/api/admin/marketplace?status=' + status + '&sort=' + sort + '&order=' + order;
    if (catId) url += '&category_id=' + catId;
    if (mode) url += '&share_mode=' + mode;
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var packs = data.packs || [];
        var tbody = document.getElementById('marketplace-list');
        if (packs.length === 0) {
            var emptyMsg = status === 'delisted' ? '暂无已下架分析包' : '暂无在售分析包';
            tbody.innerHTML = '<tr><td colspan="9" style="text-align:center;color:#999;">' + emptyMsg + '</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < packs.length; i++) {
            var p = packs[i];
            var priceText = p.share_mode === 'free' ? '免费' : p.credits_price + ' Credits';
            html += '<tr>';
            html += '<td>' + p.id + '</td>';
            html += '<td>' + escHtml(p.pack_name) + '</td>';
            html += '<td>' + escHtml(p.category_name) + '</td>';
            html += '<td>' + escHtml(p.author_name || '-') + '</td>';
            html += '<td>' + shareModeLabel(p.share_mode) + '</td>';
            html += '<td>' + priceText + '</td>';
            html += '<td>' + p.download_count + '</td>';
            html += '<td>' + p.created_at + '</td>';
            if (status === 'delisted') {
                html += '<td><button class="btn btn-primary btn-sm" onclick="relistPack(' + p.id + ',\'' + escAttr(p.pack_name) + '\')">恢复在售</button></td>';
            } else {
                html += '<td><button class="btn btn-danger btn-sm" onclick="delistPack(' + p.id + ',\'' + escAttr(p.pack_name) + '\')">下架</button></td>';
            }
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载市场列表失败: ' + err, true); });
}

function delistPack(id, name) {
    if (!confirm('确定要下架 "' + name + '" 吗？（下架后不删除，可在数据库中恢复）')) return;
    apiFetch('/api/admin/marketplace/' + id + '/delist', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('已下架', false); loadMarketplacePacks(); }
            else { showMsg(res.data.error || '下架失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function relistPack(id, name) {
    if (!confirm('确定要恢复 "' + name + '" 为在售状态吗？')) return;
    apiFetch('/api/admin/marketplace/' + id + '/relist', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('已恢复在售', false); loadMarketplacePacks(); }
            else { showMsg(res.data.error || '恢复失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Author Management ---
function loadAuthors() {
    var email = document.getElementById('author-email-search').value.trim();
    var sort = document.getElementById('author-sort').value;
    var order = document.getElementById('author-order').value;
    var url = '/api/admin/authors?sort=' + sort + '&order=' + order;
    if (email) url += '&email=' + encodeURIComponent(email);
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var authors = data.authors || [];
        var tbody = document.getElementById('author-list');
        if (authors.length === 0) {
            tbody.innerHTML = '<tr><td colspan="11" style="text-align:center;color:#999;">暂无作者数据</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < authors.length; i++) {
            var a = authors[i];
            html += '<tr>';
            html += '<td>' + a.user_id + '</td>';
            html += '<td>' + escHtml(a.display_name) + '</td>';
            html += '<td>' + escHtml(a.email || '-') + '</td>';
            html += '<td>' + a.total_packs + '</td>';
            html += '<td>' + a.total_downloads + '</td>';
            html += '<td>' + a.total_revenue + '</td>';
            html += '<td>' + a.year_downloads + '</td>';
            html += '<td>' + a.year_revenue + '</td>';
            html += '<td>' + a.month_downloads + '</td>';
            html += '<td>' + a.month_revenue + '</td>';
            html += '<td><button class="btn btn-primary btn-sm" onclick="showAuthorDetail(' + a.user_id + ')">详情</button></td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载作者列表失败: ' + err, true); });
}

function showAuthorDetail(userId) {
    apiFetch('/api/admin/authors/' + userId).then(function(r) { return r.json(); }).then(function(data) {
        document.getElementById('author-detail-title').textContent = escHtml(data.display_name) + ' 的销售详情';
        document.getElementById('author-detail-info').textContent = '邮箱: ' + (data.email || '-');
        var packs = data.packs || [];
        var tbody = document.getElementById('author-detail-packs');
        if (packs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#999;">暂无分析包</td></tr>';
        } else {
            var html = '';
            for (var i = 0; i < packs.length; i++) {
                var p = packs[i];
                var statusLabel = p.status === 'published' ? '在售' : '已下架';
                html += '<tr>';
                html += '<td>' + escHtml(p.pack_name) + '</td>';
                html += '<td>' + escHtml(p.category_name) + '</td>';
                html += '<td>' + shareModeLabel(p.share_mode) + '</td>';
                html += '<td>' + (p.share_mode === 'free' ? '免费' : p.credits_price + ' Credits') + '</td>';
                html += '<td>' + p.download_count + '</td>';
                html += '<td>' + p.total_revenue + '</td>';
                html += '<td>' + statusLabel + '</td>';
                html += '</tr>';
            }
            tbody.innerHTML = html;
        }
        document.getElementById('author-detail-modal').className = 'modal-overlay show';
    }).catch(function(err) { showMsg('加载作者详情失败: ' + err, true); });
}

function hideAuthorDetailModal() {
    document.getElementById('author-detail-modal').className = 'modal-overlay';
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

// Init: show first available section based on permissions
(function initDefaultSection() {
    var order = ['categories', 'marketplace', 'authors', 'review', 'settings'];
    for (var i = 0; i < order.length; i++) {
        if (hasPerm(order[i])) {
            showSection(order[i]);
            return;
        }
    }
    // Fallback: every admin can access profile
    showSection('profile');
})();
</script>
</body>
</html>`
