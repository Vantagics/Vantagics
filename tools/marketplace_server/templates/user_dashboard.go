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
            background: #0f172a;
            min-height: 100vh;
            color: #f1f5f9;
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
        .header-title h1 { font-size: 20px; font-weight: 700; color: #f1f5f9; }
        /* User info bar */
        .user-info {
            background: #1e293b;
            border-radius: 12px;
            padding: 24px 28px;
            margin-bottom: 28px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            box-shadow: 0 4px 24px rgba(0,0,0,0.3);
            border: 1px solid rgba(255,255,255,0.06);
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
            color: #e2e8f0;
        }
        .user-email .label {
            font-size: 12px;
            color: #94a3b8;
            display: block;
            margin-bottom: 2px;
        }
        .credits-info {
            font-size: 15px;
            color: #e2e8f0;
        }
        .credits-info .label {
            font-size: 12px;
            color: #94a3b8;
            display: block;
            margin-bottom: 2px;
        }
        .credits-info .balance {
            color: #fbbf24;
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
            background: #3b82f6;
            color: #fff;
            border: none;
            border-radius: 6px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            transition: background 0.2s;
        }
        .btn-recharge:hover { background: #2563eb; }
        .btn-logout {
            padding: 8px 16px;
            background: none;
            color: #ef4444;
            border: 1px solid rgba(239,68,68,0.3);
            border-radius: 6px;
            font-size: 13px;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.2s;
        }
        .btn-logout:hover { background: rgba(239,68,68,0.1); border-color: #ef4444; }
        /* Section title */
        .section-title {
            font-size: 16px;
            font-weight: 600;
            color: #f1f5f9;
            margin-bottom: 16px;
        }

        /* Pack cards grid */
        .pack-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 16px;
        }
        .pack-card {
            background: #1e293b;
            border-radius: 10px;
            padding: 20px;
            border: 1px solid rgba(255,255,255,0.06);
            box-shadow: 0 2px 12px rgba(0,0,0,0.2);
            transition: border-color 0.2s;
        }
        .pack-card:hover { border-color: rgba(255,255,255,0.12); }
        .pack-card .pack-name {
            font-size: 15px;
            font-weight: 600;
            color: #f1f5f9;
            margin-bottom: 8px;
        }
        .pack-card .pack-category {
            font-size: 12px;
            color: #94a3b8;
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
        .tag-free { background: rgba(34,197,94,0.15); color: #4ade80; }
        .tag-per-use { background: rgba(59,130,246,0.15); color: #60a5fa; }
        .tag-time-limited { background: rgba(251,191,36,0.15); color: #fbbf24; }
        .tag-subscription { background: rgba(168,85,247,0.15); color: #c084fc; }
        .pack-card .pack-date {
            font-size: 12px;
            color: #64748b;
        }
        .pack-card .pack-expires {
            font-size: 12px;
            color: #94a3b8;
            margin-top: 4px;
        }

        /* Empty state */
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #64748b;
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
