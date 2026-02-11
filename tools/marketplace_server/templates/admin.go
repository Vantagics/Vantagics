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
        <a href="#categories" class="active" onclick="showSection('categories')">分类管理</a>
        <a href="#settings" onclick="showSection('settings')">系统设置</a>
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
    document.getElementById('section-categories').style.display = name === 'categories' ? '' : 'none';
    document.getElementById('section-settings').style.display = name === 'settings' ? '' : 'none';
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

// --- Category Management ---
function loadCategories() {
    fetch('/api/categories').then(function(r) { return r.json(); }).then(function(data) {
        var cats = data.categories || [];
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
            html += '<button class="btn btn-danger" onclick="deleteCategory(' + c.id + ',\'' + escAttr(c.name) + '\',' + c.pack_count + ')">删除</button>';
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
    fetch(url, {
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
    fetch('/api/admin/categories/' + id, { method: 'DELETE' })
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
    fetch('/admin/settings/initial-credits', {
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

// Init
loadCategories();
</script>
</body>
</html>`
