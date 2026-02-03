package templates

// InitScripts contains the initialization JavaScript
const InitScripts = `
// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    // Load all groups first
    Promise.all([
        fetch('/api/llm-groups').then(function(r) { return r.json(); }),
        fetch('/api/search-groups').then(function(r) { return r.json(); }),
        fetch('/api/license-groups').then(function(r) { return r.json(); }),
        fetch('/api/product-types').then(function(r) { return r.json(); })
    ]).then(function(results) {
        llmGroups = results[0] || [];
        searchGroups = results[1] || [];
        licenseGroups = results[2] || [];
        productTypes = results[3] || [];
        
        // Initialize product filter dropdown for email records
        initEmailProductFilter();
        
        // Load all data
        loadLLMGroups();
        loadSearchGroups();
        loadLicenseGroups();
        loadProductTypes();
        loadLicenses();
        loadLLMConfigs();
        loadSearchConfigs();
        loadEmailRecords();
        loadAPIKeys();
        loadSSLConfig();
        loadSMTPConfig();
        loadFilterSettings();
        loadBlacklist();
        loadWhitelist();
        loadConditions();
        loadBackupInfo();
    });
});
`
