package templates

// AdminHTML contains the marketplace admin panel HTML template.
const AdminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>市场管理后台</title>
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
        <h1>📦 市场管理</h1>
        <span>Marketplace Admin</span>
    </div>
    <nav class="sidebar-nav">
        <a href="#categories" data-perm="categories" onclick="showSection('categories')" style="display:none;"><span class="nav-icon">📂</span>分类管理</a>
        <a href="#marketplace" data-perm="marketplace" onclick="showSection('marketplace')" style="display:none;"><span class="nav-icon">🏪</span>市场管理</a>
        <a href="#authors" data-perm="authors" onclick="showSection('authors')" style="display:none;"><span class="nav-icon">✍️</span>作者管理</a>
        <a href="#customers" data-perm="customers" onclick="showSection('customers')" style="display:none;"><span class="nav-icon">👥</span>客户管理</a>
        <a href="#review" data-perm="review" onclick="showSection('review')" style="display:none;"><span class="nav-icon">📋</span>审核管理</a>
        <a href="#settings" data-perm="settings" onclick="showSection('settings')" style="display:none;"><span class="nav-icon">⚙️</span>系统设置</a>
        <a href="#notifications" data-perm="notifications" onclick="showSection('notifications')" style="display:none;"><span class="nav-icon">📢</span>消息管理</a>
        <a href="#withdrawals" data-perm="settings" onclick="showSection('withdrawals')" style="display:none;"><span class="nav-icon">💰</span>提现管理</a>
        <a href="#sales" data-perm="sales" onclick="showSection('sales')" style="display:none;"><span class="nav-icon">📊</span>销售管理</a>
        <a href="#admins" data-perm="admin_manage" onclick="showSection('admins')" style="display:none;"><span class="nav-icon">🔑</span>管理员管理</a>
        <div class="nav-divider"></div>
        <a href="#profile" onclick="showSection('profile')"><span class="nav-icon">👤</span>修改资料</a>
    </nav>
    <div class="sidebar-footer">
        <a href="/admin/logout">⏻ 退出登录</a>
    </div>
</aside>

