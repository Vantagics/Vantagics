package templates

import "html/template"

// UserDashboardTmpl is the parsed user dashboard page template.
var UserDashboardTmpl = template.Must(template.New("user_dashboard").Parse(userDashboardHTML))

// UserCustomProductOrdersTmpl is the parsed user custom product orders page template.
var UserCustomProductOrdersTmpl = template.Must(template.New("user_custom_product_orders").Parse(userCustomProductOrdersHTML))

const userDashboardHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="default-lang" content="{{.DefaultLang}}">
    <title data-i18n="personal_center">个人中心 - 分析技能包市场</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", sans-serif;
            background: #f0f2f5;
            min-height: 100vh;
            color: #1e293b;
            line-height: 1.6;
        }
        .dashboard-wrap {
            max-width: 1020px;
            margin: 0 auto;
            padding: 36px 24px;
        }

        /* Header */
        .header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 32px;
        }
        .header-title {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .header-title .logo {
            width: 38px; height: 38px;
            background: #4f46e5;
            border-radius: 10px;
            display: flex; align-items: center; justify-content: center;
            font-size: 18px;
            border: none;
            color: #fff;
        }
        .header-title h1 {
            font-size: 22px;
            font-weight: 700;
            color: #1e293b;
            letter-spacing: -0.3px;
        }
        .header-lang {
            display: flex;
            gap: 4px;
            background: #fff;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 3px;
        }
        .header-lang a {
            padding: 5px 14px;
            border-radius: 6px;
            text-decoration: none;
            font-size: 13px;
            font-weight: 600;
            color: #64748b;
            transition: all 0.15s;
        }
        .header-lang a:hover { color: #334155; background: #f8fafc; }
        .header-lang a.active { background: #4f46e5; color: #fff; }

        /* User info card */
        .user-info {
            background: #fff;
            border-radius: 12px;
            padding: 24px 28px;
            margin-bottom: 28px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 4px 16px rgba(0,0,0,0.04);
            border: 1px solid #e2e8f0;
        }
        .user-detail {
            display: flex;
            align-items: center;
            gap: 24px;
            flex-wrap: wrap;
        }
        .user-avatar {
            width: 46px; height: 46px;
            background: #4f46e5;
            border-radius: 12px;
            display: flex; align-items: center; justify-content: center;
            font-size: 20px;
            border: none;
            color: #fff;
        }
        .user-email {
            font-size: 14px;
            color: #334155;
        }
        .user-email .label {
            font-size: 11px;
            color: #64748b;
            display: block;
            margin-bottom: 2px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            font-weight: 600;
        }
        .credits-info {
            font-size: 14px;
            color: #334155;
        }
        .credits-info .label {
            font-size: 11px;
            color: #64748b;
            display: block;
            margin-bottom: 2px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            font-weight: 600;
        }
        .credits-info .balance {
            color: #4f46e5;
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
            border-radius: 7px;
            font-size: 13px;
            font-weight: 600;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.15s ease;
            display: inline-flex;
            align-items: center;
            gap: 5px;
            letter-spacing: 0.1px;
        }
        .btn-primary {
            background: #4f46e5;
            color: #fff;
            border-color: #4338ca;
        }
        .btn-primary:hover { background: #4338ca; border-color: #3730a3; }
        .btn-secondary {
            background: #fff;
            color: #475569;
            border-color: #cbd5e1;
        }
        .btn-secondary:hover { background: #f8fafc; border-color: #94a3b8; color: #1e293b; }
        .btn-accent {
            background: #059669;
            color: #fff;
            border-color: #047857;
        }
        .btn-accent:hover { background: #047857; border-color: #065f46; }
        .btn-warm {
            background: #f59e0b;
            color: #fff;
            border-color: #d97706;
        }
        .btn-warm:hover { background: #d97706; border-color: #b45309; }
        .btn-danger-outline {
            background: #fff;
            color: #dc2626;
            border: 1px solid #fca5a5;
        }
        .btn-danger-outline:hover { background: #fef2f2; border-color: #f87171; color: #b91c1c; }
        .btn-ghost {
            background: #f5f3ff;
            color: #6d28d9;
            border: 1px solid #ddd6fe;
        }
        .btn-ghost:hover { background: #ede9fe; border-color: #c4b5fd; color: #5b21b6; }
        .btn-sm { padding: 5px 12px; font-size: 12px; border-radius: 6px; }
        .btn-danger-sm {
            padding: 5px 12px;
            font-size: 12px;
            border-radius: 6px;
            background: #fff;
            color: #dc2626;
            border: 1px solid #fca5a5;
            cursor: pointer;
            font-weight: 600;
            transition: all 0.15s ease;
        }
        .btn-danger-sm:hover { background: #fef2f2; border-color: #f87171; color: #b91c1c; }

        /* Section */
        .section {
            margin-bottom: 32px;
        }
        .section-title {
            font-size: 15px;
            font-weight: 700;
            color: #1e293b;
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
            grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
            gap: 18px;
        }
        .pack-card {
            background: #fff;
            border-radius: 12px;
            padding: 0;
            border: 1px solid #e2e8f0;
            box-shadow: 0 1px 4px rgba(0,0,0,0.04), 0 2px 8px rgba(0,0,0,0.02);
            transition: transform 0.2s ease, box-shadow 0.2s ease;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        .pack-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04);
        }
        .pack-card-accent {
            height: 4px;
            border-radius: 0;
        }
        .pack-card-accent.accent-free { background: linear-gradient(90deg, #10b981, #34d399); }
        .pack-card-accent.accent-per-use { background: linear-gradient(90deg, #4f46e5, #818cf8); }
        .pack-card-accent.accent-time-limited { background: linear-gradient(90deg, #f59e0b, #fbbf24); }
        .pack-card-accent.accent-subscription { background: linear-gradient(90deg, #7c3aed, #a78bfa); }
        .pack-card-body {
            padding: 18px 20px 0 20px;
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        .pack-card .pack-header {
            display: flex;
            align-items: flex-start;
            justify-content: space-between;
            gap: 8px;
            margin-bottom: 10px;
        }
        .pack-card .pack-name {
            font-size: 15px;
            font-weight: 700;
            color: #1e293b;
            line-height: 1.4;
            flex: 1;
            min-width: 0;
            word-break: break-word;
        }
        .pack-card .pack-category {
            font-size: 11px;
            color: #94a3b8;
            text-transform: uppercase;
            letter-spacing: 0.4px;
            font-weight: 600;
            margin-bottom: 12px;
        }
        .pack-card .pack-info-grid {
            display: flex;
            flex-wrap: wrap;
            gap: 0;
            margin-bottom: 12px;
            padding: 8px 12px;
            background: #f8fafc;
            border-radius: 8px;
            font-size: 12px;
            color: #64748b;
            line-height: 1.8;
        }
        .pack-info-item {
            display: inline;
        }
        .pack-info-item .info-value {
            color: #334155;
            font-weight: 600;
        }
        .pack-info-sep {
            margin: 0 8px;
            color: #cbd5e1;
        }
        .pack-card .pack-meta {
            display: flex;
            align-items: center;
            gap: 8px;
            flex-wrap: wrap;
            margin-bottom: 8px;
        }
        .tag {
            display: inline-flex;
            align-items: center;
            padding: 3px 10px;
            border-radius: 20px;
            font-size: 11px;
            font-weight: 700;
            letter-spacing: 0.2px;
        }
        .tag-free { background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; }
        .tag-per-use { background: #eef2ff; color: #4338ca; border: 1px solid #c7d2fe; }
        .tag-time-limited { background: #fffbeb; color: #b45309; border: 1px solid #fde68a; }
        .tag-subscription { background: #f5f3ff; color: #7c3aed; border: 1px solid #ddd6fe; }
        .usage-progress {
            font-size: 12px;
            color: #334155;
            font-weight: 600;
        }
        .usage-exhausted { color: #dc2626; }
        .pack-usage {
            margin-top: 4px;
            font-size: 12px;
        }
        .pack-card .pack-date {
            font-size: 12px;
            color: #64748b;
        }
        .pack-card .pack-expires {
            font-size: 12px;
            color: #475569;
            margin-top: 4px;
        }
        .pack-card .pack-expires.subscription-expires {
            color: #7c3aed;
            font-weight: 600;
        }
        .pack-actions {
            display: flex;
            gap: 8px;
            margin-top: auto;
            padding: 14px 20px;
            border-top: 1px solid #f1f5f9;
            background: #fafbfc;
        }

        /* Empty state */
        .empty-state {
            text-align: center;
            padding: 48px 20px;
            color: #64748b;
            background: #fff;
            border-radius: 10px;
            border: 1px dashed #cbd5e1;
        }
        .empty-state .icon { font-size: 36px; margin-bottom: 12px; opacity: 0.7; }
        .empty-state p { font-size: 14px; }
        /* Author panel */
        .author-panel {
            margin-top: 8px;
            padding-top: 28px;
            border-top: 1px solid #e2e8f0;
        }
        .author-panel-title {
            font-size: 17px;
            font-weight: 700;
            color: #1e293b;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .author-stats {
            display: flex;
            gap: 14px;
            margin-bottom: 28px;
            flex-wrap: wrap;
        }
        .stat-card {
            background: #fff;
            border-radius: 10px;
            padding: 22px 26px;
            border: 1px solid #e2e8f0;
            border-left: 4px solid #4f46e5;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
            flex: 1;
            min-width: 200px;
        }
        .stat-card .stat-label {
            font-size: 12px;
            color: #64748b;
            margin-bottom: 8px;
            text-transform: uppercase;
            letter-spacing: 0.4px;
            font-weight: 600;
        }
        .stat-card .stat-value {
            font-size: 28px;
            font-weight: 800;
            color: #1e293b;
        }
        .stat-card .stat-value.revenue { color: #059669; }
        .stat-card .stat-value.unwithdrawn { color: #d97706; }
        .stat-actions {
            display: flex;
            gap: 10px;
            margin-top: 14px;
            align-items: center;
        }
        .withdraw-hint {
            font-size: 12px;
            color: #64748b;
        }

        /* Author pack table */
        .author-table-wrap {
            background: #fff;
            border-radius: 10px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
            overflow-x: auto;
        }
        .author-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 13px;
        }
        .author-table th {
            background: #f8fafc;
            padding: 12px 16px;
            text-align: left;
            font-weight: 700;
            color: #475569;
            border-bottom: 2px solid #e2e8f0;
            white-space: nowrap;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.3px;
        }
        .author-table td {
            padding: 14px 16px;
            border-bottom: 1px solid #f1f5f9;
            color: #334155;
        }
        .author-table tr:last-child td { border-bottom: none; }
        .author-table tr:hover td { background: #f8fafc; }
        .status-badge {
            display: inline-block;
            padding: 3px 10px;
            border-radius: 5px;
            font-size: 11px;
            font-weight: 700;
        }
        .status-pending { background: #fff7ed; color: #c2410c; border: 1px solid #fed7aa; }
        .status-published { background: #ecfdf5; color: #047857; border: 1px solid #a7f3d0; }
        .status-rejected { background: #fef2f2; color: #dc2626; border: 1px solid #fecaca; }
        .status-delisted { background: #f8fafc; color: #64748b; border: 1px solid #e2e8f0; }
        .version-badge {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 700;
            font-family: "SF Mono", "Cascadia Code", "Consolas", monospace;
            background: #f1f5f9;
            color: #475569;
            border: 1px solid #e2e8f0;
            letter-spacing: 0.3px;
        }
        .td-actions {
            display: flex;
            gap: 6px;
            align-items: center;
            flex-wrap: wrap;
        }
        .btn-share-link {
            background: #fff;
            color: #059669;
            border: 1px solid #a7f3d0;
            border-radius: 6px;
            cursor: pointer;
            font-size: 13px;
            padding: 4px 8px;
            transition: all 0.15s ease;
            line-height: 1;
        }
        .btn-share-link:hover { background: #ecfdf5; border-color: #059669; }
        .storefront-share-btn {
            width: 32px; height: 32px;
            border-radius: 7px;
            border: 1px solid #e2e8f0;
            background: #fff;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            transition: all 0.15s;
            color: #94a3b8;
            text-decoration: none;
        }
        .storefront-share-btn:hover { background: #f8fafc; color: #475569; border-color: #cbd5e1; box-shadow: 0 1px 3px rgba(0,0,0,0.06); }
        .storefront-share-btn svg { width: 16px; height: 16px; }
        .btn-share-link.copied { background: #ecfdf5; border-color: #059669; }
        .share-toast {
            position: fixed;
            bottom: 30px;
            left: 50%;
            transform: translateX(-50%) translateY(20px);
            background: #1e293b;
            color: #fff;
            padding: 10px 24px;
            border-radius: 8px;
            font-size: 13px;
            font-weight: 600;
            opacity: 0;
            transition: opacity 0.25s, transform 0.25s;
            z-index: 9999;
            pointer-events: none;
            box-shadow: 0 4px 16px rgba(0,0,0,0.2);
        }
        .share-toast.show { opacity: 1; transform: translateX(-50%) translateY(0); }
        /* Notification cards */
        .notification-section {
            margin-bottom: 28px;
        }
        .notification-card {
            background: #fff;
            border: 1px solid #e2e8f0;
            border-left: 4px solid #4f46e5;
            border-radius: 8px;
            padding: 14px 20px;
            margin-bottom: 10px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04);
        }
        .notification-card .notif-title {
            font-size: 14px;
            font-weight: 700;
            color: #1e293b;
            margin-bottom: 4px;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .notification-card .notif-content {
            font-size: 13px;
            color: #475569;
            line-height: 1.7;
        }

        /* Modal overlay */
        .modal-overlay {
            display: none;
            position: fixed;
            top: 0; left: 0;
            width: 100%; height: 100%;
            background: rgba(15,23,42,0.4);
            backdrop-filter: blur(4px);
            z-index: 1000;
            align-items: center;
            justify-content: center;
        }
        .modal-box {
            background: #fff;
            border-radius: 14px;
            padding: 28px 32px;
            max-width: 480px;
            width: 90%;
            box-shadow: 0 20px 60px rgba(0,0,0,0.15);
            position: relative;
            max-height: 90vh;
            overflow-y: auto;
            border: 1px solid #e2e8f0;
        }
        .modal-close {
            position: absolute;
            top: 14px; right: 18px;
            background: none;
            border: none;
            font-size: 20px;
            cursor: pointer;
            color: #64748b;
            width: 32px; height: 32px;
            border-radius: 8px;
            display: flex; align-items: center; justify-content: center;
            transition: background 0.15s;
        }
        .modal-close:hover { background: #f1f5f9; color: #1e293b; }
        .modal-title {
            font-size: 17px;
            font-weight: 700;
            color: #1e293b;
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
            color: #334155;
            display: block;
            margin-bottom: 5px;
            font-weight: 600;
        }
        .field-group input, .field-group select, .field-group textarea {
            width: 100%;
            padding: 9px 14px;
            border: 1px solid #cbd5e1;
            border-radius: 8px;
            font-size: 14px;
            background: #fff;
            transition: border-color 0.15s, box-shadow 0.15s;
            color: #1e293b;
        }
        .field-group input:focus, .field-group select:focus, .field-group textarea:focus {
            outline: none;
            border-color: #4f46e5;
            box-shadow: 0 0 0 3px rgba(79,70,229,0.12);
        }
        .field-group input.field-error { border-color: #dc2626; }
        .field-error-msg { font-size: 12px; color: #dc2626; margin-top: 3px; display: none; }
        .field-hint { font-size: 11px; color: #64748b; margin-top: 3px; }
        .msg-box {
            display: none;
            padding: 10px 14px;
            border-radius: 8px;
            font-size: 13px;
            margin-bottom: 14px;
            font-weight: 500;
        }
        .msg-success { background: #ecfdf5; color: #047857; border: 1px solid #a7f3d0; }
        .msg-error { background: #fef2f2; color: #b91c1c; border: 1px solid #fecaca; }

        /* Tab navigation */
        .tab-nav {
            display: flex;
            gap: 0;
            margin-bottom: 24px;
            border-bottom: 2px solid #e2e8f0;
        }
        .tab-btn {
            padding: 12px 28px;
            font-size: 14px;
            font-weight: 700;
            color: #64748b;
            background: none;
            border: none;
            border-bottom: 2px solid transparent;
            margin-bottom: -2px;
            cursor: pointer;
            transition: all 0.15s ease;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .tab-btn:hover { color: #334155; }
        .tab-btn.active {
            color: #4f46e5;
            border-bottom-color: #4f46e5;
        }
        .tab-panel { display: none; }
        .tab-panel.active { display: block; }

        /* TOP packs styles */
        .top-sort-toggle {
            display: flex;
            gap: 4px;
            margin-bottom: 18px;
            background: #f1f5f9;
            border-radius: 8px;
            padding: 3px;
            width: fit-content;
        }
        .top-sort-btn {
            padding: 7px 18px;
            border: none;
            border-radius: 6px;
            font-size: 13px;
            font-weight: 600;
            cursor: pointer;
            background: transparent;
            color: #64748b;
            transition: all 0.15s ease;
        }
        .top-sort-btn:hover { color: #334155; }
        .top-sort-btn.active {
            background: #4f46e5;
            color: #fff;
            box-shadow: 0 1px 3px rgba(79,70,229,0.3);
        }
        .top-table {
            width: 100%;
            border-collapse: collapse;
            background: #fff;
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06);
            border: 1px solid #e2e8f0;
        }
        .top-table thead th {
            background: #f8fafc;
            padding: 10px 14px;
            text-align: left;
            font-size: 12px;
            font-weight: 700;
            color: #64748b;
            text-transform: uppercase;
            letter-spacing: 0.3px;
            border-bottom: 1px solid #e2e8f0;
        }
        .top-table tbody td {
            padding: 10px 14px;
            font-size: 13px;
            color: #334155;
            border-bottom: 1px solid #f1f5f9;
        }
        .top-table tbody tr:hover { background: #f8fafc; }
        .top-table tbody tr:last-child td { border-bottom: none; }
        .top-rank {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            width: 28px;
            height: 28px;
            border-radius: 50%;
            font-size: 12px;
            font-weight: 700;
            background: #f1f5f9;
            color: #64748b;
        }
        .top-rank-1 { background: linear-gradient(135deg, #fbbf24, #f59e0b); color: #fff; }
        .top-rank-2 { background: linear-gradient(135deg, #cbd5e1, #94a3b8); color: #fff; }
        .top-rank-3 { background: linear-gradient(135deg, #d97706, #b45309); color: #fff; }
        .top-pack-name { font-weight: 600; color: #1e293b; }
    </style>
</head>
<body>
<div class="dashboard-wrap">
    {{if eq .SuccessMsg "withdraw"}}
    <div class="msg-box msg-success" style="display:block;margin-bottom:16px;" data-i18n="err_withdraw_submitted">✅ 提现申请已提交，请等待管理员审核付款。</div>
    {{end}}
    {{if eq .ErrorMsg "no_payment_info"}}
    <div class="msg-box msg-error" style="display:block;margin-bottom:16px;" data-i18n="err_no_payment_info">⚠️ 请先设置收款信息后再进行提现操作。</div>
    {{else if eq .ErrorMsg "not_author"}}
    <div class="msg-box msg-error" style="display:block;margin-bottom:16px;" data-i18n="err_not_author">⚠️ 仅作者可以申请提现。</div>
    {{else if eq .ErrorMsg "invalid_withdraw_amount"}}
    <div class="msg-box msg-error" style="display:block;margin-bottom:16px;" data-i18n="err_invalid_withdraw_amount">⚠️ 提现金额无效，请输入正确的数量。</div>
    {{else if eq .ErrorMsg "withdraw_disabled"}}
    <div class="msg-box msg-error" style="display:block;margin-bottom:16px;" data-i18n="err_withdraw_disabled">⚠️ 提现功能暂未开放。</div>
    {{else if eq .ErrorMsg "withdraw_exceeds_balance"}}
    <div class="msg-box msg-error" style="display:block;margin-bottom:16px;" data-i18n="err_withdraw_exceeds">⚠️ 提现数量超过可提现余额。</div>
    {{else if eq .ErrorMsg "withdraw_below_minimum"}}
    <div class="msg-box msg-error" style="display:block;margin-bottom:16px;" data-i18n="err_withdraw_below_min">⚠️ 扣除手续费后实付金额低于最低提现金额 100 元。</div>
    {{else if eq .ErrorMsg "internal"}}
    <div class="msg-box msg-error" style="display:block;margin-bottom:16px;" data-i18n="err_system">⚠️ 系统错误，请稍后重试。</div>
    {{end}}
    <div class="header">
        <div class="header-title">
            <span class="logo">📦</span>
            <h1 data-i18n="personal_center">个人中心</h1>
        </div>
        <div class="header-lang" id="headerLangSwitcher">
            <a href="/set-lang?lang=zh-CN&amp;redirect=%2Fuser%2F" class="{{if ne .DefaultLang "en-US"}}active{{end}}">中文</a>
            <a href="/set-lang?lang=en-US&amp;redirect=%2Fuser%2F" class="{{if eq .DefaultLang "en-US"}}active{{end}}">EN</a>
        </div>
    </div>
    <div class="user-info">
        <div class="user-detail">
            <div class="user-avatar">👤</div>
            <div class="user-email">
                <span class="label" data-i18n="email">邮箱</span>
                {{.User.Email}}
            </div>
            <div class="credits-info">
                <span class="label" data-i18n="credits_balance">Credits 余额</span>
                <span class="balance">{{printf "%.0f" .User.CreditsBalance}}</span>
            </div>
            {{if .AuthorData.StorefrontSlug}}
            <button class="btn-share-link" style="padding:7px 14px;font-size:13px;font-weight:600;border-radius:7px;" data-storefront-slug="{{.AuthorData.StorefrontSlug}}" onclick="copyStorefrontLink(this)" data-i18n="share_storefront">🏪 分享小铺</button>
            <a class="storefront-share-btn" id="storefrontShareX" href="#" target="_blank" rel="noopener" title="X (Twitter)"><svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/></svg></a>
            <a class="storefront-share-btn" id="storefrontShareLI" href="#" target="_blank" rel="noopener" title="LinkedIn"><svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433a2.062 2.062 0 01-2.063-2.065 2.064 2.064 0 112.063 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/></svg></a>
            {{end}}
            {{if .HasPassword}}
            <a class="btn btn-accent" href="/user/change-password" data-i18n="change_password">修改密码</a>
            {{else}}
            <a class="btn btn-accent" href="/user/set-password" data-i18n="set_password">设置密码</a>
            {{end}}
            <a class="btn btn-primary" href="/user/billing" data-i18n="billing_records">帐单记录</a>
            <a class="btn btn-ghost" href="/user/custom-product-orders" data-i18n="custom_product_orders">🛒 自定义商品购买记录</a>
            <button class="btn btn-warm" onclick="openPaymentSettingsModal()" data-i18n="payment_settings">收款设置</button>
            <button class="btn btn-secondary" onclick="alert(window._i18n('topup_coming_soon','功能开发中'))" data-i18n="topup">充值</button>
            <a class="btn btn-danger-outline" href="/user/logout" data-i18n="logout">退出登录</a>
        </div>
    </div>

    {{if .Notifications}}
    <div class="notification-section">
        <div class="section-title"><span class="icon">📢</span> <span data-i18n="system_messages">系统消息</span></div>
        {{range .Notifications}}
        <div class="notification-card">
            <div class="notif-title">📌 {{.Title}}</div>
            <div class="notif-content">{{.Content}}</div>
        </div>
        {{end}}
    </div>
    {{end}}

    <div class="tab-nav">
        <button class="tab-btn active" onclick="switchTab('customer')" id="tabBtnCustomer">🛒 <span data-i18n="customer_view">用户面板</span></button>
        {{if .AuthorData.IsAuthor}}
        <button class="tab-btn" onclick="switchTab('author')" id="tabBtnAuthor">✍️ <span data-i18n="author_view">作者面板</span></button>
        <a class="tab-btn" href="/user/storefront/" id="tabBtnStorefront" style="text-decoration:none;">🏪 <span data-i18n="storefront_manage">小铺管理</span></a>
        {{end}}
        <button class="tab-btn" onclick="switchTab('top')" id="tabBtnTop">🏆 <span data-i18n="top_packs_tab">TOP分析包</span></button>
    </div>

    <div id="tabCustomer" class="tab-panel active">
    <div class="section">
        <div class="section-title"><span class="icon">🛒</span> <span data-i18n="purchased_packs">已购买的分析包</span></div>
        {{if .PurchasedPacks}}
        <div class="pack-grid">
            {{range .PurchasedPacks}}
            <div class="pack-card">
                <div class="pack-card-accent{{if eq .ShareMode "free"}} accent-free{{else if eq .ShareMode "per_use"}} accent-per-use{{else if eq .ShareMode "time_limited"}} accent-time-limited{{else if eq .ShareMode "subscription"}} accent-subscription{{end}}"></div>
                <div class="pack-card-body">
                    <div class="pack-header">
                        <div class="pack-name">{{.PackName}}</div>
                        {{if eq .ShareMode "free"}}<span class="tag tag-free" data-i18n="free">免费</span>
                        {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use" data-i18n="per_use">按次付费</span>
                        {{else if eq .ShareMode "time_limited"}}<span class="tag tag-time-limited" data-i18n="time_limited">限时</span>
                        {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription" data-i18n="subscription">订阅</span>
                        {{end}}
                    </div>
                    <div class="pack-category">{{.CategoryName}} <span class="version-badge">v{{.Version}}</span></div>
                    <div class="pack-info-grid">
                        {{if .AuthorName}}<span class="pack-info-item"><span data-i18n="author">作者：</span><span class="info-value">{{.AuthorName}}</span></span><span class="pack-info-sep">·</span>{{end}}
                        <span class="pack-info-item"><span data-i18n="download_count">下载</span> <span class="info-value">{{.DownloadCount}}</span></span>
                        {{if .SourceName}}<br><span class="pack-info-item"><span data-i18n="datasource">数据源：</span><span class="info-value">{{.SourceName}}</span></span>{{end}}
                    </div>
                    <div class="pack-date">{{if eq .ShareMode "subscription"}}<span data-i18n="subscription_start">订阅起始：</span>{{else}}<span data-i18n="download_time">下载时间：</span>{{end}}{{.PurchaseDate}}</div>
                    {{if .ExpiresAt}}<div class="pack-expires{{if eq .ShareMode "subscription"}} subscription-expires{{end}}">{{if eq .ShareMode "subscription"}}<span data-i18n="subscription_expires">订阅到期：</span>{{else}}<span data-i18n="expires_at">到期时间：</span>{{end}}{{.ExpiresAt}}</div>{{end}}
                    {{if eq .ShareMode "per_use"}}<div class="pack-usage"><span class="usage-progress{{if eq .UsedCount .TotalPurchased}} usage-exhausted{{end}}"><span data-i18n="used_count">已使用</span> {{.UsedCount}}/{{.TotalPurchased}}</span></div>{{end}}
                </div>
                <div class="pack-actions">
                    {{if or (eq .ShareMode "per_use") (eq .ShareMode "subscription")}}
                    <button class="btn btn-primary btn-sm"
                        data-listing-id="{{.ListingID}}"
                        data-pack-name="{{.PackName}}"
                        data-share-mode="{{.ShareMode}}"
                        data-credits-price="{{.CreditsPrice}}"
                        onclick="openRenewModal(this)" data-i18n="renew">续费</button>
                    {{end}}
                    <button class="btn-danger-sm"
                        data-listing-id="{{.ListingID}}"
                        data-pack-name="{{.PackName}}"
                        onclick="openDeleteModal(this)" data-i18n="delete">删除</button>
                </div>
            </div>
            {{end}}
        </div>
        {{else}}
        <div class="empty-state">
            <div class="icon">📭</div>
            <p data-i18n="no_purchased_packs">暂无已购买的分析包</p>
        </div>
        {{end}}
    </div>
    </div><!-- end tabCustomer -->

    {{if .AuthorData.IsAuthor}}
    <div id="tabAuthor" class="tab-panel">
    <div class="author-panel" style="border-top:none;margin-top:0;padding-top:0;">
        <div class="author-panel-title">✍️ <span data-i18n="author_panel">作者面板</span></div>

        <div class="author-stats">
            <div class="stat-card">
                <div class="stat-label" data-i18n="actual_revenue">实际收入 Credits（分成 {{printf "%.0f" .AuthorData.RevenueSplitPct}}%）</div>
                <div class="stat-value revenue">{{printf "%.0f" .AuthorData.TotalRevenue}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label" data-i18n="unwithdrawn_credits">未提现 Credits</div>
                <div class="stat-value unwithdrawn">{{printf "%.0f" .AuthorData.UnwithdrawnCredits}}</div>
                <div class="stat-actions">
                    {{if .AuthorData.WithdrawalEnabled}}
                    <button class="btn btn-warm" onclick="openWithdrawModal()" data-i18n="withdraw">提现</button>
                    {{else}}
                    <button class="btn btn-secondary" disabled data-i18n-title="withdraw_disabled" title="提现功能暂未开放" data-i18n="withdraw">提现</button>
                    <span class="withdraw-hint" data-i18n="withdraw_disabled">提现功能暂未开放</span>
                    {{end}}
                    <a class="btn btn-ghost" href="javascript:void(0)" onclick="openWithdrawRecordsModal()" data-i18n="withdraw_records">提现记录</a>
                </div>
            </div>
        </div>

        <div class="section-title"><span class="icon">📤</span> <span data-i18n="shared_packs">已共享分析包</span></div>
        {{if .AuthorData.AuthorPacks}}
        <div class="author-table-wrap">
            <table class="author-table">
                <thead>
                    <tr>
                        <th data-i18n="pack_name">名称</th>
                        <th data-i18n="version">版本</th>
                        <th data-i18n="pricing_model">定价模式</th>
                        <th data-i18n="unit_price">单价</th>
                        <th data-i18n="review_status">审核状态</th>
                        <th data-i18n="sales_count">销量</th>
                        <th data-i18n="actual_income">实际收入</th>
                        <th data-i18n="actions">操作</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .AuthorData.AuthorPacks}}
                    <tr>
                        <td style="font-weight:500;color:#475569;">{{.PackName}}</td>
                        <td><span class="version-badge">v{{.Version}}</span></td>
                        <td>
                            {{if eq .ShareMode "free"}}<span data-i18n="free">免费</span>
                            {{else if eq .ShareMode "per_use"}}<span data-i18n="per_use">按次付费</span>
                            {{else if eq .ShareMode "subscription"}}<span data-i18n="subscription">订阅</span>
                            {{else}}{{.ShareMode}}
                            {{end}}
                        </td>
                        <td>{{if eq .ShareMode "free"}}-{{else}}{{.CreditsPrice}} Credits{{end}}</td>
                        <td>
                            {{if eq .Status "pending"}}<span class="status-badge status-pending" data-i18n="pending_review">待审核</span>
                            {{else if eq .Status "published"}}<span class="status-badge status-published" data-i18n="published">已发布</span>
                            {{else if eq .Status "rejected"}}<span class="status-badge status-rejected" data-i18n="rejected">已拒绝</span>
                            {{else if eq .Status "delisted"}}<span class="status-badge status-delisted" data-i18n="delisted">已下架</span>
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
                                    onclick="openPurchaseDetailsModal(this)" data-i18n="details">明细</button>
                                <button class="btn btn-ghost btn-sm"
                                    data-listing-id="{{.ListingID}}"
                                    data-pack-name="{{.PackName}}"
                                    data-pack-desc="{{.PackDesc}}"
                                    data-share-mode="{{.ShareMode}}"
                                    data-credits-price="{{.CreditsPrice}}"
                                    onclick="openEditPackModal(this)" data-i18n="edit">编辑</button>
                                {{if eq .Status "published"}}
                                <button class="btn-danger-sm"
                                    data-listing-id="{{.ListingID}}"
                                    data-pack-name="{{.PackName}}"
                                    onclick="openAuthorDelistModal(this)" data-i18n="delist">下架</button>
                                <button class="btn-share-link btn-sm" title="复制分享链接"
                                    data-share-token="{{.ShareToken}}"
                                    onclick="copyShareLink(this)">🔗</button>
                                {{end}}
                                {{if eq .Status "rejected"}}
                                <button class="btn-danger-sm"
                                    data-listing-id="{{.ListingID}}"
                                    data-pack-name="{{.PackName}}"
                                    onclick="openAuthorDeleteModal(this)" data-i18n="delete">删除</button>
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
            <p data-i18n="no_shared_packs">暂无已共享的分析包</p>
        </div>
        {{end}}
    </div>
    </div><!-- end tabAuthor -->
    {{end}}

    <div id="tabTop" class="tab-panel">
    <div class="section">
        <div class="section-title"><span class="icon">🏆</span> <span data-i18n="top_packs_title">TOP 分析包排行榜</span></div>
        <div class="top-sort-toggle">
            <button class="top-sort-btn active" id="topSortDownloads" onclick="switchTopSort('downloads')" data-i18n="top_sort_downloads">按下载次数</button>
            <button class="top-sort-btn" id="topSortRevenue" onclick="switchTopSort('revenue')" data-i18n="top_sort_revenue">按销售额</button>
        </div>

        <div id="topListDownloads" class="top-list-container">
        {{if .TopPacksByDownloads}}
        <table class="top-table">
            <thead>
                <tr>
                    <th style="width:50px;" data-i18n="rank_col">排名</th>
                    <th data-i18n="pack_name">包名</th>
                    <th data-i18n="author">作者</th>
                    <th data-i18n="category">分类</th>
                    <th data-i18n="payment_mode_col">定价模式</th>
                    <th data-i18n="price_col">单价</th>
                    <th data-i18n="download_count_col">下载量</th>
                    <th data-i18n="total_revenue_col">销售额</th>
                </tr>
            </thead>
            <tbody>
                {{range .TopPacksByDownloads}}
                <tr>
                    <td><span class="top-rank{{if le .Rank 3}} top-rank-{{.Rank}}{{end}}">{{.Rank}}</span></td>
                    <td class="top-pack-name">{{.PackName}}</td>
                    <td>{{.AuthorName}}</td>
                    <td>{{.CategoryName}}</td>
                    <td>{{if eq .ShareMode "free"}}<span class="tag tag-free" data-i18n="free">免费</span>
                        {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use" data-i18n="per_use">按次付费</span>
                        {{else if eq .ShareMode "time_limited"}}<span class="tag tag-time-limited" data-i18n="time_limited">限时</span>
                        {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription" data-i18n="subscription">订阅</span>
                        {{end}}</td>
                    <td>{{if eq .ShareMode "free"}}-{{else}}{{.CreditsPrice}} Credits{{end}}</td>
                    <td style="font-weight:700;color:#4f46e5;">{{.DownloadCount}}</td>
                    <td>{{printf "%.0f" .TotalRevenue}} Credits</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{else}}
        <div class="empty-state">
            <div class="icon">📊</div>
            <p data-i18n="no_top_packs">暂无排行数据</p>
        </div>
        {{end}}
        </div>

        <div id="topListRevenue" class="top-list-container" style="display:none;">
        {{if .TopPacksByRevenue}}
        <table class="top-table">
            <thead>
                <tr>
                    <th style="width:50px;" data-i18n="rank_col">排名</th>
                    <th data-i18n="pack_name">包名</th>
                    <th data-i18n="author">作者</th>
                    <th data-i18n="category">分类</th>
                    <th data-i18n="payment_mode_col">定价模式</th>
                    <th data-i18n="price_col">单价</th>
                    <th data-i18n="download_count_col">下载量</th>
                    <th data-i18n="total_revenue_col">销售额</th>
                </tr>
            </thead>
            <tbody>
                {{range .TopPacksByRevenue}}
                <tr>
                    <td><span class="top-rank{{if le .Rank 3}} top-rank-{{.Rank}}{{end}}">{{.Rank}}</span></td>
                    <td class="top-pack-name">{{.PackName}}</td>
                    <td>{{.AuthorName}}</td>
                    <td>{{.CategoryName}}</td>
                    <td>{{if eq .ShareMode "free"}}<span class="tag tag-free" data-i18n="free">免费</span>
                        {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use" data-i18n="per_use">按次付费</span>
                        {{else if eq .ShareMode "time_limited"}}<span class="tag tag-time-limited" data-i18n="time_limited">限时</span>
                        {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription" data-i18n="subscription">订阅</span>
                        {{end}}</td>
                    <td>{{if eq .ShareMode "free"}}-{{else}}{{.CreditsPrice}} Credits{{end}}</td>
                    <td>{{.DownloadCount}}</td>
                    <td style="font-weight:700;color:#059669;">{{printf "%.0f" .TotalRevenue}} Credits</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{else}}
        <div class="empty-state">
            <div class="icon">📊</div>
            <p data-i18n="no_top_packs">暂无排行数据</p>
        </div>
        {{end}}
        </div>
    </div>
    </div><!-- end tabTop -->
</div>

<!-- Payment Settings Modal -->
<div id="paymentSettingsModal" class="modal-overlay">
  <div class="modal-box">
    <button onclick="closePaymentSettingsModal()" class="modal-close">&times;</button>
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:4px;">
      <div style="display:flex;align-items:center;gap:10px;">
        <h3 class="modal-title" style="margin-bottom:0;" data-i18n="payment_settings">收款设置</h3>
        <span style="font-size:13px;color:#6366f1;font-weight:600;background:#eef2ff;padding:2px 10px;border-radius:8px;white-space:nowrap;"><span data-i18n="revenue_split">分成</span> {{printf "%.0f" .AuthorData.RevenueSplitPct}}%</span>
      </div>
      <button class="btn btn-sm" style="font-size:12px;padding:4px 12px;background:#f0f9ff;color:#0369a1;border:1px solid #bae6fd;border-radius:6px;cursor:pointer;" onclick="openFeeRatesDialog()" data-i18n="view_fee_rates">查看费率</button>
    </div>
    <div id="paymentSettingsMsg" class="msg-box"></div>
    <div class="field-group">
      <label data-i18n="payment_method">收款方式</label>
      <select id="paymentType" onchange="onPaymentTypeChange()">
        <option value="" data-i18n="select_payment_method">请选择收款方式</option>
        <option value="paypal" data-i18n="paypal">PayPal</option>
        <option value="wechat" data-i18n="wechat">微信</option>
        <option value="alipay" data-i18n="alipay">AliPay</option>
        <option value="check" data-i18n="check">支票</option>
        <option value="wire_transfer" data-i18n="wire_transfer">国际电汇 (SWIFT)</option>
        <option value="bank_card_us" data-i18n="bank_card_us">美国银行卡 (ACH)</option>
        <option value="bank_card_eu" data-i18n="bank_card_eu">欧洲银行卡 (SEPA)</option>
        <option value="bank_card_cn" data-i18n="bank_card_cn">中国银行卡 (CNAPS)</option>
      </select>
    </div>
    <div id="paymentFieldsAccount" style="display:none;">
      <div class="field-group">
        <label data-i18n="account">帐号</label>
        <input type="text" id="paymentAccount" data-i18n-placeholder="enter_account" placeholder="请输入帐号">
        <div class="field-error-msg" id="paymentAccountError" data-i18n="account_required">帐号不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="username">用户名</label>
        <input type="text" id="paymentUsername" data-i18n-placeholder="enter_account" placeholder="请输入用户名">
        <div class="field-error-msg" id="paymentUsernameError" data-i18n="username_required">用户名不能为空</div>
      </div>
    </div>

    <div id="paymentFieldsCheck" style="display:none;">
      <div class="field-group">
        <label data-i18n="full_legal_name">法定全名</label>
        <input type="text" id="paymentCheckFullLegalName" data-i18n-placeholder="enter_full_legal_name" placeholder="请输入法定全名">
        <div class="field-hint" data-i18n="must_match_bank">必须与银行账户名一致，避免缩写</div>
        <div class="field-error-msg" id="paymentCheckFullLegalNameError" data-i18n="full_legal_name_required">法定全名不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="province">省份</label>
        <input type="text" id="paymentCheckProvince" data-i18n-placeholder="enter_province" placeholder="请输入省份">
        <div class="field-error-msg" id="paymentCheckProvinceError" data-i18n="province_required">省份不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="city">城市</label>
        <input type="text" id="paymentCheckCity" data-i18n-placeholder="enter_city" placeholder="请输入城市">
        <div class="field-error-msg" id="paymentCheckCityError" data-i18n="city_required">城市不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="district">区县</label>
        <input type="text" id="paymentCheckDistrict" data-i18n-placeholder="enter_district" placeholder="请输入区县">
        <div class="field-error-msg" id="paymentCheckDistrictError" data-i18n="district_required">区县不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="street_address">街道地址</label>
        <input type="text" id="paymentCheckStreetAddress" data-i18n-placeholder="enter_street_address" placeholder="请输入街道地址">
        <div class="field-error-msg" id="paymentCheckStreetAddressError" data-i18n="street_address_required">街道地址不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="postal_code">邮政编码</label>
        <input type="text" id="paymentCheckPostalCode" data-i18n-placeholder="enter_postal_code" placeholder="请输入邮政编码">
        <div class="field-error-msg" id="paymentCheckPostalCodeError" data-i18n="postal_code_required">邮政编码不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="recipient_phone">收件人电话</label>
        <input type="text" id="paymentCheckPhone" data-i18n-placeholder="enter_phone" placeholder="请输入收件人电话">
        <div class="field-error-msg" id="paymentCheckPhoneError" data-i18n="phone_required">收件人电话不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="memo_optional">备注（可选）</label>
        <input type="text" id="paymentCheckMemo" data-i18n-placeholder="memo_placeholder" placeholder="如支付房租、还款等用途">
      </div>
    </div>
    <div id="paymentFieldsWireTransfer" style="display:none;">
      <div class="field-group">
        <label data-i18n="beneficiary_name">收款人全名 (Full Name)</label>
        <input type="text" id="paymentBeneficiaryName" data-i18n-placeholder="beneficiary_name_placeholder" placeholder="必须与银行开户证件一致">
        <div class="field-error-msg" id="paymentBeneficiaryNameError" data-i18n="beneficiary_name_required">收款人全名不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="beneficiary_address">收款人地址 (Full Address)</label>
        <input type="text" id="paymentBeneficiaryAddress" data-i18n-placeholder="beneficiary_address_placeholder" placeholder="街道、城市、邮编、国家">
        <div class="field-error-msg" id="paymentBeneficiaryAddressError" data-i18n="beneficiary_address_required">收款人地址不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="bank_name">银行名称 (Bank Name)</label>
        <input type="text" id="paymentWireBankName" data-i18n-placeholder="bank_name_placeholder" placeholder="英文全称">
        <div class="field-error-msg" id="paymentWireBankNameError" data-i18n="bank_name_required">银行名称不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="swift_code">SWIFT / BIC Code</label>
        <input type="text" id="paymentSwiftCode" data-i18n-placeholder="swift_code_placeholder" placeholder="8或11位国际银行识别码">
        <div class="field-error-msg" id="paymentSwiftCodeError" data-i18n="swift_code_required">SWIFT Code不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="account_or_iban">收款人账号 / IBAN</label>
        <input type="text" id="paymentWireAccountNumber" data-i18n-placeholder="account_or_iban_placeholder" placeholder="账号或IBAN码">
        <div class="field-error-msg" id="paymentWireAccountNumberError" data-i18n="account_number_required">账号不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="bank_branch_address">银行分行地址（选填）</label>
        <input type="text" id="paymentBankBranchAddress" data-i18n-placeholder="bank_branch_placeholder" placeholder="城市名和具体分行">
      </div>
    </div>
    <div id="paymentFieldsBankUS" style="display:none;">
      <div class="field-group">
        <label data-i18n="legal_name">收款人姓名 (Legal Name)</label>
        <input type="text" id="paymentUSLegalName" data-i18n-placeholder="enter_legal_name" placeholder="请输入Legal Name">
        <div class="field-error-msg" id="paymentUSLegalNameError" data-i18n="name_required">姓名不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="routing_number">路由号码 (Routing Number)</label>
        <input type="text" id="paymentRoutingNumber" data-i18n-placeholder="routing_placeholder" placeholder="9位数字">
        <div class="field-error-msg" id="paymentRoutingNumberError" data-i18n="routing_required">路由号码不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="account_number">账号 (Account Number)</label>
        <input type="text" id="paymentUSAccountNumber" data-i18n-placeholder="enter_account_number" placeholder="请输入账号">
        <div class="field-error-msg" id="paymentUSAccountNumberError" data-i18n="account_number_required">账号不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="account_type">账户类型</label>
        <select id="paymentUSAccountType">
          <option value="checking" data-i18n="checking_account">Checking (支票账户)</option>
          <option value="savings" data-i18n="savings_account">Savings (储蓄账户)</option>
        </select>
        <div class="field-error-msg" id="paymentUSAccountTypeError" data-i18n="select_account_type">请选择账户类型</div>
      </div>
    </div>
    <div id="paymentFieldsBankEU" style="display:none;">
      <div class="field-group">
        <label data-i18n="legal_name">收款人姓名 (Legal Name)</label>
        <input type="text" id="paymentEULegalName" data-i18n-placeholder="enter_legal_name" placeholder="请输入Legal Name">
        <div class="field-error-msg" id="paymentEULegalNameError" data-i18n="name_required">姓名不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="iban">IBAN</label>
        <input type="text" id="paymentIBAN" data-i18n-placeholder="iban_placeholder" placeholder="以国家代码开头（如 DE..., FR...）">
        <div class="field-error-msg" id="paymentIBANError" data-i18n="iban_required">IBAN不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="bic_swift">BIC / SWIFT</label>
        <input type="text" id="paymentEUBicSwift" data-i18n-placeholder="bic_swift_placeholder" placeholder="银行识别码">
        <div class="field-error-msg" id="paymentEUBicSwiftError" data-i18n="bic_swift_required">BIC/SWIFT不能为空</div>
      </div>
    </div>
    <div id="paymentFieldsBankCN" style="display:none;">
      <div class="field-group">
        <label data-i18n="cn_real_name">收款人姓名（中文实名）</label>
        <input type="text" id="paymentCNRealName" data-i18n-placeholder="cn_real_name_placeholder" placeholder="必须为中文实名">
        <div class="field-error-msg" id="paymentCNRealNameError" data-i18n="cn_real_name_required">姓名不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="cn_card_number">收款卡号</label>
        <input type="text" id="paymentCNCardNumber" data-i18n-placeholder="cn_card_placeholder" placeholder="16-19位银行卡号">
        <div class="field-error-msg" id="paymentCNCardNumberError" data-i18n="cn_card_required">卡号不能为空</div>
      </div>
      <div class="field-group">
        <label data-i18n="cn_bank_branch">开户银行（具体到分行）</label>
        <input type="text" id="paymentCNBankBranch" data-i18n-placeholder="cn_bank_placeholder" placeholder="如：中国银行北京分行XX支行">
        <div class="field-error-msg" id="paymentCNBankBranchError" data-i18n="cn_bank_required">开户银行不能为空</div>
      </div>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closePaymentSettingsModal()" data-i18n="cancel">取消</button>
      <button class="btn btn-warm" onclick="savePaymentSettings()" data-i18n="save">保存</button>
    </div>
  </div>
</div>

<!-- Fee Rates Dialog -->
<div id="feeRatesDialog" class="modal-overlay" style="display:none;z-index:1100;">
  <div class="modal-box" style="max-width:400px;">
    <button onclick="closeFeeRatesDialog()" class="modal-close">&times;</button>
    <h3 class="modal-title" data-i18n="fee_rates">提现费率</h3>
    <div id="feeRatesCurrentType" style="font-size:13px;color:#64748b;margin-bottom:10px;"></div>
    <div id="feeRatesDialogContent" style="font-size:13px;color:#475569;" data-i18n="loading">加载中...</div>
    <div class="modal-actions" style="margin-top:12px;">
      <button class="btn btn-secondary" onclick="closeFeeRatesDialog()" data-i18n="close">关闭</button>
    </div>
  </div>
</div>

<!-- Withdraw Modal -->
<div id="withdrawModal" class="modal-overlay">
  <div class="modal-box" style="max-width:400px;padding:24px;">
    <button onclick="closeWithdrawModal()" class="modal-close">&times;</button>
    <h3 style="font-size:16px;font-weight:600;color:#334155;margin-bottom:14px;" data-i18n="credits_withdraw">Credits 提现</h3>
    <div id="withdrawNoPaymentWarning" style="display:none;padding:10px 14px;background:#fff7ed;border:1px solid #fed7aa;border-radius:8px;margin-bottom:12px;">
      <div style="font-size:13px;color:#ea580c;font-weight:600;margin-bottom:2px;">⚠️ <span data-i18n="no_payment_warning">未设置收款信息</span></div>
      <div style="font-size:12px;color:#9a3412;" data-i18n="set_payment_first">请先设置收款方式后再进行提现操作。</div>
      <button class="btn btn-warm btn-sm" style="margin-top:6px;font-size:11px;" onclick="closeWithdrawModal();openPaymentSettingsModal();" data-i18n="go_set">去设置</button>
    </div>
    <div id="withdrawFormContent">
      <div id="withdrawPaymentInfo" style="display:none;padding:8px 12px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:8px;margin-bottom:10px;font-size:12px;color:#475569;justify-content:space-between;flex-wrap:wrap;gap:4px;">
        <span><span data-i18n="revenue_split">分成</span> <strong id="withdrawSplitPctLabel" style="color:#4338ca;">{{printf "%.0f" .AuthorData.RevenueSplitPct}}%</strong></span>
        <span><span data-i18n="payment_method">收款</span> <strong id="withdrawPaymentTypeLabel" style="color:#166534;"></strong></span>
        <span><span data-i18n="fee">费率</span> <strong id="withdrawFeeRateLabel" style="color:#ea580c;"></strong></span>
      </div>
      <div style="display:flex;gap:12px;font-size:12px;color:#718096;margin-bottom:10px;">
        <span><span data-i18n="withdrawable">可提现</span>：<span style="color:#f59e0b;font-weight:600;">{{printf "%.0f" .AuthorData.UnwithdrawnCredits}}</span> Credits</span>
        <span><span data-i18n="exchange_rate">汇率</span>：1C = <span id="withdrawCashRate" style="font-weight:500;">{{printf "%.2f" .AuthorData.CreditCashRate}}</span><span data-i18n="yuan">元</span></span>
      </div>
      <div style="margin-bottom:10px;">
        <label style="font-size:12px;color:#4a5568;display:block;margin-bottom:4px;font-weight:500;" data-i18n="withdraw_amount">提现数量</label>
        <input id="withdrawCreditsInput" type="number" min="1" max="{{printf "%.0f" .AuthorData.UnwithdrawnCredits}}" step="1" data-i18n-placeholder="enter_credits_amount" placeholder="输入 Credits 数量" oninput="calcWithdrawCash()" style="width:100%;padding:8px 12px;border:1px solid #e2e8f0;border-radius:8px;font-size:13px;">
      </div>
      <div id="withdrawFormulaBox" style="display:none;padding:10px 12px;background:#fafbfe;border:1px solid #eef2ff;border-radius:8px;margin-bottom:10px;font-size:12px;font-family:monospace;color:#475569;line-height:1.8;"></div>
      <div id="withdrawNetResult" style="display:none;font-size:15px;font-weight:700;color:#10b981;margin-bottom:6px;"></div>
      <div id="withdrawWarning" style="display:none;padding:6px 10px;background:#fff7ed;border:1px solid #fed7aa;border-radius:8px;margin-bottom:8px;font-size:12px;color:#9a3412;"></div>
      <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:12px;">
        <button class="btn btn-secondary" onclick="closeWithdrawModal()" style="padding:6px 14px;font-size:13px;" data-i18n="cancel">取消</button>
        <button class="btn btn-warm" id="withdrawSubmitBtn" onclick="submitWithdraw()" style="padding:6px 14px;font-size:13px;" data-i18n="confirm_withdraw">确认提现</button>
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
    <h3 style="font-size:16px;font-weight:700;color:#1e293b;margin-bottom:16px;">💰 <span data-i18n="withdraw_records">提现记录</span></h3>
    <div id="withdrawRecordsContent" style="max-height:400px;overflow-y:auto;">
      <div style="text-align:center;padding:30px;color:#94a3b8;" data-i18n="loading">加载中...</div>
    </div>
    <div id="withdrawRecordsTotalRow" style="display:none;text-align:right;padding:12px 0 0;border-top:2px solid #e2e8f0;margin-top:12px;font-size:14px;font-weight:600;color:#1e293b;">
      <span data-i18n="total_cash_withdrawn">总计提现现金</span>：<span id="withdrawRecordsTotalCash" style="color:#059669;font-size:16px;"></span>
    </div>
  </div>
</div>

<!-- Edit Pack Modal -->
<div id="editPackModal" class="modal-overlay">
  <div class="modal-box">
    <button onclick="closeEditPackModal()" class="modal-close">&times;</button>
    <h3 class="modal-title" data-i18n="edit_pack">编辑分析包</h3>
    <form id="editPackForm" method="POST" action="/user/author/edit-pack">
      <input type="hidden" name="listing_id" id="editListingId">
      <div class="field-group">
        <label data-i18n="pack_name">名称</label>
        <input type="text" name="pack_name" id="editPackName" required>
      </div>
      <div class="field-group">
        <label data-i18n="description">描述</label>
        <textarea name="pack_description" id="editPackDesc" rows="3" style="resize:vertical;"></textarea>
      </div>
      <div class="field-group">
        <label data-i18n="pricing_model">定价模式</label>
        <select name="share_mode" id="editShareMode" onchange="onEditShareModeChange()">
          <option value="free" data-i18n="free">免费</option>
          <option value="per_use" data-i18n="per_use">按次付费</option>
          <option value="subscription" data-i18n="subscription">订阅</option>
        </select>
      </div>
      <div id="editPriceSection" style="display:none;">
        <div class="field-group">
          <label data-i18n="price_credits">价格 (Credits)</label>
          <input type="number" name="credits_price" id="editCreditsPrice" min="0">
          <div class="field-hint" id="editPriceHint"></div>
        </div>
      </div>
      <div style="margin-top:12px;padding:10px 12px;background:#fffbeb;border:1px solid #fde68a;border-radius:6px;">
        <p style="font-size:12px;color:#92400e;margin:0;" data-i18n="edit_warning">⚠ 修改已上架的分析包信息后，该分析包将被下架并需要重新提交审核后才能再次上架。</p>
      </div>
      <div class="modal-actions">
        <button type="button" class="btn btn-secondary" onclick="closeEditPackModal()" data-i18n="cancel">取消</button>
        <button type="button" class="btn btn-primary" onclick="confirmEditPack()" data-i18n="confirm_edit">确认修改</button>
      </div>
    </form>
  </div>
</div>

<!-- Renew Modal -->
<div id="renewModal" class="modal-overlay">
  <div class="modal-box" style="max-width:420px;">
    <button onclick="closeRenewModal()" class="modal-close">&times;</button>
    <h3 id="renewTitle" class="modal-title" data-i18n="renew">续费</h3>
    <div id="renewPackName" style="font-size:14px;color:#4a5568;margin-bottom:12px;"></div>
    <div id="renewUnitPrice" style="font-size:13px;color:#718096;margin-bottom:16px;"></div>
    <div id="renewPerUseSection" style="display:none;">
      <div class="field-group">
        <label data-i18n="buy_count">购买次数</label>
        <input id="renewQuantity" type="number" min="1" value="1" oninput="calcPerUseCost()">
      </div>
    </div>
    <div id="renewSubSection" style="display:none;">
      <label style="font-size:13px;color:#4a5568;display:block;margin-bottom:10px;" data-i18n="renew_duration">续费时长</label>
      <div style="display:flex;flex-direction:column;gap:10px;margin-bottom:16px;">
        <label style="display:flex;align-items:center;gap:8px;font-size:14px;color:#4a5568;cursor:pointer;">
          <input type="radio" name="renewMonths" value="1" checked onchange="calcSubCost()"> <span data-i18n="monthly_1">按月（1个月）</span>
        </label>
        <label style="display:flex;align-items:center;gap:8px;font-size:14px;color:#4a5568;cursor:pointer;">
          <input type="radio" name="renewMonths" value="12" onchange="calcSubCost()"> <span data-i18n="yearly_12">按年（12个月付费，赠送2个月）</span>
        </label>
      </div>
    </div>
    <div id="renewTotalCost" style="font-size:16px;font-weight:700;color:#818cf8;margin-bottom:20px;"></div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeRenewModal()" data-i18n="cancel">取消</button>
      <button class="btn btn-primary" onclick="submitRenew()" data-i18n="confirm_renew">确认续费</button>
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
    <h3 class="modal-title" data-i18n="delete_pack">删除分析包</h3>
    <div style="font-size:14px;color:#4a5568;margin-bottom:8px;"><span data-i18n="pack_label">分析包</span>：<span id="deletePackName" style="font-weight:600;"></span></div>
    <div style="font-size:13px;color:#ef4444;margin-bottom:20px;" data-i18n="delete_pack_confirm">确定要删除该分析包吗？删除后将不再显示在已购列表中。</div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeDeleteModal()" data-i18n="cancel">取消</button>
      <button class="btn btn-danger-outline" onclick="submitDelete()" style="background:#ef4444;color:#fff;border:none;" data-i18n="confirm_delete">确认删除</button>
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
    <h3 class="modal-title"><span data-i18n="purchase_details">购买明细</span> - <span id="purchaseDetailsPackName"></span></h3>
    <div id="purchaseDetailsLoading" style="text-align:center;padding:20px;color:#94a3b8;" data-i18n="loading">加载中...</div>
    <div id="purchaseDetailsContent" style="display:none;">
      <div id="purchaseDetailsSplitInfo" style="font-size:12px;color:#718096;margin-bottom:12px;"></div>
      <div style="overflow-x:auto;">
        <table class="author-table" style="font-size:12px;">
          <thead>
            <tr>
              <th data-i18n="buyer">买家</th>
              <th data-i18n="payment_amount">支付金额</th>
              <th data-i18n="author_income">作者收入</th>
              <th data-i18n="time">时间</th>
            </tr>
          </thead>
          <tbody id="purchaseDetailsBody"></tbody>
        </table>
      </div>
      <div id="purchaseDetailsEmpty" style="display:none;text-align:center;padding:20px;color:#a0aec0;font-size:13px;" data-i18n="no_purchase_records">暂无购买记录</div>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closePurchaseDetailsModal()" data-i18n="close">关闭</button>
    </div>
  </div>
</div>

<!-- Author Delete Shared Pack Modal -->
<div id="authorDeleteModal" class="modal-overlay">
  <div class="modal-box" style="max-width:420px;">
    <button onclick="closeAuthorDeleteModal()" class="modal-close">&times;</button>
    <h3 class="modal-title" data-i18n="delete_shared_pack">删除已共享分析包</h3>
    <div style="font-size:14px;color:#4a5568;margin-bottom:8px;"><span data-i18n="pack_label">分析包</span>：<span id="authorDeletePackName" style="font-weight:600;"></span></div>
    <div style="font-size:13px;color:#ef4444;margin-bottom:20px;" data-i18n="delete_rejected_confirm">确定要删除该已拒绝的分析包吗？删除后将无法恢复。</div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeAuthorDeleteModal()" data-i18n="cancel">取消</button>
      <button class="btn btn-danger-outline" onclick="submitAuthorDelete()" style="background:#ef4444;color:#fff;border:none;" data-i18n="confirm_delete">确认删除</button>
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
    <h3 class="modal-title" data-i18n="delist_pack">下架分析包</h3>
    <div style="font-size:14px;color:#4a5568;margin-bottom:8px;"><span data-i18n="pack_label">分析包</span>：<span id="delistPackName" style="font-weight:600;"></span></div>
    <div style="font-size:13px;color:#ef4444;margin-bottom:20px;" data-i18n="delist_confirm">确认要下架此分析包吗？下架后用户将无法在市场中看到此分析包。</div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeAuthorDelistModal()" data-i18n="cancel">取消</button>
      <button class="btn btn-danger-outline" onclick="submitAuthorDelist()" style="background:#ef4444;color:#fff;border:none;" data-i18n="confirm_delist">确认下架</button>
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

/* TOP packs sort switching */
function switchTopSort(mode) {
    document.getElementById('topListDownloads').style.display = mode === 'downloads' ? '' : 'none';
    document.getElementById('topListRevenue').style.display = mode === 'revenue' ? '' : 'none';
    document.getElementById('topSortDownloads').classList.toggle('active', mode === 'downloads');
    document.getElementById('topSortRevenue').classList.toggle('active', mode === 'revenue');
}

/* Purchase Details Modal */
function openPurchaseDetailsModal(btn) {
    var listingId = btn.getAttribute('data-listing-id');
    var packName = btn.getAttribute('data-pack-name');
    document.getElementById('purchaseDetailsPackName').innerText = packName;
    document.getElementById('purchaseDetailsLoading').style.display = 'block';
    document.getElementById('purchaseDetailsLoading').innerText = window._i18n('loading','加载中...');
    document.getElementById('purchaseDetailsContent').style.display = 'none';
    document.getElementById('purchaseDetailsModal').style.display = 'flex';
    fetch('/user/author/pack-purchases?listing_id=' + encodeURIComponent(listingId), {credentials:'same-origin'})
        .then(function(r){ return r.json(); })
        .then(function(data){
            document.getElementById('purchaseDetailsLoading').style.display = 'none';
            document.getElementById('purchaseDetailsContent').style.display = 'block';
            document.getElementById('purchaseDetailsSplitInfo').innerText = window._i18n('split_ratio','分成比例') + '：' + (data.split_pct || 70) + '%';
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
            document.getElementById('purchaseDetailsLoading').innerText = window._i18n('load_failed','加载失败，请重试');
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
var _paymentTypeLabels = {"paypal":"PayPal","wechat":window._i18n("wechat","微信"),"alipay":"AliPay","check":window._i18n("check","支票"),"wire_transfer":window._i18n("wire_transfer","国际电汇"),"bank_card_us":window._i18n("bank_card_us","美国银行卡"),"bank_card_eu":window._i18n("bank_card_eu","欧洲银行卡"),"bank_card_cn":window._i18n("bank_card_cn","中国银行卡")};
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
    document.getElementById("feeRatesCurrentType").innerHTML = currentLabel ? window._i18n('current_method','当前收款方式') + '：<span style="font-weight:600;color:#ea580c;">'+currentLabel+'</span>' : '<span style="color:#94a3b8;">'+window._i18n('no_method_selected','尚未选择收款方式')+'</span>';
    document.getElementById("feeRatesDialogContent").innerText = window._i18n('loading','加载中...');
    fetch("/user/payment-info/fee-rates", {credentials:"same-origin"})
        .then(function(r){ return r.json(); })
        .then(function(data){
            var html = '<table style="width:100%;border-collapse:collapse;">';
            var types = [["paypal","PayPal"],["wechat",window._i18n("wechat","微信")],["alipay","AliPay"],["check",window._i18n("check","支票")],["wire_transfer",window._i18n("wire_transfer","国际电汇 (SWIFT)")],["bank_card_us",window._i18n("bank_card_us","美国银行卡 (ACH)")],["bank_card_eu",window._i18n("bank_card_eu","欧洲银行卡 (SEPA)")],["bank_card_cn",window._i18n("bank_card_cn","中国银行卡 (CNAPS)")]];
            for (var i=0;i<types.length;i++) {
                var rate = data[types[i][0]] || 0;
                var pct = rate.toFixed(1) + "%";
                var isActive = types[i][0] === currentType;
                var bg = isActive ? "#fff7ed" : (i % 2 === 0 ? "#f8fafc" : "#ffffff");
                var borderLeft = isActive ? "border-left:3px solid #f97316;" : "";
                var labelExtra = isActive ? ' <span style="font-size:11px;color:#ea580c;font-weight:600;">✓ '+window._i18n('current','当前')+'</span>' : '';
                var fontStyle = isActive ? "color:#9a3412;font-weight:600;" : "";
                html += '<tr style="background:'+bg+';'+borderLeft+'"><td style="padding:6px 10px;border-bottom:1px solid #e2e8f0;'+fontStyle+'">'+types[i][1]+labelExtra+'</td><td style="padding:6px 10px;border-bottom:1px solid #e2e8f0;text-align:right;font-weight:500;'+fontStyle+'">'+pct+'</td></tr>';
            }
            html += '</table>';
            document.getElementById("feeRatesDialogContent").innerHTML = html;
        }).catch(function(){ document.getElementById("feeRatesDialogContent").innerText = window._i18n('load_failed','加载失败，请重试'); });
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
    if(!t){showPaymentMsg(window._i18n("select_payment_method","请选择收款方式"),"error");return false;}
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
        if(res.ok&&res.data.ok){showPaymentMsg(window._i18n("payment_save_success","收款信息保存成功"),"success");setTimeout(function(){closePaymentSettingsModal();},1200);}
        else{showPaymentMsg(res.data.error||window._i18n("save_failed","保存失败，请重试"),"error");}
    }).catch(function(){showPaymentMsg(window._i18n("network_error","网络错误，请重试"),"error");});
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
    document.getElementById("withdrawRecordsContent").innerHTML='<div style="text-align:center;padding:30px;color:#94a3b8;">'+window._i18n('loading','加载中...')+'</div>';
    document.getElementById("withdrawRecordsTotalRow").style.display="none";
    fetch("/user/author/withdrawals",{credentials:"same-origin",headers:{"Accept":"application/json"}})
    .then(function(r){return r.json();})
    .then(function(data){
        var list=data.records||[];
        if(list.length===0){
            document.getElementById("withdrawRecordsContent").innerHTML='<div style="text-align:center;padding:40px 20px;color:#94a3b8;"><div style="font-size:40px;margin-bottom:12px;">📭</div><p>'+window._i18n('no_withdraw_records','暂无提现记录')+'</p></div>';
            return;
        }
        var html='<table style="width:100%;border-collapse:collapse;font-size:13px;">';
        html+='<thead><tr style="border-bottom:2px solid #e2e8f0;">';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">'+window._i18n('credits_col','Credits')+'</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">'+window._i18n('rate_col','汇率')+'</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">'+window._i18n('cash_amount_col','提现金额')+'</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">'+window._i18n('fee_col','手续费')+'</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">'+window._i18n('net_col','实付')+'</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">'+window._i18n('status_col','状态')+'</th>';
        html+='<th style="text-align:left;padding:8px;color:#64748b;font-size:12px;">'+window._i18n('time_col','时间')+'</th>';
        html+='</tr></thead><tbody>';
        for(var i=0;i<list.length;i++){
            var r=list[i];
            var st=r.status==='pending'?'<span style="background:#fef3c7;color:#92400e;padding:2px 8px;border-radius:4px;font-size:11px;">'+window._i18n('pending_payment','待付款')+'</span>':'<span style="background:#ecfdf5;color:#065f46;padding:2px 8px;border-radius:4px;font-size:11px;">'+window._i18n('paid','已付款')+'</span>';
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
        document.getElementById("withdrawRecordsContent").innerHTML='<div style="text-align:center;padding:30px;color:#ef4444;">'+window._i18n('load_failed','加载失败，请重试')+'</div>';
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
    var _yuan=window._i18n('yuan','元');
    var lines=[];
    lines.push('<span style="color:#94a3b8;">① '+window._i18n('formula_step1','分成后可提现余额已含分成比例')+' '+splitPct+'%</span>');
    lines.push('<span style="color:#334155;">② '+window._i18n('formula_step2','提现金额')+' = '+credits+' × '+rate.toFixed(2)+' = <b>'+cash.toFixed(2)+'</b> '+_yuan+'</span>');
    if(_withdrawFeeRate>0){
        lines.push('<span style="color:#334155;">③ '+window._i18n('formula_step3','手续费')+' = '+cash.toFixed(2)+' × '+_withdrawFeeRate.toFixed(1)+'% = <b>'+fee.toFixed(2)+'</b> '+_yuan+'</span>');
        lines.push('<span style="color:#10b981;font-weight:600;">④ '+window._i18n('formula_step4','实付')+' = '+cash.toFixed(2)+' − '+fee.toFixed(2)+' = <b>'+net.toFixed(2)+'</b> '+_yuan+'</span>');
    } else {
        lines.push('<span style="color:#10b981;font-weight:600;">③ '+window._i18n('formula_step4','实付')+' = <b>'+cash.toFixed(2)+'</b> '+_yuan+'（'+window._i18n('no_fee','无手续费')+'）</span>');
    }
    formulaBox.innerHTML=lines.join('<br>');
    formulaBox.style.display="block";
    netEl.innerText=window._i18n('net_amount','实付金额')+'：'+net.toFixed(2)+' '+_yuan;
    netEl.style.display="block";
    if(credits>maxCredits){
        warning.innerHTML='⚠️ '+window._i18n('exceeds_balance','提现 Credits 数量不能超过可提现余额')+'（'+maxCredits+' Credits）';
        warning.style.display="block";
        submitBtn.disabled=true;submitBtn.style.opacity="0.5";
    } else if(net<100){
        warning.innerHTML='⚠️ '+window._i18n('formula_step4','实付')+' '+net.toFixed(2)+' '+_yuan+'，'+window._i18n('below_minimum','扣除手续费后实付金额低于最低提现金额 100 元');
        warning.style.display="block";
        submitBtn.disabled=true;submitBtn.style.opacity="0.5";
    }
}
function submitWithdraw() {
    var credits=parseFloat(document.getElementById("withdrawCreditsInput").value)||0;
    if(credits<=0){alert(window._i18n("enter_valid_amount","请输入有效的提现数量"));return;}
    var maxCredits=parseFloat(document.getElementById("withdrawCreditsInput").max)||0;
    if(credits>maxCredits){alert(window._i18n("exceeds_balance","提现 Credits 数量不能超过可提现余额")+"（"+maxCredits+" Credits）");return;}
    var rate=parseFloat(document.getElementById("withdrawCashRate").innerText)||0;
    var cash=credits*rate;
    var fee=cash*_withdrawFeeRate/100;
    var net=cash-fee;
    var _yuan=window._i18n('yuan','元');
    if(net<100){alert(window._i18n("below_minimum","扣除手续费后实付金额低于最低提现金额 100 元"));return;}
    if(!confirm(window._i18n("confirm_withdraw","确认提现")+" "+credits+" Credits？\n\n"+window._i18n("withdraw_amount_label","提现金额")+"："+cash.toFixed(2)+" "+_yuan+"\n"+window._i18n("fee","手续费")+"："+fee.toFixed(2)+" "+_yuan+"\n"+window._i18n("net_amount","实付金额")+"："+net.toFixed(2)+" "+_yuan)){return;}
    var btn=document.getElementById("withdrawSubmitBtn");
    btn.disabled=true;btn.innerText=window._i18n("submitting","提交中...");
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
            alert("✅ "+window._i18n("withdraw_submitted","提现申请已提交，请等待管理员审核付款。"));
            window.location.href="/user/?success=withdraw";
        } else {
            alert("⚠️ "+window._i18n("withdraw_failed","提现失败")+"：" + (data.message||data.error||window._i18n("system_error","系统错误")));
            btn.disabled=false;btn.innerText=window._i18n("confirm_withdraw","确认提现");
        }
    })
    .catch(function(err){
        alert("⚠️ "+window._i18n("withdraw_failed","提现失败")+"："+err);
        btn.disabled=false;btn.innerText=window._i18n("confirm_withdraw","确认提现");
    });
}

/* Helper: find author pack table row by listing ID */
function findAuthorPackRow(listingId) {
    var rows=document.querySelectorAll(".author-table tbody tr");
    for(var i=0;i<rows.length;i++){
        var btn=rows[i].querySelector("[data-listing-id='"+listingId+"']");
        if(btn) return rows[i];
    }
    return null;
}
function showAuthorToast(msg) {
    var t=document.querySelector(".share-toast");
    if(t){t.innerText=msg;t.classList.add("show");setTimeout(function(){t.classList.remove("show");},2500);}
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
    if(confirm(window._i18n("confirm_edit_warning","修改已上架的分析包信息后，该分析包将被下架并需要重新提交审核后才能再次上架。\n\n确定要继续修改吗？"))){
        var form=document.getElementById("editPackForm");
        var formData=new FormData(form);
        fetch(form.action,{method:"POST",credentials:"same-origin",headers:{"X-Requested-With":"XMLHttpRequest"},body:formData})
        .then(function(r){return r.json();})
        .then(function(data){
            if(data.ok){
                var lid=document.getElementById("editListingId").value;
                var row=findAuthorPackRow(lid);
                if(row){
                    row.querySelector("td:first-child").innerText=document.getElementById("editPackName").value;
                    var modeSelect=document.getElementById("editShareMode");
                    var modeText=modeSelect.options[modeSelect.selectedIndex].textContent;
                    row.querySelectorAll("td")[2].innerText=modeText;
                    var price=document.getElementById("editCreditsPrice").value;
                    row.querySelectorAll("td")[3].innerText=(modeSelect.value==="free")?"-":price+" Credits";
                    var statusCell=row.querySelectorAll("td")[4];
                    statusCell.innerHTML='<span class="status-badge status-pending" data-i18n="pending_review">'+window._i18n("pending_review","待审核")+'</span>';
                }
                closeEditPackModal();
                showAuthorToast(window._i18n("edit_success","分析包信息已更新，等待重新审核"));
            } else {
                alert(data.error||window._i18n("save_failed","保存失败，请重试"));
            }
        }).catch(function(){alert(window._i18n("network_error","网络错误，请重试"));});
    }
}
function onEditShareModeChange() {
    var mode=document.getElementById("editShareMode").value;
    var ps=document.getElementById("editPriceSection");
    var pi=document.getElementById("editCreditsPrice");
    var hint=document.getElementById("editPriceHint");
    if(mode==="free"){ps.style.display="none";pi.value=0;}
    else if(mode==="per_use"){ps.style.display="block";pi.min=1;pi.max=100;hint.innerText=window._i18n("per_use_price_hint","按次付费：1-100 Credits");}
    else if(mode==="subscription"){ps.style.display="block";pi.min=100;pi.max=1000;hint.innerText=window._i18n("subscription_price_hint","订阅：100-1000 Credits");}
}

/* Renew Modal */
function openRenewModal(btn) {
    var listingId=btn.getAttribute("data-listing-id");
    var packName=btn.getAttribute("data-pack-name");
    var shareMode=btn.getAttribute("data-share-mode");
    var creditsPrice=parseFloat(btn.getAttribute("data-credits-price"))||0;
    _renewState={listingId:listingId,shareMode:shareMode,creditsPrice:creditsPrice};
    document.getElementById("renewPackName").innerText=window._i18n("pack_label","分析包")+"："+packName;
    if(shareMode==="per_use"){
        document.getElementById("renewTitle").innerText=window._i18n("per_use_renew","按次续费");
        document.getElementById("renewUnitPrice").innerText=window._i18n("per_use_price","单次价格")+"："+creditsPrice+" Credits";
        document.getElementById("renewPerUseSection").style.display="block";
        document.getElementById("renewSubSection").style.display="none";
        document.getElementById("renewQuantity").value=1;
        calcPerUseCost();
    } else if(shareMode==="subscription"){
        document.getElementById("renewTitle").innerText=window._i18n("subscription_renew","订阅续费");
        document.getElementById("renewUnitPrice").innerText=window._i18n("monthly_price","月度价格")+"："+creditsPrice+" Credits";
        document.getElementById("renewPerUseSection").style.display="none";
        document.getElementById("renewSubSection").style.display="block";
        var radios=document.getElementsByName("renewMonths");
        for(var i=0;i<radios.length;i++){if(radios[i].value==="1")radios[i].checked=true;}
        calcSubCost();
    }
    document.getElementById("renewModal").style.display="flex";
}
function closeRenewModal(){document.getElementById("renewModal").style.display="none";}
function calcPerUseCost(){var qty=parseInt(document.getElementById("renewQuantity").value)||1;if(qty<1)qty=1;document.getElementById("renewTotalCost").innerText=window._i18n("total_cost","总费用")+"："+(_renewState.creditsPrice*qty)+" Credits";}
function calcSubCost(){var radios=document.getElementsByName("renewMonths");var m=1;for(var i=0;i<radios.length;i++){if(radios[i].checked){m=parseInt(radios[i].value);break;}}document.getElementById("renewTotalCost").innerText=window._i18n("total_cost","总费用")+"："+(_renewState.creditsPrice*m)+" Credits";}
function submitRenew(){
    if(_renewState.shareMode==="per_use"){var qty=parseInt(document.getElementById("renewQuantity").value)||1;if(qty<1){alert(window._i18n("enter_valid_count","请输入有效的次数"));return;}document.getElementById("renewPerUseListingId").value=_renewState.listingId;document.getElementById("renewPerUseQuantity").value=qty;document.getElementById("renewPerUseForm").submit();}
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
function submitAuthorDelete(){
    var lid=document.getElementById("authorDeleteListingId").value;
    var formData=new FormData(document.getElementById("authorDeleteForm"));
    fetch("/user/author/delete-pack",{method:"POST",credentials:"same-origin",headers:{"X-Requested-With":"XMLHttpRequest"},body:formData})
    .then(function(r){return r.json();})
    .then(function(data){
        if(data.ok){
            var row=findAuthorPackRow(lid);
            if(row) row.remove();
            closeAuthorDeleteModal();
            showAuthorToast(window._i18n("delete_success","分析包已删除"));
        } else { alert(data.error||window._i18n("delete_failed","删除失败")); }
    }).catch(function(){alert(window._i18n("network_error","网络错误，请重试"));});
}

/* Copy Share Link */
function copyShareLink(btn){
    var token=btn.getAttribute("data-share-token");
    var url=window.location.origin+"/pack/"+token;
    if(navigator.clipboard&&navigator.clipboard.writeText){
        navigator.clipboard.writeText(url).then(function(){showShareToast(btn)}).catch(function(){fallbackCopy(url,btn)});
    }else{fallbackCopy(url,btn)}
}
function copyStorefrontLink(btn){
    var slug=btn.getAttribute("data-storefront-slug");
    var url=window.location.origin+"/store/"+slug;
    if(navigator.clipboard&&navigator.clipboard.writeText){
        navigator.clipboard.writeText(url).then(function(){showShareToast(btn)}).catch(function(){fallbackCopy(url,btn)});
    }else{fallbackCopy(url,btn)}
}
(function(){
    var shareBtn=document.querySelector("[data-storefront-slug]");
    if(!shareBtn)return;
    var slug=shareBtn.getAttribute("data-storefront-slug");
    var url=encodeURIComponent(window.location.origin+"/store/"+slug);
    var title=encodeURIComponent(document.title);
    var x=document.getElementById("storefrontShareX"),l=document.getElementById("storefrontShareLI");
    if(x)x.href="https://twitter.com/intent/tweet?text="+title+"&url="+url;
    if(l)l.href="https://www.linkedin.com/sharing/share-offsite/?url="+url;
})();
function fallbackCopy(text,btn){
    var ta=document.createElement("textarea");ta.value=text;ta.style.cssText="position:fixed;left:-9999px;";
    document.body.appendChild(ta);ta.select();
    try{document.execCommand("copy");showShareToast(btn)}catch(e){}
    document.body.removeChild(ta);
}
function showShareToast(btn){
    btn.classList.add("copied");
    setTimeout(function(){btn.classList.remove("copied")},1500);
    var t=document.getElementById("shareToast");
    if(t){t.classList.add("show");setTimeout(function(){t.classList.remove("show")},2000)}
}

/* Author Delist Published Pack Modal */
function openAuthorDelistModal(btn){
    document.getElementById("delistListingId").value=btn.getAttribute("data-listing-id");
    document.getElementById("delistPackName").textContent=btn.getAttribute("data-pack-name");
    document.getElementById("authorDelistModal").style.display="flex";
}
function closeAuthorDelistModal(){document.getElementById("authorDelistModal").style.display="none";}
function submitAuthorDelist(){
    var lid=document.getElementById("delistListingId").value;
    var formData=new FormData(document.getElementById("authorDelistForm"));
    fetch("/user/author/delist-pack",{method:"POST",credentials:"same-origin",headers:{"X-Requested-With":"XMLHttpRequest"},body:formData})
    .then(function(r){return r.json();})
    .then(function(data){
        if(data.ok){
            var row=findAuthorPackRow(lid);
            if(row){
                var statusCell=row.querySelectorAll("td")[4];
                statusCell.innerHTML='<span class="status-badge status-delisted" data-i18n="delisted">'+window._i18n("delisted","已下架")+'</span>';
                var actionsCell=row.querySelectorAll("td")[7];
                if(actionsCell){
                    var delistBtn=actionsCell.querySelector("[onclick*='openAuthorDelistModal']");
                    if(delistBtn) delistBtn.remove();
                    var shareBtn=actionsCell.querySelector(".btn-share-link");
                    if(shareBtn) shareBtn.remove();
                }
            }
            closeAuthorDelistModal();
            showAuthorToast(window._i18n("delist_success","分析包已下架"));
        } else { alert(data.error||window._i18n("delist_failed","下架失败")); }
    }).catch(function(){alert(window._i18n("network_error","网络错误，请重试"));});
}

/* Header Language Switcher */
(function(){
    var sw=document.getElementById("headerLangSwitcher");
    if(!sw)return;
    var lang=(document.cookie.match(/(?:^|;\s*)lang=([^;]*)/)||[])[1]||'';
    var dl=(document.querySelector('meta[name="default-lang"]')||{}).content||'zh-CN';
    if(!lang)lang=dl;
    var path=encodeURIComponent(window.location.pathname+window.location.search);
    // Update the server-rendered links with correct redirect path and active state
    var links=sw.querySelectorAll("a");
    for(var i=0;i<links.length;i++){
        var href=links[i].href;
        if(href.indexOf("lang=zh-CN")!==-1){
            links[i].href="/set-lang?lang=zh-CN&redirect="+path;
            links[i].className=(lang==="zh-CN")?"active":"";
        } else if(href.indexOf("lang=en-US")!==-1){
            links[i].href="/set-lang?lang=en-US&redirect="+path;
            links[i].className=(lang==="en-US")?"active":"";
        }
    }
    // Hide floating lang-switcher since header has one
    var floatSw=document.getElementById("lang-switcher");
    if(floatSw)floatSw.style.display="none";
})();
</script>
<div id="shareToast" class="share-toast" data-i18n="link_copied">✅ 分享链接已复制</div>
` + I18nJS + `
</body>
</html>`

const userCustomProductOrdersHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title data-i18n="custom_product_orders">自定义商品购买记录</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", sans-serif;
            background: #f0f2f5; min-height: 100vh; color: #1e293b; line-height: 1.6;
        }
        .page { max-width: 960px; margin: 0 auto; padding: 24px 20px 36px; }
        .nav {
            display: flex; align-items: center; justify-content: space-between; margin-bottom: 24px;
        }
        .logo-link { display: flex; align-items: center; gap: 10px; text-decoration: none; }
        .logo-mark {
            width: 36px; height: 36px; border-radius: 10px;
            display: flex; align-items: center; justify-content: center;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            font-size: 18px; box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .logo-text { font-size: 15px; font-weight: 700; color: #1e293b; }
        .nav-link {
            padding: 7px 16px; font-size: 13px; font-weight: 500; color: #64748b;
            background: #fff; border: 1px solid #e2e8f0; border-radius: 8px;
            text-decoration: none; transition: all .2s;
        }
        .nav-link:hover { color: #1e293b; border-color: #cbd5e1; }
        .page-title { font-size: 22px; font-weight: 800; color: #0f172a; margin-bottom: 20px; display: flex; align-items: center; gap: 10px; }
        .card {
            background: #fff; border-radius: 12px; padding: 24px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 4px 16px rgba(0,0,0,0.04);
            margin-bottom: 20px; border: 1px solid #e2e8f0;
        }
        .card-title { font-size: 15px; font-weight: 700; color: #1e293b; margin-bottom: 16px; display: flex; align-items: center; gap: 8px; }
        .order-table { width: 100%; border-collapse: collapse; }
        .order-table th {
            text-align: left; padding: 10px 12px; font-size: 12px; font-weight: 700;
            color: #64748b; border-bottom: 2px solid #e2e8f0; white-space: nowrap;
        }
        .order-table td {
            padding: 12px; font-size: 13px; color: #1e293b;
            border-bottom: 1px solid #f1f5f9; vertical-align: top;
        }
        .order-table tr:hover td { background: #f8fafc; }
        .status-badge {
            display: inline-block; padding: 2px 10px; border-radius: 20px;
            font-size: 11px; font-weight: 700;
        }
        .status-pending { background: #fef3c7; color: #d97706; border: 1px solid #fde68a; }
        .status-paid { background: #dbeafe; color: #2563eb; border: 1px solid #bfdbfe; }
        .status-fulfilled { background: #dcfce7; color: #16a34a; border: 1px solid #bbf7d0; }
        .status-failed { background: #fef2f2; color: #dc2626; border: 1px solid #fecaca; }
        .sn-info { font-size: 12px; color: #6366f1; margin-top: 4px; word-break: break-all; }
        .credits-info { font-size: 12px; color: #059669; margin-top: 4px; font-weight: 600; }
        .type-tag {
            display: inline-block; padding: 2px 8px; border-radius: 4px;
            font-size: 11px; font-weight: 700;
        }
        .type-credits { background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; }
        .type-virtual { background: #eef2ff; color: #4338ca; border: 1px solid #c7d2fe; }
        .empty-state { text-align: center; padding: 40px 20px; color: #94a3b8; font-size: 13px; }
        .empty-state .icon { font-size: 28px; margin-bottom: 8px; opacity: 0.7; }
        .foot { text-align: center; margin-top: 28px; padding-top: 16px; border-top: 1px solid #e2e8f0; }
        .foot-text { font-size: 11px; color: #94a3b8; }
        .foot-text a { color: #6366f1; text-decoration: none; }
        @media (max-width: 640px) {
            .order-table { font-size: 12px; }
            .order-table th, .order-table td { padding: 8px 6px; }
        }
    </style>
</head>
<body>
<div class="page">
    <nav class="nav">
        <a class="logo-link" href="/"><span class="logo-mark">📦</span><span class="logo-text" data-i18n="site_name">分析技能包市场</span></a>
        <a class="nav-link" href="/user/" data-i18n="back_to_center_link">← 返回个人中心</a>
    </nav>

    <h1 class="page-title">🛒 <span data-i18n="custom_product_orders">自定义商品购买记录</span></h1>

    <div class="card">
        <div class="card-title"><span>📋</span> <span data-i18n="cp_my_orders">我的购买记录</span></div>
        {{if .Orders}}
        <div style="overflow-x:auto;">
            <table class="order-table">
                <thead>
                    <tr>
                        <th data-i18n="product_name_col">商品名称</th>
                        <th data-i18n="cp_product_type">商品类型</th>
                        <th data-i18n="cp_purchase_time">购买时间</th>
                        <th data-i18n="cp_payment_amount">支付金额</th>
                        <th data-i18n="cp_order_status">订单状态</th>
                        <th data-i18n="details">详情</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Orders}}
                    <tr>
                        <td style="font-weight:600;">{{.ProductName}}</td>
                        <td>
                            {{if eq .ProductType "credits"}}<span class="type-tag type-credits" data-i18n="product_type_credits">积分充值</span>
                            {{else if eq .ProductType "virtual_goods"}}<span class="type-tag type-virtual" data-i18n="product_type_virtual">虚拟商品</span>
                            {{else}}{{.ProductType}}{{end}}
                        </td>
                        <td>{{.CreatedAt}}</td>
                        <td>$ {{printf "%.2f" .AmountUSD}}</td>
                        <td>
                            <span class="status-badge status-{{.Status}}">
                                {{if eq .Status "pending"}}<span data-i18n="cp_status_pending">待支付</span>{{end}}
                                {{if eq .Status "paid"}}<span data-i18n="cp_status_paid">已支付</span>{{end}}
                                {{if eq .Status "fulfilled"}}<span data-i18n="cp_status_fulfilled">已完成</span>{{end}}
                                {{if eq .Status "failed"}}<span data-i18n="cp_status_failed">失败</span>{{end}}
                            </span>
                        </td>
                        <td>
                            {{if and (eq .ProductType "virtual_goods") (eq .Status "fulfilled") (ne .LicenseSN "")}}
                            <div class="sn-info">🔑 授权 SN: {{.LicenseSN}}</div>
                            <div class="sn-info">📧 邮箱: {{.LicenseEmail}}</div>
                            {{end}}
                            {{if and (eq .ProductType "credits") (eq .Status "fulfilled") (gt .CreditsAmount 0)}}
                            <div class="credits-info">💰 充值积分: {{.CreditsAmount}}</div>
                            {{end}}
                            {{if and (ne .Status "fulfilled") (ne .Status "failed")}}
                            <span style="font-size:12px;color:#94a3b8;">—</span>
                            {{end}}
                            {{if eq .Status "failed"}}
                            <span style="font-size:12px;color:#94a3b8;">—</span>
                            {{end}}
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{else}}
        <div class="empty-state">
            <div class="icon">📭</div>
            <p data-i18n="cp_no_orders">暂无购买记录</p>
        </div>
        {{end}}
    </div>

    <div class="foot">
        <p class="foot-text">Vantagics <span data-i18n="site_name">分析技能包市场</span> · <a href="/" data-i18n="browse_more">浏览更多</a></p>
    </div>
</div>
</body>
</html>`
