package templates

// BackupHTML contains the backup and restore panel HTML
const BackupHTML = `
<div id="panel-backup" class="tab-panel hidden">
    <div class="grid grid-cols-1 gap-6">
        <!-- Backup Section -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <h2 class="text-lg font-bold text-slate-800 mb-4">📦 数据备份</h2>
            <div class="grid grid-cols-2 gap-6">
                <!-- Full Backup -->
                <div class="border rounded-lg p-4">
                    <h3 class="font-semibold text-slate-700 mb-2">🗄️ 完全备份</h3>
                    <p class="text-sm text-slate-500 mb-4">备份数据库中的所有数据，包括序列号、配置、邮件记录等。</p>
                    <button onclick="createBackup('full')" class="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700">
                        创建完全备份
                    </button>
                </div>
                
                <!-- Incremental Backup -->
                <div class="border rounded-lg p-4">
                    <h3 class="font-semibold text-slate-700 mb-2">📈 增量备份</h3>
                    <p class="text-sm text-slate-500 mb-4">仅备份自上次备份以来新增或修改的数据。</p>
                    <div id="last-backup-info" class="text-xs text-slate-400 mb-2"></div>
                    <button onclick="createBackup('incremental')" class="w-full bg-green-600 text-white py-2 rounded-lg hover:bg-green-700">
                        创建增量备份
                    </button>
                </div>
            </div>
            
            <!-- Backup Domain Setting -->
            <div class="mt-4 p-4 bg-slate-50 rounded-lg">
                <label class="text-sm text-slate-600 font-medium">备份标识（域名/服务器名）</label>
                <input type="text" id="backup-domain" placeholder="例如: license.example.com" 
                    class="w-full mt-2 px-3 py-2 border rounded-lg" value="">
                <p class="text-xs text-slate-400 mt-1">此标识将包含在备份文件名中，便于区分不同服务器的备份</p>
            </div>
        </div>
        
        <!-- Restore Section -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <h2 class="text-lg font-bold text-slate-800 mb-4">🔄 数据恢复</h2>
            <div class="space-y-4">
                <div class="border-2 border-dashed border-slate-300 rounded-lg p-6 text-center" id="restore-drop-zone">
                    <input type="file" id="restore-file" accept=".json" class="hidden" onchange="handleRestoreFile(this)">
                    <div class="text-slate-500">
                        <svg class="w-12 h-12 mx-auto mb-2 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"></path>
                        </svg>
                        <p class="mb-2">拖拽备份文件到此处，或</p>
                        <button onclick="document.getElementById('restore-file').click()" class="px-4 py-2 bg-slate-200 rounded-lg hover:bg-slate-300">
                            选择文件
                        </button>
                    </div>
                </div>
                
                <!-- Restore Options -->
                <div id="restore-options" class="hidden">
                    <div class="bg-slate-50 rounded-lg p-4">
                        <h4 class="font-medium text-slate-700 mb-2">备份文件信息</h4>
                        <div id="backup-file-info" class="text-sm text-slate-600 space-y-1"></div>
                    </div>
                    
                    <div class="mt-4">
                        <label class="text-sm text-slate-600 font-medium">恢复类型</label>
                        <div class="flex gap-4 mt-2">
                            <label class="flex items-center gap-2">
                                <input type="radio" name="restore-type" value="full" class="w-4 h-4">
                                <span class="text-sm">完全恢复（删除现有数据）</span>
                            </label>
                            <label class="flex items-center gap-2">
                                <input type="radio" name="restore-type" value="incremental" class="w-4 h-4">
                                <span class="text-sm">增量恢复（合并数据）</span>
                            </label>
                        </div>
                        <p id="restore-type-warning" class="text-xs text-red-500 mt-2 hidden"></p>
                    </div>
                    
                    <div class="flex gap-3 mt-4">
                        <button onclick="cancelRestore()" class="flex-1 py-2 bg-slate-200 rounded-lg hover:bg-slate-300">取消</button>
                        <button onclick="executeRestore()" class="flex-1 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700">执行恢复</button>
                    </div>
                </div>
            </div>
            
            <div class="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                <p class="text-xs text-yellow-700">
                    <strong>⚠️ 注意：</strong>
                    <br>• 完全恢复会删除所有现有数据，请谨慎操作
                    <br>• 恢复类型必须与备份文件类型匹配
                    <br>• 建议在恢复前先创建一个完全备份
                </p>
            </div>
        </div>
        
        <!-- Backup History -->
        <div class="bg-white rounded-xl shadow-sm p-6">
            <h2 class="text-lg font-bold text-slate-800 mb-4">📋 备份记录</h2>
            <div id="backup-history" class="space-y-2">
                <p class="text-sm text-slate-500">加载中...</p>
            </div>
        </div>
    </div>
</div>
`


