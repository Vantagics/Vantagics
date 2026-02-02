package templates

// ProductTypesHTML contains the product types panel HTML
const ProductTypesHTML = `
<div id="panel-product-types" class="tab-panel">
    <div class="bg-white rounded-xl shadow-sm p-6">
        <div class="flex justify-between items-center mb-4">
            <h2 class="text-lg font-bold text-slate-800">产品类型管理</h2>
            <button onclick="showProductTypeForm()" class="px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm">+ 添加产品类型</button>
        </div>
        <p class="text-xs text-slate-500 mb-4">* 产品类型用于区分不同产品的序列号。ID=0 为默认产品 VantageData（不可删除）。集成时使用产品 ID 进行区分。</p>
        <div id="product-types-list" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"></div>
    </div>
</div>
`

// ProductTypesScripts contains the product types JavaScript
const ProductTypesScripts = `
var productTypes = [];

function loadProductTypes() {
    fetch('/api/product-types').then(function(resp) { return resp.json(); }).then(function(data) {
        productTypes = data || [];
        var list = document.getElementById('product-types-list');
        
        // Always show "VantageData" as the first item (ID=0, default product)
        var html = '<div class="flex items-center justify-between p-3 bg-blue-50 rounded-lg border-2 border-blue-200">';
        html += '<div><div class="flex items-center gap-2"><span class="font-bold text-sm text-blue-700">VantageData</span><span class="px-2 py-0.5 bg-blue-600 text-white text-xs rounded font-mono">ID: 0</span></div>';
        html += '<p class="text-xs text-slate-500 mt-1">默认产品（不可删除）</p></div>';
        html += '<div class="flex gap-1">';
        html += '<button onclick="showExtraInfoModal(0, \'VantageData\')" class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs">额外信息</button>';
        html += '</div></div>';
        
        if (productTypes && productTypes.length > 0) { 
            productTypes.forEach(function(p, idx) { 
                html += '<div class="flex items-center justify-between p-3 bg-amber-50 rounded-lg">';
                html += '<div><div class="flex items-center gap-2"><span class="font-bold text-sm">' + escapeHtml(p.name) + '</span><span class="px-2 py-0.5 bg-amber-600 text-white text-xs rounded font-mono">ID: ' + p.id + '</span></div>';
                html += '<p class="text-xs text-slate-400 mt-1">' + escapeHtml(p.description || '无描述') + '</p></div>';
                html += '<div class="flex gap-1">';
                html += '<button onclick="showExtraInfoModal(' + p.id + ', \'' + escapeJs(p.name) + '\')" class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs">额外信息</button>';
                html += '<button data-action="edit-product-type" data-idx="' + idx + '" class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">编辑</button>';
                html += '<button data-action="delete-product-type" data-idx="' + idx + '" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs">删除</button>';
                html += '</div></div>'; 
            });
        }
        list.innerHTML = html;
        
        // Update filter dropdown in licenses panel
        var filterSelect = document.getElementById('product-filter');
        if (filterSelect) {
            var currentValue = filterSelect.value;
            var opts = '<option value="">全部产品</option><option value="0">VantageData (ID: 0)</option>';
            productTypes.forEach(function(p) { opts += '<option value="' + p.id + '">' + escapeHtml(p.name) + ' (ID: ' + p.id + ')</option>'; });
            filterSelect.innerHTML = opts;
            filterSelect.value = currentValue;
        }
    });
}

function getProductTypeName(id) {
    if (!id || id === 0) return 'VantageData';
    var p = productTypes.find(function(pt) { return pt.id === id; });
    return p ? p.name : '';
}

// Event delegation for product types
document.getElementById('product-types-list').addEventListener('click', function(e) {
    var btn = e.target.closest('button[data-action]');
    if (!btn) return;
    var action = btn.getAttribute('data-action');
    var idx = parseInt(btn.getAttribute('data-idx'));
    var product = productTypes[idx];
    if (!product) return;
    
    if (action === 'edit-product-type') {
        showProductTypeForm(product);
    } else if (action === 'delete-product-type') {
        deleteProductType(product.id);
    }
});

function showProductTypeForm(product) {
    var p = product || {id: 0, name: '', description: ''};
    var isEdit = p.id > 0;
    showModal('<div class="p-6"><h3 class="text-lg font-bold mb-4">' + (isEdit ? '编辑' : '添加') + '产品类型</h3><div class="space-y-3">' +
        '<input type="hidden" id="product-type-id" value="' + (p.id || 0) + '">' +
        (isEdit ? '<p class="text-sm text-slate-600">产品 ID: <span class="font-mono font-bold text-amber-600">' + p.id + '</span></p>' : '') +
        '<div><label class="text-sm text-slate-600">产品名称</label>' +
        '<input type="text" id="product-type-name" value="' + escapeHtml(p.name) + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div><label class="text-sm text-slate-600">描述</label>' +
        '<input type="text" id="product-type-desc" value="' + escapeHtml(p.description || '') + '" class="w-full px-3 py-2 border rounded-lg"></div>' +
        '<div class="flex gap-2"><button onclick="hideModal()" class="flex-1 py-2 bg-slate-200 rounded-lg">取消</button>' +
        '<button onclick="saveProductType()" class="flex-1 py-2 bg-blue-600 text-white rounded-lg">保存</button></div>' +
        '</div></div>');
}

function saveProductType() {
    var product = {
        id: parseInt(document.getElementById('product-type-id').value) || 0,
        name: document.getElementById('product-type-name').value,
        description: document.getElementById('product-type-desc').value
    };
    if (!product.name) { alert('产品名称不能为空'); return; }
    
    fetch('/api/product-types', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(product)})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            hideModal(); 
            if (result.success) {
                loadProductTypes(); 
                loadLicenses(licenseCurrentPage, licenseSearchTerm); 
            } else {
                alert('保存失败: ' + (result.error || '未知错误'));
            }
        });
}

function deleteProductType(id) {
    if (!confirm('确定要删除此产品类型吗？')) return;
    fetch('/api/product-types', {method: 'DELETE', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: id})})
        .then(function(resp) { return resp.json(); })
        .then(function(result) { 
            if (result.success) {
                loadProductTypes(); 
                loadLicenses(licenseCurrentPage, licenseSearchTerm); 
            } else {
                alert('删除失败: ' + (result.error || '未知错误'));
            }
        });
}

// Extra Info Management
var currentExtraInfoProductId = 0;
var currentExtraInfoProductName = '';

function showExtraInfoModal(productId, productName) {
    currentExtraInfoProductId = productId;
    currentExtraInfoProductName = productName;
    
    var html = '<div class="p-6" style="min-width: 500px;">';
    html += '<h3 class="text-lg font-bold mb-2">额外授权信息 - ' + escapeHtml(productName) + '</h3>';
    html += '<p class="text-xs text-slate-500 mb-4">这些信息会在激活时发送给客户端，用于扩展授权功能。</p>';
    html += '<div id="extra-info-list" class="space-y-2 mb-4 max-h-64 overflow-y-auto"></div>';
    html += '<div class="border-t pt-4">';
    html += '<p class="text-sm font-medium mb-2">添加新项</p>';
    html += '<div class="flex gap-2">';
    html += '<input type="text" id="new-extra-key" placeholder="Key" class="flex-1 px-3 py-2 border rounded-lg text-sm">';
    html += '<input type="text" id="new-extra-value" placeholder="Value" class="flex-1 px-3 py-2 border rounded-lg text-sm">';
    html += '<select id="new-extra-type" class="px-3 py-2 border rounded-lg text-sm">';
    html += '<option value="string">字符串</option>';
    html += '<option value="number">数字</option>';
    html += '</select>';
    html += '<button onclick="addExtraInfo()" class="px-4 py-2 bg-green-600 text-white rounded-lg text-sm">添加</button>';
    html += '</div></div>';
    html += '<div class="flex justify-end mt-4"><button onclick="hideModal()" class="px-4 py-2 bg-slate-200 rounded-lg">关闭</button></div>';
    html += '</div>';
    
    showModal(html);
    loadExtraInfo(productId);
}

function loadExtraInfo(productId) {
    fetch('/api/product-extra-info?product_id=' + productId)
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            var list = document.getElementById('extra-info-list');
            if (!data || data.length === 0) {
                list.innerHTML = '<p class="text-slate-400 text-sm text-center py-4">暂无额外信息</p>';
                return;
            }
            var html = '';
            data.forEach(function(item) {
                html += '<div class="flex items-center gap-2 p-2 bg-slate-50 rounded">';
                html += '<code class="text-blue-600 font-mono text-sm flex-shrink-0">' + escapeHtml(item.key) + '</code>';
                html += '<span class="text-slate-400">=</span>';
                html += '<span class="text-sm flex-1 truncate ' + (item.value_type === 'number' ? 'text-orange-600' : 'text-green-600') + '">' + escapeHtml(item.value) + '</span>';
                html += '<span class="text-xs text-slate-400">(' + (item.value_type === 'number' ? '数字' : '字符串') + ')</span>';
                html += '<button onclick="deleteExtraInfo(' + item.id + ')" class="px-2 py-1 bg-red-100 text-red-700 rounded text-xs hover:bg-red-200">删除</button>';
                html += '</div>';
            });
            list.innerHTML = html;
        });
}

function addExtraInfo() {
    var key = document.getElementById('new-extra-key').value.trim();
    var value = document.getElementById('new-extra-value').value.trim();
    var valueType = document.getElementById('new-extra-type').value;
    
    if (!key) { alert('Key 不能为空'); return; }
    if (valueType === 'number' && isNaN(parseFloat(value))) {
        alert('数字类型的值必须是有效数字');
        return;
    }
    
    fetch('/api/product-extra-info', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
            product_id: currentExtraInfoProductId,
            key: key,
            value: value,
            value_type: valueType
        })
    })
    .then(function(resp) { return resp.json(); })
    .then(function(result) {
        if (result.success) {
            document.getElementById('new-extra-key').value = '';
            document.getElementById('new-extra-value').value = '';
            loadExtraInfo(currentExtraInfoProductId);
        } else {
            alert('添加失败: ' + (result.error || '未知错误'));
        }
    });
}

function deleteExtraInfo(id) {
    if (!confirm('确定要删除此项吗？')) return;
    fetch('/api/product-extra-info', {
        method: 'DELETE',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({id: id})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(result) {
        if (result.success) {
            loadExtraInfo(currentExtraInfoProductId);
        } else {
            alert('删除失败: ' + (result.error || '未知错误'));
        }
    });
}
`
