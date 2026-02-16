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
        .btn-password {
            padding: 8px 16px;
            background: linear-gradient(135deg, #10b981, #059669);
            color: #fff;
            border: none;
            border-radius: 8px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            text-decoration: none;
            transition: opacity 0.2s;
        }
        .btn-password:hover { opacity: 0.9; }
        .btn-billing {
            padding: 8px 16px;
            background: linear-gradient(135deg, #3b82f6, #2563eb);
            color: #fff;
            border: none;
            border-radius: 8px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            text-decoration: none;
            transition: opacity 0.2s;
        }
        .btn-billing:hover { opacity: 0.9; }
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
        .usage-progress { font-size: 12px; color: #334155; }
        .usage-exhausted { color: #ef4444; font-weight: 600; }
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

        /* Pack action buttons */
        .pack-actions {
            display: flex;
            gap: 8px;
            margin-top: 14px;
            padding-top: 12px;
            border-top: 1px solid #f1f5f9;
        }
        .btn-renew {
            padding: 6px 14px;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            color: #fff;
            border: none;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 500;
            cursor: pointer;
            transition: opacity 0.2s;
        }
        .btn-renew:hover { opacity: 0.85; }
        .btn-delete {
            padding: 6px 14px;
            background: none;
            color: #ef4444;
            border: 1px solid #fecaca;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn-delete:hover { background: #fef2f2; border-color: #ef4444; }

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
            {{if .HasPassword}}
            <a class="btn-password" href="/user/change-password">修改密码</a>
            {{else}}
            <a class="btn-password" href="/user/set-password">设置密码</a>
            {{end}}
            <a class="btn-billing" href="/user/billing">帐单记录</a>
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
                <span class="usage-progress{{if eq .UsedCount .TotalPurchased}} usage-exhausted{{end}}">已使用 {{.UsedCount}}/{{.TotalPurchased}} 次</span>
                {{else if eq .ShareMode "time_limited"}}<span class="tag tag-time-limited">限时</span>
                {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription">订阅</span>
                {{end}}
            </div>
            <div class="pack-date">下载时间：{{.PurchaseDate}}</div>
            {{if .ExpiresAt}}<div class="pack-expires">到期时间：{{.ExpiresAt}}</div>{{end}}
            <div class="pack-actions">
                {{if or (eq .ShareMode "per_use") (eq .ShareMode "subscription")}}
                <button class="btn-renew"
                    data-listing-id="{{.ListingID}}"
                    data-pack-name="{{.PackName}}"
                    data-share-mode="{{.ShareMode}}"
                    data-credits-price="{{.CreditsPrice}}"
                    onclick="openRenewModal(this)">续费</button>
                {{end}}
                <button class="btn-delete"
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

<!-- Renew Modal -->
<div id="renewModal" style="display:none;position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.4);z-index:1000;align-items:center;justify-content:center;">
  <div style="background:#fff;border-radius:16px;padding:28px 32px;max-width:420px;width:90%;box-shadow:0 8px 32px rgba(0,0,0,0.15);position:relative;">
    <button onclick="closeRenewModal()" style="position:absolute;top:12px;right:16px;background:none;border:none;font-size:20px;cursor:pointer;color:#94a3b8;">&times;</button>
    <h3 id="renewTitle" style="font-size:17px;font-weight:600;color:#1e293b;margin-bottom:18px;">续费</h3>
    <div id="renewPackName" style="font-size:14px;color:#334155;margin-bottom:12px;"></div>
    <div id="renewUnitPrice" style="font-size:13px;color:#64748b;margin-bottom:16px;"></div>

    <!-- Per-use form -->
    <div id="renewPerUseSection" style="display:none;">
      <label style="font-size:13px;color:#334155;display:block;margin-bottom:6px;">购买次数</label>
      <input id="renewQuantity" type="number" min="1" value="1" style="width:100%;padding:8px 12px;border:1px solid #e2e8f0;border-radius:8px;font-size:14px;margin-bottom:14px;" oninput="calcPerUseCost()">
    </div>

    <!-- Subscription form -->
    <div id="renewSubSection" style="display:none;">
      <label style="font-size:13px;color:#334155;display:block;margin-bottom:8px;">续费时长</label>
      <div style="display:flex;flex-direction:column;gap:8px;margin-bottom:14px;">
        <label style="display:flex;align-items:center;gap:8px;font-size:14px;color:#334155;cursor:pointer;">
          <input type="radio" name="renewMonths" value="1" checked onchange="calcSubCost()"> 按月（1个月）
        </label>
        <label style="display:flex;align-items:center;gap:8px;font-size:14px;color:#334155;cursor:pointer;">
          <input type="radio" name="renewMonths" value="12" onchange="calcSubCost()"> 按年（12个月付费，赠送2个月）
        </label>
      </div>
    </div>

    <div id="renewTotalCost" style="font-size:15px;font-weight:600;color:#d97706;margin-bottom:18px;"></div>
    <div style="display:flex;gap:10px;justify-content:flex-end;">
      <button onclick="closeRenewModal()" style="padding:8px 18px;background:none;border:1px solid #e2e8f0;border-radius:8px;font-size:13px;cursor:pointer;color:#64748b;">取消</button>
      <button onclick="submitRenew()" style="padding:8px 18px;background:linear-gradient(135deg,#6366f1,#8b5cf6);color:#fff;border:none;border-radius:8px;font-size:13px;font-weight:500;cursor:pointer;">确认续费</button>
    </div>
  </div>
</div>

<!-- Hidden forms for renew submission -->
<form id="renewPerUseForm" method="POST" action="/user/pack/renew-uses" style="display:none;">
  <input type="hidden" name="listing_id" id="renewPerUseListingId">
  <input type="hidden" name="quantity" id="renewPerUseQuantity">
</form>
<form id="renewSubForm" method="POST" action="/user/pack/renew-subscription" style="display:none;">
  <input type="hidden" name="listing_id" id="renewSubListingId">
  <input type="hidden" name="months" id="renewSubMonths">
</form>

<script>
var _renewState = {listingId:"", shareMode:"", creditsPrice:0};

function openRenewModal(btn) {
    var listingId = btn.getAttribute("data-listing-id");
    var packName = btn.getAttribute("data-pack-name");
    var shareMode = btn.getAttribute("data-share-mode");
    var creditsPrice = parseFloat(btn.getAttribute("data-credits-price")) || 0;

    _renewState.listingId = listingId;
    _renewState.shareMode = shareMode;
    _renewState.creditsPrice = creditsPrice;

    document.getElementById("renewPackName").innerText = "分析包：" + packName;

    var perUseSection = document.getElementById("renewPerUseSection");
    var subSection = document.getElementById("renewSubSection");

    if (shareMode === "per_use") {
        document.getElementById("renewTitle").innerText = "按次续费";
        document.getElementById("renewUnitPrice").innerText = "单次价格：" + creditsPrice + " Credits";
        perUseSection.style.display = "block";
        subSection.style.display = "none";
        document.getElementById("renewQuantity").value = 1;
        calcPerUseCost();
    } else if (shareMode === "subscription") {
        document.getElementById("renewTitle").innerText = "订阅续费";
        document.getElementById("renewUnitPrice").innerText = "月度价格：" + creditsPrice + " Credits";
        perUseSection.style.display = "none";
        subSection.style.display = "block";
        var radios = document.getElementsByName("renewMonths");
        for (var i = 0; i < radios.length; i++) { if (radios[i].value === "1") radios[i].checked = true; }
        calcSubCost();
    }

    var modal = document.getElementById("renewModal");
    modal.style.display = "flex";
}

function closeRenewModal() {
    document.getElementById("renewModal").style.display = "none";
}

function calcPerUseCost() {
    var qty = parseInt(document.getElementById("renewQuantity").value) || 1;
    if (qty < 1) qty = 1;
    var total = _renewState.creditsPrice * qty;
    document.getElementById("renewTotalCost").innerText = "总费用：" + total + " Credits";
}

function calcSubCost() {
    var radios = document.getElementsByName("renewMonths");
    var months = 1;
    for (var i = 0; i < radios.length; i++) { if (radios[i].checked) { months = parseInt(radios[i].value); break; } }
    var total = _renewState.creditsPrice * months;
    document.getElementById("renewTotalCost").innerText = "总费用：" + total + " Credits";
}

function submitRenew() {
    if (_renewState.shareMode === "per_use") {
        var qty = parseInt(document.getElementById("renewQuantity").value) || 1;
        if (qty < 1) { alert("请输入有效的次数"); return; }
        document.getElementById("renewPerUseListingId").value = _renewState.listingId;
        document.getElementById("renewPerUseQuantity").value = qty;
        document.getElementById("renewPerUseForm").submit();
    } else if (_renewState.shareMode === "subscription") {
        var radios = document.getElementsByName("renewMonths");
        var months = 1;
        for (var i = 0; i < radios.length; i++) { if (radios[i].checked) { months = parseInt(radios[i].value); break; } }
        document.getElementById("renewSubListingId").value = _renewState.listingId;
        document.getElementById("renewSubMonths").value = months;
        document.getElementById("renewSubForm").submit();
    }
}
function openDeleteModal(btn) {
    var listingId = btn.getAttribute("data-listing-id");
    var packName = btn.getAttribute("data-pack-name");
    document.getElementById("deletePackName").innerText = packName;
    document.getElementById("deleteListingId").value = listingId;
    document.getElementById("deleteModal").style.display = "flex";
}

function closeDeleteModal() {
    document.getElementById("deleteModal").style.display = "none";
}

function submitDelete() {
    document.getElementById("deleteForm").submit();
}
</script>

<!-- Delete Confirmation Modal -->
<div id="deleteModal" style="display:none;position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.4);z-index:1000;align-items:center;justify-content:center;">
  <div style="background:#fff;border-radius:16px;padding:28px 32px;max-width:420px;width:90%;box-shadow:0 8px 32px rgba(0,0,0,0.15);position:relative;">
    <button onclick="closeDeleteModal()" style="position:absolute;top:12px;right:16px;background:none;border:none;font-size:20px;cursor:pointer;color:#94a3b8;">&times;</button>
    <h3 style="font-size:17px;font-weight:600;color:#1e293b;margin-bottom:18px;">删除分析包</h3>
    <div style="font-size:14px;color:#334155;margin-bottom:8px;">分析包：<span id="deletePackName"></span></div>
    <div style="font-size:13px;color:#ef4444;margin-bottom:20px;">确定要删除该分析包吗？删除后将不再显示在已购列表中。</div>
    <div style="display:flex;gap:10px;justify-content:flex-end;">
      <button onclick="closeDeleteModal()" style="padding:8px 18px;background:none;border:1px solid #e2e8f0;border-radius:8px;font-size:13px;cursor:pointer;color:#64748b;">取消</button>
      <button onclick="submitDelete()" style="padding:8px 18px;background:#ef4444;color:#fff;border:none;border-radius:8px;font-size:13px;font-weight:500;cursor:pointer;">确认删除</button>
    </div>
  </div>
</div>

<!-- Hidden form for delete submission -->
<form id="deleteForm" method="POST" action="/user/pack/delete" style="display:none;">
  <input type="hidden" name="listing_id" id="deleteListingId">
</form>

</body>
</html>`
