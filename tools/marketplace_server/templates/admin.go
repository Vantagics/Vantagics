package templates

// AdminHTML contains the marketplace admin panel HTML template.
const AdminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title data-i18n="admin_panel_title">市场管理后台</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f0f2f5; color: #1f2937; display: flex; min-height: 100vh; }

        /* Sidebar */
        .sidebar { width: 220px; background: #1e293b; color: #cbd5e1; display: flex; flex-direction: column; position: fixed; top: 0; left: 0; bottom: 0; z-index: 50; }
        .sidebar-brand { padding: 20px 20px 16px; border-bottom: 1px solid rgba(255,255,255,0.08); }
        .sidebar-brand h1 { font-size: 16px; color: #f1f5f9; font-weight: 700; letter-spacing: 0.5px; }
        .sidebar-brand span { font-size: 12px; color: #64748b; margin-top: 2px; display: block; }
        .sidebar-nav { flex: 1; padding: 12px 0; }
        .sidebar-nav a { display: flex; align-items: center; gap: 10px; padding: 10px 20px; color: #94a3b8; text-decoration: none; font-size: 14px; transition: all 0.15s; border-left: 3px solid transparent; }
        .sidebar-nav a:hover { color: #e2e8f0; background: rgba(255,255,255,0.04); }
        .sidebar-nav a.active { color: #fff; background: rgba(255,255,255,0.08); border-left-color: #3b82f6; }
        .sidebar-nav .nav-icon { width: 18px; text-align: center; font-size: 15px; }
        .sidebar-nav .nav-divider { height: 1px; background: rgba(255,255,255,0.06); margin: 8px 20px; }
        .sidebar-footer { padding: 16px 20px; border-top: 1px solid rgba(255,255,255,0.08); }
        .sidebar-footer a { display: flex; align-items: center; gap: 8px; color: #ef4444; text-decoration: none; font-size: 13px; padding: 8px 12px; border-radius: 6px; transition: background 0.15s; }
        .sidebar-footer a:hover { background: rgba(239,68,68,0.1); }

        /* Main content */
        .main-wrap { margin-left: 220px; flex: 1; min-height: 100vh; display: flex; flex-direction: column; }
        .topbar { background: #fff; border-bottom: 1px solid #e5e7eb; padding: 0 32px; height: 56px; display: flex; align-items: center; justify-content: space-between; position: sticky; top: 0; z-index: 40; }
        .topbar-title { font-size: 15px; font-weight: 600; color: #374151; }
        .topbar-user { font-size: 13px; color: #6b7280; display: flex; align-items: center; gap: 8px; }
        .topbar-user .avatar { width: 30px; height: 30px; border-radius: 50%; background: #3b82f6; color: #fff; display: flex; align-items: center; justify-content: center; font-size: 13px; font-weight: 600; }
        .content { flex: 1; padding: 24px 32px; }

        /* Cards & Typography */
        h2 { font-size: 16px; margin-bottom: 16px; color: #111827; font-weight: 600; }
        .card { background: #fff; border-radius: 10px; padding: 24px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04); border: 1px solid #e5e7eb; }
        .card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
        .card-header h2 { margin-bottom: 0; }

        /* Table */
        table { width: 100%; border-collapse: collapse; }
        th, td { text-align: left; padding: 12px 16px; font-size: 13px; }
        th { background: #f9fafb; font-weight: 600; color: #6b7280; text-transform: uppercase; font-size: 11px; letter-spacing: 0.5px; border-bottom: 2px solid #e5e7eb; }
        td { border-bottom: 1px solid #f3f4f6; color: #374151; }
        tr:hover td { background: #f9fafb; }

        /* Buttons */
        .btn { display: inline-flex; align-items: center; gap: 6px; padding: 7px 16px; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: 500; text-decoration: none; transition: all 0.15s; }
        .btn-primary { background: #3b82f6; color: #fff; }
        .btn-primary:hover { background: #2563eb; }
        .btn-danger { background: #ef4444; color: #fff; }
        .btn-danger:hover { background: #dc2626; }
        .btn-secondary { background: #f3f4f6; color: #374151; border: 1px solid #d1d5db; }
        .btn-secondary:hover { background: #e5e7eb; }
        .btn-sm { padding: 5px 12px; font-size: 12px; }

        /* Forms */
        input[type="text"], input[type="number"], input[type="password"], textarea { width: 100%; padding: 9px 12px; border: 1px solid #d1d5db; border-radius: 6px; font-size: 14px; color: #1f2937; background: #fff; transition: border-color 0.15s, box-shadow 0.15s; }
        input[type="text"]:focus, input[type="number"]:focus, input[type="password"]:focus, textarea:focus { outline: none; border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,0.1); }
        textarea { resize: vertical; min-height: 60px; }
        .form-group { margin-bottom: 16px; }
        .form-group label { display: block; font-size: 13px; color: #374151; margin-bottom: 6px; font-weight: 500; }
        .form-hint { font-size: 12px; color: #9ca3af; margin-top: 4px; }

        /* Withdrawal Tabs */
        .wd-tabs { display: flex; gap: 0; margin-bottom: 20px; border-bottom: 2px solid #e5e7eb; }
        .wd-tab { padding: 10px 24px; font-size: 14px; font-weight: 500; color: #6b7280; background: none; border: none; border-bottom: 2px solid transparent; margin-bottom: -2px; cursor: pointer; transition: all 0.15s; }
        .wd-tab:hover { color: #3b82f6; }
        .wd-tab.active { color: #3b82f6; border-bottom-color: #3b82f6; font-weight: 600; }

        /* Messages */
        .msg { padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 13px; display: flex; align-items: center; gap: 8px; }
        .msg-success { background: #ecfdf5; color: #065f46; border: 1px solid #a7f3d0; }
        .msg-error { background: #fef2f2; color: #991b1b; border: 1px solid #fecaca; }

        /* Badges */
        .actions { display: flex; gap: 6px; }
        .badge { display: inline-block; padding: 3px 10px; border-radius: 12px; font-size: 11px; font-weight: 600; }
        .badge-preset { background: #ede9fe; color: #5b21b6; }

        /* Modal */
        .modal-overlay { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); backdrop-filter: blur(2px); z-index: 100; justify-content: center; align-items: center; }
        .modal-overlay.show { display: flex; }
        .modal { background: #fff; border-radius: 12px; padding: 28px; width: 440px; max-width: 90%; box-shadow: 0 20px 60px rgba(0,0,0,0.15); }
        .modal h3 { margin-bottom: 20px; font-size: 16px; font-weight: 600; color: #111827; }
        .modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 20px; }

        /* Profile section layout */
        .profile-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
        @media (max-width: 768px) { .profile-grid { grid-template-columns: 1fr; } }
        .profile-card { background: #fff; border-radius: 10px; padding: 24px; box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04); border: 1px solid #e5e7eb; }
        .profile-card h3 { font-size: 15px; font-weight: 600; color: #111827; margin-bottom: 16px; padding-bottom: 12px; border-bottom: 1px solid #f3f4f6; }
        .profile-card .icon-header { display: flex; align-items: center; gap: 8px; }
        .profile-card .icon-header span { font-size: 18px; }
    </style>
</head>
<body>
<!-- Sidebar -->
<aside class="sidebar">
    <div class="sidebar-brand">
        <h1>📦 <span data-i18n="market_admin">市场管理</span></h1>
        <span>Marketplace Admin</span>
    </div>
    <nav class="sidebar-nav">
        <a href="#categories" data-perm="categories" onclick="showSection('categories')" style="display:none;"><span class="nav-icon">📂</span><span data-i18n="category_mgmt">分类管理</span></a>
        <a href="#marketplace" data-perm="marketplace" onclick="showSection('marketplace')" style="display:none;"><span class="nav-icon">🏪</span><span data-i18n="marketplace_mgmt">市场管理</span></a>
        <a href="#accounts" data-perm="accounts" onclick="showSection('accounts')" style="display:none;"><span class="nav-icon">👤</span><span data-i18n="account_mgmt">账号管理</span></a>
        <a href="#review" data-perm="review" onclick="showSection('review')" style="display:none;"><span class="nav-icon">📋</span><span data-i18n="review_mgmt">审核管理</span></a>
        <a href="#settings" data-perm="settings" onclick="showSection('settings')" style="display:none;"><span class="nav-icon">⚙️</span><span data-i18n="system_settings">系统设置</span></a>
        <a href="#notifications" data-perm="notifications" onclick="showSection('notifications')" style="display:none;"><span class="nav-icon">📢</span><span data-i18n="notification_mgmt">消息管理</span></a>
        <a href="#withdrawals" data-perm="settings" onclick="showSection('withdrawals')" style="display:none;"><span class="nav-icon">💰</span><span data-i18n="withdraw_mgmt">提现管理</span></a>
        <a href="#featured" data-perm="settings" onclick="showSection('featured')" style="display:none;"><span class="nav-icon">⭐</span><span data-i18n="featured_stores_mgmt">明星店铺</span></a>
        <a href="#sales" data-perm="sales" onclick="showSection('sales')" style="display:none;"><span class="nav-icon">📊</span><span data-i18n="sales_mgmt">销售管理</span></a>
        <a href="#billing" data-perm="billing" onclick="showSection('billing')" style="display:none;"><span class="nav-icon">💳</span><span data-i18n="billing_mgmt">收费管理</span></a>
        <a href="#admins" data-perm="admin_manage" onclick="showSection('admins')" style="display:none;"><span class="nav-icon">🔑</span><span data-i18n="admin_mgmt">管理员管理</span></a>
        <div class="nav-divider"></div>
        <a href="#profile" onclick="showSection('profile')"><span class="nav-icon">👤</span><span data-i18n="edit_profile">修改资料</span></a>
    </nav>
    <div class="sidebar-footer">
        <a href="/admin/logout">⏻ <span data-i18n="logout">退出登录</span></a>
    </div>
</aside>

<!-- Main -->
<div class="main-wrap">
    <header class="topbar">
        <div class="topbar-title" id="topbar-title" data-i18n="admin_panel_title">管理面板</div>
        <div class="topbar-user">
            <div class="avatar">A</div>
            <span data-i18n="admin">管理员</span>
        </div>
    </header>
    <main class="content">
    <div id="msg-area"></div>

    <!-- Categories Section -->
    <div id="section-categories">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="category_mgmt">分类管理</h2>
                <button class="btn btn-primary" onclick="showCreateCategory()">+ <span data-i18n="new_category">新建分类</span></button>
            </div>
            <table>
                <thead>
                    <tr><th data-i18n="id_col">ID</th><th data-i18n="name_col">名称</th><th data-i18n="desc_col">描述</th><th data-i18n="pack_count_col">分析包数</th><th data-i18n="type_col_cat">类型</th><th data-i18n="actions">操作</th></tr>
                </thead>
                <tbody id="category-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Settings Section -->
    <div id="section-settings" style="display:none;">
        <div class="card">
            <h2 data-i18n="default_language">默认语言</h2>
            <p class="form-hint" style="margin-bottom:16px;" data-i18n="default_language_desc">设置系统默认显示语言，用户未手动选择语言时将使用此设置</p>
            <form id="default-lang-form" onsubmit="saveDefaultLanguage(event)">
                <div class="form-group">
                    <label for="default-lang-select" data-i18n="default_language">默认语言</label>
                    <select id="default-lang-select" style="padding:9px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:14px;">
                        <option value="zh-CN" data-i18n="chinese">中文</option>
                        <option value="en-US" data-i18n="english">English</option>
                    </select>
                </div>
                <button type="submit" class="btn btn-primary" data-i18n="save_settings">保存设置</button>
            </form>
        </div>
        <div class="card">
            <h2 data-i18n="initial_credits">初始 Credits 余额</h2>
            <p class="form-hint" style="margin-bottom:16px;" data-i18n="initial_credits_desc">新用户注册时自动获得的 Credits 数量</p>
            <form id="credits-form" onsubmit="saveInitialCredits(event)">
                <div class="form-group">
                    <label for="initial-credits" data-i18n="initial_balance">初始余额</label>
                    <input type="number" id="initial-credits" min="0" step="1" value="{{.InitialCredits}}" />
                </div>
                <button type="submit" class="btn btn-primary" data-i18n="save_settings">保存设置</button>
            </form>
        </div>
        <div class="card">
            <h2 data-i18n="download_urls">下载地址设置</h2>
            <p class="form-hint" style="margin-bottom:16px;" data-i18n="download_urls_desc">设置客户端软件的下载地址，将显示在分析包分享页面上</p>
            <form id="download-urls-form" onsubmit="saveDownloadURLs(event)">
                <div class="form-group">
                    <label for="download-url-windows" data-i18n="download_url_windows">Windows 下载地址</label>
                    <input type="url" id="download-url-windows" placeholder="https://example.com/download/windows" value="{{.DownloadURLWindows}}" />
                </div>
                <div class="form-group">
                    <label for="download-url-macos" data-i18n="download_url_macos">macOS 下载地址</label>
                    <input type="url" id="download-url-macos" placeholder="https://example.com/download/macos" value="{{.DownloadURLMacOS}}" />
                </div>
                <button type="submit" class="btn btn-primary" data-i18n="save_settings">保存设置</button>
            </form>
        </div>
        <div class="card">
            <h2 data-i18n="smtp_settings">邮件服务器设置 (SMTP)</h2>
            <p class="form-hint" style="margin-bottom:16px;" data-i18n="smtp_settings_desc">配置 SMTP 邮件服务器，用于小铺店主向客户发送邮件通知</p>
            <form id="smtp-form" onsubmit="saveSMTPConfig(event)">
                <div class="form-group">
                    <label style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                        <input type="checkbox" id="smtp-enabled" style="width:auto;" />
                        <span data-i18n="smtp_enabled">启用邮件服务</span>
                    </label>
                </div>
                <div class="form-group">
                    <label for="smtp-host" data-i18n="smtp_host">SMTP 服务器地址</label>
                    <input type="text" id="smtp-host" placeholder="smtp.example.com" />
                </div>
                <div class="form-group">
                    <label for="smtp-port" data-i18n="smtp_port">端口</label>
                    <input type="number" id="smtp-port" min="1" max="65535" value="587" placeholder="587" />
                </div>
                <div class="form-group">
                    <label style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                        <input type="checkbox" id="smtp-use-tls" style="width:auto;" />
                        <span data-i18n="smtp_use_tls">使用 TLS 加密</span>
                    </label>
                </div>
                <div class="form-group">
                    <label for="smtp-username" data-i18n="smtp_username">用户名</label>
                    <input type="text" id="smtp-username" placeholder="user@example.com" />
                </div>
                <div class="form-group">
                    <label for="smtp-password" data-i18n="smtp_password">密码</label>
                    <input type="password" id="smtp-password" placeholder="••••••••" />
                </div>
                <div class="form-group">
                    <label for="smtp-from-email" data-i18n="smtp_from_email">发件人邮箱</label>
                    <input type="text" id="smtp-from-email" placeholder="noreply@example.com" />
                </div>
                <div class="form-group">
                    <label for="smtp-from-name" data-i18n="smtp_from_name">发件人名称</label>
                    <input type="text" id="smtp-from-name" placeholder="默认名称（店主发信时自动使用店铺名称）" />
                </div>
                <div style="display:flex;gap:8px;flex-wrap:wrap;">
                    <button type="submit" class="btn btn-primary" data-i18n="save_settings">保存设置</button>
                    <button type="button" class="btn btn-secondary" onclick="showSMTPTestModal()" data-i18n="smtp_test_btn">发送测试邮件</button>
                </div>
            </form>
        </div>
        <div class="card">
            <h2>PayPal 支付配置</h2>
            <p class="form-hint" style="margin-bottom:16px;">配置 PayPal 支付参数，用于自定义商品的在线支付功能</p>
            <form id="paypal-config-form" onsubmit="savePayPalConfig(event)">
                <div class="form-group">
                    <label for="paypal-client-id">Client ID</label>
                    <input type="text" id="paypal-client-id" placeholder="PayPal Client ID" />
                </div>
                <div class="form-group">
                    <label for="paypal-client-secret">Client Secret</label>
                    <input type="password" id="paypal-client-secret" placeholder="••••••••" />
                </div>
                <div class="form-group">
                    <label for="paypal-mode">运行模式</label>
                    <select id="paypal-mode" style="padding:9px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:14px;">
                        <option value="sandbox">Sandbox</option>
                        <option value="live">Live</option>
                    </select>
                </div>
                <button type="submit" class="btn btn-primary">保存设置</button>
            </form>
        </div>
    </div>

    <!-- SMTP Test Modal -->
    <div id="smtp-test-modal" class="modal-overlay">
        <div class="modal">
            <h3 data-i18n="smtp_test_title">发送测试邮件</h3>
            <p class="form-hint" style="margin-bottom:16px;" data-i18n="smtp_test_desc">输入收件邮箱地址，系统将发送一封测试邮件以验证 SMTP 配置是否正确</p>
            <div class="form-group">
                <label for="smtp-test-email" data-i18n="smtp_test_email">测试收件邮箱</label>
                <input type="email" id="smtp-test-email" placeholder="test@example.com" />
            </div>
            <div id="smtp-test-result" style="display:none;margin-bottom:12px;"></div>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="hideSMTPTestModal()" data-i18n="cancel">取消</button>
                <button class="btn btn-primary" id="smtp-test-send-btn" onclick="sendSMTPTest()" data-i18n="smtp_send_test">发送测试</button>
            </div>
        </div>
    </div>

    <!-- Review Section (all admins) -->
    <div id="section-review" style="display:none;">
        <!-- Tab Navigation -->
        <div class="wd-tabs">
            <button class="wd-tab active" onclick="switchReviewTab('review-tab-packs', this)">📋 <span data-i18n="review_packs">待审核分析包</span></button>
            <button class="wd-tab" onclick="switchReviewTab('review-tab-custom-products', this)">🛍️ <span data-i18n="review_custom_products">待审核商品</span></button>
        </div>

        <!-- Tab: 待审核分析包 -->
        <div id="review-tab-packs" class="wd-tab-content">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="review_packs">待审核分析包</h2>
                <button class="btn btn-secondary" onclick="loadPendingPacks()">↻ <span data-i18n="refresh">刷新</span></button>
            </div>
            <table>
                <thead>
                    <tr><th data-i18n="id_col">ID</th><th data-i18n="name_col">名称</th><th data-i18n="category">分类</th><th data-i18n="author_col">作者</th><th data-i18n="mode_col">模式</th><th data-i18n="price_col">价格</th><th data-i18n="upload_time_col">上传时间</th><th data-i18n="actions">操作</th></tr>
                </thead>
                <tbody id="pending-list"></tbody>
            </table>
        </div>
        </div>

        <!-- Tab: 待审核商品 -->
        <div id="review-tab-custom-products" class="wd-tab-content" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="review_custom_products">待审核商品</h2>
                <button class="btn btn-secondary" onclick="loadPendingCustomProducts()">↻ <span data-i18n="refresh">刷新</span></button>
            </div>
            <table>
                <thead>
                    <tr>
                        <th data-i18n="id_col">ID</th>
                        <th data-i18n="product_name_col">商品名称</th>
                        <th data-i18n="store_name_col">店铺</th>
                        <th data-i18n="type_col">类型</th>
                        <th data-i18n="price_col">价格</th>
                        <th data-i18n="desc_col">描述</th>
                        <th data-i18n="upload_time_col">提交时间</th>
                        <th data-i18n="actions">操作</th>
                    </tr>
                </thead>
                <tbody id="pending-custom-products-list"></tbody>
            </table>
        </div>
        </div>
    </div>

    <!-- Marketplace Management Section -->
    <div id="section-marketplace" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="marketplace_packs">市场管理 - 在售分析包</h2>
                <button class="btn btn-secondary" onclick="loadMarketplacePacks()">↻ <span data-i18n="refresh">刷新</span></button>
            </div>
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <select id="mp-status-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="published" data-i18n="on_sale">在售</option>
                    <option value="delisted" data-i18n="delisted">已下架</option>
                </select>
                <select id="mp-category-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="" data-i18n="all_categories">全部分类</option>
                </select>
                <select id="mp-mode-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="" data-i18n="all_payment_modes">全部付费方式</option>
                    <option value="free" data-i18n="free">免费</option>
                    <option value="per_use" data-i18n="per_use">按次付费</option>
                    <option value="subscription" data-i18n="subscription">订阅</option>
                </select>
                <select id="mp-sort" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="downloads" data-i18n="sort_by_downloads">按下载量排序</option>
                    <option value="price" data-i18n="sort_by_price">按价格排序</option>
                    <option value="name" data-i18n="sort_by_name">按名称排序</option>
                </select>
                <select id="mp-order" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="desc" data-i18n="desc">降序</option>
                    <option value="asc" data-i18n="asc">升序</option>
                </select>
            </div>
            <table>
                <thead>
                    <tr><th data-i18n="id_col">ID</th><th data-i18n="name_col">名称</th><th data-i18n="category">分类</th><th data-i18n="author_col">作者</th><th data-i18n="payment_mode_col">付费方式</th><th data-i18n="price_col">价格</th><th data-i18n="download_count_col">下载量</th><th data-i18n="list_time_col">上架时间</th><th data-i18n="actions">操作</th></tr>
                </thead>
                <tbody id="marketplace-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Unified Account Management Section -->
    <div id="section-accounts" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="account_mgmt">账号管理</h2>
                <button class="btn btn-secondary" onclick="loadAccounts()">↻ <span data-i18n="refresh">刷新</span></button>
            </div>
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <input type="text" id="account-search" placeholder="搜索邮箱/名称/SN..." data-i18n-placeholder="search_email_name_sn" style="width:260px;" onkeydown="if(event.key==='Enter')loadAccounts()" />
                <button class="btn btn-primary btn-sm" onclick="loadAccounts()" data-i18n="search">搜索</button>
                <select id="account-role-filter" onchange="loadAccounts()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="" data-i18n="all_roles">全部</option>
                    <option value="author" data-i18n="authors_only">仅作者</option>
                    <option value="customer" data-i18n="customers_only">仅客户</option>
                </select>
                <select id="account-sort" onchange="loadAccounts()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="created_at" data-i18n="sort_by_register_time">按注册时间</option>
                    <option value="credits" data-i18n="sort_by_balance">按余额</option>
                    <option value="downloads" data-i18n="sort_by_downloads">按下载量</option>
                    <option value="spent" data-i18n="sort_by_spent">按消费额</option>
                    <option value="packs" data-i18n="sort_by_packs">按发布包数</option>
                    <option value="revenue" data-i18n="sort_by_revenue">按作者收入</option>
                    <option value="name" data-i18n="sort_by_name">按名称</option>
                </select>
                <select id="account-order" onchange="loadAccounts()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="desc" data-i18n="desc">降序</option>
                    <option value="asc" data-i18n="asc">升序</option>
                </select>
            </div>
            <table>
                <thead>
                    <tr>
                        <th data-i18n="email_col">邮箱</th>
                        <th data-i18n="name_col">名称</th>
                        <th data-i18n="role_col">角色</th>
                        <th data-i18n="balance_col">余额</th>
                        <th data-i18n="published_packs_col">发布包数</th>
                        <th data-i18n="author_revenue_col">作者收入</th>
                        <th data-i18n="download_count_col">下载数</th>
                        <th data-i18n="spent_col">消费额</th>
                        <th data-i18n="status">状态</th>
                        <th data-i18n="actions">操作</th>
                    </tr>
                </thead>
                <tbody id="account-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Account Detail Modal -->
    <div id="account-detail-modal" class="modal-overlay">
        <div class="modal" style="width:720px;">
            <h3 id="account-detail-title" data-i18n="account_detail">账号详情</h3>
            <div id="account-detail-info" style="margin-bottom:12px;font-size:13px;color:#6b7280;"></div>
            <div id="account-detail-sub-accounts" style="margin-bottom:16px;"></div>
            <div id="account-detail-tabs" style="display:flex;gap:8px;margin-bottom:12px;">
                <button class="btn btn-primary btn-sm" id="acct-tab-packs-btn" onclick="switchAccountDetailTab('packs')" data-i18n="author_packs">发布的分析包</button>
                <button class="btn btn-secondary btn-sm" id="acct-tab-tx-btn" onclick="switchAccountDetailTab('tx')" data-i18n="transaction_records">交易记录</button>
            </div>
            <div id="acct-tab-packs">
                <table>
                    <thead>
                        <tr><th data-i18n="pack_name">包名</th><th data-i18n="category">分类</th><th data-i18n="payment_mode_col">付费方式</th><th data-i18n="unit_price">单价</th><th data-i18n="download_count_col">下载量</th><th data-i18n="total_revenue_col">总收入</th><th data-i18n="status">状态</th></tr>
                    </thead>
                    <tbody id="account-detail-packs"></tbody>
                </table>
                <div style="display:flex;align-items:center;justify-content:space-between;margin-top:12px;flex-wrap:wrap;gap:8px;">
                    <span id="account-detail-page-info" style="font-size:12px;color:#6b7280;"></span>
                    <div style="display:flex;align-items:center;gap:4px;">
                        <button id="account-detail-prev-btn" class="btn btn-secondary btn-sm" onclick="accountDetailGoPage(accountDetailCurrentPage-1)" disabled>‹ <span data-i18n="prev_page">上一页</span></button>
                        <span id="account-detail-page-nums" style="display:inline-flex;gap:4px;"></span>
                        <button id="account-detail-next-btn" class="btn btn-secondary btn-sm" onclick="accountDetailGoPage(accountDetailCurrentPage+1)" disabled><span data-i18n="next_page">下一页</span> ›</button>
                    </div>
                </div>
            </div>
            <div id="acct-tab-tx" style="display:none;">
                <div style="max-height:400px;overflow-y:auto;">
                    <table>
                        <thead>
                            <tr><th data-i18n="id_col">ID</th><th data-i18n="type_col">类型</th><th data-i18n="amount">金额</th><th data-i18n="description">描述</th><th data-i18n="time">时间</th></tr>
                        </thead>
                        <tbody id="account-detail-tx-list"></tbody>
                    </table>
                </div>
                <div id="account-detail-tx-pagination" style="display:flex;align-items:center;justify-content:center;gap:8px;margin-top:12px;font-size:13px;"></div>
            </div>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="hideAccountDetailModal()" data-i18n="close">关闭</button>
            </div>
        </div>
    </div>

    <!-- Topup Modal (reused) -->
    <div id="topup-modal" class="modal-overlay">
        <div class="modal">
            <h3 data-i18n="topup_credits">充值 Credits</h3>
            <input type="hidden" id="topup-email" value="" />
            <div id="topup-user-info" style="margin-bottom:16px;font-size:13px;color:#6b7280;"></div>
            <div class="form-group">
                <label for="topup-amount" data-i18n="topup_amount">充值数量</label>
                <input type="number" id="topup-amount" min="1" step="1" placeholder="输入充值 Credits 数量" data-i18n-placeholder="enter_topup_amount" />
            </div>
            <div class="form-group">
                <label for="topup-reason" data-i18n="topup_reason">备注（可选）</label>
                <input type="text" id="topup-reason" placeholder="充值原因" data-i18n-placeholder="topup_reason_placeholder" />
            </div>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="hideTopupModal()" data-i18n="cancel">取消</button>
                <button class="btn btn-primary" onclick="submitTopup()" data-i18n="confirm_topup">确认充值</button>
            </div>
        </div>
    </div>

    <!-- Admin Management Section (id=1 only) -->
    <div id="section-admins" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="admin_mgmt">管理员管理</h2>
                <button class="btn btn-primary" onclick="showAddAdminModal()">+ <span data-i18n="add_admin">添加管理员</span></button>
            </div>
            <table>
                <thead>
                    <tr><th data-i18n="id_col">ID</th><th data-i18n="username">用户名</th><th data-i18n="permissions_col">权限</th><th data-i18n="created_at_col">创建时间</th></tr>
                </thead>
                <tbody id="admin-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Notifications Management Section -->
    <div id="section-notifications" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="notification_mgmt">消息管理</h2>
                <button class="btn btn-primary" onclick="showCreateNotification()">+ <span data-i18n="send_notification">发送消息</span></button>
            </div>
            <table>
                <thead>
                    <tr><th data-i18n="id_col">ID</th><th data-i18n="title_col">标题</th><th data-i18n="type_col">类型</th><th data-i18n="status">状态</th><th data-i18n="effective_date">生效日期</th><th data-i18n="duration_col">时长</th><th data-i18n="created_at_col">创建时间</th><th data-i18n="actions">操作</th></tr>
                </thead>
                <tbody id="notifications-tbody"></tbody>
            </table>
        </div>
    </div>

    <!-- Withdrawals Management Section -->
    <div id="section-withdrawals" style="display:none;">
        <!-- Tab Navigation -->
        <div class="wd-tabs">
            <button class="wd-tab active" onclick="switchWdTab('wd-tab-settings', this)">⚙️ <span data-i18n="withdraw_settings">提现设置</span></button>
            <button class="wd-tab" onclick="switchWdTab('wd-tab-records', this)">📋 <span data-i18n="withdraw_records_tab">提现记录</span></button>
        </div>

        <!-- Tab: 提现设置 -->
        <div id="wd-tab-settings" class="wd-tab-content">
            <div class="card">
                <h2 data-i18n="credit_cash_rate">Credit 提现价格</h2>
                <p class="form-hint" style="margin-bottom:16px;" data-i18n="credit_cash_rate_desc">每个 Credit 兑换的现金金额（单位：元），设为 0 表示提现功能未启用</p>
                <form id="cash-rate-form" onsubmit="saveCreditCashRate(event)">
                    <div class="form-group">
                        <label for="credit-cash-rate" data-i18n="cash_rate_label">提现价格（元/Credit）</label>
                        <input type="number" id="credit-cash-rate" min="0" step="0.01" value="{{.CreditCashRate}}" />
                    </div>
                    <button type="submit" class="btn btn-primary" data-i18n="save_settings">保存设置</button>
                </form>
            </div>
            <div class="card">
                <h2 data-i18n="revenue_split_settings">收入分成比例设置</h2>
                <p class="form-hint" style="margin-bottom:16px;" data-i18n="revenue_split_desc">设置发布者（作者）获得的收入比例，平台获得剩余部分。默认 70 表示发布者获得 70%，平台获得 30%</p>
                <form id="revenue-split-form" onsubmit="saveRevenueSplit(event)">
                    <div class="form-group">
                        <label for="revenue-split-publisher-pct" data-i18n="publisher_split_pct">发布者分成比例（%）</label>
                        <div style="display:flex;align-items:center;gap:12px;">
                            <input type="number" id="revenue-split-publisher-pct" min="0" max="100" step="1" value="{{.RevenueSplitPublisherPct}}" style="flex:1;" oninput="updateSplitPreview()" />
                            <span id="split-preview" style="font-size:13px;color:#6366f1;font-weight:600;white-space:nowrap;">发布者 {{.RevenueSplitPublisherPct}}% : 平台 {{.RevenueSplitPlatformPct}}%</span>
                        </div>
                    </div>
                    <button type="submit" class="btn btn-primary" data-i18n="save_split_settings">保存分成设置</button>
                </form>
            </div>
            <div class="card">
                <h2 data-i18n="fee_rate_settings">提现手续费率设置</h2>
                <p class="form-hint" style="margin-bottom:16px;" data-i18n="fee_rate_desc">为每种收款方式设置提现手续费率（百分比），例如输入 3 表示 3%</p>
                <form id="withdrawal-fees-form" onsubmit="saveWithdrawalFees(event)">
                    <div class="form-group">
                        <label for="fee-rate-paypal" data-i18n="fee_rate_paypal">PayPal 手续费率（%）</label>
                        <input type="number" id="fee-rate-paypal" min="0" step="0.01" value="{{.FeeRatePaypal}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-wechat" data-i18n="fee_rate_wechat">微信 手续费率（%）</label>
                        <input type="number" id="fee-rate-wechat" min="0" step="0.01" value="{{.FeeRateWechat}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-alipay" data-i18n="fee_rate_alipay">AliPay 手续费率（%）</label>
                        <input type="number" id="fee-rate-alipay" min="0" step="0.01" value="{{.FeeRateAlipay}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-check" data-i18n="fee_rate_check">支票 手续费率（%）</label>
                        <input type="number" id="fee-rate-check" min="0" step="0.01" value="{{.FeeRateCheck}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-wire-transfer" data-i18n="fee_rate_wire">国际电汇 手续费率（%）</label>
                        <input type="number" id="fee-rate-wire-transfer" min="0" step="0.01" value="{{.FeeRateWireTransfer}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-bank-card-us" data-i18n="fee_rate_us">美国银行卡 手续费率（%）</label>
                        <input type="number" id="fee-rate-bank-card-us" min="0" step="0.01" value="{{.FeeRateBankCardUS}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-bank-card-eu" data-i18n="fee_rate_eu">欧洲银行卡 手续费率（%）</label>
                        <input type="number" id="fee-rate-bank-card-eu" min="0" step="0.01" value="{{.FeeRateBankCardEU}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-bank-card-cn" data-i18n="fee_rate_cn">中国银行卡 手续费率（%）</label>
                        <input type="number" id="fee-rate-bank-card-cn" min="0" step="0.01" value="{{.FeeRateBankCardCN}}" />
                    </div>
                    <button type="submit" class="btn btn-primary" data-i18n="save_fee_settings">保存手续费设置</button>
                </form>
            </div>
        </div>

        <!-- Tab: 提现记录 -->
        <div id="wd-tab-records" class="wd-tab-content" style="display:none;">
            <div class="card">
                <div class="card-header">
                    <h2 data-i18n="withdraw_mgmt">提现管理</h2>
                    <div style="display:flex;gap:8px;">
                        <button class="btn btn-secondary" onclick="exportWithdrawals()">📥 <span data-i18n="export_excel">导出 Excel</span></button>
                        <button class="btn btn-primary" onclick="exportAndApproveWithdrawals()">📥 <span data-i18n="export_and_approve">导出并标记已付款</span></button>
                        <button class="btn btn-primary" id="btn-batch-approve" onclick="batchApproveWithdrawals()" style="display:none;" data-i18n="batch_approve">批量标记已付款</button>
                        <button class="btn btn-secondary" onclick="loadWithdrawals()">↻ <span data-i18n="refresh">刷新</span></button>
                    </div>
                </div>
                <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                    <select id="wd-status-filter" onchange="loadWithdrawals()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                        <option value="" data-i18n="all">全部</option>
                        <option value="pending" data-i18n="applied_withdraw">已申请提现</option>
                        <option value="paid" data-i18n="paid">已付款</option>
                    </select>
                    <input type="text" id="wd-author-filter" placeholder="按作者名过滤" data-i18n-placeholder="filter_by_author" oninput="loadWithdrawals()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;width:160px;" />
                </div>
                <table>
                    <thead>
                        <tr>
                            <th><input type="checkbox" id="wd-select-all" onchange="toggleSelectAllWithdrawals()" /></th>
                            <th data-i18n="author_col">作者</th>
                            <th data-i18n="payment_method_col">收款方式</th>
                            <th data-i18n="payment_detail_col">收款详情</th>
                            <th data-i18n="withdraw_amount_col">提现金额</th>
                            <th data-i18n="fee_rate_col">手续费率</th>
                            <th data-i18n="fee_amount_col">手续费</th>
                            <th data-i18n="net_amount_col">实付金额</th>
                            <th data-i18n="status">状态</th>
                            <th data-i18n="time">时间</th>
                        </tr>
                    </thead>
                    <tbody id="withdrawals-list"></tbody>
                </table>
            </div>
        </div>
    </div>

    <!-- Sales Management Section -->
    <div id="section-sales" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="sales_mgmt">销售管理</h2>
                <div style="display:flex;gap:8px;">
                    <button class="btn btn-secondary" onclick="loadSalesData(1)">↻ <span data-i18n="refresh">刷新</span></button>
                    <button class="btn btn-primary" onclick="exportSalesExcel()">📥 <span data-i18n="export_sales_excel">导出 Excel</span></button>
                </div>
            </div>
            <!-- Filters -->
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <select id="sales-category-filter" onchange="loadSalesData(1)" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="" data-i18n="all_categories_filter">全部分类</option>
                </select>
                <select id="sales-author-filter" onchange="loadSalesData(1)" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="" data-i18n="all_authors_filter">全部作者</option>
                </select>
                <input type="date" id="sales-date-from" onchange="loadSalesData(1)" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;" />
                <input type="date" id="sales-date-to" onchange="loadSalesData(1)" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;" />
                <button class="btn btn-secondary btn-sm" onclick="clearSalesFilters()" data-i18n="clear_filters">清除筛选</button>
            </div>
            <!-- Summary Stats -->
            <div id="sales-summary" style="display:flex;gap:16px;margin-bottom:20px;flex-wrap:wrap;">
                <div style="background:#f0f9ff;border:1px solid #bae6fd;border-radius:8px;padding:16px 24px;flex:1;min-width:180px;">
                    <div style="font-size:12px;color:#0369a1;font-weight:600;" data-i18n="total_orders">订单总数</div>
                    <div id="sales-total-orders" style="font-size:24px;font-weight:700;color:#0c4a6e;margin-top:4px;">0</div>
                </div>
                <div style="background:#f0fdf4;border:1px solid #bbf7d0;border-radius:8px;padding:16px 24px;flex:1;min-width:180px;">
                    <div style="font-size:12px;color:#15803d;font-weight:600;" data-i18n="total_sales_credits">总销售额 (Credits)</div>
                    <div id="sales-total-credits" style="font-size:24px;font-weight:700;color:#14532d;margin-top:4px;">0</div>
                </div>
                <div style="background:#fefce8;border:1px solid #fde68a;border-radius:8px;padding:16px 24px;flex:1;min-width:180px;">
                    <div style="font-size:12px;color:#a16207;font-weight:600;" data-i18n="total_users">涉及用户数</div>
                    <div id="sales-total-users" style="font-size:24px;font-weight:700;color:#713f12;margin-top:4px;">0</div>
                </div>
                <div style="background:#fdf4ff;border:1px solid #f0abfc;border-radius:8px;padding:16px 24px;flex:1;min-width:180px;">
                    <div style="font-size:12px;color:#86198f;font-weight:600;" data-i18n="total_authors">涉及作者数</div>
                    <div id="sales-total-authors" style="font-size:24px;font-weight:700;color:#4a044e;margin-top:4px;">0</div>
                </div>
            </div>
            <!-- Orders Table -->
            <table>
                <thead>
                    <tr><th data-i18n="order_id">订单ID</th><th data-i18n="buyer_col">买家</th><th data-i18n="buyer_email_col">买家邮箱</th><th data-i18n="pack_col">分析包</th><th data-i18n="category_col">分类</th><th data-i18n="author_col">作者</th><th data-i18n="amount_credits_col">金额(Credits)</th><th data-i18n="type_col">类型</th><th data-i18n="buyer_ip_col">买家IP</th><th data-i18n="time_col_admin">时间</th></tr>
                </thead>
                <tbody id="sales-order-list"></tbody>
            </table>
            <!-- Pagination -->
            <div id="sales-pagination" style="display:flex;justify-content:space-between;align-items:center;margin-top:16px;padding-top:16px;border-top:1px solid #e5e7eb;">
                <div id="sales-page-info" style="font-size:13px;color:#6b7280;"></div>
                <div style="display:flex;gap:6px;align-items:center;">
                    <button class="btn btn-secondary btn-sm" id="sales-prev-btn" onclick="salesGoPage(salesCurrentPage-1)" disabled>‹ <span data-i18n="prev_page">上一页</span></button>
                    <span id="sales-page-nums" style="display:flex;gap:4px;"></span>
                    <button class="btn btn-secondary btn-sm" id="sales-next-btn" onclick="salesGoPage(salesCurrentPage+1)" disabled><span data-i18n="next_page">下一页</span> ›</button>
                </div>
            </div>
        </div>
    </div>

    <!-- Billing Management Section -->
    <div id="section-billing" style="display:none;">
        <!-- Tab Navigation -->
        <div class="wd-tabs">
            <button class="wd-tab active" onclick="switchBillingTab('billing-tab-email', this)">📧 邮件发送</button>
            <button class="wd-tab" onclick="switchBillingTab('billing-tab-storefront', this)">🎨 店铺装修</button>
        </div>

        <!-- Tab: 邮件发送 -->
        <div id="billing-tab-email" class="wd-tab-content">
            <div class="card">
                <div class="card-header">
                    <h2>📧 邮件发送收费明细</h2>
                    <div style="display:flex;gap:8px;">
                        <button class="btn btn-secondary" onclick="loadBillingData(1)">↻ 刷新</button>
                        <button class="btn btn-primary" onclick="exportBillingExcel()">📥 导出 Excel</button>
                    </div>
                </div>
                <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                    <input type="text" id="billing-store-filter" placeholder="按店铺名称搜索..." style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;width:200px;" onkeyup="if(event.key==='Enter')loadBillingData(1)" />
                    <button class="btn btn-secondary btn-sm" onclick="document.getElementById('billing-store-filter').value='';loadBillingData(1)">清除</button>
                </div>
                <div id="billing-summary" style="display:flex;gap:24px;margin-bottom:16px;font-size:14px;color:#374151;"></div>
                <table>
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>店铺名称</th>
                            <th>收件人数</th>
                            <th>消耗 Credits</th>
                            <th>描述</th>
                            <th>时间</th>
                        </tr>
                    </thead>
                    <tbody id="billing-list"></tbody>
                </table>
                <div id="billing-pagination" style="display:flex;justify-content:space-between;align-items:center;margin-top:16px;padding-top:16px;border-top:1px solid #e5e7eb;">
                    <div id="billing-page-info" style="font-size:13px;color:#6b7280;"></div>
                    <div style="display:flex;gap:6px;align-items:center;">
                        <button class="btn btn-secondary btn-sm" id="billing-prev-btn" onclick="billingGoPage(billingCurrentPage-1)" disabled>‹ 上一页</button>
                        <span id="billing-page-nums" style="display:flex;gap:4px;"></span>
                        <button class="btn btn-secondary btn-sm" id="billing-next-btn" onclick="billingGoPage(billingCurrentPage+1)" disabled>下一页 ›</button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Tab: 店铺装修 -->
        <div id="billing-tab-storefront" class="wd-tab-content" style="display:none;">
            <div class="card">
                <div style="text-align:center;padding:48px 24px;color:#9ca3af;">
                    <div style="font-size:48px;margin-bottom:16px;">🎨</div>
                    <h2 style="color:#6b7280;margin-bottom:8px;">店铺装修收费</h2>
                    <p style="font-size:14px;">此功能正在开发中，敬请期待...</p>
                </div>
            </div>
        </div>
    </div>

    <!-- Featured Storefronts Management Section -->
    <div id="section-featured" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2 data-i18n="featured_stores_mgmt">明星店铺管理</h2>
                <span id="featured-count" style="font-size:13px;color:#6b7280;"></span>
            </div>
            <p style="font-size:13px;color:#9ca3af;margin-bottom:16px;" data-i18n="featured_stores_hint">管理首页展示的明星店铺，最多可设置 16 个。拖拽排序可调整展示顺序。</p>
            <!-- Search to add -->
            <div style="position:relative;margin-bottom:20px;">
                <div style="display:flex;gap:8px;">
                    <input type="text" id="featured-search-input" placeholder="搜索用户名或店铺名称..." data-i18n-placeholder="search_store_placeholder" oninput="searchFeaturedStores()" style="flex:1;" />
                </div>
                <div id="featured-search-results" style="display:none;position:absolute;top:100%;left:0;right:0;background:#fff;border:1px solid #d1d5db;border-radius:6px;box-shadow:0 4px 12px rgba(0,0,0,0.1);max-height:240px;overflow-y:auto;z-index:10;margin-top:4px;"></div>
            </div>
            <!-- Featured list table -->
            <table>
                <thead>
                    <tr>
                        <th style="width:60px;" data-i18n="sort_order_col">排序</th>
                        <th data-i18n="store_name_col">店铺名称</th>
                        <th style="width:160px;" data-i18n="actions">操作</th>
                    </tr>
                </thead>
                <tbody id="featured-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Profile Section (all admins) -->
    <div id="section-profile" style="display:none;">
        <div class="profile-grid">
            <div class="profile-card">
                <h3><span class="icon-header"><span>👤</span> <span data-i18n="edit_profile_title">修改资料</span></span></h3>
                <div class="form-group">
                    <label for="profile-username" data-i18n="username">用户名</label>
                    <input type="text" id="profile-username" placeholder="新用户名（留空不修改）" data-i18n-placeholder="new_username_optional" />
                    <div class="form-hint" data-i18n="change_requires_relogin">修改后需要重新登录</div>
                </div>
            </div>
            <div class="profile-card">
                <h3><span class="icon-header"><span>🔒</span> <span data-i18n="change_password_admin">修改密码</span></span></h3>
                <div class="form-group">
                    <label for="profile-old-password" data-i18n="current_password">当前密码</label>
                    <input type="password" id="profile-old-password" placeholder="输入当前密码" data-i18n-placeholder="enter_current_pw_admin" />
                </div>
                <div class="form-group">
                    <label for="profile-new-password" data-i18n="new_password">新密码</label>
                    <input type="password" id="profile-new-password" placeholder="输入新密码" data-i18n-placeholder="enter_new_pw" />
                    <div class="form-hint" data-i18n="min_6_chars_admin">至少 6 个字符</div>
                </div>
            </div>
        </div>
        <div style="margin-top: 20px; display: flex; justify-content: flex-end;">
            <button class="btn btn-primary" onclick="saveProfile()" style="padding: 10px 28px; font-size: 14px;" data-i18n="save_changes">保存修改</button>
        </div>
    </div>

    </main>
</div>

<!-- Reject Reason Modal -->
<div id="reject-modal" class="modal-overlay">
    <div class="modal">
        <h3 data-i18n="reject_review">拒绝审核</h3>
        <input type="hidden" id="reject-pack-id" value="" />
        <div class="form-group">
            <label for="reject-reason" data-i18n="reject_reason_required">拒绝原因（必填）</label>
            <textarea id="reject-reason" placeholder="请输入拒绝原因" data-i18n-placeholder="enter_reject_reason_ph"></textarea>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideRejectModal()" data-i18n="cancel">取消</button>
            <button class="btn btn-danger" onclick="submitReject()" data-i18n="confirm_reject">确认拒绝</button>
        </div>
    </div>
</div>

<!-- Reject Custom Product Modal -->
<div id="reject-custom-product-modal" class="modal-overlay">
    <div class="modal">
        <h3 data-i18n="reject_custom_product">拒绝商品</h3>
        <input type="hidden" id="reject-custom-product-id" value="" />
        <div class="form-group">
            <label for="reject-custom-product-reason" data-i18n="reject_reason_required">拒绝原因（必填）</label>
            <textarea id="reject-custom-product-reason" placeholder="请输入拒绝原因" data-i18n-placeholder="enter_reject_reason_ph"></textarea>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideRejectCustomProductModal()" data-i18n="cancel">取消</button>
            <button class="btn btn-danger" onclick="submitRejectCustomProduct()" data-i18n="confirm_reject">确认拒绝</button>
        </div>
    </div>
</div>

<!-- Add Admin Modal -->
<div id="add-admin-modal" class="modal-overlay">
    <div class="modal">
        <h3 data-i18n="add_admin_title">添加管理员</h3>
        <div class="form-group">
            <label for="new-admin-username" data-i18n="username_min3_label">用户名（至少3个字符）</label>
            <input type="text" id="new-admin-username" placeholder="输入用户名" data-i18n-placeholder="enter_username_ph" />
        </div>
        <div class="form-group">
            <label for="new-admin-password" data-i18n="password_min6_label">密码（至少6个字符）</label>
            <input type="text" id="new-admin-password" placeholder="输入密码" data-i18n-placeholder="enter_password_ph" />
        </div>
        <div class="form-group">
            <label data-i18n="permission_settings">权限设置</label>
            <div style="display:flex;flex-wrap:wrap;gap:12px;margin-top:6px;">
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="categories" class="new-admin-perm" /> <span data-i18n="category_mgmt">分类管理</span>
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="marketplace" class="new-admin-perm" /> <span data-i18n="marketplace_mgmt">市场管理</span>
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="authors" class="new-admin-perm" /> <span data-i18n="author_mgmt">作者管理</span>
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="customers" class="new-admin-perm" /> <span data-i18n="customer_mgmt">客户管理</span>
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="review" class="new-admin-perm" /> <span data-i18n="review_mgmt">审核管理</span>
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="settings" class="new-admin-perm" /> <span data-i18n="system_settings">系统设置</span>
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="notifications" class="new-admin-perm" /> <span data-i18n="notification_mgmt">消息管理</span>
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="sales" class="new-admin-perm" /> <span data-i18n="sales_mgmt">销售管理</span>
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="billing" class="new-admin-perm" /> 收费管理
                </label>
            </div>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideAddAdminModal()" data-i18n="cancel">取消</button>
            <button class="btn btn-primary" onclick="submitAddAdmin()" data-i18n="add">添加</button>
        </div>
    </div>
</div>

<!-- Create/Edit Category Modal -->
<div id="category-modal" class="modal-overlay">
    <div class="modal">
        <h3 id="modal-title" data-i18n="new_category_title">新建分类</h3>
        <input type="hidden" id="edit-category-id" value="" />
        <div class="form-group">
            <label for="cat-name" data-i18n="category_name">分类名称</label>
            <input type="text" id="cat-name" placeholder="输入分类名称" data-i18n-placeholder="enter_category_name_ph" />
        </div>
        <div class="form-group">
            <label for="cat-desc" data-i18n="category_desc_optional">描述（可选）</label>
            <textarea id="cat-desc" placeholder="输入分类描述" data-i18n-placeholder="enter_category_desc_ph"></textarea>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideModal()" data-i18n="cancel">取消</button>
            <button class="btn btn-primary" onclick="saveCategory()" data-i18n="save">保存</button>
        </div>
    </div>
</div>

<!-- Create Notification Modal -->
<div id="create-notif-modal" class="modal-overlay">
    <div class="modal" style="width:520px;">
        <h3 data-i18n="send_notification">发送消息</h3>
        <div class="form-group">
            <label for="notif-title" data-i18n="message_title">消息标题</label>
            <input type="text" id="notif-title" placeholder="消息标题" data-i18n-placeholder="message_title" />
        </div>
        <div class="form-group">
            <label for="notif-content" data-i18n="message_content">消息内容</label>
            <textarea id="notif-content" placeholder="消息内容" data-i18n-placeholder="message_content" rows="4"></textarea>
        </div>
        <div class="form-group">
            <label for="notif-target-type" data-i18n="message_type">消息类型</label>
            <select id="notif-target-type" onchange="toggleTargetUsers()" style="width:100%;padding:9px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:14px;">
                <option value="broadcast" data-i18n="broadcast_all">广播（所有用户）</option>
                <option value="targeted" data-i18n="targeted_specific">定向（指定用户）</option>
            </select>
        </div>
        <div id="notif-target-section" style="display:none;">
            <div class="form-group">
                <label data-i18n="search_users">搜索用户</label>
                <div style="display:flex;gap:8px;">
                    <input type="text" id="notif-user-search" placeholder="输入邮箱/名称搜索..." data-i18n-placeholder="search_email_name" style="flex:1;" onkeydown="if(event.key==='Enter')searchNotifUsers()" />
                    <button class="btn btn-primary btn-sm" onclick="searchNotifUsers()" data-i18n="search">搜索</button>
                </div>
                <div id="notif-user-results" style="max-height:120px;overflow-y:auto;margin-top:8px;"></div>
            </div>
            <div class="form-group">
                <label data-i18n="no_users_selected">已选用户</label>
                <div id="notif-selected-users" style="min-height:32px;padding:8px;background:#f9fafb;border:1px solid #e5e7eb;border-radius:6px;font-size:13px;color:#6b7280;" data-i18n="no_users_selected">未选择用户</div>
            </div>
        </div>
        <div class="form-group">
            <label for="notif-effective-date" data-i18n="effective_date">生效日期（可选，留空立即生效）</label>
            <input type="datetime-local" id="notif-effective-date" style="width:100%;padding:9px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:14px;" />
        </div>
        <div class="form-group">
            <label for="notif-duration" data-i18n="display_duration">显示时长</label>
            <div style="display:flex;align-items:center;gap:8px;">
                <input type="number" id="notif-duration" value="0" min="0" style="width:120px;" />
                <span style="font-size:13px;color:#6b7280;" data-i18n="days_0_permanent">天 (0=永久)</span>
            </div>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideCreateNotification()" data-i18n="cancel">取消</button>
            <button class="btn btn-primary" onclick="createNotification()" data-i18n="send_notification">发送</button>
        </div>
    </div>
</div>

<script>
// Permission system
var adminID = {{.AdminID}};
var permissions = {{.PermissionsJSON}};
// Lazy _i18n wrapper: safe to call before I18nJS loads
if (!window._i18n) { window._i18n = function(key, fallback) { return fallback || key; }; }
var permLabels = { categories: window._i18n("category_mgmt","分类管理"), marketplace: window._i18n("marketplace_mgmt","市场管理"), accounts: window._i18n("account_mgmt","账号管理"), authors: window._i18n("author_mgmt","作者管理"), customers: window._i18n("customer_mgmt","客户管理"), review: window._i18n("review_mgmt","审核管理"), settings: window._i18n("system_settings","系统设置"), notifications: window._i18n("notification_mgmt","消息管理"), sales: window._i18n("sales_mgmt","销售管理"), billing: window._i18n("billing_mgmt","收费管理") };

function hasPerm(p) {
    if (p === 'accounts') return permissions.indexOf('accounts') !== -1 || permissions.indexOf('authors') !== -1 || permissions.indexOf('customers') !== -1;
    return permissions.indexOf(p) !== -1;
}
function isSuperAdmin() { return adminID === 1; }

// Initialize sidebar visibility based on permissions
(function initSidebar() {
    var links = document.querySelectorAll('.sidebar-nav a[data-perm]');
    for (var i = 0; i < links.length; i++) {
        var perm = links[i].getAttribute('data-perm');
        if (perm === 'admin_manage') {
            if (isSuperAdmin()) links[i].style.display = '';
        } else {
            if (hasPerm(perm)) links[i].style.display = '';
        }
    }
})();

function showSection(name) {
    // Map old section names to new unified accounts section
    if (name === 'authors' || name === 'customers') name = 'accounts';
    var sections = ['categories', 'marketplace', 'accounts', 'settings', 'admins', 'review', 'profile', 'notifications', 'withdrawals', 'featured', 'sales', 'billing'];
    var titles = { categories: window._i18n("category_mgmt","分类管理"), marketplace: window._i18n("marketplace_mgmt","市场管理"), accounts: window._i18n("account_mgmt","账号管理"), settings: window._i18n("system_settings","系统设置"), admins: window._i18n("admin_mgmt","管理员管理"), review: window._i18n("review_mgmt","审核管理"), profile: window._i18n("edit_profile","修改资料"), notifications: window._i18n("notification_mgmt","消息管理"), withdrawals: window._i18n("withdraw_mgmt","提现管理"), featured: window._i18n("featured_stores_mgmt","明星店铺"), sales: window._i18n("sales_mgmt","销售管理"), billing: window._i18n("billing_mgmt","收费管理") };
    for (var i = 0; i < sections.length; i++) {
        var el = document.getElementById('section-' + sections[i]);
        if (el) el.style.display = sections[i] === name ? '' : 'none';
    }
    var links = document.querySelectorAll('.sidebar-nav a');
    for (var i = 0; i < links.length; i++) {
        var href = links[i].getAttribute('href');
        if (href) links[i].className = href === '#' + name ? 'active' : '';
    }
    document.getElementById('topbar-title').textContent = titles[name] || window._i18n("admin_panel_title","管理面板");
    if (name === 'categories') loadCategories();
    if (name === 'marketplace') loadMarketplacePacks();
    if (name === 'accounts') loadAccounts();
    if (name === 'admins') loadAdmins();
    if (name === 'review') { loadPendingPacks(); loadPendingCustomProducts(); }
    if (name === 'notifications') loadNotifications();
    if (name === 'withdrawals') loadWithdrawals();
    if (name === 'sales') loadSalesData(1);
    if (name === 'billing') loadBillingData(1);
    if (name === 'featured') loadFeaturedStorefronts();
    if (name === 'settings') { loadSMTPConfig(); loadPayPalConfig(); }
}

function showMsg(text, isError) {
    var area = document.getElementById('msg-area');
    area.innerHTML = '<div class="msg ' + (isError ? 'msg-error' : 'msg-success') + '">' + text + '</div>';
    setTimeout(function() { area.innerHTML = ''; }, 4000);
}

function apiFetch(url, opts) {
    return fetch(url, opts).then(function(r) {
        if (r.status === 401) {
            showMsg(window._i18n("session_expired","会话已过期，正在跳转到登录页..."), true);
            setTimeout(function() { window.location.href = '/admin/login'; }, 1500);
            return Promise.reject(new Error('session_expired'));
        }
        return r;
    });
}

// --- Category Management ---
function loadCategories() {
    apiFetch('/api/categories').then(function(r) { return r.json(); }).then(function(data) {
        var cats = Array.isArray(data) ? data : (data.categories || []);
        var tbody = document.getElementById('category-list');
        if (cats.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#999;">' + window._i18n("no_categories","暂无分类") + '</td></tr>';
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
            html += '<td>' + (c.is_preset ? '<span class="badge badge-preset">' + window._i18n("preset","预设") + '</span>' : window._i18n("custom","自定义")) + '</td>';
            html += '<td class="actions">';
            html += '<button class="btn btn-primary" onclick="showEditCategory(' + c.id + ',\'' + escAttr(c.name) + '\',\'' + escAttr(c.description || '') + '\')">' + window._i18n("edit","编辑") + '</button> ';
            if (!c.is_preset) {
                html += '<button class="btn btn-danger" onclick="deleteCategory(' + c.id + ',\'' + escAttr(c.name) + '\',' + c.pack_count + ')">' + window._i18n("delete","删除") + '</button>';
            }
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg(window._i18n("load_categories_failed","加载分类失败") + ': ' + err, true); });
}

function showCreateCategory() {
    document.getElementById('modal-title').textContent = window._i18n("new_category_title","新建分类");
    document.getElementById('edit-category-id').value = '';
    document.getElementById('cat-name').value = '';
    document.getElementById('cat-desc').value = '';
    document.getElementById('category-modal').className = 'modal-overlay show';
}

function showEditCategory(id, name, desc) {
    document.getElementById('modal-title').textContent = window._i18n("edit_category_title","编辑分类");
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
    if (!name) { alert(window._i18n("enter_category_name","请输入分类名称")); return; }

    var url, method;
    if (id) {
        url = '/api/admin/categories/' + id;
        method = 'PUT';
    } else {
        url = '/api/admin/categories';
        method = 'POST';
    }
    apiFetch(url, {
        method: method,
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({name: name, description: desc})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) {
            hideModal();
            showMsg(id ? window._i18n("category_updated","分类已更新") : window._i18n("category_created","分类已创建"), false);
            loadCategories();
        } else {
            showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true);
        }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function deleteCategory(id, name, packCount) {
    if (packCount > 0) {
        alert(window._i18n("category_has_packs","分类 \"{name}\" 下有 {count} 个分析包，请先迁移后再删除。").replace("{name}", name).replace("{count}", packCount));
        return;
    }
    if (!confirm(window._i18n("confirm_delete_category","确定要删除分类 \"{name}\" 吗？").replace("{name}", name))) return;
    apiFetch('/api/admin/categories/' + id, { method: 'DELETE' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg(window._i18n("category_deleted","分类已删除"), false); loadCategories(); }
            else { showMsg(res.data.error || window._i18n("delete_failed","删除失败"), true); }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Settings ---
function saveDefaultLanguage(e) {
    e.preventDefault();
    var val = document.getElementById('default-lang-select').value;
    apiFetch('/admin/api/settings/default-language', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: 'value=' + encodeURIComponent(val)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg(window._i18n("default_lang_updated","默认语言已更新"), false); }
        else { showMsg(res.data.error || window._i18n("save_failed","保存失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function saveInitialCredits(e) {
    e.preventDefault();
    var val = document.getElementById('initial-credits').value;
    apiFetch('/admin/settings/initial-credits', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: 'value=' + encodeURIComponent(val)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg(window._i18n("initial_balance_updated","初始余额已更新为") + ' ' + val, false); }
        else { showMsg(res.data.error || window._i18n("save_failed","保存失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function saveDownloadURLs(e) {
    e.preventDefault();
    var winURL = document.getElementById('download-url-windows').value.trim();
    var macURL = document.getElementById('download-url-macos').value.trim();
    if (winURL && !/^https?:\/\//.test(winURL)) { showMsg(window._i18n("invalid_url","请输入有效的 URL（以 http:// 或 https:// 开头）"), true); return; }
    if (macURL && !/^https?:\/\//.test(macURL)) { showMsg(window._i18n("invalid_url","请输入有效的 URL（以 http:// 或 https:// 开头）"), true); return; }
    apiFetch('/admin/api/settings/download-urls', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({windows_url: winURL, macos_url: macURL})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg(window._i18n("download_urls_updated","下载地址已更新"), false); }
        else { showMsg(res.data.error || window._i18n("save_failed","保存失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- SMTP Config ---
function loadSMTPConfig() {
    var raw = '{{.SMTPConfigJSON}}';
    if (!raw) return;
    try {
        var cfg = JSON.parse(raw);
        document.getElementById('smtp-enabled').checked = cfg.enabled || false;
        document.getElementById('smtp-host').value = cfg.host || '';
        document.getElementById('smtp-port').value = cfg.port || 587;
        document.getElementById('smtp-use-tls').checked = cfg.use_tls || false;
        document.getElementById('smtp-username').value = cfg.username || '';
        document.getElementById('smtp-password').value = cfg.password || '';
        document.getElementById('smtp-from-email').value = cfg.from_email || '';
        document.getElementById('smtp-from-name').value = cfg.from_name || '';
    } catch(e) {}
}

function saveSMTPConfig(e) {
    e.preventDefault();
    var config = {
        enabled: document.getElementById('smtp-enabled').checked,
        host: document.getElementById('smtp-host').value.trim(),
        port: parseInt(document.getElementById('smtp-port').value) || 587,
        use_tls: document.getElementById('smtp-use-tls').checked,
        username: document.getElementById('smtp-username').value.trim(),
        password: document.getElementById('smtp-password').value,
        from_email: document.getElementById('smtp-from-email').value.trim(),
        from_name: document.getElementById('smtp-from-name').value.trim()
    };
    apiFetch('/admin/api/settings/smtp', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(config)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg(window._i18n("smtp_saved","SMTP 配置已保存"), false); }
        else { showMsg(res.data.error || window._i18n("save_failed","保存失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function showSMTPTestModal() {
    document.getElementById('smtp-test-result').style.display = 'none';
    document.getElementById('smtp-test-email').value = '';
    document.getElementById('smtp-test-modal').className = 'modal-overlay show';
}

function hideSMTPTestModal() {
    document.getElementById('smtp-test-modal').className = 'modal-overlay';
}

function sendSMTPTest() {
    var email = document.getElementById('smtp-test-email').value.trim();
    if (!email) { alert(window._i18n("enter_test_email","请输入测试收件邮箱")); return; }
    var btn = document.getElementById('smtp-test-send-btn');
    btn.disabled = true;
    btn.textContent = window._i18n("sending","发送中...");
    var resultEl = document.getElementById('smtp-test-result');
    resultEl.style.display = 'none';
    apiFetch('/admin/api/settings/smtp-test', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({test_email: email})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        btn.disabled = false;
        btn.textContent = window._i18n("smtp_send_test","发送测试");
        resultEl.style.display = '';
        if (res.ok) {
            resultEl.className = 'msg msg-success';
            resultEl.textContent = window._i18n("smtp_test_success","测试邮件发送成功，请检查收件箱");
        } else {
            resultEl.className = 'msg msg-error';
            resultEl.textContent = res.data.error || window._i18n("smtp_test_failed","测试邮件发送失败");
        }
    }).catch(function(err) {
        btn.disabled = false;
        btn.textContent = window._i18n("smtp_send_test","发送测试");
        resultEl.style.display = '';
        resultEl.className = 'msg msg-error';
        resultEl.textContent = window._i18n("request_failed","请求失败") + ': ' + err;
    });
}

// --- PayPal Config ---
function loadPayPalConfig() {
    apiFetch('/admin/settings/paypal').then(function(r) { return r.json(); })
    .then(function(cfg) {
        document.getElementById('paypal-client-id').value = cfg.client_id || '';
        document.getElementById('paypal-client-secret').value = '';
        document.getElementById('paypal-client-secret').placeholder = cfg.client_secret || '••••••••';
        if (cfg.mode) document.getElementById('paypal-mode').value = cfg.mode;
    }).catch(function(err) {});
}

function savePayPalConfig(e) {
    e.preventDefault();
    var data = new URLSearchParams();
    data.append('client_id', document.getElementById('paypal-client-id').value.trim());
    data.append('client_secret', document.getElementById('paypal-client-secret').value);
    data.append('mode', document.getElementById('paypal-mode').value);
    apiFetch('/admin/settings/paypal', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: data.toString()
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg('PayPal 配置已保存', false); loadPayPalConfig(); }
        else { showMsg(res.data.error || '保存失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function saveCreditCashRate(e) {
    e.preventDefault();
    var val = document.getElementById('credit-cash-rate').value;
    apiFetch('/admin/settings/credit-cash-rate', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: 'value=' + encodeURIComponent(val)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg(window._i18n("credit_rate_updated","Credit 提现价格已更新为") + ' ' + val + ' ' + window._i18n("yuan","元") + '/Credit', false); }
        else { showMsg(res.data.error || window._i18n("save_failed","保存失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function switchWdTab(tabId, btn) {
    var contents = document.querySelectorAll('#section-withdrawals .wd-tab-content');
    for (var i = 0; i < contents.length; i++) { contents[i].style.display = 'none'; }
    document.getElementById(tabId).style.display = '';
    var tabs = document.querySelectorAll('#section-withdrawals .wd-tab');
    for (var i = 0; i < tabs.length; i++) { tabs[i].classList.remove('active'); }
    btn.classList.add('active');
    if (tabId === 'wd-tab-records') { loadWithdrawals(); }
}

function updateSplitPreview() {
    var pub = parseFloat(document.getElementById('revenue-split-publisher-pct').value) || 0;
    var plat = 100 - pub;
    var preview = document.getElementById('split-preview');
    preview.innerHTML = window._i18n("publisher_pct_platform","发布者 {pub}% : 平台 {plat}%").replace("{pub}", pub).replace("{plat}", plat);
}
// Initialize preview on page load
(function(){ var el = document.getElementById('split-preview'); if(el){ updateSplitPreview(); } })();
// Initialize default language select from server value
(function(){ var sel = document.getElementById('default-lang-select'); if(sel){ var sv = '{{.DefaultLang}}'; if(sv === 'en-US' || sv === 'zh-CN') sel.value = sv; else sel.value = 'zh-CN'; } })();

function saveRevenueSplit(e) {
    e.preventDefault();
    var val = document.getElementById('revenue-split-publisher-pct').value;
    var pct = parseFloat(val);
    if (isNaN(pct) || pct < 0 || pct > 100) { showMsg(window._i18n("split_must_0_100","分成比例必须在 0-100 之间"), true); return; }
    apiFetch('/admin/api/settings/revenue-split', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({publisher_pct: pct})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok || res.data.ok) { showMsg(window._i18n("split_saved","收入分成比例已保存：发布者 {pub}% / 平台 {plat}%").replace("{pub}", pct).replace("{plat}", 100 - pct), false); }
        else { showMsg(res.data.error || window._i18n("save_failed","保存失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function saveWithdrawalFees(e) {
    e.preventDefault();
    var body = {
        paypal_fee_rate: parseFloat(document.getElementById('fee-rate-paypal').value) || 0,
        wechat_fee_rate: parseFloat(document.getElementById('fee-rate-wechat').value) || 0,
        alipay_fee_rate: parseFloat(document.getElementById('fee-rate-alipay').value) || 0,
        check_fee_rate: parseFloat(document.getElementById('fee-rate-check').value) || 0,
        wire_transfer_fee_rate: parseFloat(document.getElementById('fee-rate-wire-transfer').value) || 0,
        bank_card_us_fee_rate: parseFloat(document.getElementById('fee-rate-bank-card-us').value) || 0,
        bank_card_eu_fee_rate: parseFloat(document.getElementById('fee-rate-bank-card-eu').value) || 0,
        bank_card_cn_fee_rate: parseFloat(document.getElementById('fee-rate-bank-card-cn').value) || 0
    };
    apiFetch('/admin/api/settings/withdrawal-fees', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(body)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok || res.data.ok) { showMsg(window._i18n("fee_saved","提现手续费率已保存"), false); }
        else { showMsg(res.data.error || window._i18n("save_failed","保存失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Helpers ---
function escHtml(s) { var d = document.createElement('div'); d.textContent = s; return d.innerHTML; }
function escAttr(s) { return s.replace(/\\/g,'\\\\').replace(/'/g,"\\'").replace(/"/g,'\\"'); }

// --- Admin Management ---
function loadAdmins() {
    apiFetch('/api/admin/admins').then(function(r) { return r.json(); }).then(function(data) {
        var admins = data.admins || [];
        var tbody = document.getElementById('admin-list');
        if (admins.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;color:#999;">' + window._i18n("no_admins","暂无管理员") + '</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < admins.length; i++) {
            var a = admins[i];
            var permDisplay;
            if (a.id === 1) {
                permDisplay = '<span class="badge badge-preset">' + window._i18n("super_admin_all_perms","超级管理员（全部权限）") + '</span>';
            } else {
                var perms = a.permissions || [];
                if (perms.length === 0) {
                    permDisplay = '<span style="color:#999;">' + window._i18n("no_permissions","无权限") + '</span>';
                } else {
                    permDisplay = '';
                    for (var j = 0; j < perms.length; j++) {
                        permDisplay += '<span class="badge" style="background:#e0f2fe;color:#0369a1;margin:2px;">' + (permLabels[perms[j]] || perms[j]) + '</span>';
                    }
                }
            }
            html += '<tr>';
            html += '<td>' + a.id + '</td>';
            html += '<td>' + escHtml(a.username) + '</td>';
            html += '<td>' + permDisplay + '</td>';
            html += '<td>' + a.created_at + '</td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg(window._i18n("load_admins_failed","加载管理员列表失败") + ': ' + err, true); });
}

function showAddAdminModal() {
    document.getElementById('new-admin-username').value = '';
    document.getElementById('new-admin-password').value = '';
    var checks = document.querySelectorAll('.new-admin-perm');
    for (var i = 0; i < checks.length; i++) checks[i].checked = false;
    document.getElementById('add-admin-modal').className = 'modal-overlay show';
}

function hideAddAdminModal() {
    document.getElementById('add-admin-modal').className = 'modal-overlay';
}

function submitAddAdmin() {
    var username = document.getElementById('new-admin-username').value.trim();
    var password = document.getElementById('new-admin-password').value;
    if (username.length < 3) { alert(window._i18n("username_min3","用户名至少3个字符")); return; }
    if (password.length < 6) { alert(window._i18n("password_min6_alert","密码至少6个字符")); return; }
    var permCheckboxes = document.querySelectorAll('.new-admin-perm:checked');
    var perms = [];
    for (var i = 0; i < permCheckboxes.length; i++) { perms.push(permCheckboxes[i].value); }
    apiFetch('/api/admin/admins', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({username: username, password: password, permissions: perms})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { hideAddAdminModal(); showMsg(window._i18n("admin_added","管理员已添加"), false); loadAdmins(); }
        else { showMsg(res.data.error || window._i18n("add_failed","添加失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Review Management ---
function switchReviewTab(tabId, btn) {
    var contents = document.querySelectorAll('#section-review .wd-tab-content');
    for (var i = 0; i < contents.length; i++) { contents[i].style.display = 'none'; }
    document.getElementById(tabId).style.display = '';
    var tabs = document.querySelectorAll('#section-review .wd-tab');
    for (var i = 0; i < tabs.length; i++) { tabs[i].classList.remove('active'); }
    btn.classList.add('active');
    if (tabId === 'review-tab-packs') { loadPendingPacks(); }
    if (tabId === 'review-tab-custom-products') { loadPendingCustomProducts(); }
}

function loadPendingPacks() {
    apiFetch('/api/admin/review/pending').then(function(r) { return r.json(); }).then(function(data) {
        var packs = data || [];
        var tbody = document.getElementById('pending-list');
        if (packs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#999;">' + window._i18n("no_pending_packs","暂无待审核分析包") + '</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < packs.length; i++) {
            var p = packs[i];
            html += '<tr>';
            html += '<td>' + p.id + '</td>';
            html += '<td>' + escHtml(p.pack_name) + '</td>';
            html += '<td>' + escHtml(p.category_name) + '</td>';
            html += '<td>' + escHtml(p.author_name || '-') + '</td>';
            html += '<td>' + p.share_mode + '</td>';
            html += '<td>' + (p.share_mode === 'free' ? window._i18n("free","免费") : p.credits_price + ' Credits') + '</td>';
            html += '<td>' + p.created_at + '</td>';
            html += '<td class="actions">';
            html += '<button class="btn btn-primary" onclick="approvePack(' + p.id + ')">' + window._i18n("approved","通过") + '</button> ';
            html += '<button class="btn btn-danger" onclick="showRejectModal(' + p.id + ')">' + window._i18n("rejected","拒绝") + '</button>';
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg(window._i18n("load_pending_failed","加载待审核列表失败") + ': ' + err, true); });
}

function approvePack(id) {
    if (!confirm(window._i18n("confirm_approve","确定通过审核？"))) return;
    apiFetch('/api/admin/review/' + id + '/approve', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg(window._i18n("approved","审核已通过"), false); loadPendingPacks(); }
            else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function showRejectModal(id) {
    document.getElementById('reject-pack-id').value = id;
    document.getElementById('reject-reason').value = '';
    document.getElementById('reject-modal').className = 'modal-overlay show';
}

function hideRejectModal() {
    document.getElementById('reject-modal').className = 'modal-overlay';
}

function submitReject() {
    var id = document.getElementById('reject-pack-id').value;
    var reason = document.getElementById('reject-reason').value.trim();
    if (!reason) { alert(window._i18n("enter_reject_reason","请输入拒绝原因")); return; }
    apiFetch('/api/admin/review/' + id + '/reject', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({reason: reason})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { hideRejectModal(); showMsg(window._i18n("rejected_done","已拒绝"), false); loadPendingPacks(); }
        else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Pending Custom Products Review ---
function loadPendingCustomProducts() {
    apiFetch('/api/admin/pending-custom-products').then(function(r) { return r.json(); }).then(function(data) {
        var products = data || [];
        var tbody = document.getElementById('pending-custom-products-list');
        if (products.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#999;">' + window._i18n("no_pending_custom_products","暂无待审核商品") + '</td></tr>';
            return;
        }
        var typeLabels = { credits: window._i18n("credits_recharge","积分充值"), virtual_goods: window._i18n("virtual_goods","虚拟商品") };
        var html = '';
        for (var i = 0; i < products.length; i++) {
            var p = products[i];
            html += '<tr>';
            html += '<td>' + p.id + '</td>';
            html += '<td>' + escHtml(p.product_name) + '</td>';
            html += '<td>' + escHtml(p.store_name || '-') + '</td>';
            html += '<td>' + (typeLabels[p.product_type] || p.product_type) + '</td>';
            html += '<td>$' + p.price_usd.toFixed(2) + '</td>';
            html += '<td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="' + escAttr(p.description || '') + '">' + escHtml(p.description || '-') + '</td>';
            html += '<td>' + p.created_at + '</td>';
            html += '<td class="actions">';
            html += '<button class="btn btn-primary btn-sm" onclick="approveCustomProduct(' + p.id + ')">' + window._i18n("approved","通过") + '</button> ';
            html += '<button class="btn btn-danger btn-sm" onclick="showRejectCustomProductModal(' + p.id + ')">' + window._i18n("rejected","拒绝") + '</button>';
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg(window._i18n("load_pending_custom_products_failed","加载待审核商品列表失败") + ': ' + err, true); });
}

function approveCustomProduct(id) {
    if (!confirm(window._i18n("confirm_approve_custom_product","确定通过该商品审核？"))) return;
    apiFetch('/admin/custom-product/' + id + '/approve', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg(window._i18n("custom_product_approved","商品审核已通过"), false); loadPendingCustomProducts(); }
            else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function showRejectCustomProductModal(id) {
    document.getElementById('reject-custom-product-id').value = id;
    document.getElementById('reject-custom-product-reason').value = '';
    document.getElementById('reject-custom-product-modal').className = 'modal-overlay show';
}

function hideRejectCustomProductModal() {
    document.getElementById('reject-custom-product-modal').className = 'modal-overlay';
}

function submitRejectCustomProduct() {
    var id = document.getElementById('reject-custom-product-id').value;
    var reason = document.getElementById('reject-custom-product-reason').value.trim();
    if (!reason) { alert(window._i18n("enter_reject_reason","请输入拒绝原因")); return; }
    var fd = new FormData();
    fd.append('reason', reason);
    apiFetch('/admin/custom-product/' + id + '/reject', { method: 'POST', body: fd })
    .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { hideRejectCustomProductModal(); showMsg(window._i18n("custom_product_rejected","商品已拒绝"), false); loadPendingCustomProducts(); }
        else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Marketplace Management ---
var mpCategoriesLoaded = false;
function loadMarketplaceCategoryFilter() {
    if (mpCategoriesLoaded) return;
    apiFetch('/api/categories').then(function(r) { return r.json(); }).then(function(data) {
        var cats = Array.isArray(data) ? data : (data.categories || []);
        var sel = document.getElementById('mp-category-filter');
        for (var i = 0; i < cats.length; i++) {
            var opt = document.createElement('option');
            opt.value = cats[i].id;
            opt.textContent = cats[i].name;
            sel.appendChild(opt);
        }
        mpCategoriesLoaded = true;
    });
}

function shareModeLabel(mode) {
    var labels = { free: window._i18n("free","免费"), per_use: window._i18n("per_use","按次"), subscription: window._i18n("subscription_mode","订阅") };
    return labels[mode] || mode;
}

function loadMarketplacePacks() {
    loadMarketplaceCategoryFilter();
    var status = document.getElementById('mp-status-filter').value;
    var catId = document.getElementById('mp-category-filter').value;
    var mode = document.getElementById('mp-mode-filter').value;
    var sort = document.getElementById('mp-sort').value;
    var order = document.getElementById('mp-order').value;
    document.querySelector('#section-marketplace .card-header h2').textContent = status === 'delisted' ? window._i18n("marketplace_delisted","市场管理 - 已下架分析包") : window._i18n("marketplace_listed","市场管理 - 在售分析包");
    var url = '/api/admin/marketplace?status=' + status + '&sort=' + sort + '&order=' + order;
    if (catId) url += '&category_id=' + catId;
    if (mode) url += '&share_mode=' + mode;
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var packs = data.packs || [];
        var tbody = document.getElementById('marketplace-list');
        if (packs.length === 0) {
            var emptyMsg = status === 'delisted' ? window._i18n("no_delisted_packs","暂无已下架分析包") : window._i18n("no_listed_packs","暂无在售分析包");
            tbody.innerHTML = '<tr><td colspan="9" style="text-align:center;color:#999;">' + emptyMsg + '</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < packs.length; i++) {
            var p = packs[i];
            var priceText = p.share_mode === 'free' ? window._i18n("free","免费") : p.credits_price + ' Credits';
            html += '<tr>';
            html += '<td>' + p.id + '</td>';
            html += '<td>' + escHtml(p.pack_name) + '</td>';
            html += '<td>' + escHtml(p.category_name) + '</td>';
            html += '<td>' + escHtml(p.author_name || '-') + '</td>';
            html += '<td>' + shareModeLabel(p.share_mode) + '</td>';
            html += '<td>' + priceText + '</td>';
            html += '<td>' + p.download_count + '</td>';
            html += '<td>' + p.created_at + '</td>';
            if (status === 'delisted') {
                html += '<td><button class="btn btn-primary btn-sm" onclick="relistPack(' + p.id + ',\'' + escAttr(p.pack_name) + '\')">' + window._i18n("restore_listing","恢复在售") + '</button></td>';
            } else {
                html += '<td><button class="btn btn-danger btn-sm" onclick="delistPack(' + p.id + ',\'' + escAttr(p.pack_name) + '\')">' + window._i18n("delist","下架") + '</button></td>';
            }
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg(window._i18n("load_failed","加载失败") + ': ' + err, true); });
}

function delistPack(id, name) {
    if (!confirm(window._i18n("confirm_delist_admin","确定要下架 \"{name}\" 吗？（下架后不删除，可在数据库中恢复）").replace("{name}", name))) return;
    apiFetch('/api/admin/marketplace/' + id + '/delist', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg(window._i18n("pack_delisted","已下架"), false); loadMarketplacePacks(); }
            else { showMsg(res.data.error || window._i18n("delist_failed","下架失败"), true); }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function relistPack(id, name) {
    if (!confirm(window._i18n("confirm_relist","确定要恢复 \"{name}\" 为在售状态吗？").replace("{name}", name))) return;
    apiFetch('/api/admin/marketplace/' + id + '/relist', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg(window._i18n("pack_relisted","已恢复在售"), false); loadMarketplacePacks(); }
            else { showMsg(res.data.error || window._i18n("relist_failed","恢复失败"), true); }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Unified Account Management ---
function loadAccounts() {
    var search = document.getElementById('account-search').value.trim();
    var role = document.getElementById('account-role-filter').value;
    var sort = document.getElementById('account-sort').value;
    var order = document.getElementById('account-order').value;
    var url = '/api/admin/accounts?sort=' + sort + '&order=' + order;
    if (search) url += '&search=' + encodeURIComponent(search);
    if (role) url += '&role=' + role;
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var accounts = data.accounts || [];
        var tbody = document.getElementById('account-list');
        if (accounts.length === 0) {
            tbody.innerHTML = '<tr><td colspan="10" style="text-align:center;color:#999;">' + window._i18n("no_accounts","暂无账号数据") + '</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < accounts.length; i++) {
            var a = accounts[i];
            var roleBadge = a.is_author
                ? '<span class="badge" style="background:#fef3c7;color:#92400e;">✍️ ' + window._i18n("author","作者") + '</span>'
                : '<span class="badge" style="background:#e0e7ff;color:#3730a3;">👤 ' + window._i18n("customer","客户") + '</span>';
            if (a.account_count > 1) {
                roleBadge += ' <span class="badge" style="background:#eff6ff;color:#1e40af;font-size:11px;">' + a.account_count + ' SN</span>';
            }
            var statusBadge = a.is_blocked
                ? '<span class="badge" style="background:#fef2f2;color:#991b1b;">' + window._i18n("blocked","已禁用") + '</span>'
                : '<span class="badge" style="background:#ecfdf5;color:#065f46;">' + window._i18n("normal","正常") + '</span>';
            html += '<tr>';
            html += '<td>' + escHtml(a.email || '-') + '</td>';
            html += '<td>' + escHtml(a.display_name) + '</td>';
            html += '<td>' + roleBadge + '</td>';
            html += '<td>' + a.total_balance.toFixed(0) + '</td>';
            html += '<td>' + (a.published_packs || 0) + '</td>';
            html += '<td>' + (a.author_revenue || 0) + '</td>';
            html += '<td>' + a.total_downloads + '</td>';
            html += '<td>' + a.total_spent.toFixed(0) + '</td>';
            html += '<td>' + statusBadge + '</td>';
            html += '<td class="actions" style="white-space:nowrap;">';
            html += '<button class="btn btn-primary btn-sm" onclick="showAccountDetail(\'' + escAttr(a.email) + '\')">' + window._i18n("details","详情") + '</button> ';
            html += '<button class="btn btn-secondary btn-sm" onclick="showAccountTopup(\'' + escAttr(a.email) + '\',\'' + escAttr(a.display_name) + '\',' + a.total_balance + ')">' + window._i18n("topup","充值") + '</button> ';
            var blockBtn = a.is_blocked
                ? '<button class="btn btn-primary btn-sm" onclick="toggleAccountBlock(\'' + escAttr(a.email) + '\',\'' + escAttr(a.display_name) + '\',true)">' + window._i18n("unblock","解禁") + '</button>'
                : '<button class="btn btn-danger btn-sm" onclick="toggleAccountBlock(\'' + escAttr(a.email) + '\',\'' + escAttr(a.display_name) + '\',false)">' + window._i18n("block","禁用") + '</button>';
            html += blockBtn;
            var emailBtn = a.email_allowed
                ? '<button class="btn btn-secondary btn-sm" style="font-size:11px;" onclick="toggleEmailPermission(\'' + escAttr(a.email) + '\',\'' + escAttr(a.display_name) + '\',true)" title="' + window._i18n("disable_email","禁用邮件") + '">📧✓</button>'
                : '<button class="btn btn-danger btn-sm" style="font-size:11px;" onclick="toggleEmailPermission(\'' + escAttr(a.email) + '\',\'' + escAttr(a.display_name) + '\',false)" title="' + window._i18n("enable_email","启用邮件") + '">📧✗</button>';
            html += ' ' + emailBtn;
            if (a.storefront_id) {
                var cpBtn = a.custom_products_enabled
                    ? '<button class="btn btn-secondary btn-sm" style="font-size:11px;" onclick="toggleCustomProducts(' + a.storefront_id + ',true)" title="' + window._i18n("disable_custom_products","关闭自定义商品") + '">🛍️✓</button>'
                    : '<button class="btn btn-danger btn-sm" style="font-size:11px;" onclick="toggleCustomProducts(' + a.storefront_id + ',false)" title="' + window._i18n("enable_custom_products","允许自定义商品") + '">🛍️✗</button>';
                html += ' ' + cpBtn;
            }
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg(window._i18n("load_accounts_failed","加载账号列表失败") + ': ' + err, true); });
}

var accountDetailCurrentPage = 1;
var accountDetailTotalPages = 1;
var accountDetailCurrentEmail = '';

function showAccountDetail(email, page) {
    if (typeof page !== 'number' || page < 1) page = 1;
    accountDetailCurrentEmail = email;
    accountDetailCurrentPage = page;
    var url = '/api/admin/accounts/detail?email=' + encodeURIComponent(email) + '&page=' + page + '&page_size=10';
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        document.getElementById('account-detail-title').textContent = escHtml(data.display_name) + ' (' + escHtml(data.email) + ')';
        document.getElementById('account-detail-info').innerHTML =
            window._i18n("wallet_balance","钱包余额") + ': <b>' + (data.wallet_balance || 0).toFixed(0) + ' Credits</b>';

        // Sub-accounts
        var subHtml = '';
        var subs = data.sub_accounts || [];
        if (subs.length > 1) {
            subHtml += '<div style="font-size:12px;color:#6b7280;margin-bottom:8px;">' + window._i18n("sub_accounts","关联账号") + ' (' + subs.length + '):</div>';
            subHtml += '<div style="display:flex;flex-wrap:wrap;gap:6px;">';
            for (var si = 0; si < subs.length; si++) {
                var s = subs[si];
                var sStatus = s.is_blocked ? '🔴' : '🟢';
                subHtml += '<span class="badge" style="background:#f1f5f9;color:#475569;font-size:11px;">' + sStatus + ' ' + escHtml(s.auth_id) + ' (' + escHtml(s.display_name) + ')</span>';
            }
            subHtml += '</div>';
        }
        document.getElementById('account-detail-sub-accounts').innerHTML = subHtml;

        // Author packs tab visibility
        var tabsEl = document.getElementById('account-detail-tabs');
        tabsEl.style.display = data.is_author ? '' : 'none';

        accountDetailCurrentPage = data.page || 1;
        accountDetailTotalPages = data.total_pages || 1;
        var packs = data.packs || [];
        var tbody = document.getElementById('account-detail-packs');
        if (packs.length === 0 && data.total_packs === 0) {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#999;">' + window._i18n("no_packs","暂无分析包") + '</td></tr>';
        } else {
            var html = '';
            for (var pi = 0; pi < packs.length; pi++) {
                var p = packs[pi];
                var statusLabel = p.status === 'published' ? window._i18n("listed_status","在售") : window._i18n("delisted","已下架");
                html += '<tr>';
                html += '<td>' + escHtml(p.pack_name) + '</td>';
                html += '<td>' + escHtml(p.category_name) + '</td>';
                html += '<td>' + shareModeLabel(p.share_mode) + '</td>';
                html += '<td>' + (p.share_mode === 'free' ? window._i18n("free","免费") : p.credits_price + ' Credits') + '</td>';
                html += '<td>' + p.download_count + '</td>';
                html += '<td>' + p.total_revenue + '</td>';
                html += '<td>' + statusLabel + '</td>';
                html += '</tr>';
            }
            tbody.innerHTML = html;
        }
        renderAccountDetailPagination(data.total_packs || 0);
        loadAccountTxPage(1);
        switchAccountDetailTab(data.is_author ? 'packs' : 'tx');
        document.getElementById('account-detail-modal').className = 'modal-overlay show';
    }).catch(function(err) { showMsg(window._i18n("load_account_detail_failed","加载账号详情失败") + ': ' + err, true); });
}

function switchAccountDetailTab(tab) {
    document.getElementById('acct-tab-packs').style.display = tab === 'packs' ? '' : 'none';
    document.getElementById('acct-tab-tx').style.display = tab === 'tx' ? '' : 'none';
    document.getElementById('acct-tab-packs-btn').className = tab === 'packs' ? 'btn btn-primary btn-sm' : 'btn btn-secondary btn-sm';
    document.getElementById('acct-tab-tx-btn').className = tab === 'tx' ? 'btn btn-primary btn-sm' : 'btn btn-secondary btn-sm';
}

function renderAccountDetailPagination(totalPacks) {
    var pageInfo = document.getElementById('account-detail-page-info');
    var prevBtn = document.getElementById('account-detail-prev-btn');
    var nextBtn = document.getElementById('account-detail-next-btn');
    var pageNums = document.getElementById('account-detail-page-nums');
    if (totalPacks === 0) { pageInfo.textContent = ''; prevBtn.disabled = true; nextBtn.disabled = true; pageNums.innerHTML = ''; return; }
    var start = (accountDetailCurrentPage - 1) * 10 + 1;
    var end = Math.min(accountDetailCurrentPage * 10, totalPacks);
    pageInfo.textContent = window._i18n("showing_range","显示 {start}-{end} 条，共 {total} 条").replace("{start}", start).replace("{end}", end).replace("{total}", totalPacks);
    prevBtn.disabled = accountDetailCurrentPage <= 1;
    nextBtn.disabled = accountDetailCurrentPage >= accountDetailTotalPages;
    var html = '';
    var lo = Math.max(1, accountDetailCurrentPage - 3);
    var hi = Math.min(accountDetailTotalPages, accountDetailCurrentPage + 3);
    if (lo > 1) html += '<button class="btn btn-secondary btn-sm" onclick="accountDetailGoPage(1)" style="min-width:32px;">1</button>';
    if (lo > 2) html += '<span style="color:#9ca3af;padding:0 4px;">…</span>';
    for (var pg = lo; pg <= hi; pg++) {
        html += pg === accountDetailCurrentPage
            ? '<button class="btn btn-primary btn-sm" style="min-width:32px;" disabled>' + pg + '</button>'
            : '<button class="btn btn-secondary btn-sm" onclick="accountDetailGoPage(' + pg + ')" style="min-width:32px;">' + pg + '</button>';
    }
    if (hi < accountDetailTotalPages - 1) html += '<span style="color:#9ca3af;padding:0 4px;">…</span>';
    if (hi < accountDetailTotalPages) html += '<button class="btn btn-secondary btn-sm" onclick="accountDetailGoPage(' + accountDetailTotalPages + ')" style="min-width:32px;">' + accountDetailTotalPages + '</button>';
    pageNums.innerHTML = html;
}

function accountDetailGoPage(page) {
    if (page < 1) page = 1;
    if (page > accountDetailTotalPages) page = accountDetailTotalPages;
    showAccountDetail(accountDetailCurrentEmail, page);
}

function hideAccountDetailModal() {
    document.getElementById('account-detail-modal').className = 'modal-overlay';
}

function loadAccountTxPage(page) {
    var email = accountDetailCurrentEmail;
    var tbody = document.getElementById('account-detail-tx-list');
    var pagination = document.getElementById('account-detail-tx-pagination');
    tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#999;">' + window._i18n("loading","加载中...") + '</td></tr>';
    pagination.innerHTML = '';
    var url = '/api/admin/accounts/transactions?email=' + encodeURIComponent(email) + '&page=' + page + '&pageSize=20';
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var txns = data.transactions || [];
        if (txns.length === 0 && page === 1) { tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#999;">' + window._i18n("no_transactions_admin","暂无交易记录") + '</td></tr>'; return; }
        var typeLabels = { download: window._i18n("tx_download","下载扣费"), admin_topup: window._i18n("tx_admin_topup","管理员充值"), initial: window._i18n("tx_initial","注册赠送"), purchase: window._i18n("tx_purchase","购买"), purchase_uses: window._i18n("tx_purchase","购买"), renew: window._i18n("tx_renew","续费") };
        var html = '';
        for (var ti = 0; ti < txns.length; ti++) {
            var t = txns[ti];
            var amountStyle = t.amount >= 0 ? 'color:#065f46;' : 'color:#991b1b;';
            var amountText = t.amount >= 0 ? '+' + t.amount : '' + t.amount;
            html += '<tr><td>' + t.id + '</td><td>' + (typeLabels[t.transaction_type] || t.transaction_type) + '</td>';
            html += '<td style="' + amountStyle + 'font-weight:600;">' + amountText + '</td>';
            html += '<td>' + escHtml(t.description || '-') + (t.account_name ? ' <span style="color:#6b7280;font-size:11px;">(' + escHtml(t.account_name) + ')</span>' : '') + '</td>';
            html += '<td>' + t.created_at + '</td></tr>';
        }
        tbody.innerHTML = html;
        var total = data.total || 0; var totalPages = data.totalPages || 1; var curPage = data.page || 1;
        var pgHtml = '<span style="color:#64748b;">' + window._i18n("total_records","共") + ' ' + total + ' ' + window._i18n("records_unit","条") + '</span>';
        if (totalPages > 1) {
            pgHtml += '<button class="btn btn-secondary btn-sm" onclick="loadAccountTxPage(1)"' + (curPage === 1 ? ' disabled style="opacity:0.4;"' : '') + '>' + window._i18n("first_page","首页") + '</button>';
            pgHtml += '<button class="btn btn-secondary btn-sm" onclick="loadAccountTxPage(' + (curPage - 1) + ')"' + (curPage === 1 ? ' disabled style="opacity:0.4;"' : '') + '>' + window._i18n("prev_page","上一页") + '</button>';
            pgHtml += '<span>' + curPage + ' / ' + totalPages + '</span>';
            pgHtml += '<button class="btn btn-secondary btn-sm" onclick="loadAccountTxPage(' + (curPage + 1) + ')"' + (curPage === totalPages ? ' disabled style="opacity:0.4;"' : '') + '>' + window._i18n("next_page","下一页") + '</button>';
            pgHtml += '<button class="btn btn-secondary btn-sm" onclick="loadAccountTxPage(' + totalPages + ')"' + (curPage === totalPages ? ' disabled style="opacity:0.4;"' : '') + '>' + window._i18n("last_page","末页") + '</button>';
        }
        pagination.innerHTML = pgHtml;
    }).catch(function(err) { tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#991b1b;">' + window._i18n("load_tx_failed","加载失败") + '</td></tr>'; });
}

function showAccountTopup(email, name, balance) {
    document.getElementById('topup-email').value = email;
    document.getElementById('topup-user-info').textContent = name + ' (' + email + ')  |  ' + window._i18n("current_balance_label","当前余额:") + ' ' + balance.toFixed(0) + ' Credits';
    document.getElementById('topup-amount').value = '';
    document.getElementById('topup-reason').value = '';
    document.getElementById('topup-modal').className = 'modal-overlay show';
}

function hideTopupModal() { document.getElementById('topup-modal').className = 'modal-overlay'; }

function submitTopup() {
    var email = document.getElementById('topup-email').value;
    var amount = parseFloat(document.getElementById('topup-amount').value);
    var reason = document.getElementById('topup-reason').value.trim();
    if (!amount || amount <= 0) { alert(window._i18n("enter_valid_topup","请输入有效的充值数量")); return; }
    apiFetch('/api/admin/accounts/topup', {
        method: 'POST', headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({email: email, amount: amount, reason: reason})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { hideTopupModal(); showMsg(window._i18n("topup_success","充值成功，新余额:") + ' ' + res.data.new_balance, false); loadAccounts(); }
        else { showMsg(res.data.error || window._i18n("topup_failed","充值失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function toggleAccountBlock(email, name, isCurrentlyBlocked) {
    var action = isCurrentlyBlocked ? window._i18n("unblock","解禁") : window._i18n("block","禁用");
    if (!confirm(window._i18n("confirm_block","确定要{action}客户 \"{name}\" 吗？").replace("{action}", action).replace("{name}", name + ' (' + email + ')'))) return;
    apiFetch('/api/admin/accounts/toggle-block', {
        method: 'POST', headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({email: email})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg(res.data.status === 'blocked' ? window._i18n("blocked_done","已禁用") : window._i18n("unblocked_done","已解禁"), false); loadAccounts(); }
        else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function toggleEmailPermission(email, name, isCurrentlyAllowed) {
    var action = isCurrentlyAllowed ? window._i18n("disable_email","禁用邮件") : window._i18n("enable_email","启用邮件");
    if (!confirm(window._i18n("confirm_email_toggle","确定要{action} \"{name}\" 的邮件发送权限吗？").replace("{action}", action).replace("{name}", name + ' (' + email + ')'))) return;
    apiFetch('/api/admin/accounts/toggle-email', {
        method: 'POST', headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({email: email})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg(res.data.status === 'allowed' ? window._i18n("email_enabled","邮件权限已启用") : window._i18n("email_disabled","邮件权限已禁用"), false); loadAccounts(); }
        else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function toggleCustomProducts(storefrontId, isCurrentlyEnabled) {
    var action = isCurrentlyEnabled ? window._i18n("disable_custom_products","关闭自定义商品") : window._i18n("enable_custom_products","允许自定义商品");
    if (!confirm(window._i18n("confirm_custom_products_toggle","确定要{action}吗？").replace("{action}", action))) return;
    var fd = new FormData();
    fd.append('enabled', isCurrentlyEnabled ? 'false' : 'true');
    apiFetch('/admin/storefront/' + storefrontId + '/custom-products-toggle', { method: 'POST', body: fd })
    .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok && res.data.ok) {
            showMsg(isCurrentlyEnabled ? window._i18n("custom_products_disabled","已关闭自定义商品") : window._i18n("custom_products_enabled","已允许自定义商品"), false);
            loadAccounts();
        } else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// Keep old function names as aliases for backward compatibility
function loadCustomers() { loadAccounts(); }
function loadAuthors() { loadAccounts(); }

function saveProfile() {
    var username = document.getElementById('profile-username').value.trim();
    var oldPassword = document.getElementById('profile-old-password').value;
    var newPassword = document.getElementById('profile-new-password').value;
    if (!username && !newPassword) { alert(window._i18n("enter_change_content","请输入要修改的内容")); return; }
    if (newPassword && !oldPassword) { alert(window._i18n("need_current_pw","修改密码需要输入当前密码")); return; }
    if (newPassword && newPassword.length < 6) { alert(window._i18n("new_pw_min6","新密码至少6个字符")); return; }
    var body = {};
    if (username) body.username = username;
    if (newPassword) { body.old_password = oldPassword; body.new_password = newPassword; }
    apiFetch('/api/admin/profile', {
        method: 'PUT',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(body)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) {
            showMsg(window._i18n("profile_updated","资料已更新"), false);
            document.getElementById('profile-username').value = '';
            document.getElementById('profile-old-password').value = '';
            document.getElementById('profile-new-password').value = '';
        } else {
            var errMsg = res.data.error;
            if (errMsg === 'invalid_old_password') errMsg = window._i18n("invalid_old_password","当前密码错误");
            else if (errMsg === 'username_already_exists') errMsg = window._i18n("username_already_exists","用户名已被使用");
            showMsg(errMsg || window._i18n("change_failed","修改失败"), true);
        }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Create Notification Modal ---
var notifSelectedUsers = [];

function showCreateNotification() {
    document.getElementById('notif-title').value = '';
    document.getElementById('notif-content').value = '';
    document.getElementById('notif-target-type').value = 'broadcast';
    document.getElementById('notif-effective-date').value = '';
    document.getElementById('notif-duration').value = '0';
    document.getElementById('notif-user-search').value = '';
    document.getElementById('notif-user-results').innerHTML = '';
    notifSelectedUsers = [];
    renderSelectedUsers();
    toggleTargetUsers();
    document.getElementById('create-notif-modal').className = 'modal-overlay show';
}

function hideCreateNotification() {
    document.getElementById('create-notif-modal').className = 'modal-overlay';
}

function toggleTargetUsers() {
    var type = document.getElementById('notif-target-type').value;
    document.getElementById('notif-target-section').style.display = type === 'targeted' ? '' : 'none';
}

function searchNotifUsers() {
    var q = document.getElementById('notif-user-search').value.trim();
    if (!q) return;
    apiFetch('/api/admin/customers?search=' + encodeURIComponent(q)).then(function(r) { return r.json(); }).then(function(data) {
        var customers = data.customers || [];
        var container = document.getElementById('notif-user-results');
        if (customers.length === 0) {
            container.innerHTML = '<div style="padding:6px;color:#999;font-size:12px;">' + window._i18n("no_users_found","未找到用户") + '</div>';
            return;
        }
        var html = '';
        for (var i = 0; i < customers.length; i++) {
            var c = customers[i];
            var alreadySelected = false;
            for (var j = 0; j < notifSelectedUsers.length; j++) {
                if (notifSelectedUsers[j].id === c.id) { alreadySelected = true; break; }
            }
            if (alreadySelected) continue;
            html += '<div style="display:flex;justify-content:space-between;align-items:center;padding:4px 8px;border-bottom:1px solid #f3f4f6;font-size:13px;">';
            html += '<span>' + escHtml(c.display_name) + ' (' + escHtml(c.email || '-') + ')</span>';
            html += '<button class="btn btn-primary btn-sm" style="padding:2px 8px;font-size:11px;" onclick="addNotifUser(' + c.id + ',\'' + escAttr(c.display_name) + '\',\'' + escAttr(c.email || '') + '\')">' + window._i18n("add","添加") + '</button>';
            html += '</div>';
        }
        container.innerHTML = html || '<div style="padding:6px;color:#999;font-size:12px;">' + window._i18n("all_added","所有搜索结果已添加") + '</div>';
    }).catch(function(err) { showMsg(window._i18n("request_failed","搜索用户失败") + ': ' + err, true); });
}

function addNotifUser(id, name, email) {
    for (var i = 0; i < notifSelectedUsers.length; i++) {
        if (notifSelectedUsers[i].id === id) return;
    }
    notifSelectedUsers.push({id: id, name: name, email: email});
    renderSelectedUsers();
    searchNotifUsers();
}

function removeNotifUser(id) {
    notifSelectedUsers = notifSelectedUsers.filter(function(u) { return u.id !== id; });
    renderSelectedUsers();
}

function renderSelectedUsers() {
    var container = document.getElementById('notif-selected-users');
    if (notifSelectedUsers.length === 0) {
        container.innerHTML = window._i18n("no_users_selected","未选择用户");
        return;
    }
    var html = '';
    for (var i = 0; i < notifSelectedUsers.length; i++) {
        var u = notifSelectedUsers[i];
        html += '<span style="display:inline-flex;align-items:center;gap:4px;background:#e0f2fe;color:#0369a1;padding:2px 8px;border-radius:12px;margin:2px;font-size:12px;">';
        html += escHtml(u.name);
        html += '<span style="cursor:pointer;font-weight:bold;" onclick="removeNotifUser(' + u.id + ')">&times;</span>';
        html += '</span>';
    }
    container.innerHTML = html;
}

// --- Notification Management ---
function loadNotifications() {
    apiFetch('/api/admin/notifications').then(function(r) { return r.json(); }).then(function(data) {
        var notifs = data.notifications || [];
        var tbody = document.getElementById('notifications-tbody');
        if (notifs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#999;">' + window._i18n("no_notifications","暂无消息") + '</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < notifs.length; i++) {
            var n = notifs[i];
            var typeText = n.target_type === 'broadcast' ? window._i18n("broadcast","广播") : window._i18n("targeted","定向({count}人)").replace("{count}", n.target_count || 0);
            var statusBadge = n.status === 'active'
                ? '<span class="badge" style="background:#ecfdf5;color:#065f46;">' + window._i18n("active","活跃") + '</span>'
                : '<span class="badge" style="background:#f3f4f6;color:#6b7280;">' + window._i18n("disabled","已禁用") + '</span>';
            var durationText = n.display_duration_days === 0 ? window._i18n("permanent","永久") : n.display_duration_days + window._i18n("days_unit","天");
            var toggleBtn = n.status === 'active'
                ? '<button class="btn btn-secondary btn-sm" onclick="disableNotification(' + n.id + ')">' + window._i18n("disable","禁用") + '</button>'
                : '<button class="btn btn-primary btn-sm" onclick="enableNotification(' + n.id + ')">' + window._i18n("enable","启用") + '</button>';
            html += '<tr>';
            html += '<td>' + n.id + '</td>';
            html += '<td>' + escHtml(n.title) + '</td>';
            html += '<td>' + typeText + '</td>';
            html += '<td>' + statusBadge + '</td>';
            html += '<td>' + escHtml(n.effective_date || '-') + '</td>';
            html += '<td>' + durationText + '</td>';
            html += '<td>' + escHtml(n.created_at || '-') + '</td>';
            html += '<td class="actions">' + toggleBtn + ' <button class="btn btn-danger btn-sm" onclick="deleteNotification(' + n.id + ')">' + window._i18n("delete","删除") + '</button></td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg(window._i18n("load_notifications_failed","加载消息列表失败") + ': ' + err, true); });
}

function createNotification() {
    var title = document.getElementById('notif-title').value.trim();
    var content = document.getElementById('notif-content').value.trim();
    var targetType = document.getElementById('notif-target-type').value;
    var effectiveDate = document.getElementById('notif-effective-date').value;
    var duration = parseInt(document.getElementById('notif-duration').value) || 0;
    if (!title || !content) { alert(window._i18n("enter_title_content","请输入消息标题和内容")); return; }
    if (targetType === 'targeted' && notifSelectedUsers.length === 0) { alert(window._i18n("select_target_users","请选择目标用户")); return; }
    var targetUserIds = [];
    for (var i = 0; i < notifSelectedUsers.length; i++) {
        targetUserIds.push(notifSelectedUsers[i].id);
    }
    var body = {
        title: title,
        content: content,
        target_type: targetType,
        target_user_ids: targetUserIds,
        effective_date: effectiveDate ? new Date(effectiveDate).toISOString() : '',
        display_duration_days: duration
    };
    apiFetch('/api/admin/notifications', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(body)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) {
            hideCreateNotification();
            showMsg(window._i18n("notification_sent","消息已发送"), false);
            loadNotifications();
        } else {
            alert(res.data.error || window._i18n("send_failed","发送失败"));
        }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function disableNotification(id) {
    apiFetch('/api/admin/notifications/' + id + '/disable', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg(window._i18n("notification_disabled","消息已禁用"), false); loadNotifications(); }
            else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function enableNotification(id) {
    apiFetch('/api/admin/notifications/' + id + '/enable', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg(window._i18n("notification_enabled","消息已启用"), false); loadNotifications(); }
            else { showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true); }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

function deleteNotification(id) {
    if (!confirm(window._i18n("confirm_delete_notif","确定要删除该消息吗？"))) return;
    apiFetch('/api/admin/notifications/' + id + '/delete', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg(window._i18n("notification_deleted","消息已删除"), false); loadNotifications(); }
            else { showMsg(res.data.error || window._i18n("delete_failed","删除失败"), true); }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Withdrawal Management ---
function paymentTypeLabel(t) {
    var labels = { paypal: 'PayPal', wechat: window._i18n("wechat","微信"), alipay: 'AliPay', check: window._i18n("check","支票"), wire_transfer: window._i18n("wire_transfer","国际电汇"), bank_card_us: window._i18n("bank_card_us","美国银行卡"), bank_card_eu: window._i18n("bank_card_eu","欧洲银行卡"), bank_card_cn: window._i18n("bank_card_cn","中国银行卡") };
    return labels[t] || t;
}

function formatPaymentDetails(type, detailsStr) {
    try {
        var d = typeof detailsStr === 'string' ? JSON.parse(detailsStr) : detailsStr;
        if (type === 'paypal' || type === 'wechat' || type === 'alipay') return escHtml(d.account || '') + ' / ' + escHtml(d.username || '');
        if (type === 'bank_card') return escHtml(d.bank_name || '') + ' ' + escHtml(d.card_number || '') + ' ' + escHtml(d.account_holder || '');
        if (type === 'check') return escHtml(d.address || '');
        if (type === 'wire_transfer') return 'SWIFT:' + escHtml(d.swift_code || '') + ' ' + escHtml(d.beneficiary_name || '') + ' ' + escHtml(d.account_number || '');
        if (type === 'bank_card_us') return 'ACH:' + escHtml(d.routing_number || '') + ' ' + escHtml(d.legal_name || '') + ' ' + escHtml(d.account_type || '');
        if (type === 'bank_card_eu') return 'IBAN:' + escHtml(d.iban || '') + ' ' + escHtml(d.legal_name || '');
        if (type === 'bank_card_cn') return escHtml(d.bank_branch || '') + ' ' + escHtml(d.card_number || '') + ' ' + escHtml(d.real_name || '');
    } catch(e) {}
    return escHtml(detailsStr);
}

function getSelectedWithdrawalIds() {
    var boxes = document.querySelectorAll('.wd-check:checked');
    var ids = [];
    for (var i = 0; i < boxes.length; i++) ids.push(parseInt(boxes[i].value));
    return ids;
}

function exportWithdrawals() {
    var ids = getSelectedWithdrawalIds();
    if (ids.length === 0) { alert(window._i18n("select_records_export","请先选择要导出的提现记录")); return; }
    window.open('/admin/api/withdrawals/export?ids=' + ids.join(','), '_blank');
}

function exportAndApproveWithdrawals() {
    var boxes = document.querySelectorAll('.wd-check:checked');
    var allIds = [];
    var pendingIds = [];
    for (var i = 0; i < boxes.length; i++) {
        allIds.push(parseInt(boxes[i].value));
        if (boxes[i].getAttribute('data-status') === 'pending') pendingIds.push(parseInt(boxes[i].value));
    }
    if (allIds.length === 0) { alert(window._i18n("select_records_export","请先选择要导出的提现记录")); return; }
    var msg = window._i18n("will_export","将导出 {count} 条记录").replace("{count}", allIds.length);
    if (pendingIds.length > 0) msg += window._i18n("and_mark_paid","并将其中 {count} 条待审核记录标记为已付款").replace("{count}", pendingIds.length);
    if (!confirm(msg + window._i18n("confirm_continue","，确定继续？"))) return;
    // Export first
    window.open('/admin/api/withdrawals/export?ids=' + allIds.join(','), '_blank');
    // Then approve pending ones
    if (pendingIds.length > 0) {
        apiFetch('/admin/api/withdrawals/approve', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({ids: pendingIds})
        }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok || res.data.ok) {
                showMsg(window._i18n("exported_marked","已导出并标记 {count} 条记录为已付款").replace("{count}", res.data.updated || pendingIds.length), false);
                loadWithdrawals();
            } else {
                showMsg(res.data.error || window._i18n("mark_paid_failed","标记付款失败"), true);
            }
        }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
    }
}

function loadWithdrawals() {
    var status = document.getElementById('wd-status-filter').value;
    var author = (document.getElementById('wd-author-filter').value || '').trim();
    var url = '/admin/api/withdrawals';
    var params = [];
    if (status) params.push('status=' + encodeURIComponent(status));
    if (author) params.push('author=' + encodeURIComponent(author));
    if (params.length > 0) url += '?' + params.join('&');
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var list = data.withdrawals || [];
        var tbody = document.getElementById('withdrawals-list');
        document.getElementById('wd-select-all').checked = false;
        if (list.length === 0) {
            tbody.innerHTML = '<tr><td colspan="10" style="text-align:center;color:#999;">' + window._i18n("no_withdraw_records_admin","暂无提现记录") + '</td></tr>';
            document.getElementById('btn-batch-approve').style.display = 'none';
            return;
        }
        var hasPending = false;
        var html = '';
        for (var i = 0; i < list.length; i++) {
            var w = list[i];
            var statusBadge = w.status === 'pending'
                ? '<span class="badge" style="background:#fef3c7;color:#92400e;">' + window._i18n("applied_withdraw","已申请提现") + '</span>'
                : '<span class="badge" style="background:#ecfdf5;color:#065f46;">' + window._i18n("paid","已付款") + '</span>';
            if (w.status === 'pending') hasPending = true;
            html += '<tr>';
            html += '<td><input type="checkbox" class="wd-check" value="' + w.id + '" data-status="' + w.status + '" onchange="updateBatchBtn()" /></td>';
            html += '<td>' + escHtml(w.display_name) + '</td>';
            html += '<td>' + paymentTypeLabel(w.payment_type) + '</td>';
            html += '<td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="' + escAttr(w.payment_details) + '">' + formatPaymentDetails(w.payment_type, w.payment_details) + '</td>';
            html += '<td>' + w.cash_amount.toFixed(2) + '</td>';
            html += '<td>' + (w.fee_rate * 100).toFixed(2) + '%</td>';
            html += '<td>' + w.fee_amount.toFixed(2) + '</td>';
            html += '<td style="font-weight:600;">' + w.net_amount.toFixed(2) + '</td>';
            html += '<td>' + statusBadge + '</td>';
            html += '<td>' + escHtml(w.created_at) + '</td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
        document.getElementById('btn-batch-approve').style.display = hasPending ? '' : 'none';
    }).catch(function(err) { showMsg(window._i18n("load_withdrawals_failed","加载提现列表失败") + ': ' + err, true); });
}

function toggleSelectAllWithdrawals() {
    var checked = document.getElementById('wd-select-all').checked;
    var boxes = document.querySelectorAll('.wd-check');
    for (var i = 0; i < boxes.length; i++) boxes[i].checked = checked;
    updateBatchBtn();
}

function updateBatchBtn() {
    var boxes = document.querySelectorAll('.wd-check:checked');
    var hasPending = false;
    for (var i = 0; i < boxes.length; i++) {
        if (boxes[i].getAttribute('data-status') === 'pending') { hasPending = true; break; }
    }
    document.getElementById('btn-batch-approve').style.display = hasPending ? '' : 'none';
}

function batchApproveWithdrawals() {
    var boxes = document.querySelectorAll('.wd-check:checked');
    var ids = [];
    for (var i = 0; i < boxes.length; i++) {
        if (boxes[i].getAttribute('data-status') === 'pending') ids.push(parseInt(boxes[i].value));
    }
    if (ids.length === 0) { alert(window._i18n("select_pending_records","请选择待审核的提现记录")); return; }
    if (!confirm(window._i18n("confirm_batch_approve","确定将选中的 {count} 条提现记录标记为已付款？").replace("{count}", ids.length))) return;
    apiFetch('/admin/api/withdrawals/approve', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({ids: ids})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok || res.data.ok) {
            showMsg(window._i18n("marked_paid","已标记 {count} 条记录为已付款").replace("{count}", res.data.updated || ids.length), false);
            loadWithdrawals();
        } else {
            showMsg(res.data.error || window._i18n("operation_failed","操作失败"), true);
        }
    }).catch(function(err) { showMsg(window._i18n("request_failed","请求失败") + ': ' + err, true); });
}

// --- Sales Management ---
var salesCategoriesLoaded = false;
var salesAuthorsLoaded = false;
var salesCurrentPage = 1;
var salesTotalPages = 1;

function salesBuildFilterParams() {
    var category = document.getElementById('sales-category-filter').value;
    var author = document.getElementById('sales-author-filter').value;
    var dateFrom = document.getElementById('sales-date-from').value;
    var dateTo = document.getElementById('sales-date-to').value;
    var params = [];
    if (category) params.push('category_id=' + encodeURIComponent(category));
    if (author) params.push('author_id=' + encodeURIComponent(author));
    if (dateFrom) params.push('date_from=' + encodeURIComponent(dateFrom));
    if (dateTo) params.push('date_to=' + encodeURIComponent(dateTo));
    return params;
}

function loadSalesData(page) {
    loadSalesFilterOptions();
    if (typeof page !== 'number' || page < 1) page = 1;
    salesCurrentPage = page;
    var params = salesBuildFilterParams();
    params.push('page=' + page);
    params.push('page_size=100');
    var qs = '?' + params.join('&');
    apiFetch('/api/admin/sales' + qs).then(function(r) { return r.json(); }).then(function(data) {
        document.getElementById('sales-total-orders').textContent = data.total_orders || 0;
        document.getElementById('sales-total-credits').textContent = data.total_credits || 0;
        document.getElementById('sales-total-users').textContent = data.total_users || 0;
        document.getElementById('sales-total-authors').textContent = data.total_authors || 0;
        salesCurrentPage = data.page || 1;
        salesTotalPages = data.total_pages || 1;
        var orders = data.orders || [];
        var tbody = document.getElementById('sales-order-list');
        if (orders.length === 0) {
            tbody.innerHTML = '<tr><td colspan="10" style="text-align:center;color:#999;">' + window._i18n("no_orders","暂无订单数据") + '</td></tr>';
            renderSalesPagination(0);
            return;
        }
        var html = '';
        var typeLabels = {purchase:window._i18n("tx_purchase","购买"),purchase_uses:window._i18n("tx_purchase_uses","购买次数"),renew:window._i18n("tx_renew","续订"),download:window._i18n("tx_free_claim","免费领取")};
        for (var i = 0; i < orders.length; i++) {
            var o = orders[i];
            html += '<tr>';
            html += '<td>' + o.id + '</td>';
            html += '<td>' + escHtml(o.buyer_name || '-') + '</td>';
            html += '<td>' + escHtml(o.buyer_email || '-') + '</td>';
            html += '<td>' + escHtml(o.pack_name || '-') + '</td>';
            html += '<td>' + escHtml(o.category_name || '-') + '</td>';
            html += '<td>' + escHtml(o.author_name || '-') + '</td>';
            html += '<td style="font-weight:600;color:#059669;">' + Math.abs(o.amount) + '</td>';
            html += '<td>' + (typeLabels[o.transaction_type] || o.transaction_type) + '</td>';
            html += '<td style="font-size:11px;color:#9ca3af;">' + escHtml(o.buyer_ip || '-') + '</td>';
            html += '<td style="font-size:12px;">' + escHtml(o.created_at || '') + '</td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
        renderSalesPagination(data.total_orders || 0);
    }).catch(function(e) { if (e.message !== 'session_expired') showMsg(window._i18n("load_sales_failed","加载销售数据失败"), true); });
}

function renderSalesPagination(totalOrders) {
    var pageInfo = document.getElementById('sales-page-info');
    var prevBtn = document.getElementById('sales-prev-btn');
    var nextBtn = document.getElementById('sales-next-btn');
    var pageNums = document.getElementById('sales-page-nums');
    if (totalOrders === 0) {
        pageInfo.textContent = '';
        prevBtn.disabled = true;
        nextBtn.disabled = true;
        pageNums.innerHTML = '';
        return;
    }
    var start = (salesCurrentPage - 1) * 100 + 1;
    var end = Math.min(salesCurrentPage * 100, totalOrders);
    pageInfo.textContent = window._i18n("showing_range","显示 {start}-{end} 条，共 {total} 条").replace("{start}", start).replace("{end}", end).replace("{total}", totalOrders);
    prevBtn.disabled = salesCurrentPage <= 1;
    nextBtn.disabled = salesCurrentPage >= salesTotalPages;
    // Render page number buttons (show max 7 pages around current)
    var html = '';
    var lo = Math.max(1, salesCurrentPage - 3);
    var hi = Math.min(salesTotalPages, salesCurrentPage + 3);
    if (lo > 1) html += '<button class="btn btn-secondary btn-sm" onclick="salesGoPage(1)" style="min-width:32px;">1</button>';
    if (lo > 2) html += '<span style="color:#9ca3af;padding:0 4px;">…</span>';
    for (var p = lo; p <= hi; p++) {
        if (p === salesCurrentPage) {
            html += '<button class="btn btn-primary btn-sm" style="min-width:32px;" disabled>' + p + '</button>';
        } else {
            html += '<button class="btn btn-secondary btn-sm" onclick="salesGoPage(' + p + ')" style="min-width:32px;">' + p + '</button>';
        }
    }
    if (hi < salesTotalPages - 1) html += '<span style="color:#9ca3af;padding:0 4px;">…</span>';
    if (hi < salesTotalPages) html += '<button class="btn btn-secondary btn-sm" onclick="salesGoPage(' + salesTotalPages + ')" style="min-width:32px;">' + salesTotalPages + '</button>';
    pageNums.innerHTML = html;
}

function salesGoPage(page) {
    if (page < 1) page = 1;
    if (page > salesTotalPages) page = salesTotalPages;
    loadSalesData(page);
}

function loadSalesFilterOptions() {
    if (!salesCategoriesLoaded) {
        apiFetch('/api/categories').then(function(r) { return r.json(); }).then(function(data) {
            var cats = Array.isArray(data) ? data : (data.categories || []);
            var sel = document.getElementById('sales-category-filter');
            var current = sel.value;
            sel.innerHTML = '<option value="">' + window._i18n("all_categories_filter","全部分类") + '</option>';
            for (var i = 0; i < cats.length; i++) {
                sel.innerHTML += '<option value="' + cats[i].id + '">' + escHtml(cats[i].name) + '</option>';
            }
            sel.value = current;
            salesCategoriesLoaded = true;
        });
    }
    if (!salesAuthorsLoaded) {
        apiFetch('/api/admin/sales/authors').then(function(r) { return r.json(); }).then(function(data) {
            var authors = data.authors || [];
            var sel = document.getElementById('sales-author-filter');
            var current = sel.value;
            sel.innerHTML = '<option value="">' + window._i18n("all_authors_filter","全部作者") + '</option>';
            for (var i = 0; i < authors.length; i++) {
                sel.innerHTML += '<option value="' + authors[i].id + '">' + escHtml(authors[i].name) + '</option>';
            }
            sel.value = current;
            salesAuthorsLoaded = true;
        });
    }
}

function clearSalesFilters() {
    document.getElementById('sales-category-filter').value = '';
    document.getElementById('sales-author-filter').value = '';
    document.getElementById('sales-date-from').value = '';
    document.getElementById('sales-date-to').value = '';
    loadSalesData(1);
}

function exportSalesExcel() {
    var params = salesBuildFilterParams();
    var qs = params.length > 0 ? '?' + params.join('&') : '';
    window.open('/api/admin/sales/export' + qs, '_blank');
}

/* ===== Billing Management ===== */
var billingCurrentPage = 1;
var billingTotalPages = 1;

function switchBillingTab(tabId, btn) {
    var contents = document.querySelectorAll('#section-billing .wd-tab-content');
    for (var i = 0; i < contents.length; i++) { contents[i].style.display = 'none'; }
    document.getElementById(tabId).style.display = '';
    var tabs = document.querySelectorAll('#section-billing .wd-tab');
    for (var i = 0; i < tabs.length; i++) { tabs[i].classList.remove('active'); }
    btn.classList.add('active');
    if (tabId === 'billing-tab-email') { loadBillingData(1); }
}

function loadBillingData(page) {
    if (typeof page !== 'number' || page < 1) page = 1;
    billingCurrentPage = page;
    var storeFilter = document.getElementById('billing-store-filter').value.trim();
    var params = ['page=' + page, 'page_size=20'];
    if (storeFilter) params.push('store_name=' + encodeURIComponent(storeFilter));
    apiFetch('/admin/api/billing?' + params.join('&'))
    .then(function(r) { return r.json(); })
    .then(function(data) {
        var records = data.records || [];
        var total = data.total || 0;
        var pageSize = data.page_size || 20;
        billingCurrentPage = data.page || 1;
        billingTotalPages = Math.ceil(total / pageSize) || 1;
        var tbody = document.getElementById('billing-list');
        if (records.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#9ca3af;padding:32px;">暂无收费记录</td></tr>';
        } else {
            tbody.innerHTML = records.map(function(r) {
                return '<tr>' +
                    '<td>' + r.id + '</td>' +
                    '<td>' + (r.store_name || '-') + '</td>' +
                    '<td>' + r.recipient_count + '</td>' +
                    '<td style="font-weight:600;color:#dc2626;">' + r.credits_used + '</td>' +
                    '<td style="max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="' + (r.description||'').replace(/"/g,'&quot;') + '">' + (r.description || '-') + '</td>' +
                    '<td>' + r.created_at + '</td>' +
                    '</tr>';
            }).join('');
        }
        // Summary
        var summaryEl = document.getElementById('billing-summary');
        summaryEl.innerHTML = '<span>共 <b>' + total + '</b> 条记录</span><span>总消耗 <b style="color:#dc2626;">' + (data.total_credits || 0) + '</b> Credits</span>';
        // Pagination
        renderBillingPagination(total, pageSize);
    }).catch(function() { showMsg('请求失败', true); });
}

function renderBillingPagination(total, pageSize) {
    var pageInfo = document.getElementById('billing-page-info');
    var prevBtn = document.getElementById('billing-prev-btn');
    var nextBtn = document.getElementById('billing-next-btn');
    var pageNums = document.getElementById('billing-page-nums');
    if (total === 0) {
        pageInfo.textContent = '';
        prevBtn.disabled = true;
        nextBtn.disabled = true;
        pageNums.innerHTML = '';
        return;
    }
    var start = (billingCurrentPage - 1) * pageSize + 1;
    var end = Math.min(billingCurrentPage * pageSize, total);
    pageInfo.textContent = '显示 ' + start + '-' + end + ' 条，共 ' + total + ' 条';
    prevBtn.disabled = billingCurrentPage <= 1;
    nextBtn.disabled = billingCurrentPage >= billingTotalPages;
    var html = '';
    var lo = Math.max(1, billingCurrentPage - 3);
    var hi = Math.min(billingTotalPages, billingCurrentPage + 3);
    if (lo > 1) html += '<button class="btn btn-secondary btn-sm" onclick="billingGoPage(1)" style="min-width:32px;">1</button>';
    if (lo > 2) html += '<span style="color:#9ca3af;padding:0 4px;">…</span>';
    for (var p = lo; p <= hi; p++) {
        if (p === billingCurrentPage) {
            html += '<button class="btn btn-primary btn-sm" style="min-width:32px;" disabled>' + p + '</button>';
        } else {
            html += '<button class="btn btn-secondary btn-sm" onclick="billingGoPage(' + p + ')" style="min-width:32px;">' + p + '</button>';
        }
    }
    if (hi < billingTotalPages - 1) html += '<span style="color:#9ca3af;padding:0 4px;">…</span>';
    if (hi < billingTotalPages) html += '<button class="btn btn-secondary btn-sm" onclick="billingGoPage(' + billingTotalPages + ')" style="min-width:32px;">' + billingTotalPages + '</button>';
    pageNums.innerHTML = html;
}

function billingGoPage(page) {
    if (page < 1 || page > billingTotalPages) return;
    loadBillingData(page);
}

function exportBillingExcel() {
    var storeFilter = document.getElementById('billing-store-filter').value.trim();
    var qs = storeFilter ? '?store_name=' + encodeURIComponent(storeFilter) : '';
    window.open('/admin/api/billing/export' + qs, '_blank');
}

// ===== Featured Storefronts Management =====
function escapeHtml(str) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str || ''));
    return div.innerHTML;
}
var featuredSearchTimer = null;

function loadFeaturedStorefronts() {
    apiFetch('/api/admin/featured-storefronts').then(function(data) {
        if (!data.ok) { showMsg(data.error || 'Failed to load', true); return; }
        var list = data.data || [];
        var countEl = document.getElementById('featured-count');
        countEl.textContent = window._i18n('featured_count_label', '已选') + ' ' + list.length + '/16 ' + window._i18n('featured_count_unit', '个明星店铺');
        countEl.style.color = list.length >= 16 ? '#ef4444' : '#6b7280';
        var tbody = document.getElementById('featured-list');
        if (list.length === 0) {
            tbody.innerHTML = '<tr><td colspan="3" style="text-align:center;color:#9ca3af;padding:32px;">' + window._i18n('no_featured_stores', '暂无明星店铺，请通过上方搜索添加') + '</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < list.length; i++) {
            var s = list[i];
            html += '<tr data-id="' + s.storefront_id + '">';
            html += '<td style="text-align:center;">';
            if (i > 0) html += '<button class="btn btn-secondary btn-sm" onclick="moveFeatured(' + i + ',-1)" title="上移" style="padding:2px 6px;margin-right:4px;">↑</button>';
            if (i < list.length - 1) html += '<button class="btn btn-secondary btn-sm" onclick="moveFeatured(' + i + ',1)" title="下移" style="padding:2px 6px;">↓</button>';
            html += '</td>';
            html += '<td>' + escapeHtml(s.store_name) + '</td>';
            html += '<td><button class="btn btn-danger btn-sm" onclick="removeFeatured(' + s.storefront_id + ')">' + window._i18n('remove', '移除') + '</button></td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    });
}

function searchFeaturedStores() {
    clearTimeout(featuredSearchTimer);
    var q = document.getElementById('featured-search-input').value.trim();
    var resultsDiv = document.getElementById('featured-search-results');
    if (q.length < 1) { resultsDiv.style.display = 'none'; return; }
    featuredSearchTimer = setTimeout(function() {
        apiFetch('/api/admin/featured-storefronts/search?q=' + encodeURIComponent(q)).then(function(data) {
            if (!data.ok || !data.data || data.data.length === 0) {
                resultsDiv.innerHTML = '<div style="padding:12px;color:#9ca3af;font-size:13px;">' + window._i18n('no_results', '无匹配结果') + '</div>';
                resultsDiv.style.display = '';
                return;
            }
            var html = '';
            for (var i = 0; i < data.data.length; i++) {
                var s = data.data[i];
                html += '<div onclick="addFeatured(' + s.id + ')" style="padding:10px 14px;cursor:pointer;font-size:13px;border-bottom:1px solid #f3f4f6;transition:background 0.1s;"';
                html += ' onmouseover="this.style.background=\'#f9fafb\'" onmouseout="this.style.background=\'#fff\'">';
                html += escapeHtml(s.store_name);
                if (s.display_name && s.display_name !== s.store_name) html += ' <span style="color:#9ca3af;font-size:12px;">(@' + escapeHtml(s.display_name) + ')</span>';
                html += '</div>';
            }
            resultsDiv.innerHTML = html;
            resultsDiv.style.display = '';
        });
    }, 300);
}

function addFeatured(storefrontId) {
    var fd = new FormData();
    fd.append('storefront_id', storefrontId);
    apiFetch('/api/admin/featured-storefronts', { method: 'POST', body: fd }).then(function(data) {
        if (!data.ok) { showMsg(data.error || 'Failed to add', true); return; }
        document.getElementById('featured-search-input').value = '';
        document.getElementById('featured-search-results').style.display = 'none';
        showMsg(window._i18n('featured_added', '已添加为明星店铺'));
        loadFeaturedStorefronts();
    });
}

function removeFeatured(storefrontId) {
    if (!confirm(window._i18n('confirm_remove_featured', '确定移除该明星店铺？'))) return;
    var fd = new FormData();
    fd.append('storefront_id', storefrontId);
    apiFetch('/api/admin/featured-storefronts/remove', { method: 'POST', body: fd }).then(function(data) {
        if (!data.ok) { showMsg(data.error || 'Failed to remove', true); return; }
        showMsg(window._i18n('featured_removed', '已移除明星店铺'));
        loadFeaturedStorefronts();
    });
}

function moveFeatured(index, direction) {
    var rows = document.querySelectorAll('#featured-list tr[data-id]');
    var ids = [];
    for (var i = 0; i < rows.length; i++) ids.push(rows[i].getAttribute('data-id'));
    var newIndex = index + direction;
    if (newIndex < 0 || newIndex >= ids.length) return;
    var tmp = ids[index];
    ids[index] = ids[newIndex];
    ids[newIndex] = tmp;
    var fd = new FormData();
    fd.append('ids', ids.join(','));
    apiFetch('/api/admin/featured-storefronts/reorder', { method: 'POST', body: fd }).then(function(data) {
        if (!data.ok) { showMsg(data.error || 'Failed to reorder', true); return; }
        loadFeaturedStorefronts();
    });
}

// Hide search results when clicking outside
document.addEventListener('click', function(e) {
    var searchArea = document.getElementById('featured-search-input');
    var resultsDiv = document.getElementById('featured-search-results');
    if (searchArea && resultsDiv && !searchArea.contains(e.target) && !resultsDiv.contains(e.target)) {
        resultsDiv.style.display = 'none';
    }
});

// Init: show first available section based on permissions
(function initDefaultSection() {
    var order = ['categories', 'marketplace', 'accounts', 'review', 'settings', 'notifications', 'featured', 'sales', 'billing'];
    for (var i = 0; i < order.length; i++) {
        if (hasPerm(order[i])) {
            showSection(order[i]);
            return;
        }
    }
    // Fallback: every admin can access profile
    showSection('profile');
})();
</script>
` + I18nJS + `
</body>
</html>`
