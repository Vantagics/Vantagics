package templates

// SettingsHTML contains the settings panel HTML
const SettingsHTML = `
<div id="section-settings" class="section">
    <div style="display:flex;gap:20px;flex-wrap:wrap;">
        <div class="card" style="flex:1;min-width:300px;">
            <h2 class="card-title mb-4">修改密码</h2>
            <div>
                <input type="password" id="old-password" placeholder="当前密码" class="form-input mb-2">
                <input type="password" id="new-password" placeholder="新密码" class="form-input mb-2">
                <button onclick="changePassword()" class="btn btn-primary w-full">修改密码</button>
            </div>
        </div>
        <div class="card" style="flex:1;min-width:300px;">
            <h2 class="card-title mb-4">端口配置</h2>
            <div>
                <div class="mb-2">
                    <label class="form-label">管理端口</label>
                    <input type="number" id="manage-port" value="{{.ManagePort}}" class="form-input">
                </div>
                <div class="mb-2">
                    <label class="form-label">授权端口</label>
                    <input type="number" id="auth-port" value="{{.AuthPort}}" class="form-input">
                </div>
                <button onclick="changePorts()" class="btn btn-primary w-full">保存端口配置</button>
                <p class="text-xs text-muted mt-2">* 修改端口后需要重启服务生效</p>
            </div>
        </div>
    </div>

    <div class="card">
        <h2 class="card-title mb-4">SSL/HTTPS 配置</h2>
        <div>
            <div class="flex items-center gap-3 mb-4">
                <input type="checkbox" id="use-ssl">
                <label class="text-sm">启用 HTTPS</label>
            </div>
            <div id="ssl-fields" class="hidden">
                <div class="mb-2">
                    <label class="form-label">SSL 证书文件路径</label>
                    <input type="text" id="ssl-cert" placeholder="/path/to/cert.pem" class="form-input">
                </div>
                <div class="mb-2">
                    <label class="form-label">SSL 密钥文件路径</label>
                    <input type="text" id="ssl-key" placeholder="/path/to/key.pem" class="form-input">
                </div>
            </div>
            <button onclick="saveSSLConfig()" class="btn btn-primary w-full">保存 SSL 配置</button>
            <p class="text-xs text-muted mt-2">* 修改 SSL 配置后需要重启服务生效</p>
        </div>
    </div>

    <div class="card">
        <h2 class="card-title mb-4">📧 SMTP 邮件配置</h2>
        <p class="text-sm text-muted mb-4">配置 SMTP 服务器后，用户申请序列号时会自动发送邮件通知</p>
        <div>
            <div class="flex items-center gap-3 mb-4">
                <input type="checkbox" id="smtp-enabled">
                <label class="text-sm">启用邮件发送</label>
            </div>
            <div id="smtp-fields" style="display:grid;grid-template-columns:1fr 1fr;gap:12px;">
                <div>
                    <label class="form-label">SMTP 服务器</label>
                    <input type="text" id="smtp-host" placeholder="smtp.example.com" class="form-input">
                </div>
                <div>
                    <label class="form-label">端口</label>
                    <input type="number" id="smtp-port" placeholder="587" class="form-input">
                </div>
                <div>
                    <label class="form-label">用户名</label>
                    <input type="text" id="smtp-username" placeholder="your@email.com" class="form-input">
                </div>
                <div>
                    <label class="form-label">密码/授权码</label>
                    <input type="password" id="smtp-password" placeholder="应用专用密码" class="form-input">
                </div>
                <div>
                    <label class="form-label">发件人邮箱</label>
                    <input type="text" id="smtp-from-email" placeholder="noreply@example.com" class="form-input">
                </div>
                <div>
                    <label class="form-label">发件人名称</label>
                    <input type="text" id="smtp-from-name" placeholder="VantageData" class="form-input">
                </div>
                <div style="grid-column:span 2;">
                    <label class="form-label">加密方式</label>
                    <div class="flex gap-3 mt-2">
                        <label class="flex items-center gap-2">
                            <input type="radio" name="smtp-encryption" value="starttls" checked>
                            <span class="text-sm">STARTTLS (端口 587)</span>
                        </label>
                        <label class="flex items-center gap-2">
                            <input type="radio" name="smtp-encryption" value="tls">
                            <span class="text-sm">SSL/TLS (端口 465)</span>
                        </label>
                        <label class="flex items-center gap-2">
                            <input type="radio" name="smtp-encryption" value="none">
                            <span class="text-sm">无加密 (不推荐)</span>
                        </label>
                    </div>
                </div>
            </div>
            <div class="flex gap-3 mt-4">
                <button onclick="saveSMTPConfig()" class="btn btn-primary" style="flex:1">保存配置</button>
                <button onclick="testSMTP()" class="btn btn-success">发送测试邮件</button>
            </div>
            <div class="mt-4" style="padding:12px;background:#f8fafc;border-radius:8px;">
                <p class="text-xs font-medium mb-2">常用 SMTP 服务器配置：</p>
                <ul class="text-xs text-muted" style="list-style:none;">
                    <li>• <strong>Gmail:</strong> smtp.gmail.com:587 (STARTTLS) - 需使用应用专用密码</li>
                    <li>• <strong>Outlook:</strong> smtp.office365.com:587 (STARTTLS)</li>
                    <li>• <strong>QQ邮箱:</strong> smtp.qq.com:587 (STARTTLS) - 需使用授权码</li>
                    <li>• <strong>163邮箱:</strong> smtp.163.com:465 (SSL/TLS) - 需使用授权码</li>
                    <li>• <strong>阿里企业邮:</strong> smtp.qiye.aliyun.com:465 (SSL/TLS)</li>
                </ul>
            </div>
        </div>
    </div>

    <div class="card">
        <h2 class="card-title text-danger mb-4">⚠️ 危险操作</h2>
        <div>
            <div class="flex items-center gap-3 mb-2">
                <button onclick="showClearIPRecords()" class="btn btn-warning">🌐 清除IP请求记录</button>
                <p class="text-xs text-muted">清除指定IP的所有SN请求次数记录，清除后该IP可重新申请序列号（方便测试）</p>
            </div>
            <div class="flex items-center gap-3 mb-2">
                <button onclick="showClearEmailRecords()" class="btn btn-warning">📧 清除邮箱记录</button>
                <p class="text-xs text-muted">清除指定邮箱的所有申请绑定记录，清除后该邮箱可重新申请序列号</p>
            </div>
            <div class="flex items-center gap-3">
                <button onclick="showForceDeleteLicense()" class="btn btn-danger">🗑️ 强制删除序列号</button>
                <p class="text-xs text-muted">强制删除指定序列号及其所有相关记录（邮箱申请记录等），此操作不可恢复</p>
            </div>
        </div>
    </div>
</div>
`