// BackupScripts contains the backup JavaScript
const BackupScripts = `
var pendingRestoreData = null;

function loadBackupInfo() {
    // Load backup domain setting
    fetch('/api/backup/settings').then(function(resp) { return resp.json(); }).then(function(data) {
        if (data.domain) {
            document.getElementById('backup-domain').value = data.domain;
        }
        if (data.last_backup_time) {
            document.getElementById('last-backup-info').innerHTML = 
                '上次备份: ' + data.last_backup_time + ' (' + data.last_backup_type + ')';
        } else {
            document.getElementById('last-backup-info').innerHTML = '尚无备份记录';
        }
    });
    
    // Load backup history
    loadBackupHistory();
}

function loadBackupHistory() {
    fetch('/api/backup/history').then(function(resp) { return resp.json(); }).then(function(data) {
        var container = document.getElementById('backup-history');
        if (!data.history || data.history.length === 0) {
            container.innerHTML = '<p class="text-sm text-slate-500">暂无备份记录</p>';
            return;
        }
        
        var html = '<table class="w-full text-sm"><thead><tr class="text-left text-slate-500 border-b">' +
            '<th class="pb-2">时间</th><th class="pb-2">类型</th><th class="pb-2">记录数</th><th class="pb-2">文件名</th></tr></thead><tbody>';
        
        data.history.forEach(function(item) {
            var typeLabel = item.type === 'full' ? '<span class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">完全</span>' : 
                '<span class="px-2 py-1 bg-green-100 text-green-700 rounded text-xs">增量</span>';
            html += '<tr class="border-b border-slate-100">' +
                '<td class="py-2">' + item.time + '</td>' +
                '<td class="py-2">' + typeLabel + '</td>' +
                '<td class="py-2">' + item.record_count + '</td>' +
                '<td class="py-2 text-xs text-slate-500 font-mono">' + item.filename + '</td>' +
                '</tr>';
        });
        
        html += '</tbody></table>';
        container.innerHTML = html;
    });
}

function createBackup(type) {
    var domain = document.getElementById('backup-domain').value.trim();
    if (!domain) {
        domain = prompt('请输入备份标识（域名/服务器名）：', 'license-server');
        if (!domain) return;
        document.getElementById('backup-domain').value = domain;
    }
    
    // Save domain setting
    fetch('/api/backup/settings', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({domain: domain})
    });
    
    var confirmMsg = type === 'full' ? 
        '确定要创建完全备份吗？这将备份所有数据。' :
        '确定要创建增量备份吗？这将仅备份自上次备份以来的变更。';
    
    if (!confirm(confirmMsg)) return;
    
    // Show loading
    var btn = event.target;
    var originalText = btn.textContent;
    btn.textContent = '备份中...';
    btn.disabled = true;
    
    fetch('/api/backup/create', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({type: type, domain: domain})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(result) {
        btn.textContent = originalText;
        btn.disabled = false;
        
        if (result.success) {
            alert('备份创建成功！\\n\\n文件名: ' + result.filename + '\\n记录数: ' + result.record_count);
            
            // Download the backup file
            var blob = new Blob([JSON.stringify(result.data, null, 2)], {type: 'application/json'});
            var url = URL.createObjectURL(blob);
            var a = document.createElement('a');
            a.href = url;
            a.download = result.filename;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            
            // Refresh backup info
            loadBackupInfo();
        } else {
            alert('备份失败: ' + result.error);
        }
    })
    .catch(function(err) {
        btn.textContent = originalText;
        btn.disabled = false;
        alert('备份失败: ' + err.message);
    });
}

function handleRestoreFile(input) {
    var file = input.files[0];
    if (!file) return;
    
    var reader = new FileReader();
    reader.onload = function(e) {
        try {
            var data = JSON.parse(e.target.result);
            
            // Validate backup file structure
            if (!data.backup_info || !data.backup_info.type) {
                alert('无效的备份文件格式');
                return;
            }
            
            pendingRestoreData = data;
            
            // Show file info
            var info = data.backup_info;
            var infoHtml = '<p><strong>备份类型:</strong> ' + (info.type === 'full' ? '完全备份' : '增量备份') + '</p>' +
                '<p><strong>备份时间:</strong> ' + info.created_at + '</p>' +
                '<p><strong>备份域名:</strong> ' + (info.domain || '未知') + '</p>' +
                '<p><strong>版本:</strong> ' + (info.version || '1.0') + '</p>';
            
            if (info.record_counts) {
                infoHtml += '<p><strong>记录数:</strong></p><ul class="ml-4 text-xs">';
                for (var table in info.record_counts) {
                    infoHtml += '<li>' + table + ': ' + info.record_counts[table] + '</li>';
                }
                infoHtml += '</ul>';
            }
            
            document.getElementById('backup-file-info').innerHTML = infoHtml;
            document.getElementById('restore-options').classList.remove('hidden');
            
            // Pre-select matching restore type
            var restoreType = info.type;
            document.querySelector('input[name="restore-type"][value="' + restoreType + '"]').checked = true;
            
        } catch (err) {
            alert('解析备份文件失败: ' + err.message);
        }
    };
    reader.readAsText(file);
}

function cancelRestore() {
    pendingRestoreData = null;
    document.getElementById('restore-options').classList.add('hidden');
    document.getElementById('restore-file').value = '';
}

function executeRestore() {
    if (!pendingRestoreData) {
        alert('请先选择备份文件');
        return;
    }
    
    var selectedType = document.querySelector('input[name="restore-type"]:checked');
    if (!selectedType) {
        alert('请选择恢复类型');
        return;
    }
    
    var restoreType = selectedType.value;
    var backupType = pendingRestoreData.backup_info.type;
    
    // Validate type match
    if (restoreType !== backupType) {
        alert('恢复类型必须与备份文件类型匹配！\\n\\n备份文件类型: ' + (backupType === 'full' ? '完全备份' : '增量备份') + 
            '\\n选择的恢复类型: ' + (restoreType === 'full' ? '完全恢复' : '增量恢复'));
        return;
    }
    
    var confirmMsg = restoreType === 'full' ?
        '⚠️ 警告：完全恢复将删除所有现有数据！\\n\\n确定要继续吗？' :
        '确定要执行增量恢复吗？这将合并备份数据到现有数据中。';
    
    if (!confirm(confirmMsg)) return;
    
    if (restoreType === 'full') {
        if (!confirm('再次确认：这将永久删除所有现有数据！\\n\\n输入 "确认删除" 继续...') || 
            prompt('请输入 "确认删除" 以继续：') !== '确认删除') {
            return;
        }
    }
    
    // Execute restore
    fetch('/api/backup/restore', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
            type: restoreType,
            data: pendingRestoreData
        })
    })
    .then(function(resp) { return resp.json(); })
    .then(function(result) {
        if (result.success) {
            alert('恢复成功！\\n\\n' + result.message);
            cancelRestore();
            // Refresh all data
            loadLicenses();
            loadLLMConfigs();
            loadSearchConfigs();
            loadEmailRecords();
            loadBackupInfo();
        } else {
            alert('恢复失败: ' + result.error);
        }
    })
    .catch(function(err) {
        alert('恢复失败: ' + err.message);
    });
}

// Setup drag and drop
document.addEventListener('DOMContentLoaded', function() {
    var dropZone = document.getElementById('restore-drop-zone');
    if (dropZone) {
        dropZone.addEventListener('dragover', function(e) {
            e.preventDefault();
            dropZone.classList.add('border-blue-500', 'bg-blue-50');
        });
        
        dropZone.addEventListener('dragleave', function(e) {
            e.preventDefault();
            dropZone.classList.remove('border-blue-500', 'bg-blue-50');
        });
        
        dropZone.addEventListener('drop', function(e) {
            e.preventDefault();
            dropZone.classList.remove('border-blue-500', 'bg-blue-50');
            
            var files = e.dataTransfer.files;
            if (files.length > 0) {
                document.getElementById('restore-file').files = files;
                handleRestoreFile(document.getElementById('restore-file'));
            }
        });
    }
});
`
