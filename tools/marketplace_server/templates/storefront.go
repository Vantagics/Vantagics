package templates

import (
	"fmt"
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
	"formatPriceUSD": func(price float64) string {
		if price == float64(int(price)) {
			return fmt.Sprintf("$%.0f", price)
		}
		return fmt.Sprintf("$%.2f", price)
	},
	"productTypeLabel": func(productType string) string {
		switch productType {
		case "credits":
			return "积分充值"
		case "virtual_goods":
			return "虚拟商品"
		default:
			return productType
		}
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
    <title>{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}} - 分析技能包市场</title>
    <meta property="og:type" content="website" />
    <meta property="og:title" content="{{if .Storefront.StoreName}}{{.Storefront.StoreName}}的小铺{{else}}小铺{{end}}" />
    <meta property="og:description" content="{{if .Storefront.Description}}{{truncateDesc .Storefront.Description 200}}{{else}}该作者暂未设置小铺描述{{end}}" />
    {{if .Storefront.HasLogo}}<meta property="og:image" content="/store/{{.Storefront.StoreSlug}}/logo" />{{end}}
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}}" />
    <meta name="twitter:description" content="{{if .Storefront.Description}}{{truncateDesc .Storefront.Description 200}}{{else}}该作者暂未设置小铺描述{{end}}" />
    {{if .Storefront.HasLogo}}<meta name="twitter:image" content="/store/{{.Storefront.StoreSlug}}/logo" />{{end}}
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap" rel="stylesheet">
    <style>:root { {{.ThemeCSS}} }</style>
    <style>
        *,*::before,*::after { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #f8f9fc;
            min-height: 100vh;
            color: #1e293b;
            line-height: 1.6;
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }
        .page { max-width: 1000px; margin: 0 auto; padding: 20px 24px 48px; }

        /* ── Nav ── */
        .nav {
            display: flex; align-items: center; justify-content: space-between;
            margin-bottom: 28px; padding: 12px 16px;
            background: #fff; border-radius: 14px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04);
        }
        .logo-link {
            display: flex; align-items: center; gap: 10px; text-decoration: none;
        }
        .logo-mark {
            width: 36px; height: 36px; border-radius: 10px;
            display: flex; align-items: center; justify-content: center;
            background: linear-gradient(135deg, #6366f1, #4f46e5);
            box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .logo-mark svg { width: 20px; height: 20px; fill: none; stroke: #fff; }
        .logo-text { font-size: 15px; font-weight: 700; color: #0f172a; letter-spacing: -0.3px; }
        .nav-actions { display: flex; align-items: center; gap: 8px; }
        .nav-link {
            padding: 8px 18px; font-size: 13px; font-weight: 600; color: #fff;
            background: linear-gradient(135deg, #312e81, #1e1b4b);
            border: 1px solid transparent; border-radius: 10px;
            text-decoration: none; transition: all .2s;
            box-shadow: 0 2px 8px rgba(49,46,129,0.3);
            text-shadow: 0 1px 2px rgba(0,0,0,0.2);
        }
        .nav-link:hover { box-shadow: 0 4px 12px rgba(49,46,129,0.4); transform: translateY(-1px); }

        /* ── Hero / Store Header ── */
        .store-hero {
            position: relative; overflow: hidden;
            background: var(--hero-gradient);
            border: 1px solid var(--card-border);
            border-radius: 20px; padding: 36px 36px 32px;
            margin-bottom: 28px;
        }
        .store-hero::before {
            content: ''; position: absolute; top: -60px; right: -60px;
            width: 200px; height: 200px; border-radius: 50%;
            background: radial-gradient(circle, rgba(99,102,241,0.08) 0%, transparent 70%);
            pointer-events: none;
        }
        .store-hero::after {
            content: ''; position: absolute; bottom: -40px; left: 30%;
            width: 160px; height: 160px; border-radius: 50%;
            background: radial-gradient(circle, rgba(139,92,246,0.06) 0%, transparent 70%);
            pointer-events: none;
        }
        .store-hero-inner {
            position: relative; z-index: 1;
            display: flex; gap: 36px; align-items: stretch;
        }
        .store-hero-inner.hero-reversed {
            flex-direction: row-reverse;
        }
        .store-profile {
            display: flex; flex-direction: column; align-items: center;
            justify-content: center; text-align: center;
            min-width: 200px; flex-shrink: 0;
        }
        .store-avatar {
            width: 88px; height: 88px; border-radius: 22px;
            margin-bottom: 16px; overflow: hidden;
            box-shadow: 0 4px 16px rgba(0,0,0,0.1);
            border: 3px solid rgba(255,255,255,0.8);
        }
        .store-avatar img { width: 100%; height: 100%; object-fit: cover; }
        .store-avatar-letter {
            width: 100%; height: 100%;
            background: linear-gradient(135deg, var(--primary-color), var(--accent-color));
            display: flex; align-items: center; justify-content: center;
            font-size: 38px; font-weight: 800; color: #fff;
        }
        .store-name {
            font-size: 22px; font-weight: 800; color: #0f172a;
            margin-bottom: 8px; letter-spacing: -0.4px;
        }
        .store-desc {
            font-size: 13px; color: #475569; line-height: 1.7;
            max-width: 220px;
        }
        .store-stats {
            display: flex; gap: 16px; margin-top: 14px;
        }
        .store-stat {
            display: flex; flex-direction: column; align-items: center;
            padding: 6px 12px; background: rgba(255,255,255,0.6);
            border-radius: 8px; border: 1px solid rgba(226,232,240,0.5);
        }
        .store-stat-val { font-size: 16px; font-weight: 800; color: var(--primary-hover); }
        .store-stat-label { font-size: 10px; color: #64748b; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }

        /* ── Featured Section ── */
        .store-featured {
            flex: 1; min-width: 0;
            display: flex; flex-direction: column;
        }
        .store-featured-title {
            font-size: 11px; font-weight: 700; color: #475569;
            margin-bottom: 12px; display: flex; align-items: center; gap: 6px;
            letter-spacing: 0.8px; text-transform: uppercase;
        }
        .store-featured-title svg { width: 14px; height: 14px; color: #f59e0b; }
        .featured-grid {
            display: grid; grid-template-columns: repeat(2, 1fr);
            gap: 10px; flex: 1;
        }
        .featured-card {
            background: linear-gradient(135deg, #f3f0ff 0%, #eef2ff 50%, #faf5ff 100%); border-radius: 14px; padding: 18px 16px 14px;
            border: 1px solid #ddd6fe;
            cursor: pointer; text-decoration: none;
            display: flex; flex-direction: column; align-items: flex-start;
            color: inherit;
            transition: all 0.25s cubic-bezier(.4,0,.2,1);
            position: relative; overflow: hidden;
        }
        .featured-card:hover {
            transform: translateY(-3px); background: #fff;
            box-shadow: 0 8px 32px rgba(99,102,241,0.1), 0 2px 8px rgba(0,0,0,0.04);
            border-color: #c7d2fe;
        }
        .featured-card-top {
            display: flex; align-items: center; gap: 10px; width: 100%; margin-bottom: 8px;
        }
        .featured-icon {
            width: 36px; height: 36px; border-radius: 10px; flex-shrink: 0;
            background: #6366f1; background: linear-gradient(135deg, var(--primary-color, #6366f1), var(--accent-color, #8b5cf6));
            display: flex; align-items: center; justify-content: center;
            box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .featured-icon svg { width: 18px; height: 18px; color: #fff; stroke: #fff; fill: none; }
        .featured-icon-img {
            width: 36px; height: 36px; border-radius: 10px;
            object-fit: cover; flex-shrink: 0;
            box-shadow: 0 2px 8px rgba(99,102,241,0.2);
            display: none;
        }
        .featured-card-title {
            flex: 1; min-width: 0;
        }
        .featured-name {
            font-size: 13px; font-weight: 700; color: #1e293b;
            line-height: 1.3;
            overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
            max-width: 100%;
        }
        .featured-tag {
            display: inline-block; padding: 1px 7px; border-radius: 10px;
            font-size: 10px; font-weight: 600; margin-top: 2px;
        }
        .featured-tag-free { background: #dcfce7; color: #15803d; }
        .featured-tag-per_use { background: #e0e7ff; color: #3730a3; }
        .featured-tag-subscription { background: #ede9fe; color: #6d28d9; }
        .featured-desc {
            font-size: 11px; color: #334155; line-height: 1.5;
            margin-bottom: 10px; flex: 1;
            overflow: hidden; text-overflow: ellipsis;
            display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
        }
        .featured-footer {
            display: flex; align-items: center; justify-content: space-between;
            width: 100%; padding-top: 8px; border-top: 1px solid #e2e8f0;
        }
        .featured-price { font-size: 12px; font-weight: 800; }
        .featured-price.price-free { color: #15803d; }
        .featured-price.price-paid { color: #4338ca; }
        .featured-downloads {
            display: flex; align-items: center; gap: 3px;
            font-size: 11px; color: #475569; font-weight: 500;
        }
        .featured-downloads svg { width: 12px; height: 12px; opacity: 0.8; }
        .featured-empty-slot {
            background: rgba(255,255,255,0.4); border-radius: 14px; padding: 16px;
            border: 1px dashed rgba(203,213,225,0.6); display: flex;
            align-items: center; justify-content: center;
            color: #cbd5e1; font-size: 20px;
        }

        /* ── Download Button (storefront hero) ── */
        .store-featured-header {
            display: flex; align-items: center; justify-content: space-between;
            margin-bottom: 12px;
        }
        .store-featured-header .store-featured-title { margin-bottom: 0; }
        .sf-dl-btn {
            display: inline-flex; align-items: center; gap: 6px;
            padding: 7px 16px; border-radius: 10px;
            font-size: 12px; font-weight: 600; text-decoration: none;
            transition: all .25s cubic-bezier(.4,0,.2,1);
            border: 1px solid var(--card-border);
            background: rgba(255,255,255,0.75); color: var(--primary-hover);
            backdrop-filter: blur(8px);
        }
        .sf-dl-btn:hover {
            background: rgba(255,255,255,0.85); border-color: #c7d2fe;
            box-shadow: 0 4px 16px rgba(99,102,241,0.15);
            transform: translateY(-1px);
            color: var(--primary-hover);
        }
        .sf-dl-btn-primary {
            background: linear-gradient(135deg, #312e81, #1e1b4b); color: #fff;
            border-color: transparent;
            box-shadow: 0 2px 8px rgba(49,46,129,0.4);
            text-shadow: 0 1px 2px rgba(0,0,0,0.2);
        }
        .sf-dl-btn-primary:hover {
            background: linear-gradient(135deg, #3730a3, #312e81);
            box-shadow: 0 4px 16px rgba(49,46,129,0.5); color: #fff;
        }
        .sf-dl-btn svg { width: 16px; height: 16px; flex-shrink: 0; }

        /* ── Filter Bar ── */
        .filter-bar {
            display: flex; align-items: center; gap: 12px;
            margin-bottom: 20px; flex-wrap: wrap;
        }
        .filter-group {
            display: flex; gap: 4px; background: #fff;
            border: 1px solid #e2e8f0; border-radius: 10px; padding: 3px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06);
        }
        .filter-btn {
            padding: 7px 16px; border: none; border-radius: 8px;
            font-size: 12px; font-weight: 600; cursor: pointer;
            background: transparent; color: #334155; transition: all 0.2s;
            text-decoration: none; display: inline-block;
        }
        .filter-btn:hover { color: #0f172a; background: #f1f5f9; }
        .filter-btn.active {
            background: linear-gradient(135deg, #312e81, #1e1b4b); color: #fff;
            box-shadow: 0 2px 8px rgba(49,46,129,0.35);
            text-shadow: 0 1px 2px rgba(0,0,0,0.2);
        }
        .search-input {
            padding: 8px 16px; border: 1px solid #cbd5e1; border-radius: 10px;
            font-size: 13px; background: #fff; min-width: 200px;
            transition: all 0.2s; color: #1e293b;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06);
        }
        .search-input:focus { outline: none; border-color: var(--primary-color); box-shadow: 0 0 0 3px rgba(99,102,241,0.1); }
        .search-input::placeholder { color: #94a3b8; }
        .sort-select {
            padding: 8px 16px; border: 1px solid #cbd5e1; border-radius: 10px;
            font-size: 13px; background: #fff; color: #1e293b; cursor: pointer;
            transition: all 0.2s; box-shadow: 0 1px 3px rgba(0,0,0,0.06);
        }
        .sort-select:focus { outline: none; border-color: var(--primary-color); }

        /* ── Pack Grid ── */
        .pack-list { display: grid; grid-template-columns: repeat(2, 1fr); gap: 14px; }
        .pack-item {
            background: linear-gradient(135deg, #fafbff 0%, #f5f7ff 100%); border-radius: 14px; padding: 22px 24px;
            border: 1px solid #e0e7ff;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04);
            display: flex; flex-direction: column; gap: 12px;
            transition: all 0.25s cubic-bezier(.4,0,.2,1);
            position: relative;
        }
        .pack-item:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 32px rgba(0,0,0,0.06), 0 2px 8px rgba(0,0,0,0.03);
            border-color: #c7d2fe;
        }
        .pack-item-body { flex: 1; min-width: 0; }
        .pack-item-header { display: flex; align-items: center; gap: 10px; margin-bottom: 4px; }
        .pack-item-tags { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-bottom: 8px; }
        .pack-item-icon {
            width: 32px; height: 32px; border-radius: 8px; flex-shrink: 0;
            background: #6366f1; background: linear-gradient(135deg, var(--primary-color, #6366f1), var(--accent-color, #8b5cf6));
            display: flex; align-items: center; justify-content: center;
            box-shadow: 0 2px 6px rgba(99,102,241,0.15);
        }
        .pack-item-icon svg { width: 16px; height: 16px; color: #fff; }
        .pack-item-icon-img {
            width: 32px; height: 32px; border-radius: 8px;
            object-fit: cover; flex-shrink: 0;
            box-shadow: 0 2px 6px rgba(99,102,241,0.15);
            display: none;
        }
        .pack-item-icon-img[src=""], .pack-item-icon-img:not([src]) { display: none !important; }
        .pack-item-icon-wrap, .featured-icon-wrap {
            position: relative; display: inline-flex; flex-shrink: 0;
        }
        .pack-item-icon-wrap .pack-item-icon-img,
        .featured-icon-wrap .featured-icon-img {
            position: absolute; top: 0; left: 0; z-index: 1;
        }
        .pack-item-name { font-size: 15px; font-weight: 700; color: #0f172a; letter-spacing: -0.2px; }
        .tag {
            display: inline-flex; align-items: center;
            padding: 3px 10px; border-radius: 20px;
            font-size: 10px; font-weight: 700; letter-spacing: 0.3px;
            text-transform: uppercase;
        }
        .tag-free { background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; }
        .tag-per-use { background: #eef2ff; color: #4338ca; border: 1px solid #c7d2fe; }
        .tag-subscription { background: #f5f3ff; color: #7c3aed; border: 1px solid #ddd6fe; }
        .tag-category { background: #f0f9ff; color: #0369a1; border: 1px solid #bae6fd; }
        .pack-item-desc {
            font-size: 13px; color: #64748b; line-height: 1.7;
            margin-bottom: 12px;
            overflow: hidden; text-overflow: ellipsis;
            display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
        }
        .pack-item-footer {
            display: flex; align-items: center; justify-content: space-between;
            padding-top: 12px; border-top: 1px solid #f1f5f9;
        }
        .pack-item-meta {
            display: flex; align-items: center; gap: 14px;
            font-size: 12px; color: #64748b;
        }
        .pack-item-meta .meta-item { display: flex; align-items: center; gap: 4px; }
        .pack-item-meta .meta-item svg { width: 14px; height: 14px; opacity: 0.7; }
        .pack-item-price { font-weight: 800; color: var(--primary-hover); font-size: 14px; letter-spacing: -0.2px; }
        .pack-item-price.price-free { color: #16a34a; }

        /* ── Buttons ── */
        .btn {
            padding: 9px 20px; border: none; border-radius: 10px;
            font-size: 13px; font-weight: 600; cursor: pointer;
            display: inline-flex; align-items: center; gap: 6px;
            text-decoration: none; transition: all 0.25s cubic-bezier(.4,0,.2,1);
            font-family: inherit;
        }
        .btn-green {
            background: linear-gradient(135deg, #22c55e, #16a34a); color: #fff;
            box-shadow: 0 2px 8px rgba(34,197,94,0.25);
        }
        .btn-green:hover { box-shadow: 0 4px 16px rgba(34,197,94,0.3); transform: translateY(-1px); }
        .btn-indigo {
            background-color: #6366f1; background-image: linear-gradient(135deg, var(--primary-color), var(--primary-hover)); color: #fff;
            box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .btn-indigo:hover { box-shadow: 0 4px 16px rgba(99,102,241,0.3); transform: translateY(-1px); }
        a.btn-indigo, a.btn-indigo:visited, a.btn-indigo:link, a.btn-indigo:active { color: #fff !important; }
        a.btn-green, a.btn-green:visited, a.btn-green:link, a.btn-green:active { color: #fff !important; }
        .btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none !important; }
        .badge-owned {
            display: inline-flex; align-items: center; gap: 6px;
            padding: 8px 16px; background: #ecfdf5; color: #059669;
            border: 1px solid #a7f3d0; border-radius: 10px;
            font-size: 12px; font-weight: 700; letter-spacing: 0.2px;
        }
        .badge-owned svg { width: 14px; height: 14px; }
        .btn-ghost {
            padding: 9px 20px; font-size: 13px; border-radius: 10px;
            background: #f8fafc; color: #64748b; border: 1px solid #e2e8f0;
            cursor: pointer; transition: all .2s; font-family: inherit; font-weight: 600;
        }
        .btn-ghost:hover { background: #f1f5f9; color: #475569; }

        /* ── Empty State ── */
        .empty-state {
            text-align: center; padding: 56px 24px; color: #64748b;
            background: #fff; border-radius: 16px; border: 1px dashed #cbd5e1;
        }
        .empty-state .icon { font-size: 40px; margin-bottom: 14px; opacity: 0.5; }
        .empty-state p { font-size: 14px; font-weight: 500; }

        /* ── Modal ── */
        .modal-overlay {
            display: none; position: fixed; top: 0; left: 0;
            width: 100%; height: 100%;
            background: rgba(15,23,42,0.5); backdrop-filter: blur(6px);
            z-index: 1000; align-items: center; justify-content: center;
        }
        .modal-overlay.show { display: flex; }
        .modal-box {
            background: #fff; border-radius: 18px; padding: 32px;
            max-width: 420px; width: 90%;
            box-shadow: 0 24px 64px rgba(0,0,0,0.15), 0 8px 24px rgba(0,0,0,0.08);
            position: relative; border: 1px solid #e2e8f0;
        }
        .modal-close {
            position: absolute; top: 16px; right: 18px;
            background: none; border: none; font-size: 18px; cursor: pointer;
            color: #94a3b8; width: 32px; height: 32px; border-radius: 8px;
            display: flex; align-items: center; justify-content: center;
            transition: all 0.15s;
        }
        .modal-close:hover { background: #f1f5f9; color: #475569; }
        .modal-title { font-size: 17px; font-weight: 700; color: #0f172a; margin-bottom: 22px; letter-spacing: -0.2px; }
        .modal-actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 22px; }

        /* ── Form Fields ── */
        .field-group { margin-bottom: 16px; }
        .field-group label {
            font-size: 12px; color: #475569; display: block;
            margin-bottom: 6px; font-weight: 600;
        }
        .field-group input, .field-group select {
            width: 100%; padding: 10px 14px;
            border: 1px solid #e2e8f0; border-radius: 10px;
            font-size: 14px; background: #f8fafc;
            transition: all 0.2s; color: #1e293b; font-family: inherit;
        }
        .field-group input:focus, .field-group select:focus {
            outline: none; border-color: var(--primary-color); background: #fff;
            box-shadow: 0 0 0 3px rgba(99,102,241,0.1);
        }
        .total-price { font-size: 18px; font-weight: 800; color: var(--primary-hover); margin-bottom: 4px; letter-spacing: -0.3px; }

        /* ── Messages ── */
        .msg { display: none; padding: 14px 18px; border-radius: 12px; font-size: 13px; margin-bottom: 16px; font-weight: 600; }
        .msg-ok { background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; }
        .msg-err { background: #fef2f2; color: #dc2626; border: 1px solid #fecaca; }

        /* ── Footer ── */
        .foot { text-align: center; margin-top: 36px; padding-top: 20px; border-top: 1px solid #e2e8f0; }
        .foot-text { font-size: 12px; color: #94a3b8; font-weight: 500; }
        .foot-text a { color: var(--primary-color); text-decoration: none; font-weight: 600; }
        .foot-text a:hover { text-decoration: underline; }
        .powered-by {
            margin-top: 10px; font-size: 11px; color: #b0b8c9; font-weight: 500;
            display: flex; align-items: center; justify-content: center; gap: 5px;
        }
        .powered-by a {
            color: var(--primary-color); text-decoration: none; font-weight: 600;
            display: inline-flex; align-items: center; gap: 4px;
        }
        .powered-by a:hover { text-decoration: underline; }
        .powered-by svg { width: 14px; height: 14px; flex-shrink: 0; }

        /* ── Toast ── */
        .toast {
            position: fixed; bottom: 32px; left: 50%;
            transform: translateX(-50%) translateY(20px);
            background: #1e293b; color: #fff;
            padding: 12px 28px; border-radius: 12px;
            font-size: 13px; font-weight: 600;
            opacity: 0; transition: all .3s; pointer-events: none; z-index: 9999;
            box-shadow: 0 8px 24px rgba(0,0,0,0.2);
        }
        .toast.show { opacity: 1; transform: translateX(-50%) translateY(0); }

        @media (max-width: 640px) {
            .page { padding: 16px 16px 36px; }
            .store-hero { padding: 24px; border-radius: 16px; }
            .store-hero-inner { flex-direction: column; gap: 24px; }
            .store-profile { min-width: auto; }
            .store-stats { justify-content: center; }
            .filter-bar { flex-direction: column; align-items: stretch; }
            .search-input { min-width: auto; }
            .pack-list { grid-template-columns: 1fr !important; }
            .featured-grid { grid-template-columns: repeat(2, 1fr); }
        }
    </style>
</head>
<body>
{{if .IsPreviewMode}}
<div class="preview-banner" style="background:#fef3c7;color:#92400e;text-align:center;padding:10px 16px;font-size:14px;font-weight:600;border-bottom:2px solid #fde68a;position:sticky;top:0;z-index:9999;" data-i18n="preview_mode_banner">
    🔍 预览模式 — 仅作者可见此提示，访客看到的页面不会包含此横幅
</div>
{{end}}
<div class="page">
    <!-- Navigation -->
    <nav class="nav">
        <a class="logo-link" href="/">
            <span class="logo-mark">
                <svg viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
            </span>
            <span class="logo-text" data-i18n="site_name">分析技能包市场</span>
        </a>
        <div class="nav-actions">
            {{if .IsLoggedIn}}<a class="nav-link" href="/user/dashboard" data-i18n="personal_center">个人中心</a>{{else}}<a class="nav-link" href="/user/login" data-i18n="login">登录</a>{{end}}
        </div>
    </nav>

    <!-- Messages -->
    <div class="msg msg-ok" id="successMsg"></div>
    <div class="msg msg-err" id="errorMsg"></div>

    <!-- Dynamic Sections -->
    {{range $index, $section := .Sections}}{{if $section.Visible}}
    {{if eq $section.Type "hero"}}
    <!-- Store Hero -->
    <div class="store-hero" data-section-type="{{.Type}}">
        <div class="store-hero-inner{{if eq $.HeroLayout "reversed"}} hero-reversed{{end}}">
            <div class="store-profile">
                <div class="store-avatar">
                    {{if $.Storefront.HasLogo}}
                    <img src="/store/{{$.Storefront.StoreSlug}}/logo" alt="{{$.Storefront.StoreName}}">
                    {{else}}
                    <div class="store-avatar-letter">{{firstChar $.Storefront.StoreName}}</div>
                    {{end}}
                </div>
                <h1 class="store-name">{{if $.Storefront.StoreName}}{{$.Storefront.StoreName}}{{else}}小铺{{end}}</h1>
                <p class="store-desc">{{if $.Storefront.Description}}{{$.Storefront.Description}}{{else}}该作者暂未设置小铺描述{{end}}</p>
                <div class="store-stats">
                    <div class="store-stat">
                        <span class="store-stat-val">{{len $.Packs}}</span>
                        <span class="store-stat-label" data-i18n="stat_packs">分析包</span>
                    </div>
                    {{if and $.FeaturedPacks $.FeaturedVisible}}
                    <div class="store-stat">
                        <span class="store-stat-val">{{len $.FeaturedPacks}}</span>
                        <span class="store-stat-label" data-i18n="stat_featured">推荐</span>
                    </div>
                    {{end}}
                </div>
            </div>
            {{if and $.FeaturedPacks $.FeaturedVisible}}
            <div class="store-featured">
                <div class="store-featured-header">
                    <div class="store-featured-title">
                        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg>
                        <span data-i18n="featured_packs">店主推荐</span>
                    </div>
                    {{if or $.DownloadURLWindows $.DownloadURLMacOS}}<span id="sfDlBtn"></span>{{end}}
                </div>
                <div class="featured-grid">
                    {{range $.FeaturedPacks}}
                    <a class="featured-card" href="/pack/{{.ShareToken}}" target="_blank" rel="noopener">
                        <div class="featured-card-top">
                            {{if .HasLogo}}
                            <span class="featured-icon-wrap">
                                <img class="featured-icon-img" src="/store/{{$.Storefront.StoreSlug}}/featured/{{.ListingID}}/logo" alt="{{.PackName}}" onload="if(this.naturalWidth>0){this.parentNode.querySelector('.featured-icon').style.display='none';this.style.display='block';}" onerror="this.style.display='none';">
                                <div class="featured-icon">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                                </div>
                            </span>
                            {{else}}
                            <div class="featured-icon">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                            </div>
                            {{end}}
                            <div class="featured-card-title">
                                <div class="featured-name" title="{{.PackName}}">{{.PackName}}</div>
                                {{if eq .ShareMode "free"}}<span class="featured-tag featured-tag-free" data-i18n="free">免费</span>
                                {{else if eq .ShareMode "per_use"}}<span class="featured-tag featured-tag-per_use" data-i18n="per_use">按次收费</span>
                                {{else if eq .ShareMode "subscription"}}<span class="featured-tag featured-tag-subscription" data-i18n="subscription">订阅制</span>
                                {{end}}
                            </div>
                        </div>
                        {{if .PackDesc}}<div class="featured-desc">{{.PackDesc}}</div>
                        {{else}}<div class="featured-desc" style="color:#94a3b8;font-style:italic;" data-i18n="no_description">暂无描述</div>
                        {{end}}
                        <div class="featured-footer">
                            {{if eq .ShareMode "free"}}
                            <span class="featured-price price-free" data-i18n="free">免费</span>
                            {{else}}
                            <span class="featured-price price-paid">{{.CreditsPrice}} Credits</span>
                            {{end}}
                            <span class="featured-downloads">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                                {{.DownloadCount}}
                            </span>
                        </div>
                    </a>
                    {{end}}
                </div>
            </div>
            {{end}}
            {{if or (not $.FeaturedPacks) (not $.FeaturedVisible)}}
            {{if or $.DownloadURLWindows $.DownloadURLMacOS}}
            <div class="store-featured" style="justify-content:flex-start;">
                <span id="sfDlBtn"></span>
            </div>
            {{end}}
            {{end}}
        </div>
    </div>
    {{else if eq .Type "featured"}}
    <!-- Featured Packs (standalone section) -->
    {{if $.FeaturedPacks}}
    <div data-section-type="{{.Type}}">
    </div>
    {{end}}
    {{else if eq .Type "filter_bar"}}
    <!-- Filter Bar -->
    <div class="filter-bar" data-section-type="{{.Type}}">
        <div class="filter-group">
            <a class="filter-btn{{if eq $.Filter ""}} active{{end}}" href="?filter=&sort={{$.Sort}}&q={{$.SearchQuery}}&cat={{$.CategoryFilter}}" data-i18n="filter_all">全部</a>
            <a class="filter-btn{{if eq $.Filter "free"}} active{{end}}" href="?filter=free&sort={{$.Sort}}&q={{$.SearchQuery}}&cat={{$.CategoryFilter}}" data-i18n="free">免费</a>
            <a class="filter-btn{{if eq $.Filter "per_use"}} active{{end}}" href="?filter=per_use&sort={{$.Sort}}&q={{$.SearchQuery}}&cat={{$.CategoryFilter}}" data-i18n="per_use">按次收费</a>
            <a class="filter-btn{{if eq $.Filter "subscription"}} active{{end}}" href="?filter=subscription&sort={{$.Sort}}&q={{$.SearchQuery}}&cat={{$.CategoryFilter}}" data-i18n="subscription">订阅制</a>
        </div>
        {{if $.Categories}}
        <select class="sort-select" id="catSelect" onchange="changeCat(this.value)">
            <option value=""{{if eq $.CategoryFilter ""}} selected{{end}} data-i18n="all_categories">全部类别</option>
            {{range $.Categories}}
            <option value="{{.}}"{{if eq $.CategoryFilter .}} selected{{end}}>{{.}}</option>
            {{end}}
        </select>
        {{end}}
        <form id="searchForm" method="GET" style="display:flex;gap:8px;align-items:center;">
            <input type="hidden" name="filter" value="{{$.Filter}}">
            <input type="hidden" name="sort" value="{{$.Sort}}">
            <input type="hidden" name="cat" value="{{$.CategoryFilter}}">
            <input class="search-input" type="text" name="q" value="{{$.SearchQuery}}" placeholder="搜索分析包..." data-i18n-placeholder="search_packs">
        </form>
        <select class="sort-select" id="sortSelect" onchange="changeSort(this.value)">
            <option value="revenue"{{if eq $.Sort "revenue"}} selected{{end}} data-i18n="sort_revenue">按销售金额</option>
            <option value="downloads"{{if eq $.Sort "downloads"}} selected{{end}} data-i18n="sort_downloads">按下载量</option>
            <option value="orders"{{if eq $.Sort "orders"}} selected{{end}} data-i18n="sort_orders">按订单数</option>
        </select>
    </div>
    {{else if eq .Type "pack_grid"}}
    <!-- Pack List -->
    <div data-section-type="{{.Type}}">
    {{if $.Packs}}
    <div class="pack-list" style="grid-template-columns: repeat({{$.PackGridColumns}}, 1fr);">
        {{range $.Packs}}
        <div class="pack-item">
            <div class="pack-item-body">
                <div class="pack-item-header">
                    {{if .HasLogo}}
                    <span class="pack-item-icon-wrap">
                        <img class="pack-item-icon-img" src="/store/{{$.Storefront.StoreSlug}}/featured/{{.ListingID}}/logo" alt="{{.PackName}}" onload="if(this.naturalWidth>0){this.parentNode.querySelector('.pack-item-icon').style.display='none';this.style.display='block';}" onerror="this.style.display='none';">
                        <div class="pack-item-icon">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                        </div>
                    </span>
                    {{else}}
                    <div class="pack-item-icon">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                    </div>
                    {{end}}
                    <span class="pack-item-name">{{.PackName}}</span>
                </div>
                <div class="pack-item-tags">
                    {{if eq .ShareMode "free"}}<span class="tag tag-free" data-i18n="free">免费</span>
                    {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use" data-i18n="per_use">按次收费</span>
                    {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription" data-i18n="subscription">订阅制</span>
                    {{end}}
                    {{if .CategoryName}}<span class="tag tag-category">{{.CategoryName}}</span>{{end}}
                </div>
                {{if .PackDesc}}<div class="pack-item-desc">{{.PackDesc}}</div>{{end}}
            </div>
            <div class="pack-item-footer">
                <div class="pack-item-meta">
                    {{if eq .ShareMode "free"}}
                    <span class="meta-item"><span class="pack-item-price price-free" data-i18n="free">免费</span></span>
                    {{else}}
                    <span class="meta-item"><span class="pack-item-price">{{.CreditsPrice}} Credits</span></span>
                    {{end}}
                    <span class="meta-item">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                        {{.DownloadCount}}
                    </span>
                </div>
                <div class="pack-item-actions">
                    {{if $.IsLoggedIn}}
                        {{if index $.PurchasedIDs .ListingID}}
                        <span class="badge-owned">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                            <span data-i18n="already_purchased">已购买</span>
                        </span>
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
        </div>
        {{end}}
    </div>
    {{else}}
    <div class="empty-state">
        <div class="icon">📭</div>
        <p data-i18n="storefront_empty">该小铺暂无分析包</p>
    </div>
    {{end}}
    </div>
    {{else if eq .Type "custom_banner"}}
    <!-- Custom Banner -->
    {{with index $.BannerData $index}}{{if .Text}}
    <div data-section-type="custom_banner" style="padding: 16px 20px; border-radius: 12px; margin-bottom: 20px; font-size: 14px; font-weight: 500; line-height: 1.6; border: 1px solid {{if eq .Style "success"}}#bbf7d0{{else if eq .Style "warning"}}#fde68a{{else}}#bfdbfe{{end}}; background: {{if eq .Style "success"}}#f0fdf4{{else if eq .Style "warning"}}#fffbeb{{else}}#eff6ff{{end}}; color: {{if eq .Style "success"}}#166534{{else if eq .Style "warning"}}#92400e{{else}}#1e40af{{end}};">{{.Text}}</div>
    {{end}}{{end}}
    {{end}}
    {{end}}{{end}}

    {{if .CustomProducts}}
    <!-- Custom Products Section -->
    <div class="custom-products-section" style="margin-top: 28px;">
        <div style="font-size: 16px; font-weight: 700; color: #0f172a; margin-bottom: 16px; display: flex; align-items: center; gap: 8px; letter-spacing: -0.2px;">
            <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/><path d="M16 10a4 4 0 0 1-8 0"/></svg>
            <span data-i18n="custom_products">自定义商品</span>
        </div>
        <div class="pack-list" style="grid-template-columns: repeat(2, 1fr);">
            {{range .CustomProducts}}
            <div class="pack-item">
                <div class="pack-item-body">
                    <div class="pack-item-header">
                        <span class="pack-item-name">{{.ProductName}}</span>
                        {{if eq .ProductType "credits"}}<span class="tag" style="background:#fef3c7;color:#b45309;border:1px solid #fde68a;" data-i18n="product_type_credits">积分充值</span>
                        {{else if eq .ProductType "virtual_goods"}}<span class="tag" style="background:#ede9fe;color:#6d28d9;border:1px solid #ddd6fe;" data-i18n="product_type_virtual">虚拟商品</span>
                        {{end}}
                    </div>
                    {{if .Description}}<div class="pack-item-desc">{{.Description}}</div>{{end}}
                </div>
                <div class="pack-item-footer">
                    <div class="pack-item-meta">
                        <span class="meta-item"><span class="pack-item-price" style="color:var(--primary-hover);">{{formatPriceUSD .PriceUSD}} USD</span></span>
                    </div>
                    <div class="pack-item-actions">
                        {{if $.IsLoggedIn}}
                        <button class="btn btn-indigo" onclick="showCustomProductPurchaseDialog({{.ID}}, '{{.ProductName}}', {{.PriceUSD}})" data-i18n="purchase">购买</button>
                        {{else}}
                        <a class="btn btn-indigo" href="/user/login?redirect=/store/{{$.Storefront.StoreSlug}}" data-i18n="login_to_buy">登录后购买</a>
                        {{end}}
                    </div>
                </div>
            </div>
            {{end}}
        </div>
    </div>
    {{end}}

    {{if .CustomProducts}}
    <!-- Custom Product Purchase Dialog -->
    <div class="modal-overlay" id="customProductPurchaseModal">
        <div class="modal-box">
            <button class="modal-close" onclick="closeCustomProductPurchaseDialog()">✕</button>
            <div class="modal-title" id="cpPurchaseTitle" data-i18n="purchase_confirm">购买确认</div>
            <div style="margin-bottom: 16px;">
                <div style="font-size: 14px; color: #475569; margin-bottom: 8px;"><span data-i18n="custom_product_label">商品</span>：<strong id="cpProductName"></strong></div>
                <div class="total-price" id="cpProductPrice"></div>
                <div style="font-size: 13px; color: #64748b; margin-top: 8px; display: flex; align-items: center; gap: 6px;">
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="1" y="4" width="22" height="16" rx="2" ry="2"/><line x1="1" y1="10" x2="23" y2="10"/></svg>
                    <span data-i18n="payment_via_paypal">支付方式：PayPal</span>
                </div>
            </div>
            <div class="modal-actions">
                <button class="btn-ghost" onclick="closeCustomProductPurchaseDialog()" data-i18n="cancel">取消</button>
                <button class="btn btn-indigo" id="cpConfirmBtn" onclick="confirmCustomProductPurchase()" data-i18n="confirm_purchase">确认购买</button>
            </div>
        </div>
    </div>
    {{end}}

    <!-- Footer -->
    <div class="foot">
        <p class="foot-text">Vantagics <span data-i18n="site_name">分析技能包市场</span> · <a href="/" data-i18n="browse_more">浏览更多</a></p>
        <div class="powered-by">
            Powered by
            <a href="https://vantagics.com" target="_blank" rel="noopener">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                Vantagics
            </a>
        </div>
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
var _dlURLWindows = "{{.DownloadURLWindows}}";
var _dlURLMacOS = "{{.DownloadURLMacOS}}";

// Fix broken logo images: show fallback icon when image fails to load
(function(){
    var imgs = document.querySelectorAll('.pack-item-icon-img, .featured-icon-img');
    for (var i = 0; i < imgs.length; i++) {
        (function(img) {
            if (img.complete) {
                if (img.naturalWidth > 0) {
                    img.style.display = 'block';
                    var fallback = img.parentNode.querySelector('.pack-item-icon, .featured-icon');
                    if (fallback) fallback.style.display = 'none';
                } else {
                    img.style.display = 'none';
                }
            }
        })(imgs[i]);
    }
})();

(function(){
    var c = document.getElementById('sfDlBtn');
    if (!c) return;
    if (!_dlURLWindows && !_dlURLMacOS) return;
    function esc(s){var d=document.createElement('div');d.appendChild(document.createTextNode(s));return d.innerHTML;}
    var ua = navigator.userAgent || navigator.platform || '';
    var isWin = /Win/.test(ua), isMac = /Mac/.test(ua);
    var winSVG = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801"/></svg>';
    var macSVG = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/></svg>';
    function mkBtn(url, svg, i18nKey, label, primary) {
        return '<a class="sf-dl-btn' + (primary ? ' sf-dl-btn-primary' : '') + '" href="' + esc(url) + '" target="_blank" rel="noopener">' + svg + ' <span data-i18n="' + i18nKey + '">' + label + '</span></a>';
    }
    var html = '';
    if (isWin) {
        if (_dlURLWindows) html = mkBtn(_dlURLWindows, winSVG, 'download_vantagics_windows', '下载 Windows 版', true);
    } else if (isMac) {
        if (_dlURLMacOS) html = mkBtn(_dlURLMacOS, macSVG, 'download_vantagics_macos', '下载 macOS 版', true);
    } else {
        if (_dlURLWindows) html += mkBtn(_dlURLWindows, winSVG, 'download_vantagics_windows', '下载 Windows 版', false);
        if (_dlURLMacOS) html += mkBtn(_dlURLMacOS, macSVG, 'download_vantagics_macos', '下载 macOS 版', false);
    }
    c.innerHTML = html;
})();

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

function changeCat(val) {
    var params = new URLSearchParams(window.location.search);
    params.set('cat', val);
    window.location.search = params.toString();
}

function claimPack(shareToken) {
    if (!confirm(window._i18n('add_to_purchased_confirm', '是否将此分析包添加到您的已购分析技能包中？'))) return;
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
                errEl.innerHTML = window._i18n('insufficient_balance', '余额不足，当前余额') + ' ' + (d.balance || 0) + ' Credits。<a href="/user/dashboard" style="color:var(--primary-hover);text-decoration:underline;font-weight:600;">' + window._i18n('go_topup', '前往充值') + '</a>';
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

var _cpCurrentProductID = 0;
function showCustomProductPurchaseDialog(productID, productName, priceUSD) {
    _cpCurrentProductID = productID;
    var nameEl = document.getElementById('cpProductName');
    var priceEl = document.getElementById('cpProductPrice');
    if (nameEl) nameEl.textContent = productName;
    if (priceEl) priceEl.textContent = '$' + priceUSD.toFixed(2) + ' USD';
    document.getElementById('customProductPurchaseModal').classList.add('show');
}
function closeCustomProductPurchaseDialog() {
    document.getElementById('customProductPurchaseModal').classList.remove('show');
}
function confirmCustomProductPurchase() {
    if (!_cpCurrentProductID) return;
    var btn = document.getElementById('cpConfirmBtn');
    if (btn) { btn.disabled = true; btn.textContent = window._i18n('processing', '处理中...'); }
    fetch('/custom-product/' + _cpCurrentProductID + '/purchase', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    }).then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.approve_url) {
            window.location.href = d.approve_url;
        } else {
            closeCustomProductPurchaseDialog();
            showMsg('error', d.error || window._i18n('purchase_failed', '购买失败'));
            if (btn) { btn.disabled = false; btn.textContent = window._i18n('confirm_purchase', '确认购买'); }
        }
    }).catch(function() {
        closeCustomProductPurchaseDialog();
        showMsg('error', window._i18n('network_error', '网络错误'));
        if (btn) { btn.disabled = false; btn.textContent = window._i18n('confirm_purchase', '确认购买'); }
    });
}
</script>
` + I18nJS + `
</body>
</html>`