<!-- Main -->
<div class="main-wrap">
    <header class="topbar">
        <div class="topbar-title" id="topbar-title">管理面板</div>
        <div class="topbar-user">
            <div class="avatar">A</div>
            <span>管理员</span>
        </div>
    </header>
    <main class="content">
    <div id="msg-area"></div>

    <!-- Categories Section -->
    <div id="section-categories">
        <div class="card">
            <div class="card-header">
                <h2>分类管理</h2>
                <button class="btn btn-primary" onclick="showCreateCategory()">+ 新建分类</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>描述</th><th>分析包数</th><th>类型</th><th>操作</th></tr>
                </thead>
                <tbody id="category-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Settings Section -->
    <div id="section-settings" style="display:none;">
        <div class="card">
            <h2>初始 Credits 余额</h2>
            <p class="form-hint" style="margin-bottom:16px;">新用户注册时自动获得的 Credits 数量</p>
            <form id="credits-form" onsubmit="saveInitialCredits(event)">
                <div class="form-group">
                    <label for="initial-credits">初始余额</label>
                    <input type="number" id="initial-credits" min="0" step="1" value="{{.InitialCredits}}" />
                </div>
                <button type="submit" class="btn btn-primary">保存设置</button>
            </form>
        </div>
    </div>

    <!-- Review Section (all admins) -->
    <div id="section-review" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>审核管理</h2>
                <button class="btn btn-secondary" onclick="loadPendingPacks()">↻ 刷新</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>分类</th><th>作者</th><th>模式</th><th>价格</th><th>上传时间</th><th>操作</th></tr>
                </thead>
                <tbody id="pending-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Marketplace Management Section -->
    <div id="section-marketplace" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>市场管理 - 在售分析包</h2>
                <button class="btn btn-secondary" onclick="loadMarketplacePacks()">↻ 刷新</button>
            </div>
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <select id="mp-status-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="published">在售</option>
                    <option value="delisted">已下架</option>
                </select>
                <select id="mp-category-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="">全部分类</option>
                </select>
                <select id="mp-mode-filter" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="">全部付费方式</option>
                    <option value="free">免费</option>
                    <option value="per_use">按次付费</option>
                    <option value="subscription">订阅</option>
                </select>
                <select id="mp-sort" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="downloads">按下载量排序</option>
                    <option value="price">按价格排序</option>
                    <option value="name">按名称排序</option>
                </select>
                <select id="mp-order" onchange="loadMarketplacePacks()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="desc">降序</option>
                    <option value="asc">升序</option>
                </select>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>分类</th><th>作者</th><th>付费方式</th><th>价格</th><th>下载量</th><th>上架时间</th><th>操作</th></tr>
                </thead>
                <tbody id="marketplace-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Author Management Section -->
    <div id="section-authors" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>作者管理</h2>
                <button class="btn btn-secondary" onclick="loadAuthors()">↻ 刷新</button>
            </div>
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <input type="text" id="author-email-search" placeholder="按邮箱搜索..." style="width:240px;" onkeydown="if(event.key==='Enter')loadAuthors()" />
                <button class="btn btn-primary btn-sm" onclick="loadAuthors()">搜索</button>
                <select id="author-sort" onchange="loadAuthors()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="total_downloads">按总下载量</option>
                    <option value="total_packs">按总包数</option>
                    <option value="year_downloads">按年下载量</option>
                    <option value="year_revenue">按年收入</option>
                    <option value="month_downloads">按月下载量</option>
                    <option value="month_revenue">按月收入</option>
                </select>
                <select id="author-order" onchange="loadAuthors()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="desc">降序</option>
                    <option value="asc">升序</option>
                </select>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>邮箱</th><th>包数</th><th>总下载</th><th>总收入</th><th>年下载</th><th>年收入</th><th>月下载</th><th>月收入</th><th>操作</th></tr>
                </thead>
                <tbody id="author-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Author Detail Modal -->
    <div id="author-detail-modal" class="modal-overlay">
        <div class="modal" style="width:640px;">
            <h3 id="author-detail-title">作者销售详情</h3>
            <div id="author-detail-info" style="margin-bottom:16px;font-size:13px;color:#6b7280;"></div>
            <table>
                <thead>
                    <tr><th>包名</th><th>分类</th><th>付费方式</th><th>单价</th><th>下载量</th><th>总收入</th><th>状态</th></tr>
                </thead>
                <tbody id="author-detail-packs"></tbody>
            </table>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="hideAuthorDetailModal()">关闭</button>
            </div>
        </div>
    </div>

    <!-- Customer Management Section -->
    <div id="section-customers" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>客户管理</h2>
                <button class="btn btn-secondary" onclick="loadCustomers()">↻ 刷新</button>
            </div>
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <input type="text" id="customer-search" placeholder="搜索邮箱/名称/SN..." style="width:260px;" onkeydown="if(event.key==='Enter')loadCustomers()" />
                <button class="btn btn-primary btn-sm" onclick="loadCustomers()">搜索</button>
                <select id="customer-sort" onchange="loadCustomers()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="created_at">按注册时间</option>
                    <option value="credits">按余额</option>
                    <option value="downloads">按下载量</option>
                    <option value="spent">按消费额</option>
                    <option value="name">按名称</option>
                </select>
                <select id="customer-order" onchange="loadCustomers()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="desc">降序</option>
                    <option value="asc">升序</option>
                </select>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>名称</th><th>邮箱</th><th>SN</th><th>余额</th><th>下载数</th><th>消费额</th><th>状态</th><th>注册时间</th><th>操作</th></tr>
                </thead>
                <tbody id="customer-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Customer Topup Modal -->
    <div id="topup-modal" class="modal-overlay">
        <div class="modal">
            <h3>充值 Credits</h3>
            <input type="hidden" id="topup-user-id" value="" />
            <div id="topup-user-info" style="margin-bottom:16px;font-size:13px;color:#6b7280;"></div>
            <div class="form-group">
                <label for="topup-amount">充值数量</label>
                <input type="number" id="topup-amount" min="1" step="1" placeholder="输入充值 Credits 数量" />
            </div>
            <div class="form-group">
                <label for="topup-reason">备注（可选）</label>
                <input type="text" id="topup-reason" placeholder="充值原因" />
            </div>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="hideTopupModal()">取消</button>
                <button class="btn btn-primary" onclick="submitTopup()">确认充值</button>
            </div>
        </div>
    </div>

    <!-- Customer Transactions Modal -->
    <div id="customer-tx-modal" class="modal-overlay">
        <div class="modal" style="width:640px;">
            <h3 id="customer-tx-title">交易记录</h3>
            <table>
                <thead>
                    <tr><th>ID</th><th>类型</th><th>金额</th><th>描述</th><th>时间</th></tr>
                </thead>
                <tbody id="customer-tx-list"></tbody>
            </table>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="hideCustomerTxModal()">关闭</button>
            </div>
        </div>
    </div>

    <!-- Admin Management Section (id=1 only) -->
    <div id="section-admins" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>管理员管理</h2>
                <button class="btn btn-primary" onclick="showAddAdminModal()">+ 添加管理员</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>用户名</th><th>权限</th><th>创建时间</th></tr>
                </thead>
                <tbody id="admin-list"></tbody>
            </table>
        </div>
    </div>

    <!-- Notifications Management Section -->
    <div id="section-notifications" style="display:none;">
        <div class="card">
            <div class="card-header">
                <h2>消息管理</h2>
                <button class="btn btn-primary" onclick="showCreateNotification()">+ 发送消息</button>
            </div>
            <table>
                <thead>
                    <tr><th>ID</th><th>标题</th><th>类型</th><th>状态</th><th>生效日期</th><th>时长</th><th>创建时间</th><th>操作</th></tr>
                </thead>
                <tbody id="notifications-tbody"></tbody>
            </table>
        </div>
    </div>

    <!-- Withdrawals Management Section -->
    <div id="section-withdrawals" style="display:none;">
        <!-- Tab Navigation -->
        <div class="wd-tabs">
            <button class="wd-tab active" onclick="switchWdTab('wd-tab-settings', this)">⚙️ 提现设置</button>
            <button class="wd-tab" onclick="switchWdTab('wd-tab-records', this)">📋 提现记录</button>
        </div>

        <!-- Tab: 提现设置 -->
        <div id="wd-tab-settings" class="wd-tab-content">
            <div class="card">
                <h2>Credit 提现价格</h2>
                <p class="form-hint" style="margin-bottom:16px;">每个 Credit 兑换的现金金额（单位：元），设为 0 表示提现功能未启用</p>
                <form id="cash-rate-form" onsubmit="saveCreditCashRate(event)">
                    <div class="form-group">
                        <label for="credit-cash-rate">提现价格（元/Credit）</label>
                        <input type="number" id="credit-cash-rate" min="0" step="0.01" value="{{.CreditCashRate}}" />
                    </div>
                    <button type="submit" class="btn btn-primary">保存设置</button>
                </form>
            </div>
            <div class="card">
                <h2>收入分成比例设置</h2>
                <p class="form-hint" style="margin-bottom:16px;">设置发布者（作者）获得的收入比例，平台获得剩余部分。默认 70 表示发布者获得 70%，平台获得 30%</p>
                <form id="revenue-split-form" onsubmit="saveRevenueSplit(event)">
                    <div class="form-group">
                        <label for="revenue-split-publisher-pct">发布者分成比例（%）</label>
                        <div style="display:flex;align-items:center;gap:12px;">
                            <input type="number" id="revenue-split-publisher-pct" min="0" max="100" step="1" value="{{.RevenueSplitPublisherPct}}" style="flex:1;" oninput="updateSplitPreview()" />
                            <span id="split-preview" style="font-size:13px;color:#6366f1;font-weight:600;white-space:nowrap;">发布者 {{.RevenueSplitPublisherPct}}% : 平台 {{.RevenueSplitPlatformPct}}%</span>
                        </div>
                    </div>
                    <button type="submit" class="btn btn-primary">保存分成设置</button>
                </form>
            </div>
            <div class="card">
                <h2>提现手续费率设置</h2>
                <p class="form-hint" style="margin-bottom:16px;">为每种收款方式设置提现手续费率（百分比），例如输入 3 表示 3%</p>
                <form id="withdrawal-fees-form" onsubmit="saveWithdrawalFees(event)">
                    <div class="form-group">
                        <label for="fee-rate-paypal">PayPal 手续费率（%）</label>
                        <input type="number" id="fee-rate-paypal" min="0" step="0.01" value="{{.FeeRatePaypal}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-wechat">微信 手续费率（%）</label>
                        <input type="number" id="fee-rate-wechat" min="0" step="0.01" value="{{.FeeRateWechat}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-alipay">AliPay 手续费率（%）</label>
                        <input type="number" id="fee-rate-alipay" min="0" step="0.01" value="{{.FeeRateAlipay}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-check">支票 手续费率（%）</label>
                        <input type="number" id="fee-rate-check" min="0" step="0.01" value="{{.FeeRateCheck}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-wire-transfer">国际电汇 手续费率（%）</label>
                        <input type="number" id="fee-rate-wire-transfer" min="0" step="0.01" value="{{.FeeRateWireTransfer}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-bank-card-us">美国银行卡 手续费率（%）</label>
                        <input type="number" id="fee-rate-bank-card-us" min="0" step="0.01" value="{{.FeeRateBankCardUS}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-bank-card-eu">欧洲银行卡 手续费率（%）</label>
                        <input type="number" id="fee-rate-bank-card-eu" min="0" step="0.01" value="{{.FeeRateBankCardEU}}" />
                    </div>
                    <div class="form-group">
                        <label for="fee-rate-bank-card-cn">中国银行卡 手续费率（%）</label>
                        <input type="number" id="fee-rate-bank-card-cn" min="0" step="0.01" value="{{.FeeRateBankCardCN}}" />
                    </div>
                    <button type="submit" class="btn btn-primary">保存手续费设置</button>
                </form>
            </div>
        </div>

        <!-- Tab: 提现记录 -->
        <div id="wd-tab-records" class="wd-tab-content" style="display:none;">
            <div class="card">
                <div class="card-header">
                    <h2>提现管理</h2>
                    <div style="display:flex;gap:8px;">
                        <button class="btn btn-secondary" onclick="exportWithdrawals()">📥 导出 Excel</button>
                        <button class="btn btn-primary" onclick="exportAndApproveWithdrawals()">📥 导出并标记已付款</button>
                        <button class="btn btn-primary" id="btn-batch-approve" onclick="batchApproveWithdrawals()" style="display:none;">批量标记已付款</button>
                        <button class="btn btn-secondary" onclick="loadWithdrawals()">↻ 刷新</button>
                    </div>
                </div>
                <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                    <select id="wd-status-filter" onchange="loadWithdrawals()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                        <option value="">全部</option>
                        <option value="pending">已申请提现</option>
                        <option value="paid">已付款</option>
                    </select>
                    <input type="text" id="wd-author-filter" placeholder="按作者名过滤" oninput="loadWithdrawals()" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;width:160px;" />
                </div>
                <table>
                    <thead>
                        <tr>
                            <th><input type="checkbox" id="wd-select-all" onchange="toggleSelectAllWithdrawals()" /></th>
                            <th>作者</th>
                            <th>收款方式</th>
                            <th>收款详情</th>
                            <th>提现金额</th>
                            <th>手续费率</th>
                            <th>手续费</th>
                            <th>实付金额</th>
                            <th>状态</th>
                            <th>时间</th>
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
                <h2>销售管理</h2>
                <div style="display:flex;gap:8px;">
                    <button class="btn btn-secondary" onclick="loadSalesData(1)">↻ 刷新</button>
                    <button class="btn btn-primary" onclick="exportSalesExcel()">📥 导出 Excel</button>
                </div>
            </div>
            <!-- Filters -->
            <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center;">
                <select id="sales-category-filter" onchange="loadSalesData(1)" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="">全部分类</option>
                </select>
                <select id="sales-author-filter" onchange="loadSalesData(1)" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;">
                    <option value="">全部作者</option>
                </select>
                <input type="date" id="sales-date-from" onchange="loadSalesData(1)" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;" />
                <input type="date" id="sales-date-to" onchange="loadSalesData(1)" style="padding:7px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:13px;" />
                <button class="btn btn-secondary btn-sm" onclick="clearSalesFilters()">清除筛选</button>
            </div>
            <!-- Summary Stats -->
            <div id="sales-summary" style="display:flex;gap:16px;margin-bottom:20px;flex-wrap:wrap;">
                <div style="background:#f0f9ff;border:1px solid #bae6fd;border-radius:8px;padding:16px 24px;flex:1;min-width:180px;">
                    <div style="font-size:12px;color:#0369a1;font-weight:600;">订单总数</div>
                    <div id="sales-total-orders" style="font-size:24px;font-weight:700;color:#0c4a6e;margin-top:4px;">0</div>
                </div>
                <div style="background:#f0fdf4;border:1px solid #bbf7d0;border-radius:8px;padding:16px 24px;flex:1;min-width:180px;">
                    <div style="font-size:12px;color:#15803d;font-weight:600;">总销售额 (Credits)</div>
                    <div id="sales-total-credits" style="font-size:24px;font-weight:700;color:#14532d;margin-top:4px;">0</div>
                </div>
                <div style="background:#fefce8;border:1px solid #fde68a;border-radius:8px;padding:16px 24px;flex:1;min-width:180px;">
                    <div style="font-size:12px;color:#a16207;font-weight:600;">涉及用户数</div>
                    <div id="sales-total-users" style="font-size:24px;font-weight:700;color:#713f12;margin-top:4px;">0</div>
                </div>
                <div style="background:#fdf4ff;border:1px solid #f0abfc;border-radius:8px;padding:16px 24px;flex:1;min-width:180px;">
                    <div style="font-size:12px;color:#86198f;font-weight:600;">涉及作者数</div>
                    <div id="sales-total-authors" style="font-size:24px;font-weight:700;color:#4a044e;margin-top:4px;">0</div>
                </div>
            </div>
            <!-- Orders Table -->
            <table>
                <thead>
                    <tr><th>订单ID</th><th>买家</th><th>买家邮箱</th><th>分析包</th><th>分类</th><th>作者</th><th>金额(Credits)</th><th>类型</th><th>买家IP</th><th>时间</th></tr>
                </thead>
                <tbody id="sales-order-list"></tbody>
            </table>
            <!-- Pagination -->
            <div id="sales-pagination" style="display:flex;justify-content:space-between;align-items:center;margin-top:16px;padding-top:16px;border-top:1px solid #e5e7eb;">
                <div id="sales-page-info" style="font-size:13px;color:#6b7280;"></div>
                <div style="display:flex;gap:6px;align-items:center;">
                    <button class="btn btn-secondary btn-sm" id="sales-prev-btn" onclick="salesGoPage(salesCurrentPage-1)" disabled>‹ 上一页</button>
                    <span id="sales-page-nums" style="display:flex;gap:4px;"></span>
                    <button class="btn btn-secondary btn-sm" id="sales-next-btn" onclick="salesGoPage(salesCurrentPage+1)" disabled>下一页 ›</button>
                </div>
            </div>
        </div>
    </div>

    <!-- Profile Section (all admins) -->
    <div id="section-profile" style="display:none;">
        <div class="profile-grid">
            <div class="profile-card">
                <h3><span class="icon-header"><span>👤</span> 修改资料</span></h3>
                <div class="form-group">
                    <label for="profile-username">用户名</label>
                    <input type="text" id="profile-username" placeholder="新用户名（留空不修改）" />
                    <div class="form-hint">修改后需要重新登录</div>
                </div>
            </div>
            <div class="profile-card">
                <h3><span class="icon-header"><span>🔒</span> 修改密码</span></h3>
                <div class="form-group">
                    <label for="profile-old-password">当前密码</label>
                    <input type="password" id="profile-old-password" placeholder="输入当前密码" />
                </div>
                <div class="form-group">
                    <label for="profile-new-password">新密码</label>
                    <input type="password" id="profile-new-password" placeholder="输入新密码" />
                    <div class="form-hint">至少 6 个字符</div>
                </div>
            </div>
        </div>
        <div style="margin-top: 20px; display: flex; justify-content: flex-end;">
            <button class="btn btn-primary" onclick="saveProfile()" style="padding: 10px 28px; font-size: 14px;">保存修改</button>
        </div>
    </div>

    </main>
