package templates

import (
	"html/template"
	"strings"
)

// storefrontFuncMap provides helper functions for the storefront template.
var storefrontFuncMap = template.FuncMap{
	"truncateDesc": func(s string, maxLen int) string {
		runes := []rune(s)
		if len(runes) <= maxLen {
			return s
		}
		return string(runes[:maxLen]) + "..."
	},
	"firstChar": func(s string) string {
		runes := []rune(strings.TrimSpace(s))
		if len(runes) == 0 {
			return "?"
		}
		return string(runes[0])
	},
}

// StorefrontTmpl is the parsed storefront public page template.
var StorefrontTmpl = template.Must(
	template.New("storefront").Funcs(storefrontFuncMap).Parse(storefrontHTML),
)

const storefrontHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="default-lang" content="{{.DefaultLang}}">
    <title>{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}} - 快捷分析包市场</title>
    <meta property="og:type" content="website" />
    <meta property="og:title" content="{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}}的小铺" />
    <meta property="og:description" content="{{if .Storefront.Description}}{{truncateDesc .Storefront.Description 200}}{{else}}该作者暂未设置小铺描述{{end}}" />
    {{if .Storefront.HasLogo}}<meta property="og:image" content="/store/{{.Storefront.StoreSlug}}/logo" />{{end}}
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}}的小铺" />
    <meta name="twitter:description" content="{{if .Storefront.Description}}{{truncateDesc .Storefront.Description 200}}{{else}}该作者暂未设置小铺描述{{end}}" />
    {{if .Storefront.HasLogo}}<meta name="twitter:image" content="/store/{{.Storefront.StoreSlug}}/logo" />{{end}}
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", sans-serif;
            background: #f0f2f5;
            min-height: 100vh;
            color: #1e293b;
            line-height: 1.6;
        }
        .page { max-width: 960px; margin: 0 auto; padding: 24px 20px 36px; }

        /* Nav */
        .nav {
            display: flex; align-items: center; justify-content: space-between;
            margin-bottom: 24px;
        }
        .logo-link {
            display: flex; align-items: center; gap: 10px; text-decoration: none;
        }
        .logo-mark {
            width: 36px; height: 36px; border-radius: 10px;
            display: flex; align-items: center; justify-content: center;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            font-size: 18px; box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .logo-text { font-size: 15px; font-weight: 700; color: #1e293b; letter-spacing: -0.2px; }
        .nav-link {
            padding: 7px 16px; font-size: 13px; font-weight: 500; color: #64748b;
            background: #fff; border: 1px solid #e2e8f0; border-radius: 8px;
            text-decoration: none; transition: all .2s;
        }
        .nav-link:hover { color: #1e293b; border-color: #cbd5e1; box-shadow: 0 1px 3px rgba(0,0,0,0.06); }

        /* Store header — left/right layout */
        .store-header {
            background: #fff; border-radius: 16px; padding: 32px;
            margin-bottom: 24px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 4px 16px rgba(0,0,0,0.04);
            border: 1px solid #e2e8f0;
            display: flex; gap: 32px; align-items: stretch;
        }
        .store-profile {
            display: flex; flex-direction: column; align-items: center;
            justify-content: center; text-align: center;
            min-width: 220px; flex-shrink: 0;
        }
        .store-avatar {
            width: 80px; height: 80px; border-radius: 20px;
            margin-bottom: 16px; overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .store-avatar img { width: 100%; height: 100%; object-fit: cover; }
        .store-avatar-letter {
            width: 100%; height: 100%;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            display: flex; align-items: center; justify-content: center;
            font-size: 36px; font-weight: 800; color: #fff;
        }
        .store-name { font-size: 22px; font-weight: 800; color: #0f172a; margin-bottom: 8px; letter-spacing: -0.3px; }
        .store-desc { font-size: 13px; color: #64748b; line-height: 1.7; }

        /* Featured section — right side of header */
        .store-featured {
            flex: 1; min-width: 0;
            display: flex; flex-direction: column;
        }
        .store-featured-title {
            font-size: 13px; font-weight: 700; color: #94a3b8;
            margin-bottom: 12px; display: flex; align-items: center; gap: 6px;
            letter-spacing: 0.3px; text-transform: uppercase;
        }
        .featured-grid {
            display: grid; grid-template-columns: repeat(2, 1fr);
            gap: 10px; flex: 1;
        }
        .featured-card {
            background: #f8fafc; border-radius: 10px; padding: 14px;
            border: 1px solid #e2e8f0; text-align: center;
            transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s;
            cursor: pointer; text-decoration: none; display: flex;
            flex-direction: column; align-items: center; justify-content: center;
            color: inherit;
        }
        .featured-card:hover {
            transform: translateY(-2px); background: #fff;
            box-shadow: 0 8px 24px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04);
        }
        .featured-icon {
            width: 40px; height: 40px; border-radius: 10px;
            background: linear-gradient(135deg, #eef2ff, #faf5ff);
            display: flex; align-items: center; justify-content: center;
            margin-bottom: 8px; font-size: 18px;
            border: 1px solid #e0e7ff;
        }
        .featured-name {
            font-size: 12px; font-weight: 700; color: #1e293b;
            margin-bottom: 4px; line-height: 1.4;
            overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
            max-width: 100%;
        }
        .featured-price { font-size: 11px; font-weight: 600; }
        .featured-price.price-free { color: #16a34a; }
        .featured-price.price-paid { color: #6366f1; }
        .featured-empty-slot {
            background: #f8fafc; border-radius: 10px; padding: 14px;
            border: 1px dashed #e2e8f0; display: flex;
            align-items: center; justify-content: center;
            color: #cbd5e1; font-size: 20px;
        }

        /* Filter bar */
        .filter-bar {
            display: flex; align-items: center; gap: 12px;
            margin-bottom: 20px; flex-wrap: wrap;
        }
        .filter-group {
            display: flex; gap: 4px; background: #fff;
            border: 1px solid #e2e8f0; border-radius: 8px; padding: 3px;
        }
        .filter-btn {
            padding: 6px 14px; border: none; border-radius: 6px;
            font-size: 12px; font-weight: 600; cursor: pointer;
            background: transparent; color: #64748b; transition: all 0.15s;
            text-decoration: none; display: inline-block;
        }
        .filter-btn:hover { color: #334155; background: #f8fafc; }
        .filter-btn.active { background: #4f46e5; color: #fff; box-shadow: 0 1px 3px rgba(79,70,229,0.3); }
        .search-input {
            padding: 7px 14px; border: 1px solid #e2e8f0; border-radius: 8px;
            font-size: 13px; background: #fff; min-width: 180px;
            transition: border-color 0.15s, box-shadow 0.15s; color: #1e293b;
        }
        .search-input:focus { outline: none; border-color: #4f46e5; box-shadow: 0 0 0 3px rgba(79,70,229,0.12); }
        .sort-select {
            padding: 7px 14px; border: 1px solid #e2e8f0; border-radius: 8px;
            font-size: 13px; background: #fff; color: #1e293b; cursor: pointer;
            transition: border-color 0.15s;
        }
        .sort-select:focus { outline: none; border-color: #4f46e5; }

        /* Pack list */
        .pack-list { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; }
        .pack-item {
            background: #fff; border-radius: 12px; padding: 18px 20px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 1px 4px rgba(0,0,0,0.04), 0 2px 8px rgba(0,0,0,0.02);
            display: flex; flex-direction: column; gap: 10px;
            transition: transform 0.15s, box-shadow 0.15s;
        }
        .pack-item:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 16px rgba(0,0,0,0.06), 0 2px 8px rgba(0,0,0,0.03);
        }
        .pack-item-body { flex: 1; min-width: 0; }
        .pack-item-header { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; flex-wrap: wrap; }
        .pack-item-name { font-size: 14px; font-weight: 700; color: #1e293b; }
        .tag {
            display: inline-flex; align-items: center;
            padding: 3px 10px; border-radius: 20px;
            font-size: 11px; font-weight: 700; letter-spacing: 0.2px;
        }
        .tag-free { background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; }
        .tag-per-use { background: #eef2ff; color: #4338ca; border: 1px solid #c7d2fe; }
        .tag-subscription { background: #f5f3ff; color: #7c3aed; border: 1px solid #ddd6fe; }
        .pack-item-desc {
            font-size: 13px; color: #64748b; line-height: 1.6;
            margin-bottom: 8px;
            overflow: hidden; text-overflow: ellipsis;
            display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
        }
        .pack-item-meta {
            display: flex; align-items: center; gap: 16px;
            font-size: 12px; color: #94a3b8;
        }
        .pack-item-meta .meta-item { display: flex; align-items: center; gap: 4px; }
        .pack-item-price { font-weight: 700; color: #6366f1; }
        .pack-item-price.price-free { color: #16a34a; }
        .pack-item-actions { flex-shrink: 0; align-self: flex-end; }

        /* Buttons */
        .btn {
            padding: 8px 18px; border: none; border-radius: 8px;
            font-size: 13px; font-weight: 600; cursor: pointer;
            display: inline-flex; align-items: center; gap: 5px;
            text-decoration: none; transition: all 0.2s; font-family: inherit;
        }
        .btn-green {
            background: linear-gradient(135deg, #22c55e, #16a34a); color: #fff;
            box-shadow: 0 2px 8px rgba(34,197,94,0.25);
        }
        .btn-green:hover { box-shadow: 0 4px 16px rgba(34,197,94,0.3); transform: translateY(-1px); }
        .btn-indigo {
            background: linear-gradient(135deg, #6366f1, #4f46e5); color: #fff;
            box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .btn-indigo:hover { box-shadow: 0 4px 16px rgba(99,102,241,0.3); transform: translateY(-1px); }
        .btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none !important; }
        .badge-owned {
            display: inline-flex; align-items: center; gap: 5px;
            padding: 8px 18px; background: #dcfce7; color: #16a34a;
            border: 1px solid #bbf7d0; border-radius: 8px;
            font-size: 13px; font-weight: 600;
        }
        .btn-ghost {
            padding: 8px 18px; font-size: 13px; border-radius: 8px;
            background: #f8fafc; color: #64748b; border: 1px solid #e2e8f0;
            cursor: pointer; transition: all .2s; font-family: inherit; font-weight: 600;
        }
        .btn-ghost:hover { background: #f1f5f9; color: #475569; }

        /* Empty state */
        .empty-state {
            text-align: center; padding: 48px 20px; color: #64748b;
            background: #fff; border-radius: 12px; border: 1px dashed #cbd5e1;
        }
        .empty-state .icon { font-size: 36px; margin-bottom: 12px; opacity: 0.7; }
        .empty-state p { font-size: 14px; }

        /* Modal overlay */
        .modal-overlay {
            display: none; position: fixed; top: 0; left: 0;
            width: 100%; height: 100%;
            background: rgba(15,23,42,0.4); backdrop-filter: blur(4px);
            z-index: 1000; align-items: center; justify-content: center;
        }
        .modal-overlay.show { display: flex; }
        .modal-box {
            background: #fff; border-radius: 14px; padding: 28px 32px;
            max-width: 420px; width: 90%;
            box-shadow: 0 20px 60px rgba(0,0,0,0.15);
            position: relative; border: 1px solid #e2e8f0;
        }
        .modal-close {
            position: absolute; top: 14px; right: 18px;
            background: none; border: none; font-size: 20px; cursor: pointer;
            color: #64748b; width: 32px; height: 32px; border-radius: 8px;
            display: flex; align-items: center; justify-content: center;
            transition: background 0.15s;
        }
        .modal-close:hover { background: #f1f5f9; color: #1e293b; }
        .modal-title { font-size: 17px; font-weight: 700; color: #1e293b; margin-bottom: 20px; }
        .modal-actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px; }

        /* Form fields in modal */
        .field-group { margin-bottom: 14px; }
        .field-group label {
            font-size: 12px; color: #334155; display: block;
            margin-bottom: 5px; font-weight: 600;
        }
        .field-group input, .field-group select {
            width: 100%; padding: 9px 14px;
            border: 1px solid #cbd5e1; border-radius: 8px;
            font-size: 14px; background: #fff;
            transition: border-color 0.15s, box-shadow 0.15s; color: #1e293b;
        }
        .field-group input:focus, .field-group select:focus {
            outline: none; border-color: #4f46e5;
            box-shadow: 0 0 0 3px rgba(79,70,229,0.12);
        }
        .total-price { font-size: 16px; font-weight: 700; color: #6366f1; margin-bottom: 4px; }

        /* Messages */
        .msg { display: none; padding: 12px 16px; border-radius: 10px; font-size: 13px; margin-bottom: 14px; font-weight: 500; }
        .msg-ok { background: #dcfce7; color: #16a34a; border: 1px solid #bbf7d0; }
        .msg-err { background: #fee2e2; color: #dc2626; border: 1px solid #fecaca; }

        /* Footer */
        .foot { text-align: center; margin-top: 28px; padding-top: 16px; border-top: 1px solid #e2e8f0; }
        .foot-text { font-size: 11px; color: #94a3b8; }
        .foot-text a { color: #6366f1; text-decoration: none; }
        .foot-text a:hover { text-decoration: underline; }

        /* Toast */
        .toast {
            position: fixed; bottom: 32px; left: 50%;
            transform: translateX(-50%) translateY(20px);
            background: #1e293b; color: #fff;
            padding: 10px 24px; border-radius: 10px;
            font-size: 13px; font-weight: 500;
            opacity: 0; transition: all .3s; pointer-events: none; z-index: 9999;
            box-shadow: 0 4px 16px rgba(0,0,0,0.2);
        }
        .toast.show { opacity: 1; transform: translateX(-50%) translateY(0); }

        @media (max-width: 640px) {
            .store-header { flex-direction: column; }
            .store-profile { min-width: auto; }
            .filter-bar { flex-direction: column; align-items: stretch; }
            .search-input { min-width: auto; }
            .pack-list { grid-template-columns: 1fr; }
            .pack-item-actions { align-self: flex-end; }
            .featured-grid { grid-template-columns: repeat(2, 1fr); }
        }
    </style>
</head>
<body>
<div class="page">
    <!-- Navigation -->
    <nav class="nav">
        <a class="logo-link" href="/"><span class="logo-mark">📦</span><span class="logo-text" data-i18n="site_name">快捷分析包市场</span></a>
        <div>{{if .IsLoggedIn}}<a class="nav-link" href="/user/dashboard" data-i18n="personal_center">个人中心</a>{{else}}<a class="nav-link" href="/user/login" data-i18n="login">登录</a>{{end}}</div>
    </nav>

    <!-- Store Header: profile left, featured right -->
    <div class="store-header">
        <div class="store-profile">
            <div class="store-avatar">
                {{if .Storefront.HasLogo}}
                <img src="/store/{{.Storefront.StoreSlug}}/logo" alt="{{.Storefront.StoreName}}">
                {{else}}
                <div class="store-avatar-letter">{{firstChar .Storefront.StoreName}}</div>
                {{end}}
            </div>
            <h1 class="store-name">{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}}</h1>
            <p class="store-desc">{{if .Storefront.Description}}{{.Storefront.Description}}{{else}}该作者暂未设置小铺描述{{end}}</p>
        </div>
        {{if .FeaturedPacks}}
        <div class="store-featured">
            <div class="store-featured-title">⭐ <span data-i18n="featured_packs">店主推荐</span></div>
            <div class="featured-grid">
                {{range .FeaturedPacks}}
                <a class="featured-card" href="/pack/{{.ShareToken}}">
                    <div class="featured-icon">📊</div>
                    <div class="featured-name" title="{{.PackName}}">{{.PackName}}</div>
                    {{if eq .ShareMode "free"}}
                    <div class="featured-price price-free" data-i18n="free">免费</div>
                    {{else}}
                    <div class="featured-price price-paid">{{.CreditsPrice}} Credits</div>
                    {{end}}
                </a>
                {{end}}
            </div>
        </div>
        {{end}}
    </div>

    <!-- Messages -->
    <div class="msg msg-ok" id="successMsg"></div>
    <div class="msg msg-err" id="errorMsg"></div>

    <!-- Filter Bar -->
    <div class="filter-bar">
        <div class="filter-group">
            <a class="filter-btn{{if eq .Filter ""}} active{{end}}" href="?filter=&sort={{.Sort}}&q={{.SearchQuery}}" data-i18n="filter_all">全部</a>
            <a class="filter-btn{{if eq .Filter "free"}} active{{end}}" href="?filter=free&sort={{.Sort}}&q={{.SearchQuery}}" data-i18n="free">免费</a>
            <a class="filter-btn{{if eq .Filter "per_use"}} active{{end}}" href="?filter=per_use&sort={{.Sort}}&q={{.SearchQuery}}" data-i18n="per_use">按次收费</a>
            <a class="filter-btn{{if eq .Filter "subscription"}} active{{end}}" href="?filter=subscription&sort={{.Sort}}&q={{.SearchQuery}}" data-i18n="subscription">订阅制</a>
        </div>
        <form id="searchForm" method="GET" style="display:flex;gap:8px;align-items:center;">
            <input type="hidden" name="filter" value="{{.Filter}}">
            <input type="hidden" name="sort" value="{{.Sort}}">
            <input class="search-input" type="text" name="q" value="{{.SearchQuery}}" placeholder="搜索分析包..." data-i18n-placeholder="search_packs">
        </form>
        <select class="sort-select" id="sortSelect" onchange="changeSort(this.value)">
            <option value="revenue"{{if eq .Sort "revenue"}} selected{{end}} data-i18n="sort_revenue">按销售金额</option>
            <option value="downloads"{{if eq .Sort "downloads"}} selected{{end}} data-i18n="sort_downloads">按下载量</option>
            <option value="orders"{{if eq .Sort "orders"}} selected{{end}} data-i18n="sort_orders">按订单数</option>
        </select>
    </div>

    <!-- Pack List -->
    {{if .Packs}}
    <div class="pack-list">
        {{range .Packs}}
        <div class="pack-item">
            <div class="pack-item-body">
                <div class="pack-item-header">
                    <span class="pack-item-name">{{.PackName}}</span>
                    {{if eq .ShareMode "free"}}<span class="tag tag-free" data-i18n="free">免费</span>
                    {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use" data-i18n="per_use">按次收费</span>
                    {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription" data-i18n="subscription">订阅制</span>
                    {{end}}
                </div>
                {{if .PackDesc}}<div class="pack-item-desc">{{.PackDesc}}</div>{{end}}
                <div class="pack-item-meta">
                    {{if eq .ShareMode "free"}}
                    <span class="meta-item"><span class="pack-item-price price-free" data-i18n="free">免费</span></span>
                    {{else}}
                    <span class="meta-item"><span class="pack-item-price">{{.CreditsPrice}} Credits</span></span>
                    {{end}}
                    <span class="meta-item">📥 {{.DownloadCount}}</span>
                </div>
            </div>
            <div class="pack-item-actions">
                {{if $.IsLoggedIn}}
                    {{if index $.PurchasedIDs .ListingID}}
                    <span class="badge-owned">✅ <span data-i18n="already_purchased">已购买</span></span>
                    {{else if eq .ShareMode "free"}}
                    <button class="btn btn-green" onclick="claimPack('{{.ShareToken}}')" data-i18n="claim_free">免费领取</button>
                    {{else}}
                    <button class="btn btn-indigo" onclick="showPurchaseDialog('{{.ShareToken}}', '{{.ShareMode}}', {{.CreditsPrice}}, '{{.PackName}}')" data-i18n="purchase">购买</button>
                    {{end}}
                {{else}}
                    {{if eq .ShareMode "free"}}
                    <a class="btn btn-green" href="/user/login?redirect=/store/{{$.Storefront.StoreSlug}}" data-i18n="login_to_claim">登录后领取</a>
                    {{else}}
                    <a class="btn btn-indigo" href="/user/login?redirect=/store/{{$.Storefront.StoreSlug}}" data-i18n="login_to_buy">登录后购买</a>
                    {{end}}
                {{end}}
            </div>
        </div>
        {{end}}
    </div>
    {{else}}
    <div class="empty-state">
        <div class="icon">📭</div>
        <p data-i18n="storefront_empty">该小铺暂无分析包</p>
    </div>
    {{end}}

    <!-- Footer -->
    <div class="foot">
        <p class="foot-text">Vantagics <span data-i18n="site_name">快捷分析包市场</span> · <a href="/" data-i18n="browse_more">浏览更多</a></p>
    </div>
</div>

<!-- Purchase Dialog Modal -->
<div class="modal-overlay" id="purchaseModal">
    <div class="modal-box">
        <button class="modal-close" onclick="closePurchaseDialog()">✕</button>
        <div class="modal-title" id="purchaseModalTitle" data-i18n="purchase">购买</div>

        <!-- Per-use fields -->
        <div id="perUseFields" style="display:none;">
            <div class="field-group">
                <label for="purchaseQuantity" data-i18n="buy_count_label">购买次数</label>
                <input type="number" id="purchaseQuantity" min="1" value="1" onchange="updatePurchaseTotal()" oninput="updatePurchaseTotal()">
            </div>
        </div>

        <!-- Subscription fields -->
        <div id="subscriptionFields" style="display:none;">
            <div class="field-group">
                <label for="purchaseDuration" data-i18n="sub_duration">订阅时长</label>
                <select id="purchaseDuration" onchange="updatePurchaseTotal()">
                    <optgroup label="按月">
                        <option value="1">1 个月</option>
                        <option value="2">2 个月</option>
                        <option value="3">3 个月</option>
                        <option value="4">4 个月</option>
                        <option value="5">5 个月</option>
                        <option value="6">6 个月</option>
                        <option value="7">7 个月</option>
                        <option value="8">8 个月</option>
                        <option value="9">9 个月</option>
                        <option value="10">10 个月</option>
                        <option value="11">11 个月</option>
                        <option value="12">12 个月</option>
                    </optgroup>
                    <optgroup label="按年">
                        <option value="12">1 年</option>
                        <option value="24">2 年</option>
                        <option value="36">3 年</option>
                    </optgroup>
                </select>
            </div>
        </div>

        <div class="total-price" id="purchaseTotal"></div>
        <div class="modal-actions">
            <button class="btn-ghost" onclick="closePurchaseDialog()" data-i18n="cancel">取消</button>
            <button class="btn btn-indigo" id="confirmPurchaseBtn" onclick="confirmPurchase()" data-i18n="confirm_purchase">确认购买</button>
        </div>
    </div>
</div>

<!-- Toast -->
<div class="toast" id="toast"></div>

<script>
var _currentShareToken = '';
var _currentShareMode = '';
var _currentCreditsPrice = 0;
var _storeSlug = '{{.Storefront.StoreSlug}}';

function showToast(msg) {
    var t = document.getElementById('toast');
    t.textContent = msg;
    t.classList.add('show');
    setTimeout(function() { t.classList.remove('show'); }, 2500);
}

function showMsg(type, msg) {
    var s = document.getElementById('successMsg');
    var e = document.getElementById('errorMsg');
    if (s) s.style.display = 'none';
    if (e) e.style.display = 'none';
    if (type === 'success' && s) { s.textContent = msg; s.style.display = 'block'; }
    else if (e) { e.textContent = msg; e.style.display = 'block'; }
}

function changeSort(val) {
    var params = new URLSearchParams(window.location.search);
    params.set('sort', val);
    window.location.search = params.toString();
}

function claimPack(shareToken) {
    if (!confirm(window._i18n('add_to_purchased_confirm', '是否将此分析包添加到您的已购快捷分析包中？'))) return;
    fetch('/pack/' + shareToken + '/claim', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    }).then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            showMsg('success', window._i18n('claim_success', '领取成功！'));
            setTimeout(function() { location.reload(); }, 1000);
        } else {
            showMsg('error', d.error || window._i18n('claim_failed', '领取失败'));
        }
    }).catch(function() {
        showMsg('error', window._i18n('network_error', '网络错误'));
    });
}

function showPurchaseDialog(shareToken, shareMode, creditsPrice, packName) {
    _currentShareToken = shareToken;
    _currentShareMode = shareMode;
    _currentCreditsPrice = creditsPrice;

    document.getElementById('purchaseModalTitle').textContent =
        window._i18n('purchase', '购买') + ' - ' + packName;

    var perUse = document.getElementById('perUseFields');
    var sub = document.getElementById('subscriptionFields');
    perUse.style.display = 'none';
    sub.style.display = 'none';

    if (shareMode === 'per_use') {
        perUse.style.display = 'block';
        document.getElementById('purchaseQuantity').value = 1;
    } else if (shareMode === 'subscription') {
        sub.style.display = 'block';
        document.getElementById('purchaseDuration').selectedIndex = 0;
    }

    updatePurchaseTotal();
    document.getElementById('purchaseModal').classList.add('show');
}

function closePurchaseDialog() {
    document.getElementById('purchaseModal').classList.remove('show');
}

function updatePurchaseTotal() {
    var total = 0;
    if (_currentShareMode === 'per_use') {
        var q = parseInt(document.getElementById('purchaseQuantity').value) || 1;
        if (q < 1) q = 1;
        total = _currentCreditsPrice * q;
    } else if (_currentShareMode === 'subscription') {
        var m = parseInt(document.getElementById('purchaseDuration').value) || 1;
        total = _currentCreditsPrice * m;
    }
    var el = document.getElementById('purchaseTotal');
    if (el) el.textContent = window._i18n('total', '合计') + '：' + total + ' Credits';
}

function confirmPurchase() {
    var body = {};
    if (_currentShareMode === 'per_use') {
        var q = parseInt(document.getElementById('purchaseQuantity').value) || 1;
        if (q < 1) { showMsg('error', window._i18n('min_1_count', '购买次数至少为 1')); return; }
        body.quantity = q;
    } else if (_currentShareMode === 'subscription') {
        body.months = parseInt(document.getElementById('purchaseDuration').value) || 1;
    }

    var btn = document.getElementById('confirmPurchaseBtn');
    if (btn) { btn.disabled = true; btn.textContent = window._i18n('processing', '处理中...'); }

    fetch('/pack/' + _currentShareToken + '/purchase', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
    }).then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            closePurchaseDialog();
            showMsg('success', window._i18n('purchase_success', '购买成功！'));
            setTimeout(function() { location.reload(); }, 1000);
        } else if (d.insufficient_balance) {
            closePurchaseDialog();
            var errEl = document.getElementById('errorMsg');
            if (errEl) {
                errEl.innerHTML = window._i18n('insufficient_balance', '余额不足，当前余额') + ' ' + (d.balance || 0) + ' Credits。<a href="/user/dashboard" style="color:#4f46e5;text-decoration:underline;font-weight:600;">' + window._i18n('go_topup', '前往充值') + '</a>';
                errEl.style.display = 'block';
            }
            if (btn) { btn.disabled = false; btn.textContent = window._i18n('confirm_purchase', '确认购买'); }
        } else {
            showMsg('error', d.error || window._i18n('purchase_failed', '购买失败'));
            if (btn) { btn.disabled = false; btn.textContent = window._i18n('confirm_purchase', '确认购买'); }
        }
    }).catch(function() {
        showMsg('error', window._i18n('network_error', '网络错误'));
        if (btn) { btn.disabled = false; btn.textContent = window._i18n('confirm_purchase', '确认购买'); }
    });
}
</script>
` + I18nJS + `
</body>
</html>`
