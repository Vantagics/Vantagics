package templates

import "html/template"

// CustomProductManageTmpl is the parsed custom product management page template.
var CustomProductManageTmpl = template.Must(template.New("custom_product_manage").Parse(customProductManageHTML))

const customProductManageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>自定义商品管理 - 分析技能包市场</title>
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
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .logo-mark img { width: 100%; height: 100%; object-fit: cover; }
        .logo-text { font-size: 15px; font-weight: 700; color: #1e293b; }
        .nav-link {
            padding: 7px 16px; font-size: 13px; font-weight: 500; color: #64748b;
            background: #fff; border: 1px solid #e2e8f0; border-radius: 8px;
            text-decoration: none; transition: all .2s;
        }
        .nav-link:hover { color: #1e293b; border-color: #cbd5e1; }
        .page-title { font-size: 22px; font-weight: 800; color: #0f172a; margin-bottom: 20px; }
        .card {
            background: #fff; border-radius: 12px; padding: 24px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06); margin-bottom: 20px;
        }
        .card-title { font-size: 16px; font-weight: 700; margin-bottom: 16px; color: #0f172a; }
        .msg { padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 14px; }
        .msg-err { background: #fef2f2; color: #dc2626; border: 1px solid #fecaca; }
        .msg-ok { background: #f0fdf4; color: #16a34a; border: 1px solid #bbf7d0; }
        .form-group { margin-bottom: 16px; }
        .form-group label { display: block; font-size: 13px; font-weight: 600; color: #475569; margin-bottom: 6px; }
        .form-group input, .form-group select, .form-group textarea {
            width: 100%; padding: 10px 14px; font-size: 14px; border: 1px solid #e2e8f0;
            border-radius: 8px; background: #f8fafc; transition: border-color .2s; font-family: inherit;
        }
        .form-group input:focus, .form-group select:focus, .form-group textarea:focus {
            outline: none; border-color: #6366f1; background: #fff;
        }
        .form-group textarea { resize: vertical; min-height: 80px; }
        .btn {
            padding: 10px 20px; font-size: 14px; font-weight: 600; border: none; border-radius: 8px;
            cursor: pointer; transition: all .2s; font-family: inherit;
        }
        .btn-primary { background: #6366f1; color: #fff; }
        .btn-primary:hover { background: #4f46e5; }
        .product-list { margin-top: 16px; }
        .product-item {
            display: flex; align-items: center; justify-content: space-between;
            padding: 14px 16px; border: 1px solid #e2e8f0; border-radius: 10px;
            margin-bottom: 10px; background: #fafbfc;
        }
        .product-item:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
        .product-info { flex: 1; }
        .product-name { font-size: 15px; font-weight: 600; color: #1e293b; }
        .product-meta { font-size: 12px; color: #94a3b8; margin-top: 4px; }
        .product-meta span { margin-right: 12px; }
        .status-badge {
            display: inline-block; padding: 2px 10px; border-radius: 12px; font-size: 12px; font-weight: 600;
        }
        .status-draft { background: #f1f5f9; color: #64748b; }
        .status-pending { background: #fef3c7; color: #d97706; }
        .status-published { background: #dcfce7; color: #16a34a; }
        .status-rejected { background: #fef2f2; color: #dc2626; }
        .type-badge {
            display: inline-block; padding: 2px 8px; border-radius: 6px; font-size: 11px; font-weight: 600; margin-right: 8px;
        }
        .type-credits { background: #dbeafe; color: #2563eb; }
        .type-virtual { background: #f3e8ff; color: #7c3aed; }
        .conditional-fields { display: none; }
        .empty-state { text-align: center; padding: 40px 20px; color: #94a3b8; font-size: 14px; }
        .reject-reason { font-size: 12px; color: #dc2626; margin-top: 4px; }
    </style>
</head>
<body>
<div class="page">
    <div class="nav">
        <a href="/" class="logo-link">
            <div class="logo-mark"><img src="/marketplace-logo.png" alt="" style="width:100%;height:100%;object-fit:cover;border-radius:inherit;"></div>
            <span class="logo-text">分析技能包市场</span>
        </a>
        <a href="/user/storefront/" class="nav-link">← 返回小铺管理</a>
    </div>
    <h1 class="page-title">🛍️ 自定义商品管理</h1>
    {{if .ErrorMsg}}<div class="msg msg-err">{{.ErrorMsg}}</div>{{end}}
    {{if .SuccessMsg}}<div class="msg msg-ok">{{.SuccessMsg}}</div>{{end}}
    <div class="card">
        <div class="card-title">商品列表</div>
        {{if .Products}}
        <div class="product-list">
            {{range .Products}}
            <div class="product-item">
                <div class="product-info">
                    <div class="product-name">
                        {{if eq .ProductType "credits"}}<span class="type-badge type-credits">积分充值</span>{{end}}
                        {{if eq .ProductType "virtual_goods"}}<span class="type-badge type-virtual">虚拟商品</span>{{end}}
                        {{.ProductName}}
                    </div>
                    <div class="product-meta">
                        <span>$ {{printf "%.2f" .PriceUSD}}</span>
                        {{if eq .ProductType "credits"}}<span>{{.CreditsAmount}} 积分</span>{{end}}
                        <span class="status-badge status-{{.Status}}">
                            {{if eq .Status "draft"}}草稿{{end}}
                            {{if eq .Status "pending"}}待审核{{end}}
                            {{if eq .Status "published"}}已上架{{end}}
                            {{if eq .Status "rejected"}}已拒绝{{end}}
                        </span>
                    </div>
                    {{if and (eq .Status "rejected") (ne .RejectReason "")}}
                    <div class="reject-reason">拒绝原因：{{.RejectReason}}</div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
        {{else}}
        <div class="empty-state">暂无自定义商品，点击下方按钮添加第一个商品</div>
        {{end}}
    </div>
    <div class="card" id="createForm">
        <div class="card-title">➕ 添加商品</div>
        <form method="POST" action="/user/storefront/custom-products/create">
            <div class="form-group">
                <label for="product_name">商品名称 (2-100 字符)</label>
                <input type="text" id="product_name" name="product_name" required minlength="2" maxlength="100" placeholder="输入商品名称">
            </div>
            <div class="form-group">
                <label for="description">商品描述</label>
                <textarea id="description" name="description" maxlength="1000" placeholder="输入商品描述（可选）"></textarea>
            </div>
            <div class="form-group">
                <label for="product_type">商品类型</label>
                <select id="product_type" name="product_type" onchange="toggleTypeFields()" required>
                    <option value="credits">积分充值</option>
                    <option value="virtual_goods">虚拟商品</option>
                </select>
            </div>
            <div class="form-group">
                <label for="price_usd">价格 (USD, 最高 9999.99)</label>
                <input type="number" id="price_usd" name="price_usd" required step="0.01" min="0.01" max="9999.99" placeholder="0.00">
            </div>
            <div id="credits-fields" class="conditional-fields" style="display:block;">
                <div class="form-group">
                    <label for="credits_amount">积分数量</label>
                    <input type="number" id="credits_amount" name="credits_amount" min="1" placeholder="购买后充值的积分数量">
                </div>
            </div>
            <div id="virtual-fields" class="conditional-fields">
                <div class="form-group">
                    <label for="license_api_endpoint">License API 地址</label>
                    <input type="url" id="license_api_endpoint" name="license_api_endpoint" placeholder="https://license.example.com/api/bind">
                </div>
                <div class="form-group">
                    <label for="license_api_key">License API 密钥</label>
                    <input type="text" id="license_api_key" name="license_api_key" placeholder="API 密钥">
                </div>
                <div class="form-group">
                    <label for="license_product_id">License 产品标识</label>
                    <input type="text" id="license_product_id" name="license_product_id" placeholder="产品标识 ID">
                </div>
            </div>
            <button type="submit" class="btn btn-primary">创建商品</button>
        </form>
    </div>
</div>
<script>
function toggleTypeFields() {
    var t = document.getElementById('product_type').value;
    document.getElementById('credits-fields').style.display = t === 'credits' ? 'block' : 'none';
    document.getElementById('virtual-fields').style.display = t === 'virtual_goods' ? 'block' : 'none';
}
</script>
</body>
</html>`