</div>

<!-- Reject Reason Modal -->
<div id="reject-modal" class="modal-overlay">
    <div class="modal">
        <h3>拒绝审核</h3>
        <input type="hidden" id="reject-pack-id" value="" />
        <div class="form-group">
            <label for="reject-reason">拒绝原因（必填）</label>
            <textarea id="reject-reason" placeholder="请输入拒绝原因"></textarea>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideRejectModal()">取消</button>
            <button class="btn btn-danger" onclick="submitReject()">确认拒绝</button>
        </div>
    </div>
</div>

<!-- Add Admin Modal -->
<div id="add-admin-modal" class="modal-overlay">
    <div class="modal">
        <h3>添加管理员</h3>
        <div class="form-group">
            <label for="new-admin-username">用户名（至少3个字符）</label>
            <input type="text" id="new-admin-username" placeholder="输入用户名" />
        </div>
        <div class="form-group">
            <label for="new-admin-password">密码（至少6个字符）</label>
            <input type="text" id="new-admin-password" placeholder="输入密码" />
        </div>
        <div class="form-group">
            <label>权限设置</label>
            <div style="display:flex;flex-wrap:wrap;gap:12px;margin-top:6px;">
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="categories" class="new-admin-perm" /> 分类管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="marketplace" class="new-admin-perm" /> 市场管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="authors" class="new-admin-perm" /> 作者管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="customers" class="new-admin-perm" /> 客户管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="review" class="new-admin-perm" /> 审核管理
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="settings" class="new-admin-perm" /> 系统设置
                </label>
                <label style="display:flex;align-items:center;gap:4px;font-size:13px;font-weight:400;cursor:pointer;">
                    <input type="checkbox" value="notifications" class="new-admin-perm" /> 消息管理
                </label>
            </div>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideAddAdminModal()">取消</button>
            <button class="btn btn-primary" onclick="submitAddAdmin()">添加</button>
        </div>
    </div>