// SettingsScripts contains the settings JavaScript
const SettingsScripts = `
function changePassword() {
    var oldPwd = document.getElementById('old-password').value;
    var newPwd = document.getElementById('new-password').value;
    if (!oldPwd || !newPwd) { alert('请输入当前密码和新密码'); return; }
    
    fetch('/api/password', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({old_password: oldPwd, new_password: newPwd})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                alert('密码修改成功'); 
                document.getElementById('old-password').value = ''; 
                document.getElementById('new-password').value = ''; 
            } else { 
                alert('修改失败: ' + result.error); 
            } 
        });
}

function changePorts() {
    var managePort = parseInt(document.getElementById('manage-port').value);
    var authPort = parseInt(document.getElementById('auth-port').value);
    if (!managePort || !authPort) { alert('请输入有效的端口号'); return; }
    
    fetch('/api/ports', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({manage_port: managePort, auth_port: authPort})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                alert('端口配置已保存，请重启服务生效'); 
            } else { 
                alert('保存失败: ' + result.error); 
            } 
        });
}

function loadSSLConfig() {
    fetch('/api/ssl').then(function(resp) { return resp.json(); }).then(function(config) {
        document.getElementById('use-ssl').checked = config.use_ssl;
        document.getElementById('ssl-cert').value = config.ssl_cert || '';
        document.getElementById('ssl-key').value = config.ssl_key || '';
        toggleSSLFields();
    });
}

function toggleSSLFields() {
    var useSSL = document.getElementById('use-ssl').checked;
    document.getElementById('ssl-fields').classList.toggle('hidden', !useSSL);
}

function saveSSLConfig() {
    var useSSL = document.getElementById('use-ssl').checked;
    var sslCert = document.getElementById('ssl-cert').value;
    var sslKey = document.getElementById('ssl-key').value;
    
    fetch('/api/ssl', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({use_ssl: useSSL, ssl_cert: sslCert, ssl_key: sslKey})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                alert(result.message); 
            } else { 
                alert('保存失败: ' + result.error); 
            } 
        });
}

function loadSMTPConfig() {
    fetch('/api/smtp').then(function(resp) { return resp.json(); }).then(function(config) {
        document.getElementById('smtp-enabled').checked = config.enabled;
        document.getElementById('smtp-host').value = config.host || '';
        document.getElementById('smtp-port').value = config.port || 587;
        document.getElementById('smtp-username').value = config.username || '';
        document.getElementById('smtp-password').value = config.password || '';
        document.getElementById('smtp-from-email').value = config.from_email || '';
        document.getElementById('smtp-from-name').value = config.from_name || '';
        
        // Set encryption radio
        if (config.use_tls) {
            document.querySelector('input[name="smtp-encryption"][value="tls"]').checked = true;
        } else if (config.use_starttls) {
            document.querySelector('input[name="smtp-encryption"][value="starttls"]').checked = true;
        } else {
            document.querySelector('input[name="smtp-encryption"][value="none"]').checked = true;
        }
        
        toggleSMTPFields();
    });
}

function toggleSMTPFields() {
    var enabled = document.getElementById('smtp-enabled').checked;
    var fields = document.getElementById('smtp-fields');
    if (enabled) {
        fields.style.opacity = '1';
        fields.style.pointerEvents = 'auto';
    } else {
        fields.style.opacity = '0.5';
        fields.style.pointerEvents = 'none';
    }
}

function saveSMTPConfig() {
    var encryption = document.querySelector('input[name="smtp-encryption"]:checked').value;
    var config = {
        enabled: document.getElementById('smtp-enabled').checked,
        host: document.getElementById('smtp-host').value,
        port: parseInt(document.getElementById('smtp-port').value) || 587,
        username: document.getElementById('smtp-username').value,
        password: document.getElementById('smtp-password').value,
        from_email: document.getElementById('smtp-from-email').value,
        from_name: document.getElementById('smtp-from-name').value,
        use_tls: encryption === 'tls',
        use_starttls: encryption === 'starttls'
    };
    
    fetch('/api/smtp', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(config)})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) { 
                alert('SMTP 配置已保存'); 
            } else { 
                alert('保存失败: ' + result.error); 
            } 
        });
}

function testSMTP() {
    var email = prompt('请输入测试邮箱地址：');
    if (!email) return;
    
    // First save the config
    saveSMTPConfig();
    
    // Then send test email
    setTimeout(function() {
        fetch('/api/smtp/test', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({email: email})})
            .then(function(resp) { return resp.json(); })
            .then(function(result) { 
                if (result.success) { 
                    alert('测试邮件已发送，请检查收件箱（包括垃圾邮件文件夹）'); 
                } else { 
                    alert('发送失败: ' + result.error); 
                } 
            });
    }, 500);
}

function showForceDeleteLicense() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold text-red-600 mb-4">⚠️ 强制删除序列号</h3><div class="space-y-3">' +
        '<div><label class="text-sm text-slate-600">输入要删除的序列号</label>' +
        '<input type="text" id="force-delete-sn" placeholder="XXXX-XXXX-XXXX-XXXX" class="w-full px-3 py-2 border rounded-lg font-mono"></div>' +
        '<p class="text-xs text-red-500">警告：此操作将永久删除该序列号及其所有相关记录（包括邮箱申请记录），不可恢复！</p>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="doForceDeleteLicense()" class="flex-1 py-2 bg-red-600 text-white rounded-lg">确认删除</button></div>' +
        '</div></div>');
}

function showClearEmailRecords() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold text-orange-600 mb-4">📧 清除邮箱记录</h3><div class="space-y-3">' +
        '<div><label class="text-sm text-slate-600">输入要清除记录的邮箱地址</label>' +
        '<input type="email" id="clear-email-input" placeholder="user@example.com" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<p class="text-xs text-orange-500">清除后，该邮箱之前绑定的序列号将被释放，邮箱可重新申请新的序列号。</p>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="doClearEmailRecords()" class="flex-1 py-2 bg-orange-600 text-white rounded-lg">确认清除</button></div>' +
        '</div></div>');
}

function showClearIPRecords() {
    showModal('<div class="p-6"><h3 class="text-lg font-bold text-yellow-600 mb-4">🌐 清除IP请求记录</h3><div class="space-y-3">' +
        '<div><label class="text-sm text-slate-600">输入要清除记录的IP地址</label>' +
        '<input type="text" id="clear-ip-input" placeholder="192.168.1.1" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<p class="text-xs text-yellow-600">清除后，该IP的每日请求次数计数将被重置，可重新申请序列号。适用于测试场景。</p>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="doClearIPRecords()" class="flex-1 py-2 bg-yellow-600 text-white rounded-lg">确认清除</button></div>' +
        '</div></div>');
}

function doClearIPRecords() {
    var ip = document.getElementById('clear-ip-input').value.trim();
    if (!ip) { alert('请输入有效的IP地址'); return; }
    if (!confirm('确定要清除IP ' + ip + ' 的所有请求记录吗？\\n\\n清除后该IP可重新申请序列号。')) return;

    fetch('/api/settings/clear-ip-records', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ip: ip})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) {
            hideModal();
            if (result.success) {
                alert(result.message);
            } else {
                alert('清除失败: ' + result.error);
            }
        })
        .catch(function(err) { hideModal(); alert('请求失败: ' + err); });
}

function doClearEmailRecords() {
    var email = document.getElementById('clear-email-input').value.trim().toLowerCase();
    if (!email || !email.includes('@')) { alert('请输入有效的邮箱地址'); return; }
    if (!confirm('确定要清除邮箱 ' + email + ' 的所有申请记录吗？\\n\\n清除后该邮箱可重新申请序列号。')) return;

    fetch('/api/email-records/clear-by-email', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({email: email})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) {
            hideModal();
            if (result.success) {
                alert(result.message);
                refreshAllPanels();
            } else {
                alert('清除失败: ' + result.error);
            }
        })
        .catch(function(err) { hideModal(); alert('请求失败: ' + err); });
}

function doForceDeleteLicense() {
    var sn = document.getElementById('force-delete-sn').value.trim().toUpperCase();
    if (!sn) { alert('请输入序列号'); return; }
    if (!confirm('确定要强制删除序列号 ' + sn + ' 吗？\\n\\n此操作将删除：\\n- 序列号本身\\n- 相关的邮箱申请记录\\n\\n此操作不可恢复！')) return;
    
    fetch('/api/licenses/force-delete', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({sn: sn})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            hideModal(); 
            if (result.success) { 
                alert('序列号 ' + sn + ' 已被强制删除\\n\\n' + result.message); 
                loadLicenses(); 
                loadEmailRecords(); 
            } else { 
                alert('删除失败: ' + result.error); 
            } 
        });
}

// Initialize SSL toggle
document.getElementById('use-ssl').addEventListener('change', toggleSSLFields);

// Initialize SMTP toggle
document.getElementById('smtp-enabled').addEventListener('change', toggleSMTPFields);
`
