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
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #f0f4ff 0%, #e8f5e9 50%, #f3e8ff 100%);
            min-height: 100vh;
            color: #1e293b;
        }
        .dashboard-wrap {
            max-width: 960px;
            margin: 0 auto;
            padding: 32px 20px;
        }
        .header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 32px;
        }
        .header-title {
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .header-title .logo { font-size: 28px; }
        .header-title h1 { font-size: 20px; font-weight: 700; color: #1e293b; }
        /* User info bar */
        .user-info {
            background: #fff;
            border-radius: 16px;
            padding: 24px 28px;
            margin-bottom: 28px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            box-shadow: 0 4px 24px rgba(0,0,0,0.06);
            border: 1px solid #e2e8f0;
            flex-wrap: wrap;
            gap: 16px;
        }
        .user-detail {
            display: flex;
            align-items: center;
            gap: 20px;
            flex-wrap: wrap;
        }
        .user-email {
            font-size: 15px;
            color: #334155;
        }
        .user-email .label {
            font-size: 12px;
            color: #64748b;
            display: block;
            margin-bottom: 2px;
        }
        .credits-info {
            font-size: 15px;
            color: #334155;
        }
        .credits-info .label {
            font-size: 12px;
            color: #64748b;
            display: block;
            margin-bottom: 2px;
        }
        .credits-info .balance {
            color: #d97706;
            font-weight: 700;
            font-size: 18px;
        }
        .user-actions {
            display: flex;
            gap: 10px;
            align-items: center;
        }
        .btn-recharge {
            padding: 8px 16px;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            color: #fff;
            border: none;
            border-radius: 8px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            transition: opacity 0.2s;
        }
        .btn-recharge:hover { opacity: 0.9; }
        .btn-logout {
            padding: 8px 16px;
            background: none;
            color: #ef4444;
            border: 1px solid #fecaca;
            border-radius: 8px;
            font-size: 13px;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.2s;
        }
        .btn-logout:hover { background: #fef2f2; border-color: #ef4444; }
        /* Section title */
        .section-title {
            font-size: 16px;
            font-weight: 600;
            color: #1e293b;
            margin-bottom: 16px;
        }

        /* Pack cards grid */
        .pack-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 16px;
        }
        .pack-card {
            background: #fff;
            border-radius: 12px;
            padding: 20px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 2px 12px rgba(0,0,0,0.04);
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        .pack-card:hover { border-color: #c7d2fe; box-shadow: 0 4px 16px rgba(99,102,241,0.08); }
        .pack-card .pack-name {
            font-size: 15px;
            font-weight: 600;
            color: #1e293b;
            margin-bottom: 8px;
        }
        .pack-card .pack-category {
            font-size: 12px;
            color: #64748b;
            margin-bottom: 10px;
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
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 500;
        }
        .tag-free { background: #ecfdf5; color: #059669; }
        .tag-per-use { background: #eff6ff; color: #2563eb; }
        .tag-time-limited { background: #fffbeb; color: #d97706; }
        .tag-subscription { background: #faf5ff; color: #7c3aed; }
        .pack-card .pack-date {
            font-size: 12px;
            color: #94a3b8;
        }
        .pack-card .pack-expires {
            font-size: 12px;
            color: #64748b;
            margin-top: 4px;
        }

        /* Empty state */
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #94a3b8;
        }
        .empty-state .icon { font-size: 48px; margin-bottom: 16px; }
        .empty-state p { font-size: 15px; }
    </style>
</head>
<body>
<div class="dashboard-wrap">
    <div class="header">
        <div class="header-title">
            <span class="logo">📦</span>
            <h1>个人中心</h1>
        </div>
    </div>
    <div class="user-info">
        <div class="user-detail">
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
            <button class="btn-recharge" onclick="alert('功能开发中')">充值</button>
            <a class="btn-logout" href="/user/logout">退出登录</a>
        </div>
    </div>

    <div class="section-title">已购买的分析包</div>

    {{if .PurchasedPacks}}
    <div class="pack-grid">
        {{range .PurchasedPacks}}
        <div class="pack-card">
            <div class="pack-name">{{.PackName}}</div>
            <div class="pack-category">{{.CategoryName}}</div>
            <div class="pack-meta">
                {{if eq .ShareMode "free"}}<span class="tag tag-free">免费</span>
                {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use">按次付费</span>
                {{else if eq .ShareMode "time_limited"}}<span class="tag tag-time-limited">限时</span>
                {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription">订阅</span>
                {{end}}
            </div>
            <div class="pack-date">下载时间：{{.PurchaseDate}}</div>
            {{if .ExpiresAt}}<div class="pack-expires">到期时间：{{.ExpiresAt}}</div>{{end}}
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
</body>
</html>`