</div>

<!-- Create/Edit Category Modal -->
<div id="category-modal" class="modal-overlay">
    <div class="modal">
        <h3 id="modal-title">新建分类</h3>
        <input type="hidden" id="edit-category-id" value="" />
        <div class="form-group">
            <label for="cat-name">分类名称</label>
            <input type="text" id="cat-name" placeholder="输入分类名称" />
        </div>
        <div class="form-group">
            <label for="cat-desc">描述（可选）</label>
            <textarea id="cat-desc" placeholder="输入分类描述"></textarea>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideModal()">取消</button>
            <button class="btn btn-primary" onclick="saveCategory()">保存</button>
        </div>
    </div>
</div>

<!-- Create Notification Modal -->
<div id="create-notif-modal" class="modal-overlay">
    <div class="modal" style="width:520px;">
        <h3>发送消息</h3>
        <div class="form-group">
            <label for="notif-title">消息标题</label>
            <input type="text" id="notif-title" placeholder="消息标题" />
        </div>
        <div class="form-group">
            <label for="notif-content">消息内容</label>
            <textarea id="notif-content" placeholder="消息内容" rows="4"></textarea>
        </div>
        <div class="form-group">
            <label for="notif-target-type">消息类型</label>
            <select id="notif-target-type" onchange="toggleTargetUsers()" style="width:100%;padding:9px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:14px;">
                <option value="broadcast">广播（所有用户）</option>
                <option value="targeted">定向（指定用户）</option>
            </select>
        </div>
        <div id="notif-target-section" style="display:none;">
            <div class="form-group">
                <label>搜索用户</label>
                <div style="display:flex;gap:8px;">
                    <input type="text" id="notif-user-search" placeholder="输入邮箱/名称搜索..." style="flex:1;" onkeydown="if(event.key==='Enter')searchNotifUsers()" />
                    <button class="btn btn-primary btn-sm" onclick="searchNotifUsers()">搜索</button>
                </div>
                <div id="notif-user-results" style="max-height:120px;overflow-y:auto;margin-top:8px;"></div>
            </div>
            <div class="form-group">
                <label>已选用户</label>
                <div id="notif-selected-users" style="min-height:32px;padding:8px;background:#f9fafb;border:1px solid #e5e7eb;border-radius:6px;font-size:13px;color:#6b7280;">未选择用户</div>
            </div>
        </div>
        <div class="form-group">
            <label for="notif-effective-date">生效日期（可选，留空立即生效）</label>
            <input type="datetime-local" id="notif-effective-date" style="width:100%;padding:9px 12px;border:1px solid #d1d5db;border-radius:6px;font-size:14px;" />
        </div>
        <div class="form-group">
            <label for="notif-duration">显示时长</label>
            <div style="display:flex;align-items:center;gap:8px;">
                <input type="number" id="notif-duration" value="0" min="0" style="width:120px;" />
                <span style="font-size:13px;color:#6b7280;">天 (0=永久)</span>
            </div>
        </div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="hideCreateNotification()">取消</button>
            <button class="btn btn-primary" onclick="createNotification()">发送</button>
        </div>
    </div>
</div>

<script>
// Permission system
var adminID = {{.AdminID}};
var permissions = {{.PermissionsJSON}};
var permLabels = { categories: '分类管理', marketplace: '市场管理', authors: '作者管理', customers: '客户管理', review: '审核管理', settings: '系统设置', notifications: '消息管理' };

