package templates

// CommonScripts contains shared JavaScript functions
const CommonScripts = `
// Global variables
var llmGroups = [];
var searchGroups = [];
var licenseGroups = [];
var licenseCurrentPage = 1;
var licenseSearchTerm = '';
var emailCurrentPage = 1;
var emailSearchTerm = '';

// Section titles mapping
var sectionTitles = {
    'licenses': '序列号管理',
    'email-records': '邮箱申请记录',
    'email-filter': '邮箱过滤',
    'product-types': '产品类型',
    'license-groups': '序列号分组',
    'api-keys': 'API Key',
    'llm': 'LLM配置',
    'search': '搜索引擎配置',
    'email-notify': '邮件发送',
    'backup': '备份恢复',
    'settings': '系统设置'
};

// Sidebar navigation switching
function showSection(name) {
    // Hide all sections
    document.querySelectorAll('.section').forEach(function(s) {
        s.classList.remove('active');
    });
    // Show target section
    var section = document.getElementById('section-' + name);
    if (section) section.classList.add('active');
    // Remove active from all nav links
    document.querySelectorAll('.sidebar-nav a').forEach(function(a) {
        a.classList.remove('active');
    });
    // Highlight current nav link
    var nav = document.getElementById('nav-' + name);
    if (nav) nav.classList.add('active');
    // Update topbar title
    var title = sectionTitles[name] || name;
    var pageTitle = document.getElementById('page-title');
    if (pageTitle) pageTitle.textContent = title;
}

// Modal functions
function showModal(content) {
    var modal = document.getElementById('modal');
    document.getElementById('modal-content').innerHTML = content;
    modal.classList.add('show');
    setTimeout(function() {
        var firstInput = document.querySelector('#modal-content input:not([type=hidden]), #modal-content select');
        if (firstInput) firstInput.focus();
    }, 100);
}

function hideModal() {
    document.getElementById('modal').classList.remove('show');
}

// Helper functions
function escapeHtml(str) {
    if (!str) return '';
    return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeJs(str) {
    if (!str) return '';
    return String(str).replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/"/g, '\\"').replace(/\n/g, '\\n').replace(/\r/g, '\\r');
}

function getLLMGroupName(id) {
    if (!id) return '';
    var g = llmGroups.find(function(g) { return g.id === id; });
    return g ? g.name : id;
}

function getSearchGroupName(id) {
    if (!id) return '';
    var g = searchGroups.find(function(g) { return g.id === id; });
    return g ? g.name : id;
}

function getLicenseGroupName(id) {
    if (!id) return '';
    var g = licenseGroups.find(function(g) { return g.id === id; });
    return g ? g.name : id;
}

// Modal click outside to close - use mousedown to prevent closing when selecting text
var modalMouseDownTarget = null;
document.getElementById('modal').addEventListener('mousedown', function(e) { 
    modalMouseDownTarget = e.target;
});
document.getElementById('modal').addEventListener('mouseup', function(e) { 
    // Only close if both mousedown and mouseup happened on the modal overlay background
    if (e.target.id === 'modal' && modalMouseDownTarget && modalMouseDownTarget.id === 'modal') {
        hideModal();
    }
    modalMouseDownTarget = null;
});

// ESC key to close modal
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' || e.keyCode === 27) {
        var modal = document.getElementById('modal');
        if (modal && modal.classList.contains('show')) {
            hideModal();
        }
    }
});
`
