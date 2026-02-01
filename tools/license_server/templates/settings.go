package templates

// SettingsHTML contains the settings panel HTML
const SettingsHTML = `
<div id="panel-settings" class="tab-panel">
    <div class="grid grid-cols-2 gap-6">
        <!-- Change Password -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <h2 class="text-lg font-bold text-slate-800 mb-4">修改密码</h2>
            <div class="space-y-3">
                <input type="password" id="old-password" placeholder="当前密码" class="w-full px-3 py-2 border rounded-lg">
                <input type="password" id="new-password" placeholder="新密码" class="w-full px-3 py-2 border rounded-lg">
                <button onclick="changePassword()" class="w-full bg-blue-600 text-white py-2 rounded-lg">修改密码</button>
            </div>
        </div>
        
        <!-- Port Configuration -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <h2 class="text-lg font-bold text-slate-800 mb-4">端口配置</h2>
            <div class="space-y-3">
                <div>
                    <label class="text-sm text-slate-600">管理端口</label>
                    <input type="number" id="manage-port" value="{{.ManagePort}}" class="w-full px-3 py-2 border rounded-lg">
                </div>
                <div>
                    <label class="text-sm text-slate-600">授权端口</label>
                    <input type="number" id="auth-port" value="{{.AuthPort}}" class="w-full px-3 py-2 border rounded-lg">
                </div>
                <button onclick="changePorts()" class="w-full bg-blue-600 text-white py-2 rounded-lg">保存端口配置</button>
                <p class="text-xs text-slate-500">* 修改端口后需要重启服务生效</p>
            </div>
        </div>
        
        <!-- SSL Configuration -->
        <div class="bg-white rounded-xl shadow-sm p-6 col-span-2">
            <h2 class="text-lg font-bold text-slate-800 mb-4">SSL/HTTPS 配置</h2>
            <div class="space-y-3">
                <div class="flex items-center gap-3">
                    <input type="checkbox" id="use-ssl" class="w-4 h-4">
                    <label class="text-sm text-slate-700">启用 HTTPS</label>
                </div>
                <div id="ssl-fields" class="space-y-3 hidden">
                    <div>
                        <label class="text-sm text-slate-600">SSL 证书文件路径</label>
                        <input type="text" id="ssl-cert" placeholder="/path/to/cert.pem" class="w-full px-3 py-2 border rounded-lg">
                    </div>
                    <div>
                        <label class="text-sm text-slate-600">SSL 密钥文件路径</label>
                        <input type="text" id="ssl-key" placeholder="/path/to/key.pem" class="w-full px-3 py-2 border rounded-lg">
                    </div>
                </div>
                <button onclick="saveSSLConfig()" class="w-full bg-blue-600 text-white py-2 rounded-lg">保存 SSL 配置</button>
                <p class="text-xs text-slate-500">* 修改 SSL 配置后需要重启服务生效</p>
            </div>
        </div>
        
        <!-- SMTP Configuration -->
        <div class="bg-white rounded-xl shadow-sm p-6 col-span-2">
            <h2 class="text-lg font-bold text-slate-800 mb-4">📧 SMTP 邮件配置</h2>
            <p class="text-sm text-slate-500 mb-4">配置 SMTP 服务器后，用户申请序列号时会自动发送邮件通知</p>
            <div class="space-y-4">
                <div class="flex items-center gap-3">
                    <input type="checkbox" id="smtp-enabled" class="w-4 h-4">
                    <label class="text-sm text-slate-700">启用邮件发送</label>
                </div>
                <div id="smtp-fields" class="grid grid-cols-2 gap-4">
                    <div>
                        <label class="text-sm text-slate-600">SMTP 服务器</label>
                        <input type="text" id="smtp-host" placeholder="smtp.example.com" class="w-full px-3 py-2 border rounded-lg">
                    </div>
                    <div>
                        <label class="text-sm text-slate-600">端口</label>
                        <input type="number" id="smtp-port" placeholder="587" class="w-full px-3 py-2 border rounded-lg">
                    </div>
                    <div>
                        <label class="text-sm text-slate-600">用户名</label>
                        <input type="text" id="smtp-username" placeholder="your@email.com" class="w-full px-3 py-2 border rounded-lg">
                    </div>
                    <div>
                        <label class="text-sm text-slate-600">密码/授权码</label>
                        <input type="password" id="smtp-password" placeholder="应用专用密码" class="w-full px-3 py-2 border rounded-lg">
                    </div>
                    <div>
                        <label class="text-sm text-slate-600">发件人邮箱</label>
                        <input type="text" id="smtp-from-email" placeholder="noreply@example.com" class="w-full px-3 py-2 border rounded-lg">
                    </div>
                    <div>
                        <label class="text-sm text-slate-600">发件人名称</label>
                        <input type="text" id="smtp-from-name" placeholder="VantageData" class="w-full px-3 py-2 border rounded-lg">
                    </div>
                    <div class="col-span-2">
                        <label class="text-sm text-slate-600">加密方式</label>
                        <div class="flex gap-4 mt-2">
                            <label class="flex items-center gap-2">
                                <input type="radio" name="smtp-encryption" value="starttls" checked class="w-4 h-4">
                                <span class="text-sm">STARTTLS (端口 587)</span>
                            </label>
                            <label class="flex items-center gap-2">
                                <input type="radio" name="smtp-encryption" value="tls" class="w-4 h-4">
                                <span class="text-sm">SSL/TLS (端口 465)</span>
                            </label>
                            <label class="flex items-center gap-2">
                                <input type="radio" name="smtp-encryption" value="none" class="w-4 h-4">
                                <span class="text-sm">无加密 (不推荐)</span>
                            </label>
                        </div>
                    </div>
                </div>
                <div class="flex gap-3">
                    <button onclick="saveSMTPConfig()" class="flex-1 bg-blue-600 text-white py-2 rounded-lg">保存配置</button>
                    <button onclick="testSMTP()" class="px-6 bg-green-600 text-white py-2 rounded-lg">发送测试邮件</button>
                </div>
                <div class="bg-slate-50 p-3 rounded-lg">
                    <p class="text-xs text-slate-600 font-medium mb-2">常用 SMTP 服务器配置：</p>
                    <ul class="text-xs text-slate-500 space-y-1">
                        <li>• <strong>Gmail:</strong> smtp.gmail.com:587 (STARTTLS) - 需使用应用专用密码</li>
                        <li>• <strong>Outlook:</strong> smtp.office365.com:587 (STARTTLS)</li>
                        <li>• <strong>QQ邮箱:</strong> smtp.qq.com:587 (STARTTLS) - 需使用授权码</li>
                        <li>• <strong>163邮箱:</strong> smtp.163.com:465 (SSL/TLS) - 需使用授权码</li>
                        <li>• <strong>阿里企业邮:</strong> smtp.qiye.aliyun.com:465 (SSL/TLS)</li>
                    </ul>
                </div>
            </div>
        </div>
        
        <!-- Danger Zone -->
        <div class="bg-white rounded-xl shadow-sm p-6 col-span-2">
            <h2 class="text-lg font-bold text-red-600 mb-4">⚠️ 危险操作</h2>
            <div class="space-y-3">
                <button onclick="showForceDeleteLicense()" class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700">🗑️ 强制删除序列号</button>
                <p class="text-xs text-slate-500">* 强制删除指定序列号及其所有相关记录（邮箱申请记录等），此操作不可恢复</p>
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
