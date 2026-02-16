package templates

import "html/template"

// UserBillingTmpl is the parsed user billing page template.
var UserBillingTmpl = template.Must(template.New("user_billing").Parse(userBillingHTML))

const userBillingHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>帐单记录 - 快捷分析包市场</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #f0f4ff 0%, #e8f5e9 50%, #f3e8ff 100%);
            min-height: 100vh;
            color: #1e293b;
        }
        .billing-wrap {
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
        .btn-back {
            padding: 8px 16px;
            background: none;
            color: #6366f1;
            border: 1px solid #c7d2fe;
            border-radius: 8px;
            font-size: 13px;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.2s;
        }
        .btn-back:hover { background: #eef2ff; border-color: #6366f1; }
        /* Table styles */
        .billing-table-wrap {
            background: #fff;
            border-radius: 16px;
            padding: 24px 28px;
            box-shadow: 0 4px 24px rgba(0,0,0,0.06);
            border: 1px solid #e2e8f0;
            overflow-x: auto;
        }
        .billing-table {
            width: 100%;
            border-collapse: collapse;
        }
        .billing-table th {
            text-align: left;
            padding: 10px 12px;
            font-size: 12px;
            font-weight: 600;
            color: #64748b;
            border-bottom: 2px solid #e2e8f0;
            white-space: nowrap;
        }
        .billing-table td {
            padding: 12px;
            font-size: 14px;
            color: #334155;
            border-bottom: 1px solid #f1f5f9;
        }
        .billing-table tr:last-child td { border-bottom: none; }
        .billing-table tr:hover td { background: #f8fafc; }
        .amount-positive { color: #059669; font-weight: 600; }
        .amount-negative { color: #ef4444; font-weight: 600; }

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
<div class="billing-wrap">
    <div class="header">
        <div class="header-title">
            <span class="logo">🧾</span>
            <h1>帐单记录</h1>
        </div>
        <a class="btn-back" href="/user/dashboard">返回个人中心</a>
    </div>

    {{if .Records}}
    <div class="billing-table-wrap">
        <table class="billing-table">
            <thead>
                <tr>
                    <th>交易类型</th>
                    <th>金额</th>
                    <th>分析包名称</th>
                    <th>描述</th>
                    <th>交易时间</th>
                </tr>
            </thead>
            <tbody>
                {{range .Records}}
                <tr>
                    <td>{{.TransactionType}}</td>
                    <td>{{if lt .Amount 0.0}}<span class="amount-negative">{{printf "%.2f" .Amount}}</span>{{else}}<span class="amount-positive">+{{printf "%.2f" .Amount}}</span>{{end}}</td>
                    <td>{{if .PackName}}{{.PackName}}{{else}}-{{end}}</td>
                    <td>{{if .Description}}{{.Description}}{{else}}-{{end}}</td>
                    <td>{{.CreatedAt}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
    {{else}}
    <div class="empty-state">
        <div class="icon">📭</div>
        <p>暂无交易记录</p>
    </div>
    {{end}}
</div>
</body>
</html>`
