package templates

// EmailNotifyHTML contains the email notification panel HTML
const EmailNotifyHTML = `
<div id="section-email-notify" class="section">

    <!-- 收件人选择区域 -->
    <div class="card">
        <div class="card-header">
            <h2 class="card-title">📬 收件人选择</h2>
            <div class="flex items-center gap-2">
                <span id="recipient-count" class="badge badge-info">已选 0 人</span>
            </div>
        </div>

        <!-- 模式切换 -->
        <div class="flex gap-2 mb-4">
            <button id="mode-product-btn" class="btn btn-primary btn-sm" onclick="switchRecipientMode('product')">按产品发送</button>
            <button id="mode-email-btn" class="btn btn-secondary btn-sm" onclick="switchRecipientMode('email')">按邮箱发送</button>
        </div>

        <!-- 按产品发送模式 -->
        <div id="recipient-mode-product">
            <div class="mb-2">
                <label class="form-label">选择产品类型</label>
                <select id="notify-product-select" class="form-select" style="width:auto;min-width:240px" onchange="onNotifyProductChange()">
                    <option value="">-- 请选择产品 --</option>
                    <option value="0">Vantagics (ID: 0)</option>
                </select>
            </div>
            <div id="product-recipient-info" class="text-sm text-muted"></div>
        </div>

        <!-- 按邮箱发送模式 -->
        <div id="recipient-mode-email" class="hidden">
            <div class="mb-2">
                <label class="form-label">搜索并选择邮箱</label>
                <div class="flex gap-2">
                    <input type="text" id="notify-email-search" class="form-input" style="flex:1" placeholder="输入邮箱关键词搜索..." onkeypress="if(event.key==='Enter')searchNotifyEmails()">
                    <button onclick="searchNotifyEmails()" class="btn btn-primary btn-sm">搜索</button>
                </div>
            </div>
            <div id="email-search-results" class="mb-2"></div>
            <div id="selected-emails-list" class="flex flex-wrap gap-2 mt-2"></div>
        </div>
    </div>

    <!-- 邮件编辑区域 -->
    <div class="card">
        <div class="card-header">
            <h2 class="card-title">✉️ 邮件编辑</h2>
        </div>

        <!-- 模板选择 -->
        <div class="mb-4">
            <label class="form-label">选择邮件模板</label>
            <select id="notify-template-select" class="form-select" style="width:auto;min-width:300px" onchange="onNotifyTemplateChange()">
                <option value="">-- 不使用模板 --</option>
            </select>
        </div>

        <!-- 标题输入 -->
        <div class="mb-4">
            <label class="form-label">邮件标题</label>
            <input type="text" id="notify-subject" class="form-input" placeholder="请输入邮件标题" oninput="updateNotifyPreview()">
        </div>

        <!-- 富文本编辑器 + 预览 -->
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px">
            <!-- 编辑器 -->
            <div>
                <label class="form-label">邮件内容</label>
                <div class="mb-2 flex gap-2 flex-wrap">
                    <button type="button" class="btn btn-secondary btn-sm" onclick="execNotifyCmd('bold')" title="加粗"><b>B</b></button>
                    <button type="button" class="btn btn-secondary btn-sm" onclick="execNotifyCmd('italic')" title="斜体"><i>I</i></button>
                    <button type="button" class="btn btn-secondary btn-sm" onclick="execNotifyCmd('insertUnorderedList')" title="无序列表">• 列表</button>
                    <button type="button" class="btn btn-secondary btn-sm" onclick="execNotifyCmd('insertOrderedList')" title="有序列表">1. 列表</button>
                    <button type="button" class="btn btn-secondary btn-sm" onclick="insertNotifyLink()" title="插入链接">🔗 链接</button>
                </div>
                <div id="notify-editor" contenteditable="true" style="min-height:200px;max-height:400px;overflow-y:auto;border:1px solid #cbd5e1;border-radius:6px;padding:12px;font-size:14px;line-height:1.6;outline:none;background:#fff" oninput="updateNotifyPreview()"></div>
            </div>
            <!-- 预览 -->
            <div>
                <label class="form-label">实时预览</label>
                <div id="notify-preview" style="min-height:200px;max-height:400px;overflow-y:auto;border:1px solid #e2e8f0;border-radius:6px;padding:12px;font-size:14px;line-height:1.6;background:#f8fafc;color:#334155"></div>
            </div>
        </div>
        <p class="text-xs text-muted mt-2">支持模板变量：{{.ProductName}}、{{.Email}}、{{.SN}}</p>
    </div>

    <!-- 发送控制区域 -->
    <div class="card">
        <div class="card-header">
            <h2 class="card-title">🚀 发送控制</h2>
        </div>

        <div class="flex items-center gap-3 mb-4">
            <button id="notify-send-btn" class="btn btn-primary" onclick="startNotifySend()" disabled>发送邮件</button>
            <button id="notify-cancel-btn" class="btn btn-danger hidden" onclick="cancelNotifySend()">取消发送</button>
        </div>

        <!-- 进度条 -->
        <div id="notify-progress-area" class="hidden">
            <div style="background:#e2e8f0;border-radius:9999px;height:20px;overflow:hidden;margin-bottom:8px">
                <div id="notify-progress-bar" style="height:100%;border-radius:9999px;background:linear-gradient(90deg,#22c55e,#3b82f6);width:0%;transition:width 0.3s"></div>
            </div>
            <div class="flex justify-between text-sm">
                <span>已发送: <strong id="notify-sent-count" class="text-success">0</strong></span>
                <span>失败: <strong id="notify-failed-count" class="text-danger">0</strong></span>
                <span>待发送: <strong id="notify-pending-count" class="text-muted">0</strong></span>
                <span>总计: <strong id="notify-total-count">0</strong></span>
            </div>
            <div id="notify-progress-status" class="text-sm text-muted mt-2"></div>
        </div>
    </div>

    <!-- 发送历史区域 -->
    <div class="card">
        <div class="card-header">
            <h2 class="card-title">📋 发送历史</h2>
            <button class="btn btn-secondary btn-sm" onclick="loadNotifyHistory()">刷新</button>
        </div>
        <div id="notify-history-list"></div>
        <div id="notify-history-pagination" class="pagination"></div>
    </div>

</div>
`

