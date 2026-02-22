package templates

import "html/template"

// UserWithdrawalRecordsTmpl is the parsed user withdrawal records page template.
var UserWithdrawalRecordsTmpl = template.Must(template.New("user_withdrawal_records").Parse(userWithdrawalRecordsHTML))

const userWithdrawalRecordsHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title data-i18n="withdrawal_records_title">提现记录 - 分析技能包市场</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #f0f4ff 0%, #e8f5e9 50%, #f3e8ff 100%);
            min-height: 100vh;
            color: #1e293b;
        }
        .records-wrap {
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
        .records-table-wrap {
            background: #fff;
            border-radius: 16px;
            padding: 24px 28px;
            box-shadow: 0 4px 24px rgba(0,0,0,0.06);
            border: 1px solid #e2e8f0;
            overflow-x: auto;
        }
        .records-table {
            width: 100%;
            border-collapse: collapse;
        }
        .records-table th {
            text-align: left;
            padding: 10px 12px;
            font-size: 12px;
            font-weight: 600;
            color: #64748b;
            border-bottom: 2px solid #e2e8f0;
            white-space: nowrap;
        }
        .records-table td {
            padding: 12px;
            font-size: 14px;
            color: #334155;
            border-bottom: 1px solid #f1f5f9;
        }
        .records-table tr:last-child td { border-bottom: none; }
        .records-table tr:hover td { background: #f8fafc; }
        .total-row {
            background: #f8fafc;
            border-top: 2px solid #e2e8f0;
            margin-top: 16px;
            padding: 16px 28px;
            border-radius: 0 0 16px 16px;
            display: flex;
            justify-content: flex-end;
            align-items: center;
            gap: 8px;
            font-size: 15px;
            font-weight: 600;
            color: #1e293b;
        }
        .total-amount { color: #059669; font-size: 18px; }
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
<div class="records-wrap">
    <div class="header">
        <div class="header-title">
            <span class="logo">💰</span>
            <h1 data-i18n="withdrawal_records_title">提现记录</h1>
        </div>
        <a class="btn-back" href="/user/" data-i18n="back_to_center_link">返回个人中心</a>
    </div>

    {{if .Records}}
    <div class="records-table-wrap">
        <table class="records-table">
            <thead>
                <tr>
                    <th data-i18n="withdraw_credits_col">提现 Credits</th>
                    <th data-i18n="exchange_rate_col">兑换比率</th>
                    <th data-i18n="cash_col">提现现金(元)</th>
                    <th data-i18n="withdraw_time_col">提现时间</th>
                </tr>
            </thead>
            <tbody>
                {{range .Records}}
                <tr>
                    <td>{{printf "%.2f" .CreditsAmount}}</td>
                    <td>{{printf "%.4f" .CashRate}}</td>
                    <td>{{printf "%.2f" .CashAmount}}</td>
                    <td>{{.CreatedAt}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        <div class="total-row">
            <span data-i18n="total_cash_label">总计提现现金：</span>
            <span class="total-amount">¥{{printf "%.2f" .TotalCash}}</span>
        </div>
    </div>
    {{else}}
    <div class="empty-state">
        <div class="icon">📭</div>
        <p data-i18n="no_withdraw_records">暂无提现记录</p>
    </div>
    {{end}}
</div>
` + I18nJS + `
</body>
</html>`
