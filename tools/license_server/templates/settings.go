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
`