// EmailNotifyScripts contains the JavaScript logic for the email notification panel
const EmailNotifyScripts = `
// ===== Email Notify State Variables =====
var notifySelectedEmails = [];
var notifyCurrentTaskId = null;
var notifyProgressTimer = null;
var notifyTemplates = [];
var notifyHistoryPage = 1;

// ===== Initialization =====
function initEmailNotify() {
    loadNotifyTemplates();
    loadNotifyProducts();
    loadNotifyHistory();
}

// ===== Recipient Mode Switching =====
function switchRecipientMode(mode) {
    var productSection = document.getElementById('recipient-mode-product');
    var emailSection = document.getElementById('recipient-mode-email');
    var productBtn = document.getElementById('mode-product-btn');
    var emailBtn = document.getElementById('mode-email-btn');

    if (mode === 'product') {
        productSection.style.display = '';
        emailSection.style.display = 'none';
        productBtn.className = 'btn btn-primary btn-sm';
        emailBtn.className = 'btn btn-secondary btn-sm';
    } else {
        productSection.style.display = 'none';
        emailSection.style.display = '';
        productBtn.className = 'btn btn-secondary btn-sm';
        emailBtn.className = 'btn btn-primary btn-sm';
    }
    // Reset selections when switching mode
    notifySelectedEmails = [];
    updateRecipientCount();
}

// ===== Product Selection -> Load Recipients =====
function onNotifyProductChange() {
    var select = document.getElementById('notify-product-select');
    var productId = select.value;
    var info = document.getElementById('product-recipient-info');

    if (!productId) {
        info.innerHTML = '';
        notifySelectedEmails = [];
        updateRecipientCount();
        return;
    }

    info.innerHTML = '<span class="text-muted">正在查询收件人...</span>';
    fetch('/api/email-notify/recipients?product_id=' + encodeURIComponent(productId))
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.error) {
                info.innerHTML = '<span class="text-danger">查询失败: ' + escapeHtml(data.error) + '</span>';
                return;
            }
            notifySelectedEmails = data.emails || [];
            info.innerHTML = '<span class="text-success">找到 <strong>' + data.count + '</strong> 个收件人</span>';
            updateRecipientCount();
        })
        .catch(function(err) {
            info.innerHTML = '<span class="text-danger">查询失败: ' + escapeHtml(err.message) + '</span>';
        });
}

// ===== Email Search and Multi-Select =====
function searchNotifyEmails() {
    var keyword = document.getElementById('notify-email-search').value.trim();
    var resultsDiv = document.getElementById('email-search-results');

    if (!keyword) {
        resultsDiv.innerHTML = '<span class="text-muted">请输入搜索关键词</span>';
        return;
    }

    resultsDiv.innerHTML = '<span class="text-muted">搜索中...</span>';
    fetch('/api/email-notify/recipients?search=' + encodeURIComponent(keyword))
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.error) {
                resultsDiv.innerHTML = '<span class="text-danger">搜索失败: ' + escapeHtml(data.error) + '</span>';
                return;
            }
            var emails = data.emails || [];
            if (emails.length === 0) {
                resultsDiv.innerHTML = '<span class="text-muted">未找到匹配的邮箱</span>';
                return;
            }
            var html = '<div style="max-height:200px;overflow-y:auto;border:1px solid #e2e8f0;border-radius:6px;padding:8px">';
            for (var i = 0; i < emails.length; i++) {
                var checked = notifySelectedEmails.indexOf(emails[i]) >= 0 ? ' checked' : '';
                html += '<label style="display:block;padding:4px 0;cursor:pointer"><input type="checkbox" value="' + escapeHtml(emails[i]) + '" onchange="toggleNotifyEmail(this)"' + checked + '> ' + escapeHtml(emails[i]) + '</label>';
            }
            html += '</div>';
            resultsDiv.innerHTML = html;
        })
        .catch(function(err) {
            resultsDiv.innerHTML = '<span class="text-danger">搜索失败: ' + escapeHtml(err.message) + '</span>';
        });
}

function toggleNotifyEmail(checkbox) {
    var email = checkbox.value;
    var idx = notifySelectedEmails.indexOf(email);
    if (checkbox.checked && idx < 0) {
        notifySelectedEmails.push(email);
    } else if (!checkbox.checked && idx >= 0) {
        notifySelectedEmails.splice(idx, 1);
    }
    updateRecipientCount();
    renderSelectedEmails();
}

function removeNotifyEmail(email) {
    var idx = notifySelectedEmails.indexOf(email);
    if (idx >= 0) {
        notifySelectedEmails.splice(idx, 1);
    }
    updateRecipientCount();
    renderSelectedEmails();
    // Update checkboxes if visible
    var checkboxes = document.querySelectorAll('#email-search-results input[type=checkbox]');
    for (var i = 0; i < checkboxes.length; i++) {
        if (checkboxes[i].value === email) {
            checkboxes[i].checked = false;
        }
    }
}

function renderSelectedEmails() {
    var container = document.getElementById('selected-emails-list');
    if (notifySelectedEmails.length === 0) {
        container.innerHTML = '';
        return;
    }
    var html = '';
    for (var i = 0; i < notifySelectedEmails.length; i++) {
        html += '<span style="display:inline-flex;align-items:center;gap:4px;background:#e0f2fe;color:#0369a1;padding:2px 8px;border-radius:12px;font-size:12px">' + escapeHtml(notifySelectedEmails[i]) + ' <span style="cursor:pointer;font-weight:bold" onclick="removeNotifyEmail(\'' + escapeJs(notifySelectedEmails[i]) + '\')">&times;</span></span>';
    }
    container.innerHTML = html;
}

function updateRecipientCount() {
    var badge = document.getElementById('recipient-count');
    var sendBtn = document.getElementById('notify-send-btn');
    var count = notifySelectedEmails.length;
    badge.textContent = '已选 ' + count + ' 人';
    sendBtn.disabled = count === 0;
}

// ===== Template Loading and Selection =====
function loadNotifyTemplates() {
    fetch('/api/email-templates')
        .then(function(r) { return r.json(); })
        .then(function(data) {
            notifyTemplates = data || [];
            var select = document.getElementById('notify-template-select');
            // Keep the first default option
            select.innerHTML = '<option value="">-- 不使用模板 --</option>';
            for (var i = 0; i < notifyTemplates.length; i++) {
                var t = notifyTemplates[i];
                var label = t.IsPreset ? '[预置] ' : '';
                select.innerHTML += '<option value="' + t.ID + '">' + escapeHtml(label + t.Name) + '</option>';
            }
        })
        .catch(function(err) {
            console.error('加载模板失败:', err);
        });
}

function onNotifyTemplateChange() {
    var select = document.getElementById('notify-template-select');
    var templateId = parseInt(select.value);
    if (!templateId) return;

    var tmpl = null;
    for (var i = 0; i < notifyTemplates.length; i++) {
        if (notifyTemplates[i].ID === templateId) {
            tmpl = notifyTemplates[i];
            break;
        }
    }
    if (!tmpl) return;

    document.getElementById('notify-subject').value = tmpl.Subject;
    document.getElementById('notify-editor').innerHTML = tmpl.Body;
    updateNotifyPreview();
}

// ===== Preview =====
function updateNotifyPreview() {
    var subject = document.getElementById('notify-subject').value;
    var body = document.getElementById('notify-editor').innerHTML;
    var preview = document.getElementById('notify-preview');

    var html = '';
    if (subject) {
        html += '<div style="font-weight:bold;font-size:16px;margin-bottom:12px;padding-bottom:8px;border-bottom:1px solid #e2e8f0">' + escapeHtml(subject) + '</div>';
    }
    html += '<div>' + body + '</div>';
    preview.innerHTML = html;
}

// ===== Rich Text Editor Commands =====
function execNotifyCmd(cmd) {
    document.execCommand(cmd, false, null);
    document.getElementById('notify-editor').focus();
    updateNotifyPreview();
}

function insertNotifyLink() {
    var url = prompt('请输入链接地址:', 'https://');
    if (url) {
        document.execCommand('createLink', false, url);
        document.getElementById('notify-editor').focus();
        updateNotifyPreview();
    }
}

// ===== Send Task =====
function startNotifySend() {
    var subject = document.getElementById('notify-subject').value.trim();
    var body = document.getElementById('notify-editor').innerHTML.trim();

    if (!subject) {
        alert('请输入邮件标题');
        return;
    }
    if (!body || body === '<br>') {
        alert('请输入邮件内容');
        return;
    }
    if (notifySelectedEmails.length === 0) {
        alert('请选择收件人');
        return;
    }

    if (!confirm('确认向 ' + notifySelectedEmails.length + ' 个收件人发送邮件？')) {
        return;
    }

    var sendBtn = document.getElementById('notify-send-btn');
    sendBtn.disabled = true;
    sendBtn.textContent = '提交中...';

    fetch('/api/email-notify/send', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            subject: subject,
            body: body,
            emails: notifySelectedEmails
        })
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.error) {
            alert('发送失败: ' + data.error);
            sendBtn.disabled = false;
            sendBtn.textContent = '发送邮件';
            return;
        }
        notifyCurrentTaskId = data.taskId;
        showNotifyProgress(data.totalCount);
        startProgressPolling();
    })
    .catch(function(err) {
        alert('发送请求失败: ' + err.message);
        sendBtn.disabled = false;
        sendBtn.textContent = '发送邮件';
    });
}

function showNotifyProgress(total) {
    var progressArea = document.getElementById('notify-progress-area');
    var cancelBtn = document.getElementById('notify-cancel-btn');
    var sendBtn = document.getElementById('notify-send-btn');

    progressArea.style.display = '';
    progressArea.classList.remove('hidden');
    cancelBtn.style.display = '';
    cancelBtn.classList.remove('hidden');
    sendBtn.style.display = 'none';

    document.getElementById('notify-sent-count').textContent = '0';
    document.getElementById('notify-failed-count').textContent = '0';
    document.getElementById('notify-pending-count').textContent = String(total);
    document.getElementById('notify-total-count').textContent = String(total);
    document.getElementById('notify-progress-bar').style.width = '0%';
    document.getElementById('notify-progress-status').textContent = '发送中...';
}

function hideNotifyProgress() {
    var cancelBtn = document.getElementById('notify-cancel-btn');
    var sendBtn = document.getElementById('notify-send-btn');

    cancelBtn.style.display = 'none';
    sendBtn.style.display = '';
    sendBtn.disabled = notifySelectedEmails.length === 0;
    sendBtn.textContent = '发送邮件';
}

// ===== Progress Polling =====
function startProgressPolling() {
    stopProgressPolling();
    notifyProgressTimer = setInterval(function() {
        pollNotifyProgress();
    }, 3000);
}

function stopProgressPolling() {
    if (notifyProgressTimer) {
        clearInterval(notifyProgressTimer);
        notifyProgressTimer = null;
    }
}

function pollNotifyProgress() {
    if (!notifyCurrentTaskId) {
        stopProgressPolling();
        return;
    }

    fetch('/api/email-notify/progress/' + notifyCurrentTaskId)
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.error) {
                console.error('查询进度失败:', data.error);
                return;
            }

            var sent = data.sent || 0;
            var failed = data.failed || 0;
            var pending = data.pending || 0;
            var cancelled = data.cancelled || 0;
            var total = data.total || 1;
            var processed = sent + failed + cancelled;
            var pct = Math.round((processed / total) * 100);

            document.getElementById('notify-sent-count').textContent = String(sent);
            document.getElementById('notify-failed-count').textContent = String(failed);
            document.getElementById('notify-pending-count').textContent = String(pending);
            document.getElementById('notify-total-count').textContent = String(total);
            document.getElementById('notify-progress-bar').style.width = pct + '%';

            if (data.status === 'completed') {
                document.getElementById('notify-progress-status').textContent = '发送完成！成功 ' + sent + ' 封，失败 ' + failed + ' 封';
                stopProgressPolling();
                hideNotifyProgress();
                notifyCurrentTaskId = null;
                loadNotifyHistory();
            } else if (data.status === 'cancelled') {
                document.getElementById('notify-progress-status').textContent = '发送已取消。已发送 ' + sent + ' 封，失败 ' + failed + ' 封';
                stopProgressPolling();
                hideNotifyProgress();
                notifyCurrentTaskId = null;
                loadNotifyHistory();
            } else {
                document.getElementById('notify-progress-status').textContent = '发送中... ' + pct + '%';
            }
        })
        .catch(function(err) {
            console.error('轮询进度失败:', err);
        });
}

// ===== Cancel Send =====
function cancelNotifySend() {
    if (!notifyCurrentTaskId) return;

    if (!confirm('确认取消发送任务？已发送的邮件不受影响。')) {
        return;
    }

    fetch('/api/email-notify/cancel/' + notifyCurrentTaskId, {
        method: 'POST'
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.error) {
            alert('取消失败: ' + data.error);
            return;
        }
        document.getElementById('notify-progress-status').textContent = '正在取消...';
    })
    .catch(function(err) {
        alert('取消请求失败: ' + err.message);
    });
}

// ===== Send History =====
function loadNotifyHistory() {
    var container = document.getElementById('notify-history-list');
    container.innerHTML = '<span class="text-muted">加载中...</span>';

    fetch('/api/email-history?page=' + notifyHistoryPage + '&pageSize=10')
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.error) {
                container.innerHTML = '<span class="text-danger">加载失败: ' + escapeHtml(data.error) + '</span>';
                return;
            }

            var tasks = data.tasks || [];
            if (tasks.length === 0) {
                container.innerHTML = '<span class="text-muted">暂无发送记录</span>';
                document.getElementById('notify-history-pagination').innerHTML = '';
                return;
            }

            var html = '<table class="table"><thead><tr><th>时间</th><th>标题</th><th>总数</th><th>成功</th><th>失败</th><th>状态</th><th>操作</th></tr></thead><tbody>';
            for (var i = 0; i < tasks.length; i++) {
                var t = tasks[i];
                var statusBadge = '';
                if (t.status === 'completed') {
                    statusBadge = '<span class="badge badge-success">已完成</span>';
                } else if (t.status === 'cancelled') {
                    statusBadge = '<span class="badge badge-warning">已取消</span>';
                } else if (t.status === 'running') {
                    statusBadge = '<span class="badge badge-info">发送中</span>';
                } else {
                    statusBadge = '<span class="badge">' + escapeHtml(t.status) + '</span>';
                }
                html += '<tr>';
                html += '<td>' + escapeHtml(t.created_at) + '</td>';
                html += '<td>' + escapeHtml(t.subject) + '</td>';
                html += '<td>' + t.total_count + '</td>';
                html += '<td class="text-success">' + t.sent_count + '</td>';
                html += '<td class="text-danger">' + t.failed_count + '</td>';
                html += '<td>' + statusBadge + '</td>';
                html += '<td><button class="btn btn-secondary btn-sm" onclick="viewNotifyHistoryDetail(' + t.id + ')">详情</button></td>';
                html += '</tr>';
            }
            html += '</tbody></table>';
            container.innerHTML = html;

            // Pagination
            renderNotifyHistoryPagination(data.page, data.totalPages);
        })
        .catch(function(err) {
            container.innerHTML = '<span class="text-danger">加载失败: ' + escapeHtml(err.message) + '</span>';
        });
}

function renderNotifyHistoryPagination(currentPage, totalPages) {
    var container = document.getElementById('notify-history-pagination');
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }
    var html = '';
    if (currentPage > 1) {
        html += '<button class="btn btn-secondary btn-sm" onclick="notifyHistoryPage=' + (currentPage - 1) + ';loadNotifyHistory()">上一页</button> ';
    }
    html += '<span class="text-sm text-muted">第 ' + currentPage + ' / ' + totalPages + ' 页</span>';
    if (currentPage < totalPages) {
        html += ' <button class="btn btn-secondary btn-sm" onclick="notifyHistoryPage=' + (currentPage + 1) + ';loadNotifyHistory()">下一页</button>';
    }
    container.innerHTML = html;
}

function viewNotifyHistoryDetail(taskId) {
    fetch('/api/email-history/' + taskId)
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.error) {
                alert('加载详情失败: ' + data.error);
                return;
            }

            var task = data.task;
            var items = data.items || [];

            var statusText = task.status === 'completed' ? '已完成' : task.status === 'cancelled' ? '已取消' : task.status === 'running' ? '发送中' : task.status;

            var html = '<div style="max-width:700px">';
            html += '<h3 style="margin:0 0 16px 0">发送任务详情</h3>';
            html += '<div style="display:grid;grid-template-columns:auto 1fr;gap:8px 16px;margin-bottom:16px">';
            html += '<span class="text-muted">标题:</span><span>' + escapeHtml(task.subject) + '</span>';
            html += '<span class="text-muted">状态:</span><span>' + escapeHtml(statusText) + '</span>';
            html += '<span class="text-muted">创建时间:</span><span>' + escapeHtml(task.created_at) + '</span>';
            if (task.completed_at) {
                html += '<span class="text-muted">完成时间:</span><span>' + escapeHtml(task.completed_at) + '</span>';
            }
            html += '<span class="text-muted">总数/成功/失败:</span><span>' + task.total_count + ' / ' + task.sent_count + ' / ' + task.failed_count + '</span>';
            html += '</div>';

            if (items.length > 0) {
                html += '<div style="max-height:400px;overflow-y:auto"><table class="table"><thead><tr><th>邮箱</th><th>状态</th><th>发送时间</th><th>错误</th></tr></thead><tbody>';
                for (var i = 0; i < items.length; i++) {
                    var item = items[i];
                    var itemStatusBadge = '';
                    if (item.status === 'sent') {
                        itemStatusBadge = '<span class="badge badge-success">已发送</span>';
                    } else if (item.status === 'failed') {
                        itemStatusBadge = '<span class="badge badge-danger">失败</span>';
                    } else if (item.status === 'cancelled') {
                        itemStatusBadge = '<span class="badge badge-warning">已取消</span>';
                    } else {
                        itemStatusBadge = '<span class="badge">待发送</span>';
                    }
                    html += '<tr>';
                    html += '<td>' + escapeHtml(item.email) + '</td>';
                    html += '<td>' + itemStatusBadge + '</td>';
                    html += '<td>' + (item.sent_at ? escapeHtml(item.sent_at) : '-') + '</td>';
                    html += '<td>' + (item.error ? escapeHtml(item.error) : '-') + '</td>';
                    html += '</tr>';
                }
                html += '</tbody></table></div>';
            }

            html += '<div style="text-align:right;margin-top:16px"><button class="btn btn-secondary" onclick="hideModal()">关闭</button></div>';
            html += '</div>';

            showModal(html);
        })
        .catch(function(err) {
            alert('加载详情失败: ' + err.message);
        });
}

// ===== Load Products for Dropdown =====
function loadNotifyProducts() {
    fetch('/api/product-types')
        .then(function(r) { return r.json(); })
        .then(function(data) {
            var products = data || [];
            var select = document.getElementById('notify-product-select');
            select.innerHTML = '<option value="">-- 请选择产品 --</option>';
            for (var i = 0; i < products.length; i++) {
                var p = products[i];
                var pid = p.id !== undefined ? p.id : p.ID;
                var pname = p.name || p.Name || ('产品 ' + pid);
                select.innerHTML += '<option value="' + pid + '">' + escapeHtml(pname) + ' (ID: ' + pid + ')</option>';
            }
        })
        .catch(function(err) {
            console.error('加载产品类型失败:', err);
        });
}
`