function hasPerm(p) { return permissions.indexOf(p) !== -1; }
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
    var sections = ['categories', 'marketplace', 'authors', 'customers', 'settings', 'admins', 'review', 'profile', 'notifications', 'withdrawals', 'sales'];
    var titles = { categories: '分类管理', marketplace: '市场管理', authors: '作者管理', customers: '客户管理', settings: '系统设置', admins: '管理员管理', review: '审核管理', profile: '修改资料', notifications: '消息管理', withdrawals: '提现管理', sales: '销售管理' };
    for (var i = 0; i < sections.length; i++) {
        var el = document.getElementById('section-' + sections[i]);
        if (el) el.style.display = sections[i] === name ? '' : 'none';
    }
    var links = document.querySelectorAll('.sidebar-nav a');
    for (var i = 0; i < links.length; i++) {
        var href = links[i].getAttribute('href');
        if (href) links[i].className = href === '#' + name ? 'active' : '';
    }
    document.getElementById('topbar-title').textContent = titles[name] || '管理面板';
    if (name === 'categories') loadCategories();
    if (name === 'marketplace') loadMarketplacePacks();
    if (name === 'authors') loadAuthors();
    if (name === 'customers') loadCustomers();
    if (name === 'admins') loadAdmins();
    if (name === 'review') loadPendingPacks();
    if (name === 'notifications') loadNotifications();
    if (name === 'withdrawals') loadWithdrawals();
    if (name === 'sales') loadSalesData(1);
}

function showMsg(text, isError) {
    var area = document.getElementById('msg-area');
    area.innerHTML = '<div class="msg ' + (isError ? 'msg-error' : 'msg-success') + '">' + text + '</div>';
    setTimeout(function() { area.innerHTML = ''; }, 4000);
}

