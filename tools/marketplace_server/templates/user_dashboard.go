package templates

import "html/template"

// UserDashboardTmpl is the parsed user dashboard page template.
var UserDashboardTmpl = template.Must(template.New("user_dashboard").Parse(userDashboardHTML))

const userDashboardHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>个人中心 - 快捷分析包市场</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", sans-serif;
            background: #f8f9fc;
            min-height: 100vh;
            color: #2d3748;
            line-height: 1.6;
        }
        .dashboard-wrap {
            max-width: 980px;
            margin: 0 auto;
            padding: 40px 24px;
        }

        /* Header */
        .header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 36px;
        }
        .header-title {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .header-title .logo {
            width: 36px; height: 36px;
            background: #eef2ff;
            border-radius: 10px;
            display: flex; align-items: center; justify-content: center;
            font-size: 18px;
            border: 1px solid #e0e7ff;
        }
        .header-title h1 {
            font-size: 21px;
            font-weight: 600;
            color: #334155;
            letter-spacing: -0.2px;
        }

        /* User info card */
        .user-info {
            background: #fff;
            border-radius: 16px;
            padding: 28px 32px;
            margin-bottom: 32px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 6px 24px rgba(0,0,0,0.03);
            border: 1px solid rgba(0,0,0,0.04);
            flex-wrap: wrap;
            gap: 20px;
        }
        .user-detail {
            display: flex;
            align-items: center;
            gap: 28px;
            flex-wrap: wrap;
        }
        .user-avatar {
            width: 48px; height: 48px;
            background: #f0f4ff;
            border-radius: 14px;
            display: flex; align-items: center; justify-content: center;
            font-size: 20px;
            border: 1px solid #e0e7ff;
        }
        .user-email {
            font-size: 14px;
            color: #4a5568;
        }
        .user-email .label {
            font-size: 11px;
            color: #a0aec0;
            display: block;
            margin-bottom: 2px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            font-weight: 600;
        }
        .credits-info {
            font-size: 14px;
            color: #4a5568;
        }
        .credits-info .label {
            font-size: 11px;
            color: #a0aec0;
            display: block;
            margin-bottom: 2px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            font-weight: 600;
        }
        .credits-info .balance {
            color: #818cf8;
            font-weight: 700;
            font-size: 22px;
        }
        .user-actions {
            display: flex;
            gap: 8px;
            align-items: center;
            flex-wrap: wrap;
        }
        /* Buttons */
        .btn {
            padding: 7px 16px;
            border: 1px solid transparent;
            border-radius: 8px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.2s ease;
            display: inline-flex;
            align-items: center;
            gap: 5px;
            letter-spacing: 0.1px;
        }
        .btn-primary {
            background: #e0e7ff;
            color: #4338ca;
            border-color: #c7d2fe;
        }
        .btn-primary:hover { background: #c7d2fe; border-color: #a5b4fc; }
        .btn-secondary {
            background: #f1f5f9;
            color: #475569;
            border-color: #cbd5e1;
        }
        .btn-secondary:hover { background: #e2e8f0; border-color: #94a3b8; color: #334155; }
        .btn-accent {
            background: #d1fae5;
            color: #047857;
            border-color: #a7f3d0;
        }
        .btn-accent:hover { background: #a7f3d0; border-color: #6ee7b7; color: #065f46; }
        .btn-warm {
            background: #fef3c7;
            color: #b45309;
            border-color: #fde68a;
        }
        .btn-warm:hover { background: #fde68a; border-color: #fcd34d; color: #92400e; }
        .btn-danger-outline {
            background: #fee2e2;
            color: #dc2626;
            border: 1px solid #fca5a5;
        }
        .btn-danger-outline:hover { background: #fecaca; border-color: #f87171; color: #b91c1c; }
        .btn-ghost {
            background: #f3e8ff;
            color: #7c3aed;
            border: 1px solid #e9d5ff;
        }
        .btn-ghost:hover { background: #e9d5ff; border-color: #d8b4fe; color: #6d28d9; }
        .btn-sm { padding: 5px 12px; font-size: 12px; border-radius: 7px; }
        .btn-danger-sm {
            padding: 5px 12px;
            font-size: 12px;
            border-radius: 7px;
            background: #fef2f2;
            color: #fca5a5;
            border: 1px solid #fee2e2;
            cursor: pointer;
            transition: all 0.2s ease;
        }
        .btn-danger-sm:hover { background: #fee2e2; border-color: #fecaca; color: #f87171; }

        /* Section */
        .section {
            margin-bottom: 36px;
        }
        .section-title {
            font-size: 15px;
            font-weight: 600;
            color: #475569;
            margin-bottom: 16px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .section-title .icon {
            font-size: 16px;
        }
        /* Pack cards grid */
        .pack-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 16px;
        }
        .pack-card {
            background: #fff;
            border-radius: 14px;
            padding: 22px;
            border: 1px solid rgba(0,0,0,0.04);
            box-shadow: 0 1px 3px rgba(0,0,0,0.03);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .pack-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(0,0,0,0.06);
        }
        .pack-card .pack-name {
            font-size: 15px;
            font-weight: 600;
            color: #334155;
            margin-bottom: 6px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .pack-card .pack-category {
            font-size: 12px;
            color: #a0aec0;
            margin-bottom: 12px;
        }
        .pack-card .pack-meta {
            display: flex;
            align-items: center;
            gap: 8px;
            flex-wrap: wrap;
            margin-bottom: 10px;
        }
        .tag {
            display: inline-block;
            padding: 3px 10px;
            border-radius: 6px;
            font-size: 11px;
            font-weight: 600;
            letter-spacing: 0.2px;
        }
        .tag-free { background: #f0fdf4; color: #4ade80; }
        .tag-per-use { background: #eef2ff; color: #818cf8; }
        .tag-time-limited { background: #fffbeb; color: #fbbf24; }
        .tag-subscription { background: #faf5ff; color: #c084fc; }
        .usage-progress { font-size: 12px; color: #4a5568; }
        .usage-exhausted { color: #ef4444; font-weight: 600; }
        .pack-card .pack-date {
            font-size: 12px;
            color: #a0aec0;
        }
        .pack-card .pack-expires {
            font-size: 12px;
            color: #718096;
            margin-top: 4px;
        }
        .pack-card .pack-expires.subscription-expires {
            color: #c084fc;
            font-weight: 500;
        }
        .pack-actions {
            display: flex;
            gap: 8px;
            margin-top: 14px;
            padding-top: 12px;
            border-top: 1px solid #f7fafc;
        }

        /* Empty state */
        .empty-state {
            text-align: center;
            padding: 56px 20px;
            color: #a0aec0;
            background: #fff;
            border-radius: 14px;
            border: 1px dashed #e2e8f0;
        }
        .empty-state .icon { font-size: 40px; margin-bottom: 12px; opacity: 0.6; }
        .empty-state p { font-size: 14px; }
        /* Author panel */
        .author-panel {
            margin-top: 8px;
            padding-top: 32px;
            border-top: 1px solid #edf2f7;
        }
        .author-panel-title {
            font-size: 17px;
            font-weight: 600;
            color: #334155;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .author-stats {
            display: flex;
            gap: 16px;
            margin-bottom: 28px;
            flex-wrap: wrap;
        }
        .stat-card {
            background: #fff;
            border-radius: 14px;
            padding: 22px 26px;
            border: 1px solid rgba(0,0,0,0.04);
            box-shadow: 0 1px 3px rgba(0,0,0,0.03);
            flex: 1;
            min-width: 200px;
        }
        .stat-card .stat-label {
            font-size: 11px;
            color: #a0aec0;
            margin-bottom: 8px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            font-weight: 600;
        }
        .stat-card .stat-value {
            font-size: 28px;
            font-weight: 700;
            color: #334155;
        }
        .stat-card .stat-value.revenue { color: #6ee7b7; }
        .stat-card .stat-value.unwithdrawn { color: #fcd34d; }
        .stat-actions {
            display: flex;
            gap: 10px;
            margin-top: 14px;
            align-items: center;
        }
        .withdraw-hint {
            font-size: 12px;
            color: #a0aec0;
        }

        /* Author pack table */
        .author-table-wrap {
            background: #fff;
            border-radius: 14px;
            border: 1px solid rgba(0,0,0,0.04);
            box-shadow: 0 1px 3px rgba(0,0,0,0.03);
            overflow-x: auto;
        }
        .author-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 13px;
        }
        .author-table th {
            background: #fafbfe;
            padding: 14px 16px;
            text-align: left;
            font-weight: 600;
            color: #94a3b8;
            border-bottom: 1px solid #f1f5f9;
            white-space: nowrap;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.3px;
        }
        .author-table td {
            padding: 14px 16px;
            border-bottom: 1px solid #f7fafc;
            color: #4a5568;
        }
        .author-table tr:last-child td { border-bottom: none; }
        .author-table tr:hover td { background: #fafbfe; }
        .status-badge {
            display: inline-block;
            padding: 3px 10px;
            border-radius: 6px;
            font-size: 11px;
            font-weight: 600;
        }
        .status-pending { background: #fff7ed; color: #fb923c; }
        .status-published { background: #f0fdf4; color: #4ade80; }
        .status-rejected { background: #fef2f2; color: #fca5a5; }
        .status-delisted { background: #f8fafc; color: #94a3b8; }
        .td-actions {
            display: flex;
            gap: 6px;
            align-items: center;
        }
        /* Notification cards */
        .notification-section {
            margin-bottom: 32px;
        }
        .notification-card {
            background: #fff;
            border-left: 3px solid #c7d2fe;
            border-radius: 10px;
            padding: 16px 20px;
            margin-bottom: 10px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.03);
        }
        .notification-card .notif-title {
            font-size: 14px;
            font-weight: 600;
            color: #334155;
            margin-bottom: 4px;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .notification-card .notif-content {
            font-size: 13px;
            color: #4a5568;
            line-height: 1.7;
        }

        /* Modal overlay */
        .modal-overlay {
            display: none;
            position: fixed;
            top: 0; left: 0;
            width: 100%; height: 100%;
            background: rgba(0,0,0,0.3);
            backdrop-filter: blur(4px);
            z-index: 1000;
            align-items: center;
            justify-content: center;
        }
        .modal-box {
            background: #fff;
            border-radius: 18px;
            padding: 32px;
            max-width: 480px;
            width: 90%;
            box-shadow: 0 20px 60px rgba(0,0,0,0.12);
            position: relative;
            max-height: 90vh;
            overflow-y: auto;
        }
        .modal-close {
            position: absolute;
            top: 16px; right: 20px;
            background: none;
            border: none;
            font-size: 20px;
            cursor: pointer;
            color: #a0aec0;
            width: 32px; height: 32px;
            border-radius: 8px;
            display: flex; align-items: center; justify-content: center;
            transition: background 0.2s;
        }
        .modal-close:hover { background: #f1f5f9; color: #64748b; }
        .modal-title {
            font-size: 17px;
            font-weight: 600;
            color: #334155;
            margin-bottom: 20px;
        }
        .modal-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
            margin-top: 20px;
        }

        /* Form fields */
        .field-group { margin-bottom: 14px; }
        .field-group label {
            font-size: 12px;
            color: #4a5568;
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
        }
        .field-group input, .field-group select, .field-group textarea {
            width: 100%;
            padding: 9px 14px;
            border: 1px solid #e2e8f0;
            border-radius: 10px;
            font-size: 14px;
            background: #fff;
            transition: border-color 0.2s, box-shadow 0.2s;
            color: #2d3748;
        }
        .field-group input:focus, .field-group select:focus, .field-group textarea:focus {
            outline: none;
            border-color: #a5b4fc;
            box-shadow: 0 0 0 3px rgba(165,180,252,0.15);
        }
        .field-group input.field-error { border-color: #ef4444; }
        .field-error-msg { font-size: 12px; color: #ef4444; margin-top: 3px; display: none; }
        .field-hint { font-size: 11px; color: #a0aec0; margin-top: 3px; }
        .msg-box {
            display: none;
            padding: 10px 14px;
            border-radius: 10px;
            font-size: 13px;
            margin-bottom: 14px;
        }
        .msg-success { background: #ecfdf5; color: #059669; border: 1px solid #bbf7d0; }
        .msg-error { background: #fef2f2; color: #dc2626; border: 1px solid #fecaca; }

        /* Tab navigation */
        .tab-nav {
            display: flex;
            gap: 0;
            margin-bottom: 28px;
            border-bottom: 2px solid #e2e8f0;
        }
        .tab-btn {
            padding: 12px 28px;
            font-size: 14px;
            font-weight: 600;
            color: #94a3b8;
            background: none;
            border: none;
            border-bottom: 2px solid transparent;
            margin-bottom: -2px;
            cursor: pointer;
            transition: all 0.2s ease;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .tab-btn:hover { color: #64748b; }
        .tab-btn.active {
            color: #4338ca;
            border-bottom-color: #4338ca;
        }
        .tab-panel { display: none; }
        .tab-panel.active { display: block; }
    </style>
</head>
<body>
<div class="dashboard-wrap">
    {{if eq .SuccessMsg "withdraw"}}
    <div class="msg-box msg-success" style="margin-bottom:16px;">✅ 提现申请已提交，请等待管理员审核付款。</div>
    {{end}}
    {{if eq .ErrorMsg "no_payment_info"}}
    <div class="msg-box msg-error" style="margin-bottom:16px;">⚠️ 请先设置收款信息后再进行提现操作。</div>
    {{else if eq .ErrorMsg "not_author"}}
    <div class="msg-box msg-error" style="margin-bottom:16px;">⚠️ 仅作者可以申请提现。</div>
    {{else if eq .ErrorMsg "invalid_withdraw_amount"}}
    <div class="msg-box msg-error" style="margin-bottom:16px;">⚠️ 提现金额无效，请输入正确的数量。</div>
    {{else if eq .ErrorMsg "withdraw_disabled"}}
    <div class="msg-box msg-error" style="margin-bottom:16px;">⚠️ 提现功能暂未开放。</div>
    {{else if eq .ErrorMsg "withdraw_exceeds_balance"}}
    <div class="msg-box msg-error" style="margin-bottom:16px;">⚠️ 提现数量超过可提现余额。</div>
    {{else if eq .ErrorMsg "withdraw_below_minimum"}}
    <div class="msg-box msg-error" style="margin-bottom:16px;">⚠️ 扣除手续费后实付金额低于最低提现金额 100 元。</div>
    {{else if eq .ErrorMsg "internal"}}
    <div class="msg-box msg-error" style="margin-bottom:16px;">⚠️ 系统错误，请稍后重试。</div>
    {{end}}
    <div class="header">
        <div class="header-title">
            <span class="logo">📦</span>
            <h1>个人中心</h1>
        </div>
    </div>
    <div class="user-info">
        <div class="user-detail">
            <div class="user-avatar">👤</div>
            <div class="user-email">
                <span class="label">邮箱</span>
                {{.User.Email}}
            </div>
            <div class="credits-info">
                <span class="label">Credits 余额</span>
                <span class="balance">{{printf "%.0f" .User.CreditsBalance}}</span>
            </div>
        </div>
        <div class="user-actions">
            {{if .HasPassword}}
            <a class="btn btn-accent" href="/user/change-password">修改密码</a>
            {{else}}
            <a class="btn btn-accent" href="/user/set-password">设置密码</a>
            {{end}}
            <a class="btn btn-primary" href="/user/billing">帐单记录</a>
            <button class="btn btn-warm" onclick="openPaymentSettingsModal()">收款设置</button>
            <button class="btn btn-secondary" onclick="alert('功能开发中')">充值</button>
            <a class="btn btn-danger-outline" href="/user/logout">退出登录</a>
        </div>
    </div>

    {{if .Notifications}}
    <div class="notification-section">
        <div class="section-title"><span class="icon">📢</span> 系统消息</div>
        {{range .Notifications}}
        <div class="notification-card">
            <div class="notif-title">📌 {{.Title}}</div>
            <div class="notif-content">{{.Content}}</div>
        </div>
        {{end}}
    </div>
    {{end}}

    <div class="tab-nav">
        <button class="tab-btn active" onclick="switchTab('customer')" id="tabBtnCustomer">🛒 客户视图</button>
        {{if .AuthorData.IsAuthor}}
        <button class="tab-btn" onclick="switchTab('author')" id="tabBtnAuthor">✍️ 作者视图</button>
        {{end}}
    </div>

    <div id="tabCustomer" class="tab-panel active">
    <div class="section">
        <div class="section-title"><span class="icon">🛒</span> 已购买的分析包</div>
        {{if .PurchasedPacks}}
        <div class="pack-grid">
            {{range .PurchasedPacks}}
            <div class="pack-card">
                <div class="pack-name">
                    <span>{{.PackName}}</span>
                    {{if eq .ShareMode "free"}}<span class="tag tag-free">免费</span>
                    {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use">按次付费</span>
                    {{else if eq .ShareMode "time_limited"}}<span class="tag tag-time-limited">限时</span>
                    {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription">订阅</span>
                    {{end}}
                </div>
                <div class="pack-category">{{.CategoryName}}</div>
                <div class="pack-meta">
                    {{if eq .ShareMode "per_use"}}
                    <span class="usage-progress{{if eq .UsedCount .TotalPurchased}} usage-exhausted{{end}}">已使用 {{.UsedCount}}/{{.TotalPurchased}} 次</span>
                    {{end}}
                </div>
                <div class="pack-date">{{if eq .ShareMode "subscription"}}订阅起始：{{else}}下载时间：{{end}}{{.PurchaseDate}}</div>
                {{if .ExpiresAt}}<div class="pack-expires{{if eq .ShareMode "subscription"}} subscription-expires{{end}}">{{if eq .ShareMode "subscription"}}订阅到期：{{else}}到期时间：{{end}}{{.ExpiresAt}}</div>{{end}}
                <div class="pack-actions">
                    {{if or (eq .ShareMode "per_use") (eq .ShareMode "subscription")}}
                    <button class="btn btn-primary btn-sm"
                        data-listing-id="{{.ListingID}}"
                        data-pack-name="{{.PackName}}"
                        data-share-mode="{{.ShareMode}}"
                        data-credits-price="{{.CreditsPrice}}"
                        onclick="openRenewModal(this)">续费</button>
                    {{end}}
                    <button class="btn-danger-sm"
                        data-listing-id="{{.ListingID}}"
                        data-pack-name="{{.PackName}}"
                        onclick="openDeleteModal(this)">删除</button>
                </div>
            </div>
            {{end}}
        </div>
        {{else}}
        <div class="empty-state">
            <div class="icon">📭</div>
            <p>暂无已购买的分析包</p>
        </div>
        {{end}}
    </div>
    </div><!-- end tabCustomer -->

    {{if .AuthorData.IsAuthor}}
    <div id="tabAuthor" class="tab-panel">
    <div class="author-panel" style="border-top:none;margin-top:0;padding-top:0;">
        <div class="author-panel-title">✍️ 作者面板</div>

        <div class="author-stats">
            <div class="stat-card">
                <div class="stat-label">实际收入 Credits（分成 {{printf "%.0f" .AuthorData.RevenueSplitPct}}%）</div>
                <div class="stat-value revenue">{{printf "%.0f" .AuthorData.TotalRevenue}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">未提现 Credits</div>
                <div class="stat-value unwithdrawn">{{printf "%.0f" .AuthorData.UnwithdrawnCredits}}</div>
                <div class="stat-actions">
                    {{if .AuthorData.WithdrawalEnabled}}
                    <button class="btn btn-warm" onclick="openWithdrawModal()">提现</button>
                    {{else}}
                    <button class="btn btn-secondary" disabled title="提现功能暂未开放">提现</button>
                    <span class="withdraw-hint">提现功能暂未开放</span>
                    {{end}}
                    <a class="btn btn-ghost" href="javascript:void(0)" onclick="openWithdrawRecordsModal()">提现记录</a>
                </div>
            </div>
        </div>

        <div class="section-title"><span class="icon">📤</span> 已共享分析包</div>
        {{if .AuthorData.AuthorPacks}}
        <div class="author-table-wrap">
            <table class="author-table">
                <thead>
                    <tr>
                        <th>名称</th>
                        <th>定价模式</th>
                        <th>单价</th>
                        <th>审核状态</th>
                        <th>销量</th>
                        <th>实际收入</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .AuthorData.AuthorPacks}}
                    <tr>
                        <td style="font-weight:500;color:#475569;">{{.PackName}}</td>
                        <td>
                            {{if eq .ShareMode "free"}}免费
                            {{else if eq .ShareMode "per_use"}}按次付费
                            {{else if eq .ShareMode "subscription"}}订阅
                            {{else}}{{.ShareMode}}
                            {{end}}
                        </td>
                        <td>{{if eq .ShareMode "free"}}-{{else}}{{.CreditsPrice}} Credits{{end}}</td>
                        <td>
                            {{if eq .Status "pending"}}<span class="status-badge status-pending">待审核</span>
                            {{else if eq .Status "published"}}<span class="status-badge status-published">已发布</span>
                            {{else if eq .Status "rejected"}}<span class="status-badge status-rejected">已拒绝</span>
                            {{else if eq .Status "delisted"}}<span class="status-badge status-delisted">已下架</span>
                            {{else}}<span class="status-badge">{{.Status}}</span>
                            {{end}}
                        </td>
                        <td>{{.SoldCount}}</td>
                        <td>{{printf "%.0f" .TotalRevenue}} Credits</td>
                        <td>
                            <div class="td-actions">
                                <button class="btn btn-primary btn-sm"
                                    data-listing-id="{{.ListingID}}"
                                    data-pack-name="{{.PackName}}"
                                    onclick="openPurchaseDetailsModal(this)">明细</button>
                                <button class="btn btn-ghost btn-sm"
                                    data-listing-id="{{.ListingID}}"
                                    data-pack-name="{{.PackName}}"
                                    data-pack-desc="{{.PackDesc}}"
                                    data-share-mode="{{.ShareMode}}"
                                    data-credits-price="{{.CreditsPrice}}"
                                    onclick="openEditPackModal(this)">编辑</button>
                                {{if eq .Status "published"}}
                                <button class="btn-danger-sm"
                                    data-listing-id="{{.ListingID}}"
                                    data-pack-name="{{.PackName}}"
                                    onclick="openAuthorDelistModal(this)">下架</button>
                                {{end}}
                                {{if eq .Status "rejected"}}
                                <button class="btn-danger-sm"
                                    data-listing-id="{{.ListingID}}"
                                    data-pack-name="{{.PackName}}"
                                    onclick="openAuthorDeleteModal(this)">删除</button>
                                {{end}}
                            </div>
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{else}}
        <div class="empty-state">
            <div class="icon">📝</div>
            <p>暂无已共享的分析包</p>
        </div>
        {{end}}
    </div>
    </div><!-- end tabAuthor -->
    {{end}}
</div>

<!-- Payment Settings Modal -->
<div id="paymentSettingsModal" class="modal-overlay">
  <div class="modal-box">
    <button onclick="closePaymentSettingsModal()" class="modal-close">&times;</button>
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:4px;">
      <div style="display:flex;align-items:center;gap:10px;">
        <h3 class="modal-title" style="margin-bottom:0;">收款设置</h3>
        <span style="font-size:13px;color:#6366f1;font-weight:600;background:#eef2ff;padding:2px 10px;border-radius:8px;white-space:nowrap;">分成 {{printf "%.0f" .AuthorData.RevenueSplitPct}}%</span>
      </div>
      <button class="btn btn-sm" style="font-size:12px;padding:4px 12px;background:#f0f9ff;color:#0369a1;border:1px solid #bae6fd;border-radius:6px;cursor:pointer;" onclick="openFeeRatesDialog()">查看费率</button>
    </div>
    <div id="paymentSettingsMsg" class="msg-box"></div>
    <div class="field-group">
      <label>收款方式</label>
      <select id="paymentType" onchange="onPaymentTypeChange()">
        <option value="">请选择收款方式</option>
        <option value="paypal">PayPal</option>
        <option value="wechat">微信</option>
        <option value="alipay">AliPay</option>
        <option value="check">支票</option>
        <option value="wire_transfer">国际电汇 (SWIFT)</option>
        <option value="bank_card_us">美国银行卡 (ACH)</option>
        <option value="bank_card_eu">欧洲银行卡 (SEPA)</option>
        <option value="bank_card_cn">中国银行卡 (CNAPS)</option>
      </select>
    </div>
    <div id="paymentFieldsAccount" style="display:none;">
      <div class="field-group">
        <label>帐号</label>
        <input type="text" id="paymentAccount" placeholder="请输入帐号">
        <div class="field-error-msg" id="paymentAccountError">帐号不能为空</div>
      </div>
      <div class="field-group">
        <label>用户名</label>
        <input type="text" id="paymentUsername" placeholder="请输入用户名">
        <div class="field-error-msg" id="paymentUsernameError">用户名不能为空</div>
      </div>
    </div>

    <div id="paymentFieldsCheck" style="display:none;">
      <div class="field-group">
        <label>法定全名</label>
        <input type="text" id="paymentCheckFullLegalName" placeholder="请输入法定全名">
        <div class="field-hint">必须与银行账户名一致，避免缩写</div>
        <div class="field-error-msg" id="paymentCheckFullLegalNameError">法定全名不能为空</div>
      </div>
      <div class="field-group">
        <label>省份</label>
        <input type="text" id="paymentCheckProvince" placeholder="请输入省份">
        <div class="field-error-msg" id="paymentCheckProvinceError">省份不能为空</div>
      </div>
      <div class="field-group">
        <label>城市</label>
        <input type="text" id="paymentCheckCity" placeholder="请输入城市">
        <div class="field-error-msg" id="paymentCheckCityError">城市不能为空</div>
      </div>
      <div class="field-group">
        <label>区县</label>
        <input type="text" id="paymentCheckDistrict" placeholder="请输入区县">
        <div class="field-error-msg" id="paymentCheckDistrictError">区县不能为空</div>
      </div>
      <div class="field-group">
        <label>街道地址</label>
        <input type="text" id="paymentCheckStreetAddress" placeholder="请输入街道地址">
        <div class="field-error-msg" id="paymentCheckStreetAddressError">街道地址不能为空</div>
      </div>
      <div class="field-group">
        <label>邮政编码</label>
        <input type="text" id="paymentCheckPostalCode" placeholder="请输入邮政编码">
        <div class="field-error-msg" id="paymentCheckPostalCodeError">邮政编码不能为空</div>
      </div>
      <div class="field-group">
        <label>收件人电话</label>
        <input type="text" id="paymentCheckPhone" placeholder="请输入收件人电话">
        <div class="field-error-msg" id="paymentCheckPhoneError">收件人电话不能为空</div>
      </div>
      <div class="field-group">
        <label>备注（可选）</label>
        <input type="text" id="paymentCheckMemo" placeholder="如支付房租、还款等用途">
      </div>
    </div>
    <div id="paymentFieldsWireTransfer" style="display:none;">
      <div class="field-group">
        <label>收款人全名 (Full Name)</label>
        <input type="text" id="paymentBeneficiaryName" placeholder="必须与银行开户证件一致">
        <div class="field-error-msg" id="paymentBeneficiaryNameError">收款人全名不能为空</div>
      </div>
      <div class="field-group">
        <label>收款人地址 (Full Address)</label>
        <input type="text" id="paymentBeneficiaryAddress" placeholder="街道、城市、邮编、国家">
        <div class="field-error-msg" id="paymentBeneficiaryAddressError">收款人地址不能为空</div>
      </div>
      <div class="field-group">
        <label>银行名称 (Bank Name)</label>
        <input type="text" id="paymentWireBankName" placeholder="英文全称">
        <div class="field-error-msg" id="paymentWireBankNameError">银行名称不能为空</div>
      </div>
      <div class="field-group">
        <label>SWIFT / BIC Code</label>
        <input type="text" id="paymentSwiftCode" placeholder="8或11位国际银行识别码">
        <div class="field-error-msg" id="paymentSwiftCodeError">SWIFT Code不能为空</div>
      </div>
      <div class="field-group">
        <label>收款人账号 / IBAN</label>
        <input type="text" id="paymentWireAccountNumber" placeholder="账号或IBAN码">
        <div class="field-error-msg" id="paymentWireAccountNumberError">账号不能为空</div>
      </div>
      <div class="field-group">
        <label>银行分行地址（选填）</label>
        <input type="text" id="paymentBankBranchAddress" placeholder="城市名和具体分行">
      </div>
    </div>
    <div id="paymentFieldsBankUS" style="display:none;">
      <div class="field-group">
        <label>收款人姓名 (Legal Name)</label>
        <input type="text" id="paymentUSLegalName" placeholder="请输入Legal Name">
        <div class="field-error-msg" id="paymentUSLegalNameError">姓名不能为空</div>
      </div>
      <div class="field-group">
        <label>路由号码 (Routing Number)</label>
        <input type="text" id="paymentRoutingNumber" placeholder="9位数字">
        <div class="field-error-msg" id="paymentRoutingNumberError">路由号码不能为空</div>
      </div>
      <div class="field-group">
        <label>账号 (Account Number)</label>
        <input type="text" id="paymentUSAccountNumber" placeholder="请输入账号">
        <div class="field-error-msg" id="paymentUSAccountNumberError">账号不能为空</div>
      </div>
      <div class="field-group">
        <label>账户类型</label>
        <select id="paymentUSAccountType">
          <option value="checking">Checking (支票账户)</option>
          <option value="savings">Savings (储蓄账户)</option>
        </select>
        <div class="field-error-msg" id="paymentUSAccountTypeError">请选择账户类型</div>
      </div>
    </div>
    <div id="paymentFieldsBankEU" style="display:none;">
      <div class="field-group">
        <label>收款人姓名 (Legal Name)</label>
        <input type="text" id="paymentEULegalName" placeholder="请输入Legal Name">
        <div class="field-error-msg" id="paymentEULegalNameError">姓名不能为空</div>
      </div>
      <div class="field-group">
        <label>IBAN</label>
        <input type="text" id="paymentIBAN" placeholder="以国家代码开头（如 DE..., FR...）">
        <div class="field-error-msg" id="paymentIBANError">IBAN不能为空</div>
      </div>
      <div class="field-group">
        <label>BIC / SWIFT</label>
        <input type="text" id="paymentEUBicSwift" placeholder="银行识别码">
        <div class="field-error-msg" id="paymentEUBicSwiftError">BIC/SWIFT不能为空</div>
      </div>
    </div>
    <div id="paymentFieldsBankCN" style="display:none;">
      <div class="field-group">
        <label>收款人姓名（中文实名）</label>
        <input type="text" id="paymentCNRealName" placeholder="必须为中文实名">
        <div class="field-error-msg" id="paymentCNRealNameError">姓名不能为空</div>
      </div>
      <div class="field-group">
        <label>收款卡号</label>
        <input type="text" id="paymentCNCardNumber" placeholder="16-19位银行卡号">
        <div class="field-error-msg" id="paymentCNCardNumberError">卡号不能为空</div>
      </div>
      <div class="field-group">
        <label>开户银行（具体到分行）</label>
        <input type="text" id="paymentCNBankBranch" placeholder="如：中国银行北京分行XX支行">
        <div class="field-error-msg" id="paymentCNBankBranchError">开户银行不能为空</div>
      </div>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closePaymentSettingsModal()">取消</button>
      <button class="btn btn-warm" onclick="savePaymentSettings()">保存</button>
    </div>
  </div>
</div>

<!-- Fee Rates Dialog -->
<div id="feeRatesDialog" class="modal-overlay" style="display:none;z-index:1100;">
  <div class="modal-box" style="max-width:400px;">
    <button onclick="closeFeeRatesDialog()" class="modal-close">&times;</button>
    <h3 class="modal-title">提现费率</h3>
    <div id="feeRatesCurrentType" style="font-size:13px;color:#64748b;margin-bottom:10px;"></div>
    <div id="feeRatesDialogContent" style="font-size:13px;color:#475569;">加载中...</div>
    <div class="modal-actions" style="margin-top:12px;">
      <button class="btn btn-secondary" onclick="closeFeeRatesDialog()">关闭</button>
    </div>
  </div>
</div>

<!-- Withdraw Modal -->
<div id="withdrawModal" class="modal-overlay">
  <div class="modal-box" style="max-width:400px;padding:24px;">
    <button onclick="closeWithdrawModal()" class="modal-close">&times;</button>
    <h3 style="font-size:16px;font-weight:600;color:#334155;margin-bottom:14px;">Credits 提现</h3>
    <div id="withdrawNoPaymentWarning" style="display:none;padding:10px 14px;background:#fff7ed;border:1px solid #fed7aa;border-radius:8px;margin-bottom:12px;">
      <div style="font-size:13px;color:#ea580c;font-weight:600;margin-bottom:2px;">⚠️ 未设置收款信息</div>
      <div style="font-size:12px;color:#9a3412;">请先设置收款方式后再进行提现操作。</div>
      <button class="btn btn-warm btn-sm" style="margin-top:6px;font-size:11px;" onclick="closeWithdrawModal();openPaymentSettingsModal();">去设置</button>
    </div>
    <div id="withdrawFormContent">
      <div id="withdrawPaymentInfo" style="display:none;padding:8px 12px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:8px;margin-bottom:10px;font-size:12px;color:#475569;display:flex;justify-content:space-between;flex-wrap:wrap;gap:4px;">
        <span>分成 <strong id="withdrawSplitPctLabel" style="color:#4338ca;">{{printf "%.0f" .AuthorData.RevenueSplitPct}}%</strong></span>
        <span>收款 <strong id="withdrawPaymentTypeLabel" style="color:#166534;"></strong></span>
        <span>费率 <strong id="withdrawFeeRateLabel" style="color:#ea580c;"></strong></span>
      </div>
      <div style="display:flex;gap:12px;font-size:12px;color:#718096;margin-bottom:10px;">
        <span>可提现：<span style="color:#f59e0b;font-weight:600;">{{printf "%.0f" .AuthorData.UnwithdrawnCredits}}</span> Credits</span>
        <span>汇率：1C = <span id="withdrawCashRate" style="font-weight:500;">{{printf "%.2f" .AuthorData.CreditCashRate}}</span>元</span>
      </div>
      <div style="margin-bottom:10px;">
        <label style="font-size:12px;color:#4a5568;display:block;margin-bottom:4px;font-weight:500;">提现数量</label>
        <input id="withdrawCreditsInput" type="number" min="1" max="{{printf "%.0f" .AuthorData.UnwithdrawnCredits}}" step="1" placeholder="输入 Credits 数量" oninput="calcWithdrawCash()" style="width:100%;padding:8px 12px;border:1px solid #e2e8f0;border-radius:8px;font-size:13px;">
      </div>
      <div id="withdrawFormulaBox" style="display:none;padding:10px 12px;background:#fafbfe;border:1px solid #eef2ff;border-radius:8px;margin-bottom:10px;font-size:12px;font-family:monospace;color:#475569;line-height:1.8;"></div>
      <div id="withdrawNetResult" style="display:none;font-size:15px;font-weight:700;color:#10b981;margin-bottom:6px;"></div>
      <div id="withdrawWarning" style="display:none;padding:6px 10px;background:#fff7ed;border:1px solid #fed7aa;border-radius:8px;margin-bottom:8px;font-size:12px;color:#9a3412;"></div>
      <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:12px;">
        <button class="btn btn-secondary" onclick="closeWithdrawModal()" style="padding:6px 14px;font-size:13px;">取消</button>
        <button class="btn btn-warm" id="withdrawSubmitBtn" onclick="submitWithdraw()" style="padding:6px 14px;font-size:13px;">确认提现</button>
      </div>
    </div>
  </div>
</div>
<form id="withdrawForm" method="POST" action="/user/author/withdraw" style="display:none;">
  <input type="hidden" name="credits_amount" id="withdrawFormCredits">
</form>

<!-- Withdrawal Records Modal -->
<div id="withdrawRecordsModal" class="modal-overlay">
  <div class="modal-box" style="max-width:700px;padding:24px;">
    <button onclick="closeWithdrawRecordsModal()" class="modal-close">&times;</button>
    <h3 style="font-size:16px;font-weight:700;color:#1e293b;margin-bottom:16px;">💰 提现记录</h3>
    <div id="withdrawRecordsContent" style="max-height:400px;overflow-y:auto;">
      <div style="text-align:center;padding:30px;color:#94a3b8;">加载中...</div>
    </div>
    <div id="withdrawRecordsTotalRow" style="display:none;text-align:right;padding:12px 0 0;border-top:2px solid #e2e8f0;margin-top:12px;font-size:14px;font-weight:600;color:#1e293b;">
      总计提现现金：<span id="withdrawRecordsTotalCash" style="color:#059669;font-size:16px;"></span>
    </div>
  </div>
</div>

<!-- Edit Pack Modal -->
<div id="editPackModal" class="modal-overlay">
  <div class="modal-box">
    <button onclick="closeEditPackModal()" class="modal-close">&times;</button>
    <h3 class="modal-title">编辑分析包</h3>
    <form id="editPackForm" method="POST" action="/user/author/edit-pack">
      <input type="hidden" name="listing_id" id="editListingId">
      <div class="field-group">
        <label>名称</label>
        <input type="text" name="pack_name" id="editPackName" required>
      </div>
      <div class="field-group">
        <label>描述</label>
        <textarea name="pack_description" id="editPackDesc" rows="3" style="resize:vertical;"></textarea>
      </div>
      <div class="field-group">
        <label>定价模式</label>
        <select name="share_mode" id="editShareMode" onchange="onEditShareModeChange()">
          <option value="free">免费</option>
          <option value="per_use">按次付费</option>
          <option value="subscription">订阅</option>
        </select>
      </div>
      <div id="editPriceSection" style="display:none;">
        <div class="field-group">
          <label>价格 (Credits)</label>
          <input type="number" name="credits_price" id="editCreditsPrice" min="0">
          <div class="field-hint" id="editPriceHint"></div>
        </div>
      </div>
      <div style="margin-top:12px;padding:10px 12px;background:#fffbeb;border:1px solid #fde68a;border-radius:6px;">
        <p style="font-size:12px;color:#92400e;margin:0;">⚠ 修改已上架的分析包信息后，该分析包将被下架并需要重新提交审核后才能再次上架。</p>
      </div>
      <div class="modal-actions">
        <button type="button" class="btn btn-secondary" onclick="closeEditPackModal()">取消</button>
        <button type="button" class="btn btn-primary" onclick="confirmEditPack()">确认修改</button>
      </div>
    </form>
  </div>
</div>

<!-- Renew Modal -->
<div id="renewModal" class="modal-overlay">
  <div class="modal-box" style="max-width:420px;">
    <button onclick="closeRenewModal()" class="modal-close">&times;</button>
    <h3 id="renewTitle" class="modal-title">续费</h3>
    <div id="renewPackName" style="font-size:14px;color:#4a5568;margin-bottom:12px;"></div>
    <div id="renewUnitPrice" style="font-size:13px;color:#718096;margin-bottom:16px;"></div>
    <div id="renewPerUseSection" style="display:none;">
      <div class="field-group">
        <label>购买次数</label>
        <input id="renewQuantity" type="number" min="1" value="1" oninput="calcPerUseCost()">
      </div>
    </div>
    <div id="renewSubSection" style="display:none;">
      <label style="font-size:13px;color:#4a5568;display:block;margin-bottom:10px;">续费时长</label>
      <div style="display:flex;flex-direction:column;gap:10px;margin-bottom:16px;">
        <label style="display:flex;align-items:center;gap:8px;font-size:14px;color:#4a5568;cursor:pointer;">
          <input type="radio" name="renewMonths" value="1" checked onchange="calcSubCost()"> 按月（1个月）
        </label>
        <label style="display:flex;align-items:center;gap:8px;font-size:14px;color:#4a5568;cursor:pointer;">
          <input type="radio" name="renewMonths" value="12" onchange="calcSubCost()"> 按年（12个月付费，赠送2个月）
        </label>
      </div>
    </div>
    <div id="renewTotalCost" style="font-size:16px;font-weight:700;color:#818cf8;margin-bottom:20px;"></div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeRenewModal()">取消</button>
      <button class="btn btn-primary" onclick="submitRenew()">确认续费</button>
    </div>
  </div>
</div>
<form id="renewPerUseForm" method="POST" action="/user/pack/renew-uses" style="display:none;">
  <input type="hidden" name="listing_id" id="renewPerUseListingId">
  <input type="hidden" name="quantity" id="renewPerUseQuantity">
</form>
<form id="renewSubForm" method="POST" action="/user/pack/renew-subscription" style="display:none;">
  <input type="hidden" name="listing_id" id="renewSubListingId">
  <input type="hidden" name="months" id="renewSubMonths">
</form>

<!-- Delete Purchased Pack Modal -->
<div id="deleteModal" class="modal-overlay">
  <div class="modal-box" style="max-width:420px;">
    <button onclick="closeDeleteModal()" class="modal-close">&times;</button>
    <h3 class="modal-title">删除分析包</h3>
    <div style="font-size:14px;color:#4a5568;margin-bottom:8px;">分析包：<span id="deletePackName" style="font-weight:600;"></span></div>
    <div style="font-size:13px;color:#ef4444;margin-bottom:20px;">确定要删除该分析包吗？删除后将不再显示在已购列表中。</div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeDeleteModal()">取消</button>
      <button class="btn btn-danger-outline" onclick="submitDelete()" style="background:#ef4444;color:#fff;border:none;">确认删除</button>
    </div>
  </div>
</div>
<form id="deleteForm" method="POST" action="/user/pack/delete" style="display:none;">
  <input type="hidden" name="listing_id" id="deleteListingId">
</form>

<!-- Purchase Details Modal -->
<div id="purchaseDetailsModal" class="modal-overlay">
  <div class="modal-box" style="max-width:600px;">
    <button onclick="closePurchaseDetailsModal()" class="modal-close">&times;</button>
    <h3 class="modal-title">购买明细 - <span id="purchaseDetailsPackName"></span></h3>
    <div id="purchaseDetailsLoading" style="text-align:center;padding:20px;color:#94a3b8;">加载中...</div>
    <div id="purchaseDetailsContent" style="display:none;">
      <div id="purchaseDetailsSplitInfo" style="font-size:12px;color:#718096;margin-bottom:12px;"></div>
      <div style="overflow-x:auto;">
        <table class="author-table" style="font-size:12px;">
          <thead>
            <tr>
              <th>买家</th>
              <th>支付金额</th>
              <th>作者收入</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody id="purchaseDetailsBody"></tbody>
        </table>
      </div>
      <div id="purchaseDetailsEmpty" style="display:none;text-align:center;padding:20px;color:#a0aec0;font-size:13px;">暂无购买记录</div>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closePurchaseDetailsModal()">关闭</button>
    </div>
  </div>
</div>

<!-- Author Delete Shared Pack Modal -->
<div id="authorDeleteModal" class="modal-overlay">
  <div class="modal-box" style="max-width:420px;">
    <button onclick="closeAuthorDeleteModal()" class="modal-close">&times;</button>
    <h3 class="modal-title">删除已共享分析包</h3>
    <div style="font-size:14px;color:#4a5568;margin-bottom:8px;">分析包：<span id="authorDeletePackName" style="font-weight:600;"></span></div>
    <div style="font-size:13px;color:#ef4444;margin-bottom:20px;">确定要删除该已拒绝的分析包吗？删除后将无法恢复。</div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeAuthorDeleteModal()">取消</button>
      <button class="btn btn-danger-outline" onclick="submitAuthorDelete()" style="background:#ef4444;color:#fff;border:none;">确认删除</button>
    </div>
  </div>
</div>
<form id="authorDeleteForm" method="POST" action="/user/author/delete-pack" style="display:none;">
  <input type="hidden" name="listing_id" id="authorDeleteListingId">
</form>

<!-- Author Delist Shared Pack Modal -->
<div id="authorDelistModal" class="modal-overlay">
  <div class="modal-box" style="max-width:420px;">
    <button onclick="closeAuthorDelistModal()" class="modal-close">&times;</button>
    <h3 class="modal-title">下架分析包</h3>
    <div style="font-size:14px;color:#4a5568;margin-bottom:8px;">分析包：<span id="delistPackName" style="font-weight:600;"></span></div>
    <div style="font-size:13px;color:#ef4444;margin-bottom:20px;">确认要下架此分析包吗？下架后用户将无法在市场中看到此分析包。</div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeAuthorDelistModal()">取消</button>
      <button class="btn btn-danger-outline" onclick="submitAuthorDelist()" style="background:#ef4444;color:#fff;border:none;">确认下架</button>
    </div>
  </div>
</div>
<form id="authorDelistForm" method="POST" action="/user/author/delist-pack" style="display:none;">
  <input type="hidden" name="listing_id" id="delistListingId">
</form>

<script>
/* Tab switching */
function switchTab(tab) {
    document.querySelectorAll('.tab-panel').forEach(function(p){p.classList.remove('active');});
    document.querySelectorAll('.tab-btn').forEach(function(b){b.classList.remove('active');});
    document.getElementById('tab' + tab.charAt(0).toUpperCase() + tab.slice(1)).classList.add('active');
    document.getElementById('tabBtn' + tab.charAt(0).toUpperCase() + tab.slice(1)).classList.add('active');
}

/* Purchase Details Modal */
function openPurchaseDetailsModal(btn) {
    var listingId = btn.getAttribute('data-listing-id');
    var packName = btn.getAttribute('data-pack-name');
    document.getElementById('purchaseDetailsPackName').innerText = packName;
    document.getElementById('purchaseDetailsLoading').style.display = 'block';
    document.getElementById('purchaseDetailsContent').style.display = 'none';
    document.getElementById('purchaseDetailsModal').style.display = 'flex';
    fetch('/user/author/pack-purchases?listing_id=' + encodeURIComponent(listingId), {credentials:'same-origin'})
        .then(function(r){ return r.json(); })
        .then(function(data){
            document.getElementById('purchaseDetailsLoading').style.display = 'none';
            document.getElementById('purchaseDetailsContent').style.display = 'block';
            document.getElementById('purchaseDetailsSplitInfo').innerText = '分成比例：' + (data.split_pct || 70) + '%';
            var tbody = document.getElementById('purchaseDetailsBody');
            tbody.innerHTML = '';
            var purchases = data.purchases || [];
            if (purchases.length === 0) {
                document.getElementById('purchaseDetailsEmpty').style.display = 'block';
            } else {
                document.getElementById('purchaseDetailsEmpty').style.display = 'none';
                for (var i = 0; i < purchases.length; i++) {
                    var p = purchases[i];
                    var tr = document.createElement('tr');
                    tr.innerHTML = '<td>' + escapeHtml(p.buyer) + '</td>' +
                        '<td>' + p.amount.toFixed(0) + ' Credits</td>' +
                        '<td style="color:#10b981;font-weight:600;">' + p.author_earning.toFixed(0) + ' Credits</td>' +
                        '<td>' + escapeHtml(p.created_at) + '</td>';
                    tbody.appendChild(tr);
                }
            }
        }).catch(function(){
            document.getElementById('purchaseDetailsLoading').innerText = '加载失败，请重试';
        });
}
function closePurchaseDetailsModal() { document.getElementById('purchaseDetailsModal').style.display = 'none'; }
function escapeHtml(str) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

var _renewState = {listingId:"", shareMode:"", creditsPrice:0};
var _withdrawPaymentInfo = null;
var _withdrawFeeRate = 0;
var _paymentTypeLabels = {"paypal":"PayPal","wechat":"微信","alipay":"AliPay","check":"支票","wire_transfer":"国际电汇","bank_card_us":"美国银行卡","bank_card_eu":"欧洲银行卡","bank_card_cn":"中国银行卡"};
var _savedPaymentType = "";
var _savedPaymentDetails = {};

/* Payment Settings Modal */
function openPaymentSettingsModal() {
    document.getElementById("paymentType").value = "";
    onPaymentTypeChange();
    clearPaymentErrors();
    showPaymentMsg("", "");
    document.getElementById("feeRatesDialog").style.display = "none";
    document.getElementById("paymentSettingsModal").style.display = "flex";
    fetch("/user/payment-info", {credentials:"same-origin"})
        .then(function(r){ return r.json(); })
        .then(function(data){
            if (data.payment_type) {
                _savedPaymentType = data.payment_type;
                _savedPaymentDetails = data.payment_details || {};
                document.getElementById("paymentType").value = data.payment_type;
                onPaymentTypeChange();
            }
        }).catch(function(){});
}
function closePaymentSettingsModal() { document.getElementById("paymentSettingsModal").style.display = "none"; }
function openFeeRatesDialog() {
    document.getElementById("feeRatesDialog").style.display = "flex";
    var currentType = document.getElementById("paymentType").value;
    var currentLabel = currentType ? (_paymentTypeLabels[currentType] || currentType) : "";
    document.getElementById("feeRatesCurrentType").innerHTML = currentLabel ? '当前收款方式：<span style="font-weight:600;color:#ea580c;">'+currentLabel+'</span>' : '<span style="color:#94a3b8;">尚未选择收款方式</span>';
    document.getElementById("feeRatesDialogContent").innerText = "加载中...";
    fetch("/user/payment-info/fee-rates", {credentials:"same-origin"})
        .then(function(r){ return r.json(); })
        .then(function(data){
            var html = '<table style="width:100%;border-collapse:collapse;">';
            var types = [["paypal","PayPal"],["wechat","微信"],["alipay","AliPay"],["check","支票"],["wire_transfer","国际电汇 (SWIFT)"],["bank_card_us","美国银行卡 (ACH)"],["bank_card_eu","欧洲银行卡 (SEPA)"],["bank_card_cn","中国银行卡 (CNAPS)"]];
            for (var i=0;i<types.length;i++) {
                var rate = data[types[i][0]] || 0;
                var pct = rate.toFixed(1) + "%";
                var isActive = types[i][0] === currentType;
                var bg = isActive ? "#fff7ed" : (i % 2 === 0 ? "#f8fafc" : "#ffffff");
                var borderLeft = isActive ? "border-left:3px solid #f97316;" : "";
                var labelExtra = isActive ? ' <span style="font-size:11px;color:#ea580c;font-weight:600;">✓ 当前</span>' : '';
                var fontStyle = isActive ? "color:#9a3412;font-weight:600;" : "";
                html += '<tr style="background:'+bg+';'+borderLeft+'"><td style="padding:6px 10px;border-bottom:1px solid #e2e8f0;'+fontStyle+'">'+types[i][1]+labelExtra+'</td><td style="padding:6px 10px;border-bottom:1px solid #e2e8f0;text-align:right;font-weight:500;'+fontStyle+'">'+pct+'</td></tr>';
            }
            html += '</table>';
            document.getElementById("feeRatesDialogContent").innerHTML = html;
        }).catch(function(){ document.getElementById("feeRatesDialogContent").innerText = "加载失败，请重试"; });
}
function closeFeeRatesDialog() { document.getElementById("feeRatesDialog").style.display = "none"; }
function onPaymentTypeChange() {
    var t = document.getElementById("paymentType").value;
    document.getElementById("paymentFieldsAccount").style.display = (t==="paypal"||t==="wechat"||t==="alipay") ? "block" : "none";
    document.getElementById("paymentFieldsCheck").style.display = (t==="check") ? "block" : "none";
    document.getElementById("paymentFieldsWireTransfer").style.display = (t==="wire_transfer") ? "block" : "none";
    document.getElementById("paymentFieldsBankUS").style.display = (t==="bank_card_us") ? "block" : "none";
    document.getElementById("paymentFieldsBankEU").style.display = (t==="bank_card_eu") ? "block" : "none";
    document.getElementById("paymentFieldsBankCN").style.display = (t==="bank_card_cn") ? "block" : "none";
    clearPaymentErrors();
    var d = (t && t === _savedPaymentType) ? _savedPaymentDetails : {};
    document.getElementById("paymentAccount").value = d.account || "";
    document.getElementById("paymentUsername").value = d.username || "";
    document.getElementById("paymentCheckFullLegalName").value = d.full_legal_name || "";
    document.getElementById("paymentCheckProvince").value = d.province || "";
    document.getElementById("paymentCheckCity").value = d.city || "";
    document.getElementById("paymentCheckDistrict").value = d.district || "";
    document.getElementById("paymentCheckStreetAddress").value = d.street_address || "";
    document.getElementById("paymentCheckPostalCode").value = d.postal_code || "";
    document.getElementById("paymentCheckPhone").value = d.phone || "";
    document.getElementById("paymentCheckMemo").value = d.memo || "";
    document.getElementById("paymentBeneficiaryName").value = d.beneficiary_name || "";
    document.getElementById("paymentBeneficiaryAddress").value = d.beneficiary_address || "";
    document.getElementById("paymentWireBankName").value = d.bank_name || "";
    document.getElementById("paymentSwiftCode").value = d.swift_code || "";
    document.getElementById("paymentWireAccountNumber").value = d.account_number || "";
    document.getElementById("paymentBankBranchAddress").value = d.bank_branch_address || "";
    document.getElementById("paymentUSLegalName").value = d.legal_name || "";
    document.getElementById("paymentRoutingNumber").value = d.routing_number || "";
    document.getElementById("paymentUSAccountNumber").value = d.account_number || "";
    document.getElementById("paymentUSAccountType").value = d.account_type || "checking";
    document.getElementById("paymentEULegalName").value = d.legal_name || "";
    document.getElementById("paymentIBAN").value = d.iban || "";
    document.getElementById("paymentEUBicSwift").value = d.bic_swift || "";
    document.getElementById("paymentCNRealName").value = d.real_name || "";
    document.getElementById("paymentCNCardNumber").value = d.card_number || "";
    document.getElementById("paymentCNBankBranch").value = d.bank_branch || "";
}
function clearPaymentErrors() {
    var errors = document.querySelectorAll(".field-error-msg");
    for (var i=0;i<errors.length;i++) errors[i].style.display="none";
    var inputs = document.querySelectorAll("#paymentSettingsModal input[type=text]");
    for (var i=0;i<inputs.length;i++) inputs[i].classList.remove("field-error");
}
function showPaymentFieldError(id) {
    var el = document.getElementById(id); if(el) el.style.display="block";
    var inp = document.getElementById(id.replace("Error","")); if(inp) inp.classList.add("field-error");
}
function showPaymentMsg(msg, type) {
    var el = document.getElementById("paymentSettingsMsg");
    if(!msg){el.style.display="none";return;}
    el.style.display="block"; el.innerText=msg;
    el.className = "msg-box " + (type==="success" ? "msg-success" : "msg-error");
}
function validatePaymentFields() {
    clearPaymentErrors();
    var t = document.getElementById("paymentType").value;
    if(!t){showPaymentMsg("请选择收款方式","error");return false;}
    var valid=true;
    if(t==="paypal"||t==="wechat"||t==="alipay"){
        if(!document.getElementById("paymentAccount").value.trim()){showPaymentFieldError("paymentAccountError");valid=false;}
        if(!document.getElementById("paymentUsername").value.trim()){showPaymentFieldError("paymentUsernameError");valid=false;}
    } else if(t==="check"){
        if(!document.getElementById("paymentCheckFullLegalName").value.trim()){showPaymentFieldError("paymentCheckFullLegalNameError");valid=false;}
        if(!document.getElementById("paymentCheckProvince").value.trim()){showPaymentFieldError("paymentCheckProvinceError");valid=false;}
        if(!document.getElementById("paymentCheckCity").value.trim()){showPaymentFieldError("paymentCheckCityError");valid=false;}
        if(!document.getElementById("paymentCheckDistrict").value.trim()){showPaymentFieldError("paymentCheckDistrictError");valid=false;}
        if(!document.getElementById("paymentCheckStreetAddress").value.trim()){showPaymentFieldError("paymentCheckStreetAddressError");valid=false;}
        if(!document.getElementById("paymentCheckPostalCode").value.trim()){showPaymentFieldError("paymentCheckPostalCodeError");valid=false;}
        if(!document.getElementById("paymentCheckPhone").value.trim()){showPaymentFieldError("paymentCheckPhoneError");valid=false;}
    } else if(t==="wire_transfer"){
        if(!document.getElementById("paymentBeneficiaryName").value.trim()){showPaymentFieldError("paymentBeneficiaryNameError");valid=false;}
        if(!document.getElementById("paymentBeneficiaryAddress").value.trim()){showPaymentFieldError("paymentBeneficiaryAddressError");valid=false;}
        if(!document.getElementById("paymentWireBankName").value.trim()){showPaymentFieldError("paymentWireBankNameError");valid=false;}
        if(!document.getElementById("paymentSwiftCode").value.trim()){showPaymentFieldError("paymentSwiftCodeError");valid=false;}
        if(!document.getElementById("paymentWireAccountNumber").value.trim()){showPaymentFieldError("paymentWireAccountNumberError");valid=false;}
    } else if(t==="bank_card_us"){
        if(!document.getElementById("paymentUSLegalName").value.trim()){showPaymentFieldError("paymentUSLegalNameError");valid=false;}
        if(!document.getElementById("paymentRoutingNumber").value.trim()){showPaymentFieldError("paymentRoutingNumberError");valid=false;}
        if(!document.getElementById("paymentUSAccountNumber").value.trim()){showPaymentFieldError("paymentUSAccountNumberError");valid=false;}
    } else if(t==="bank_card_eu"){
        if(!document.getElementById("paymentEULegalName").value.trim()){showPaymentFieldError("paymentEULegalNameError");valid=false;}
        if(!document.getElementById("paymentIBAN").value.trim()){showPaymentFieldError("paymentIBANError");valid=false;}
        if(!document.getElementById("paymentEUBicSwift").value.trim()){showPaymentFieldError("paymentEUBicSwiftError");valid=false;}
    } else if(t==="bank_card_cn"){
        if(!document.getElementById("paymentCNRealName").value.trim()){showPaymentFieldError("paymentCNRealNameError");valid=false;}
        if(!document.getElementById("paymentCNCardNumber").value.trim()){showPaymentFieldError("paymentCNCardNumberError");valid=false;}
        if(!document.getElementById("paymentCNBankBranch").value.trim()){showPaymentFieldError("paymentCNBankBranchError");valid=false;}
    }
    return valid;
}
function savePaymentSettings() {
    showPaymentMsg("","");
    if(!validatePaymentFields()) return;
    var t = document.getElementById("paymentType").value;
    var details = {};
    if(t==="paypal"||t==="wechat"||t==="alipay"){
        details={account:document.getElementById("paymentAccount").value.trim(),username:document.getElementById("paymentUsername").value.trim()};
    } else if(t==="check"){
        details={full_legal_name:document.getElementById("paymentCheckFullLegalName").value.trim(),province:document.getElementById("paymentCheckProvince").value.trim(),city:document.getElementById("paymentCheckCity").value.trim(),district:document.getElementById("paymentCheckDistrict").value.trim(),street_address:document.getElementById("paymentCheckStreetAddress").value.trim(),postal_code:document.getElementById("paymentCheckPostalCode").value.trim(),phone:document.getElementById("paymentCheckPhone").value.trim(),memo:document.getElementById("paymentCheckMemo").value.trim()};
    } else if(t==="wire_transfer"){
        details={beneficiary_name:document.getElementById("paymentBeneficiaryName").value.trim(),beneficiary_address:document.getElementById("paymentBeneficiaryAddress").value.trim(),bank_name:document.getElementById("paymentWireBankName").value.trim(),swift_code:document.getElementById("paymentSwiftCode").value.trim(),account_number:document.getElementById("paymentWireAccountNumber").value.trim(),bank_branch_address:document.getElementById("paymentBankBranchAddress").value.trim()};
    } else if(t==="bank_card_us"){
        details={legal_name:document.getElementById("paymentUSLegalName").value.trim(),routing_number:document.getElementById("paymentRoutingNumber").value.trim(),account_number:document.getElementById("paymentUSAccountNumber").value.trim(),account_type:document.getElementById("paymentUSAccountType").value};
    } else if(t==="bank_card_eu"){
        details={legal_name:document.getElementById("paymentEULegalName").value.trim(),iban:document.getElementById("paymentIBAN").value.trim(),bic_swift:document.getElementById("paymentEUBicSwift").value.trim()};
    } else if(t==="bank_card_cn"){
        details={real_name:document.getElementById("paymentCNRealName").value.trim(),card_number:document.getElementById("paymentCNCardNumber").value.trim(),bank_branch:document.getElementById("paymentCNBankBranch").value.trim()};
    }
    fetch("/user/payment-info",{method:"POST",credentials:"same-origin",headers:{"Content-Type":"application/json"},body:JSON.stringify({payment_type:t,payment_details:details})})
    .then(function(r){return r.json().then(function(d){return{ok:r.ok,data:d};});})
    .then(function(res){
        if(res.ok&&res.data.ok){showPaymentMsg("收款信息保存成功","success");setTimeout(function(){closePaymentSettingsModal();},1200);}
        else{showPaymentMsg(res.data.error||"保存失败，请重试","error");}
    }).catch(function(){showPaymentMsg("网络错误，请重试","error");});
}

/* Withdraw Modal */
function openWithdrawModal() {
    document.getElementById("withdrawCreditsInput").value="";
    document.getElementById("withdrawFormulaBox").style.display="none";
    document.getElementById("withdrawNetResult").style.display="none";
    document.getElementById("withdrawNoPaymentWarning").style.display="none";
    document.getElementById("withdrawFormContent").style.display="block";
    document.getElementById("withdrawPaymentInfo").style.display="none";
    _withdrawPaymentInfo=null; _withdrawFeeRate=0;
    document.getElementById("withdrawModal").style.display="flex";
    fetch("/user/payment-info",{credentials:"same-origin"})
        .then(function(r){return r.json();})
        .then(function(data){
            if(!data.payment_type){
                document.getElementById("withdrawNoPaymentWarning").style.display="block";
                document.getElementById("withdrawFormContent").style.display="none";
            } else {
                _withdrawPaymentInfo=data;
                document.getElementById("withdrawPaymentTypeLabel").innerText=_paymentTypeLabels[data.payment_type]||data.payment_type;
                fetch("/user/payment-info/fee-rate?type="+encodeURIComponent(data.payment_type),{credentials:"same-origin"})
                    .then(function(r){return r.json();})
                    .then(function(feeData){
                        _withdrawFeeRate=feeData.fee_rate||0;
                        document.getElementById("withdrawFeeRateLabel").innerText=_withdrawFeeRate.toFixed(1)+"%";
                        document.getElementById("withdrawPaymentInfo").style.display="flex";
                        calcWithdrawCash();
                    }).catch(function(){_withdrawFeeRate=0;document.getElementById("withdrawFeeRateLabel").innerText="0%";document.getElementById("withdrawPaymentInfo").style.display="flex";});
            }
        }).catch(function(){});
}
function closeWithdrawModal(){document.getElementById("withdrawModal").style.display="none";}
function openWithdrawRecordsModal(){
    document.getElementById("withdrawRecordsModal").style.display="flex";
    document.getElementById("withdrawRecordsContent").innerHTML='<div style="text-align:center;padding:30px;color:#94a3b8;">加载中...</div>';
    document.getElementById("withdrawRecordsTotalRow").style.display="none";
    fetch("/user/author/withdrawals",{credentials:"same-origin",headers:{"Accept":"application/json"}})
    .then(function(r){return r.json();})
    .then(function(data){
        var list=data.records||[];
        if(list.length===0){
            document.getElementById("withdrawRecordsContent").innerHTML='<div style="text-align:center;padding:40px 20px;color:#94a3b8;"><div style="font-size:40px;margin-bottom:12px;">📭</div><p>暂无提现记录</p></div>';
            return;
        }
        var html='<table style="width:100%;border-collapse:collapse;font-size:13px;">';
        html+='<thead><tr style="border-bottom:2px solid #e2e8f0;">';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">Credits</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">汇率</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">提现金额</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">手续费</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">实付</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">状态</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">时间</th>';
        html+='</tr></thead><tbody>';
        for(var i=0;i<list.length;i++){
            var r=list[i];
            var st=r.status==='pending'?'<span style="background:#fef3c7;color:#92400e;padding:2px 8px;border-radius:4px;font-size:11px;">待付款</span>':'<span style="background:#ecfdf5;color:#065f46;padding:2px 8px;border-radius:4px;font-size:11px;">已付款</span>';
            html+='<tr style="border-bottom:1px solid #f1f5f9;">';
            html+='<td style="padding:10px 8px;">'+r.credits_amount.toFixed(0)+'</td>';
            html+='<td style="padding:10px 8px;">'+r.cash_rate.toFixed(2)+'</td>';
            html+='<td style="padding:10px 8px;">¥'+r.cash_amount.toFixed(2)+'</td>';
            html+='<td style="padding:10px 8px;">¥'+r.fee_amount.toFixed(2)+'</td>';
            html+='<td style="padding:10px 8px;font-weight:600;">¥'+r.net_amount.toFixed(2)+'</td>';
            html+='<td style="padding:10px 8px;">'+st+'</td>';
            html+='<td style="padding:10px 8px;color:#94a3b8;font-size:12px;">'+r.created_at+'</td>';
            html+='</tr>';
        }
        html+='</tbody></table>';
        document.getElementById("withdrawRecordsContent").innerHTML=html;
        document.getElementById("withdrawRecordsTotalCash").innerText='¥'+(data.total_cash||0).toFixed(2);
        document.getElementById("withdrawRecordsTotalRow").style.display="block";
    })
    .catch(function(err){
        document.getElementById("withdrawRecordsContent").innerHTML='<div style="text-align:center;padding:30px;color:#ef4444;">加载失败：'+err+'</div>';
    });
}
function closeWithdrawRecordsModal(){document.getElementById("withdrawRecordsModal").style.display="none";}
function calcWithdrawCash() {
    var credits=parseFloat(document.getElementById("withdrawCreditsInput").value)||0;
    var rate=parseFloat(document.getElementById("withdrawCashRate").innerText)||0;
    var maxCredits=parseFloat(document.getElementById("withdrawCreditsInput").max)||0;
    var splitPct=parseFloat(document.getElementById("withdrawSplitPctLabel").innerText)||0;
    var cash=credits*rate;
    var warning=document.getElementById("withdrawWarning");
    var submitBtn=document.getElementById("withdrawSubmitBtn");
    var formulaBox=document.getElementById("withdrawFormulaBox");
    var netEl=document.getElementById("withdrawNetResult");
    warning.style.display="none";
    submitBtn.disabled=false;
    submitBtn.style.opacity="1";
    if(credits<=0){formulaBox.style.display="none";netEl.style.display="none";return;}
    var fee=cash*_withdrawFeeRate/100;
    var net=cash-fee;
    var lines=[];
    lines.push('<span style="color:#94a3b8;">① 分成后可提现余额已含分成比例 '+splitPct+'%</span>');
    lines.push('<span style="color:#334155;">② 提现金额 = '+credits+' × '+rate.toFixed(2)+' = <b>'+cash.toFixed(2)+'</b> 元</span>');
    if(_withdrawFeeRate>0){
        lines.push('<span style="color:#334155;">③ 手续费 = '+cash.toFixed(2)+' × '+_withdrawFeeRate.toFixed(1)+'% = <b>'+fee.toFixed(2)+'</b> 元</span>');
        lines.push('<span style="color:#10b981;font-weight:600;">④ 实付 = '+cash.toFixed(2)+' − '+fee.toFixed(2)+' = <b>'+net.toFixed(2)+'</b> 元</span>');
    } else {
        lines.push('<span style="color:#10b981;font-weight:600;">③ 实付 = <b>'+cash.toFixed(2)+'</b> 元（无手续费）</span>');
    }
    formulaBox.innerHTML=lines.join('<br>');
    formulaBox.style.display="block";
    netEl.innerText="实付金额："+net.toFixed(2)+" 元";
    netEl.style.display="block";
    if(credits>maxCredits){
        warning.innerHTML="⚠️ 超过可提现余额（"+maxCredits+" Credits）";
        warning.style.display="block";
        submitBtn.disabled=true;submitBtn.style.opacity="0.5";
    } else if(net<100){
        warning.innerHTML="⚠️ 实付 "+net.toFixed(2)+" 元，低于最低提现 100 元";
        warning.style.display="block";
        submitBtn.disabled=true;submitBtn.style.opacity="0.5";
    }
}
function submitWithdraw() {
    var credits=parseFloat(document.getElementById("withdrawCreditsInput").value)||0;
    if(credits<=0){alert("请输入有效的提现数量");return;}
    var maxCredits=parseFloat(document.getElementById("withdrawCreditsInput").max)||0;
    if(credits>maxCredits){alert("提现 Credits 数量不能超过可提现余额（"+maxCredits+" Credits）");return;}
    var rate=parseFloat(document.getElementById("withdrawCashRate").innerText)||0;
    var cash=credits*rate;
    var fee=cash*_withdrawFeeRate/100;
    var net=cash-fee;
    if(net<100){alert("扣除手续费后实付金额为 "+net.toFixed(2)+" 元，低于最低提现金额 100 元。请继续积累帐户余额后再提现。");return;}
    if(!confirm("确认提现 "+credits+" Credits？\n\n提现金额："+cash.toFixed(2)+" 元\n手续费："+fee.toFixed(2)+" 元\n实付金额："+net.toFixed(2)+" 元")){return;}
    var btn=document.getElementById("withdrawSubmitBtn");
    btn.disabled=true;btn.innerText="提交中...";
    var formData=new FormData();
    formData.append("credits_amount",credits);
    fetch("/user/author/withdraw",{
        method:"POST",
        body:formData,
        credentials:"same-origin",
        headers:{"Accept":"application/json","X-Requested-With":"XMLHttpRequest"}
    })
    .then(function(r){return r.json();})
    .then(function(data){
        if(data.ok){
            alert("✅ 提现申请已提交，请等待管理员审核付款。");
            window.location.href="/user/?success=withdraw";
        } else {
            alert("⚠️ 提现失败：" + (data.message||data.error||"未知错误"));
            btn.disabled=false;btn.innerText="确认提现";
        }
    })
    .catch(function(err){
        alert("⚠️ 提现请求失败："+err);
        btn.disabled=false;btn.innerText="确认提现";
    });
}

/* Edit Pack Modal */
function openEditPackModal(btn) {
    document.getElementById("editListingId").value=btn.getAttribute("data-listing-id");
    document.getElementById("editPackName").value=btn.getAttribute("data-pack-name");
    document.getElementById("editPackDesc").value=btn.getAttribute("data-pack-desc")||"";
    document.getElementById("editShareMode").value=btn.getAttribute("data-share-mode");
    document.getElementById("editCreditsPrice").value=btn.getAttribute("data-credits-price")||0;
    onEditShareModeChange();
    document.getElementById("editPackModal").style.display="flex";
}
function closeEditPackModal(){document.getElementById("editPackModal").style.display="none";}
function confirmEditPack(){
    if(confirm("修改已上架的分析包信息后，该分析包将被下架并需要重新提交审核后才能再次上架。\n\n确定要继续修改吗？")){
        document.getElementById("editPackForm").submit();
    }
}
function onEditShareModeChange() {
    var mode=document.getElementById("editShareMode").value;
    var ps=document.getElementById("editPriceSection");
    var pi=document.getElementById("editCreditsPrice");
    var hint=document.getElementById("editPriceHint");
    if(mode==="free"){ps.style.display="none";pi.value=0;}
    else if(mode==="per_use"){ps.style.display="block";pi.min=1;pi.max=100;hint.innerText="按次付费：1-100 Credits";}
    else if(mode==="subscription"){ps.style.display="block";pi.min=100;pi.max=1000;hint.innerText="订阅：100-1000 Credits";}
}

/* Renew Modal */
function openRenewModal(btn) {
    var listingId=btn.getAttribute("data-listing-id");
    var packName=btn.getAttribute("data-pack-name");
    var shareMode=btn.getAttribute("data-share-mode");
    var creditsPrice=parseFloat(btn.getAttribute("data-credits-price"))||0;
    _renewState={listingId:listingId,shareMode:shareMode,creditsPrice:creditsPrice};
    document.getElementById("renewPackName").innerText="分析包："+packName;
    if(shareMode==="per_use"){
        document.getElementById("renewTitle").innerText="按次续费";
        document.getElementById("renewUnitPrice").innerText="单次价格："+creditsPrice+" Credits";
        document.getElementById("renewPerUseSection").style.display="block";
        document.getElementById("renewSubSection").style.display="none";
        document.getElementById("renewQuantity").value=1;
        calcPerUseCost();
    } else if(shareMode==="subscription"){
        document.getElementById("renewTitle").innerText="订阅续费";
        document.getElementById("renewUnitPrice").innerText="月度价格："+creditsPrice+" Credits";
        document.getElementById("renewPerUseSection").style.display="none";
        document.getElementById("renewSubSection").style.display="block";
        var radios=document.getElementsByName("renewMonths");
        for(var i=0;i<radios.length;i++){if(radios[i].value==="1")radios[i].checked=true;}
        calcSubCost();
    }
    document.getElementById("renewModal").style.display="flex";
}
function closeRenewModal(){document.getElementById("renewModal").style.display="none";}
function calcPerUseCost(){var qty=parseInt(document.getElementById("renewQuantity").value)||1;if(qty<1)qty=1;document.getElementById("renewTotalCost").innerText="总费用："+(_renewState.creditsPrice*qty)+" Credits";}
function calcSubCost(){var radios=document.getElementsByName("renewMonths");var m=1;for(var i=0;i<radios.length;i++){if(radios[i].checked){m=parseInt(radios[i].value);break;}}document.getElementById("renewTotalCost").innerText="总费用："+(_renewState.creditsPrice*m)+" Credits";}
function submitRenew(){
    if(_renewState.shareMode==="per_use"){var qty=parseInt(document.getElementById("renewQuantity").value)||1;if(qty<1){alert("请输入有效的次数");return;}document.getElementById("renewPerUseListingId").value=_renewState.listingId;document.getElementById("renewPerUseQuantity").value=qty;document.getElementById("renewPerUseForm").submit();}
    else if(_renewState.shareMode==="subscription"){var radios=document.getElementsByName("renewMonths");var m=1;for(var i=0;i<radios.length;i++){if(radios[i].checked){m=parseInt(radios[i].value);break;}}document.getElementById("renewSubListingId").value=_renewState.listingId;document.getElementById("renewSubMonths").value=m;document.getElementById("renewSubForm").submit();}
}
/* Delete Purchased Pack Modal */
function openDeleteModal(btn){
    document.getElementById("deletePackName").innerText=btn.getAttribute("data-pack-name");
    document.getElementById("deleteListingId").value=btn.getAttribute("data-listing-id");
    document.getElementById("deleteModal").style.display="flex";
}
function closeDeleteModal(){document.getElementById("deleteModal").style.display="none";}
function submitDelete(){document.getElementById("deleteForm").submit();}

/* Author Delete Rejected Pack Modal */
function openAuthorDeleteModal(btn){
    document.getElementById("authorDeletePackName").innerText=btn.getAttribute("data-pack-name");
    document.getElementById("authorDeleteListingId").value=btn.getAttribute("data-listing-id");
    document.getElementById("authorDeleteModal").style.display="flex";
}
function closeAuthorDeleteModal(){document.getElementById("authorDeleteModal").style.display="none";}
function submitAuthorDelete(){document.getElementById("authorDeleteForm").submit();}

/* Author Delist Published Pack Modal */
function openAuthorDelistModal(btn){
    document.getElementById("delistListingId").value=btn.getAttribute("data-listing-id");
    document.getElementById("delistPackName").textContent=btn.getAttribute("data-pack-name");
    document.getElementById("authorDelistModal").style.display="flex";
}
function closeAuthorDelistModal(){document.getElementById("authorDelistModal").style.display="none";}
function submitAuthorDelist(){document.getElementById("authorDelistForm").submit();}
</script>
</body>
</html>`
