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

// Tab switching
function showTab(name) {
    document.querySelectorAll('.tab-panel').forEach(function(p) { 
        p.classList.remove('active'); 
    });
    document.querySelectorAll('.tab-btn').forEach(function(b) {
        b.classList.remove('bg-blue-600', 'text-white');
        b.classList.add('bg-slate-200', 'text-slate-700');
    });
    var panel = document.getElementById('panel-' + name);
    if (panel) panel.classList.add('active');
    var btn = document.getElementById('tab-' + name);
    if (btn) {
        btn.classList.remove('bg-slate-200', 'text-slate-700');
        btn.classList.add('bg-blue-600', 'text-white');
    }
}

// Modal functions
function showModal(content) {
    document.getElementById('modal-content').innerHTML = content;
    document.getElementById('modal').classList.remove('hidden');
    document.getElementById('modal').classList.add('flex');
    setTimeout(function() {
        var firstInput = document.querySelector('#modal-content input:not([type=hidden]), #modal-content select');
        if (firstInput) firstInput.focus();
    }, 100);
}

function hideModal() {
    document.getElementById('modal').classList.add('hidden');
    document.getElementById('modal').classList.remove('flex');
}

// Helper functions
function escapeHtml(str) {
    if (!str) return '';
    return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeJs(str) {
    if (!str) return '';
    return String(str).replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/"/g, '\\"');
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
    // Only close if both mousedown and mouseup happened on the modal background
    if (e.target.id === 'modal' && modalMouseDownTarget && modalMouseDownTarget.id === 'modal') {
        hideModal();
    }
    modalMouseDownTarget = null;
});

// ESC key to close modal
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' || e.keyCode === 27) {
        var modal = document.getElementById('modal');
        if (modal && !modal.classList.contains('hidden')) {
            hideModal();
        }
    }
});
`