function apiFetch(url, opts) {
    return fetch(url, opts).then(function(r) {
        if (r.status === 401) {
            showMsg('会话已过期，正在跳转到登录页...', true);
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
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:#999;">暂无分类</td></tr>';
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
            html += '<td>' + (c.is_preset ? '<span class="badge badge-preset">预设</span>' : '自定义') + '</td>';
            html += '<td class="actions">';
            html += '<button class="btn btn-primary" onclick="showEditCategory(' + c.id + ',\'' + escAttr(c.name) + '\',\'' + escAttr(c.description || '') + '\')">编辑</button> ';
            if (!c.is_preset) {
                html += '<button class="btn btn-danger" onclick="deleteCategory(' + c.id + ',\'' + escAttr(c.name) + '\',' + c.pack_count + ')">删除</button>';
            }
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载分类失败: ' + err, true); });
}

function showCreateCategory() {
    document.getElementById('modal-title').textContent = '新建分类';
    document.getElementById('edit-category-id').value = '';
    document.getElementById('cat-name').value = '';
    document.getElementById('cat-desc').value = '';
    document.getElementById('category-modal').className = 'modal-overlay show';
}

function showEditCategory(id, name, desc) {
    document.getElementById('modal-title').textContent = '编辑分类';
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
    if (!name) { alert('请输入分类名称'); return; }

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
            showMsg(id ? '分类已更新' : '分类已创建', false);
            loadCategories();
        } else {
            showMsg(res.data.error || '操作失败', true);
        }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function deleteCategory(id, name, packCount) {
    if (packCount > 0) {
        alert('分类 "' + name + '" 下有 ' + packCount + ' 个分析包，请先迁移后再删除。');
        return;
    }
    if (!confirm('确定要删除分类 "' + name + '" 吗？')) return;
    apiFetch('/api/admin/categories/' + id, { method: 'DELETE' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('分类已删除', false); loadCategories(); }
            else { showMsg(res.data.error || '删除失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Settings ---
function saveInitialCredits(e) {
    e.preventDefault();
    var val = document.getElementById('initial-credits').value;
    apiFetch('/admin/settings/initial-credits', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: 'value=' + encodeURIComponent(val)
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { showMsg('初始余额已更新为 ' + val, false); }
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
        if (res.ok) { showMsg('Credit 提现价格已更新为 ' + val + ' 元/Credit', false); }
        else { showMsg(res.data.error || '保存失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
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
    preview.innerHTML = '发布者 ' + pub + '% : 平台 ' + plat + '%';
}
// Initialize preview on page load
(function(){ var el = document.getElementById('split-preview'); if(el){ updateSplitPreview(); } })();

function saveRevenueSplit(e) {
    e.preventDefault();
    var val = document.getElementById('revenue-split-publisher-pct').value;
    var pct = parseFloat(val);
    if (isNaN(pct) || pct < 0 || pct > 100) { showMsg('分成比例必须在 0-100 之间', true); return; }
    apiFetch('/admin/api/settings/revenue-split', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({publisher_pct: pct})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok || res.data.ok) { showMsg('收入分成比例已保存：发布者 ' + pct + '% / 平台 ' + (100 - pct) + '%', false); }
        else { showMsg(res.data.error || '保存失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
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
        if (res.ok || res.data.ok) { showMsg('提现手续费率已保存', false); }
        else { showMsg(res.data.error || '保存失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
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
            tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;color:#999;">暂无管理员</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < admins.length; i++) {
            var a = admins[i];
            var permDisplay;
            if (a.id === 1) {
                permDisplay = '<span class="badge badge-preset">超级管理员（全部权限）</span>';
            } else {
                var perms = a.permissions || [];
                if (perms.length === 0) {
                    permDisplay = '<span style="color:#999;">无权限</span>';
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
    }).catch(function(err) { showMsg('加载管理员列表失败: ' + err, true); });
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
    if (username.length < 3) { alert('用户名至少3个字符'); return; }
    if (password.length < 6) { alert('密码至少6个字符'); return; }
    var permCheckboxes = document.querySelectorAll('.new-admin-perm:checked');
    var perms = [];
    for (var i = 0; i < permCheckboxes.length; i++) { perms.push(permCheckboxes[i].value); }
    apiFetch('/api/admin/admins', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({username: username, password: password, permissions: perms})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { hideAddAdminModal(); showMsg('管理员已添加', false); loadAdmins(); }
        else { showMsg(res.data.error || '添加失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Review Management ---
function loadPendingPacks() {
    apiFetch('/api/admin/review/pending').then(function(r) { return r.json(); }).then(function(data) {
        var packs = data || [];
        var tbody = document.getElementById('pending-list');
        if (packs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#999;">暂无待审核分析包</td></tr>';
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
            html += '<td>' + (p.share_mode === 'free' ? '免费' : p.credits_price + ' Credits') + '</td>';
            html += '<td>' + p.created_at + '</td>';
            html += '<td class="actions">';
            html += '<button class="btn btn-primary" onclick="approvePack(' + p.id + ')">通过</button> ';
            html += '<button class="btn btn-danger" onclick="showRejectModal(' + p.id + ')">拒绝</button>';
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载待审核列表失败: ' + err, true); });
}

function approvePack(id) {
    if (!confirm('确定通过审核？')) return;
    apiFetch('/api/admin/review/' + id + '/approve', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('审核已通过', false); loadPendingPacks(); }
            else { showMsg(res.data.error || '操作失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
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
    if (!reason) { alert('请输入拒绝原因'); return; }
    apiFetch('/api/admin/review/' + id + '/reject', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({reason: reason})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) { hideRejectModal(); showMsg('已拒绝', false); loadPendingPacks(); }
        else { showMsg(res.data.error || '操作失败', true); }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
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
    var labels = { free: '免费', per_use: '按次', subscription: '订阅' };
    return labels[mode] || mode;
}

function loadMarketplacePacks() {
    loadMarketplaceCategoryFilter();
    var status = document.getElementById('mp-status-filter').value;
    var catId = document.getElementById('mp-category-filter').value;
    var mode = document.getElementById('mp-mode-filter').value;
    var sort = document.getElementById('mp-sort').value;
    var order = document.getElementById('mp-order').value;
    document.querySelector('#section-marketplace .card-header h2').textContent = status === 'delisted' ? '市场管理 - 已下架分析包' : '市场管理 - 在售分析包';
    var url = '/api/admin/marketplace?status=' + status + '&sort=' + sort + '&order=' + order;
    if (catId) url += '&category_id=' + catId;
    if (mode) url += '&share_mode=' + mode;
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var packs = data.packs || [];
        var tbody = document.getElementById('marketplace-list');
        if (packs.length === 0) {
            var emptyMsg = status === 'delisted' ? '暂无已下架分析包' : '暂无在售分析包';
            tbody.innerHTML = '<tr><td colspan="9" style="text-align:center;color:#999;">' + emptyMsg + '</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < packs.length; i++) {
            var p = packs[i];
            var priceText = p.share_mode === 'free' ? '免费' : p.credits_price + ' Credits';
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
                html += '<td><button class="btn btn-primary btn-sm" onclick="relistPack(' + p.id + ',\'' + escAttr(p.pack_name) + '\')">恢复在售</button></td>';
            } else {
                html += '<td><button class="btn btn-danger btn-sm" onclick="delistPack(' + p.id + ',\'' + escAttr(p.pack_name) + '\')">下架</button></td>';
            }
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载市场列表失败: ' + err, true); });
}

function delistPack(id, name) {
    if (!confirm('确定要下架 "' + name + '" 吗？（下架后不删除，可在数据库中恢复）')) return;
    apiFetch('/api/admin/marketplace/' + id + '/delist', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('已下架', false); loadMarketplacePacks(); }
            else { showMsg(res.data.error || '下架失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function relistPack(id, name) {
    if (!confirm('确定要恢复 "' + name + '" 为在售状态吗？')) return;
    apiFetch('/api/admin/marketplace/' + id + '/relist', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('已恢复在售', false); loadMarketplacePacks(); }
            else { showMsg(res.data.error || '恢复失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Author Management ---
function loadAuthors() {
    var email = document.getElementById('author-email-search').value.trim();
    var sort = document.getElementById('author-sort').value;
    var order = document.getElementById('author-order').value;
    var url = '/api/admin/authors?sort=' + sort + '&order=' + order;
    if (email) url += '&email=' + encodeURIComponent(email);
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var authors = data.authors || [];
        var tbody = document.getElementById('author-list');
        if (authors.length === 0) {
            tbody.innerHTML = '<tr><td colspan="11" style="text-align:center;color:#999;">暂无作者数据</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < authors.length; i++) {
            var a = authors[i];
            html += '<tr>';
            html += '<td>' + a.user_id + '</td>';
            html += '<td>' + escHtml(a.display_name) + '</td>';
            html += '<td>' + escHtml(a.email || '-') + '</td>';
            html += '<td>' + a.total_packs + '</td>';
            html += '<td>' + a.total_downloads + '</td>';
            html += '<td>' + a.total_revenue + '</td>';
            html += '<td>' + a.year_downloads + '</td>';
            html += '<td>' + a.year_revenue + '</td>';
            html += '<td>' + a.month_downloads + '</td>';
            html += '<td>' + a.month_revenue + '</td>';
            html += '<td><button class="btn btn-primary btn-sm" onclick="showAuthorDetail(' + a.user_id + ')">详情</button></td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载作者列表失败: ' + err, true); });
}

function showAuthorDetail(userId) {
    apiFetch('/api/admin/authors/' + userId).then(function(r) { return r.json(); }).then(function(data) {
        document.getElementById('author-detail-title').textContent = escHtml(data.display_name) + ' 的销售详情';
        document.getElementById('author-detail-info').textContent = '邮箱: ' + (data.email || '-');
        var packs = data.packs || [];
        var tbody = document.getElementById('author-detail-packs');
        if (packs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#999;">暂无分析包</td></tr>';
        } else {
            var html = '';
            for (var i = 0; i < packs.length; i++) {
                var p = packs[i];
                var statusLabel = p.status === 'published' ? '在售' : '已下架';
                html += '<tr>';
                html += '<td>' + escHtml(p.pack_name) + '</td>';
                html += '<td>' + escHtml(p.category_name) + '</td>';
                html += '<td>' + shareModeLabel(p.share_mode) + '</td>';
                html += '<td>' + (p.share_mode === 'free' ? '免费' : p.credits_price + ' Credits') + '</td>';
                html += '<td>' + p.download_count + '</td>';
                html += '<td>' + p.total_revenue + '</td>';
                html += '<td>' + statusLabel + '</td>';
                html += '</tr>';
            }
            tbody.innerHTML = html;
        }
        document.getElementById('author-detail-modal').className = 'modal-overlay show';
    }).catch(function(err) { showMsg('加载作者详情失败: ' + err, true); });
}

function hideAuthorDetailModal() {
    document.getElementById('author-detail-modal').className = 'modal-overlay';
}

// --- Customer Management ---
function loadCustomers() {
    var search = document.getElementById('customer-search').value.trim();
    var sort = document.getElementById('customer-sort').value;
    var order = document.getElementById('customer-order').value;
    var url = '/api/admin/customers?sort=' + sort + '&order=' + order;
    if (search) url += '&search=' + encodeURIComponent(search);
    apiFetch(url).then(function(r) { return r.json(); }).then(function(data) {
        var customers = data.customers || [];
        var tbody = document.getElementById('customer-list');
        if (customers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="10" style="text-align:center;color:#999;">暂无客户数据</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < customers.length; i++) {
            var c = customers[i];
            var statusBadge = c.is_blocked
                ? '<span class="badge" style="background:#fef2f2;color:#991b1b;">已禁用</span>'
                : '<span class="badge" style="background:#ecfdf5;color:#065f46;">正常</span>';
            var blockBtn = c.is_blocked
                ? '<button class="btn btn-primary btn-sm" onclick="toggleBlock(' + c.id + ',\'' + escAttr(c.display_name) + '\',true)">解禁</button>'
                : '<button class="btn btn-danger btn-sm" onclick="toggleBlock(' + c.id + ',\'' + escAttr(c.display_name) + '\',false)">禁用</button>';
            html += '<tr>';
            html += '<td>' + c.id + '</td>';
            html += '<td>' + escHtml(c.display_name) + '</td>';
            html += '<td>' + escHtml(c.email || '-') + '</td>';
            html += '<td>' + escHtml(c.auth_id || '-') + '</td>';
            html += '<td>' + c.credits_balance.toFixed(0) + '</td>';
            html += '<td>' + c.download_count + '</td>';
            html += '<td>' + c.total_spent.toFixed(0) + '</td>';
            html += '<td>' + statusBadge + '</td>';
            html += '<td>' + c.created_at + '</td>';
            html += '<td class="actions" style="white-space:nowrap;">';
            html += '<button class="btn btn-primary btn-sm" onclick="showTopupModal(' + c.id + ',\'' + escAttr(c.display_name) + '\',' + c.credits_balance + ')">充值</button> ';
            html += '<button class="btn btn-secondary btn-sm" onclick="showCustomerTx(' + c.id + ',\'' + escAttr(c.display_name) + '\')">记录</button> ';
            html += blockBtn;
            html += '</td></tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载客户列表失败: ' + err, true); });
}

function showTopupModal(userId, name, balance) {
    document.getElementById('topup-user-id').value = userId;
    document.getElementById('topup-user-info').textContent = '客户: ' + name + '  |  当前余额: ' + balance.toFixed(0) + ' Credits';
    document.getElementById('topup-amount').value = '';
    document.getElementById('topup-reason').value = '';
    document.getElementById('topup-modal').className = 'modal-overlay show';
}

function hideTopupModal() {
    document.getElementById('topup-modal').className = 'modal-overlay';
}

function submitTopup() {
    var userId = document.getElementById('topup-user-id').value;
    var amount = parseFloat(document.getElementById('topup-amount').value);
    var reason = document.getElementById('topup-reason').value.trim();
    if (!amount || amount <= 0) { alert('请输入有效的充值数量'); return; }
    apiFetch('/api/admin/customers/' + userId + '/topup', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({amount: amount, reason: reason})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok) {
            hideTopupModal();
            showMsg('充值成功，新余额: ' + res.data.new_balance, false);
            loadCustomers();
        } else {
            showMsg(res.data.error || '充值失败', true);
        }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function toggleBlock(userId, name, isCurrentlyBlocked) {
    var action = isCurrentlyBlocked ? '解禁' : '禁用';
    if (!confirm('确定要' + action + '客户 "' + name + '" 吗？')) return;
    apiFetch('/api/admin/customers/' + userId + '/toggle-block', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) {
                showMsg('已' + (res.data.status === 'blocked' ? '禁用' : '解禁'), false);
                loadCustomers();
            } else {
                showMsg(res.data.error || '操作失败', true);
            }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function showCustomerTx(userId, name) {
    document.getElementById('customer-tx-title').textContent = escHtml(name) + ' 的交易记录';
    var tbody = document.getElementById('customer-tx-list');
    tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#999;">加载中...</td></tr>';
    document.getElementById('customer-tx-modal').className = 'modal-overlay show';
    apiFetch('/api/admin/customers/' + userId + '/transactions').then(function(r) { return r.json(); }).then(function(data) {
        var txns = data.transactions || [];
        if (txns.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#999;">暂无交易记录</td></tr>';
            return;
        }
        var typeLabels = { download: '下载扣费', admin_topup: '管理员充值', initial: '注册赠送', purchase: '购买' };
        var html = '';
        for (var i = 0; i < txns.length; i++) {
            var t = txns[i];
            var amountStyle = t.amount >= 0 ? 'color:#065f46;' : 'color:#991b1b;';
            var amountText = t.amount >= 0 ? '+' + t.amount : '' + t.amount;
            html += '<tr>';
            html += '<td>' + t.id + '</td>';
            html += '<td>' + (typeLabels[t.transaction_type] || t.transaction_type) + '</td>';
            html += '<td style="' + amountStyle + 'font-weight:600;">' + amountText + '</td>';
            html += '<td>' + escHtml(t.description || '-') + '</td>';
            html += '<td>' + t.created_at + '</td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#991b1b;">加载失败</td></tr>';
    });
}

function hideCustomerTxModal() {
    document.getElementById('customer-tx-modal').className = 'modal-overlay';
}

// --- Profile ---
function saveProfile() {
    var username = document.getElementById('profile-username').value.trim();
    var oldPassword = document.getElementById('profile-old-password').value;
    var newPassword = document.getElementById('profile-new-password').value;
    if (!username && !newPassword) { alert('请输入要修改的内容'); return; }
    if (newPassword && !oldPassword) { alert('修改密码需要输入当前密码'); return; }
    if (newPassword && newPassword.length < 6) { alert('新密码至少6个字符'); return; }
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
            showMsg('资料已更新', false);
            document.getElementById('profile-username').value = '';
            document.getElementById('profile-old-password').value = '';
            document.getElementById('profile-new-password').value = '';
        } else {
            var errMsg = res.data.error;
            if (errMsg === 'invalid_old_password') errMsg = '当前密码错误';
            else if (errMsg === 'username_already_exists') errMsg = '用户名已被使用';
            showMsg(errMsg || '修改失败', true);
        }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
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
            container.innerHTML = '<div style="padding:6px;color:#999;font-size:12px;">未找到用户</div>';
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
            html += '<button class="btn btn-primary btn-sm" style="padding:2px 8px;font-size:11px;" onclick="addNotifUser(' + c.id + ',\'' + escAttr(c.display_name) + '\',\'' + escAttr(c.email || '') + '\')">添加</button>';
            html += '</div>';
        }
        container.innerHTML = html || '<div style="padding:6px;color:#999;font-size:12px;">所有搜索结果已添加</div>';
    }).catch(function(err) { showMsg('搜索用户失败: ' + err, true); });
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
        container.innerHTML = '未选择用户';
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
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#999;">暂无消息</td></tr>';
            return;
        }
        var html = '';
        for (var i = 0; i < notifs.length; i++) {
            var n = notifs[i];
            var typeText = n.target_type === 'broadcast' ? '广播' : '定向(' + (n.target_count || 0) + '人)';
            var statusBadge = n.status === 'active'
                ? '<span class="badge" style="background:#ecfdf5;color:#065f46;">活跃</span>'
                : '<span class="badge" style="background:#f3f4f6;color:#6b7280;">已禁用</span>';
            var durationText = n.display_duration_days === 0 ? '永久' : n.display_duration_days + '天';
            var toggleBtn = n.status === 'active'
                ? '<button class="btn btn-secondary btn-sm" onclick="disableNotification(' + n.id + ')">禁用</button>'
                : '<button class="btn btn-primary btn-sm" onclick="enableNotification(' + n.id + ')">启用</button>';
            html += '<tr>';
            html += '<td>' + n.id + '</td>';
            html += '<td>' + escHtml(n.title) + '</td>';
            html += '<td>' + typeText + '</td>';
            html += '<td>' + statusBadge + '</td>';
            html += '<td>' + escHtml(n.effective_date || '-') + '</td>';
            html += '<td>' + durationText + '</td>';
            html += '<td>' + escHtml(n.created_at || '-') + '</td>';
            html += '<td class="actions">' + toggleBtn + ' <button class="btn btn-danger btn-sm" onclick="deleteNotification(' + n.id + ')">删除</button></td>';
            html += '</tr>';
        }
        tbody.innerHTML = html;
    }).catch(function(err) { showMsg('加载消息列表失败: ' + err, true); });
}

function createNotification() {
    var title = document.getElementById('notif-title').value.trim();
    var content = document.getElementById('notif-content').value.trim();
    var targetType = document.getElementById('notif-target-type').value;
    var effectiveDate = document.getElementById('notif-effective-date').value;
    var duration = parseInt(document.getElementById('notif-duration').value) || 0;
    if (!title || !content) { alert('请输入消息标题和内容'); return; }
    if (targetType === 'targeted' && notifSelectedUsers.length === 0) { alert('请选择目标用户'); return; }
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
            showMsg('消息已发送', false);
            loadNotifications();
        } else {
            alert(res.data.error || '发送失败');
        }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function disableNotification(id) {
    apiFetch('/api/admin/notifications/' + id + '/disable', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('消息已禁用', false); loadNotifications(); }
            else { showMsg(res.data.error || '操作失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function enableNotification(id) {
    apiFetch('/api/admin/notifications/' + id + '/enable', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('消息已启用', false); loadNotifications(); }
            else { showMsg(res.data.error || '操作失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

function deleteNotification(id) {
    if (!confirm('确定要删除该消息吗？')) return;
    apiFetch('/api/admin/notifications/' + id + '/delete', { method: 'POST' })
        .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
        .then(function(res) {
            if (res.ok) { showMsg('消息已删除', false); loadNotifications(); }
            else { showMsg(res.data.error || '删除失败', true); }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
}

// --- Withdrawal Management ---
function paymentTypeLabel(t) {
    var labels = { paypal: 'PayPal', wechat: '微信', alipay: 'AliPay', check: '支票', wire_transfer: '国际电汇', bank_card_us: '美国银行卡', bank_card_eu: '欧洲银行卡', bank_card_cn: '中国银行卡' };
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
    if (ids.length === 0) { alert('请先选择要导出的提现记录'); return; }
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
    if (allIds.length === 0) { alert('请先选择要导出的提现记录'); return; }
    var msg = '将导出 ' + allIds.length + ' 条记录';
    if (pendingIds.length > 0) msg += '，并将其中 ' + pendingIds.length + ' 条待审核记录标记为已付款';
    if (!confirm(msg + '，确定继续？')) return;
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
                showMsg('已导出并标记 ' + (res.data.updated || pendingIds.length) + ' 条记录为已付款', false);
                loadWithdrawals();
            } else {
                showMsg(res.data.error || '标记付款失败', true);
            }
        }).catch(function(err) { showMsg('请求失败: ' + err, true); });
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
            tbody.innerHTML = '<tr><td colspan="10" style="text-align:center;color:#999;">暂无提现记录</td></tr>';
            document.getElementById('btn-batch-approve').style.display = 'none';
            return;
        }
        var hasPending = false;
        var html = '';
        for (var i = 0; i < list.length; i++) {
            var w = list[i];
            var statusBadge = w.status === 'pending'
                ? '<span class="badge" style="background:#fef3c7;color:#92400e;">已申请提现</span>'
                : '<span class="badge" style="background:#ecfdf5;color:#065f46;">已付款</span>';
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
    }).catch(function(err) { showMsg('加载提现列表失败: ' + err, true); });
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
    if (ids.length === 0) { alert('请选择待审核的提现记录'); return; }
    if (!confirm('确定将选中的 ' + ids.length + ' 条提现记录标记为已付款？')) return;
    apiFetch('/admin/api/withdrawals/approve', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({ids: ids})
    }).then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok || res.data.ok) {
            showMsg('已标记 ' + (res.data.updated || ids.length) + ' 条记录为已付款', false);
            loadWithdrawals();
        } else {
            showMsg(res.data.error || '操作失败', true);
        }
    }).catch(function(err) { showMsg('请求失败: ' + err, true); });
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
            tbody.innerHTML = '<tr><td colspan="10" style="text-align:center;color:#999;">暂无订单数据</td></tr>';
            renderSalesPagination(0);
            return;
        }
        var html = '';
        var typeLabels = {purchase:'购买',purchase_uses:'购买次数',renew:'续订',download:'免费领取'};
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
    }).catch(function(e) { if (e.message !== 'session_expired') showMsg('加载销售数据失败', true); });
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
    pageInfo.textContent = '显示 ' + start + '-' + end + ' 条，共 ' + totalOrders + ' 条';
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
            sel.innerHTML = '<option value="">全部分类</option>';
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
            sel.innerHTML = '<option value="">全部作者</option>';
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

// Init: show first available section based on permissions
(function initDefaultSection() {
    var order = ['categories', 'marketplace', 'authors', 'customers', 'review', 'settings', 'notifications', 'sales'];
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
</body>
</html>`
