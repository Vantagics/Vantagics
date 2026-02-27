package templates

import (
	"html/template"
	"strings"
)

// homepageFuncMap provides helper functions for the homepage template.
var homepageFuncMap = template.FuncMap{
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
	"logoURL": func() string { return LogoURL },
}

// HomepageTmpl is the parsed template for the marketplace homepage.
var HomepageTmpl = template.Must(
	template.New("homepage").Funcs(homepageFuncMap).Parse(homepageHTML),
)

const homepageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="default-lang" content="{{.DefaultLang}}">
    <title data-i18n="homepage.title">分析技能包市场</title>
    <style>
        *,*::before,*::after { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Microsoft YaHei", sans-serif;
            background: #f8f9fc;
            min-height: 100vh;
            color: #1e293b;
            line-height: 1.6;
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }
        .page { max-width: 1100px; margin: 0 auto; padding: 0 24px 48px; }

        /* ── Hero (nav integrated) ── */
        .hero {
            position: relative; overflow: hidden;
            background: linear-gradient(135deg, #eef2ff 0%, #e0e7ff 50%, #c7d2fe 100%);
            border: 1px solid #e2e8f0;
            border-radius: 0 0 20px 20px; padding: 0 36px;
            margin-bottom: 32px;
        }
        .hero::before {
            content: ''; position: absolute; top: -60px; right: -60px;
            width: 200px; height: 200px; border-radius: 50%;
            background: radial-gradient(circle, rgba(99,102,241,0.1) 0%, transparent 70%);
            pointer-events: none;
        }

        /* ── Nav (inside hero) ── */
        .nav {
            display: flex; align-items: center; justify-content: space-between;
            padding: 16px 0 14px;
        }
        .hero-banner {
            display: flex; align-items: center; justify-content: center; gap: 16px;
            padding: 0 0 20px; flex-wrap: wrap;
        }
        .hero-tagline {
            font-size: 13px; font-weight: 500; color: #94a3b8; letter-spacing: 0.2px;
        }
        .hero-subtitle {
            font-size: 13px; color: #64748b; font-weight: 500;
        }
        .nav-center {
            display: flex; align-items: center; gap: 12px; flex: 1;
            justify-content: center;
        }
        .logo-link {
            display: flex; align-items: center; gap: 10px; text-decoration: none;
        }
        .logo-mark {
            width: 36px; height: 36px; border-radius: 10px;
            display: flex; align-items: center; justify-content: center;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .logo-mark img { width: 100%; height: 100%; object-fit: cover; }
        .logo-text { font-size: 15px; font-weight: 700; color: #1e293b; letter-spacing: -0.3px; }
        .nav-actions { display: flex; align-items: center; gap: 8px; }
        .nav-link {
            padding: 8px 18px; font-size: 13px; font-weight: 600; color: #4f46e5;
            background: rgba(255,255,255,0.8); border: 1px solid rgba(226,232,240,0.6); border-radius: 10px;
            text-decoration: none; transition: all .2s; backdrop-filter: blur(4px);
        }
        .nav-link:hover { background: #fff; border-color: #c7d2fe; box-shadow: 0 2px 8px rgba(99,102,241,0.1); }

        .hero-sep { width: 1px; height: 20px; background: #cbd5e1; flex-shrink: 0; }
        .hero-buttons { display: flex; gap: 10px; flex-wrap: wrap; }
        .dl-btn {
            display: inline-flex; align-items: center; gap: 8px;
            padding: 8px 20px; border-radius: 10px;
            font-size: 12px; font-weight: 600; text-decoration: none;
            transition: all .25s cubic-bezier(.4,0,.2,1);
        }
        .dl-btn-win {
            background: linear-gradient(135deg, #6366f1, #4f46e5); color: #fff;
            box-shadow: 0 2px 8px rgba(99,102,241,0.3);
        }
        .dl-btn-win:hover { box-shadow: 0 4px 16px rgba(99,102,241,0.4); transform: translateY(-1px); }
        .dl-btn-mac {
            background: #fff; color: #4f46e5; border: 1px solid #e2e8f0;
        }
        .dl-btn-mac:hover { background: #eef2ff; border-color: #c7d2fe; transform: translateY(-1px); }
        .dl-btn svg { width: 18px; height: 18px; flex-shrink: 0; }

        /* ── Section ── */
        .section { margin-bottom: 32px; }
        .section-title {
            font-size: 18px; font-weight: 700; color: #0f172a;
            margin-bottom: 16px; letter-spacing: -0.3px;
            display: flex; align-items: center; gap: 8px;
        }
        .section-title svg { width: 20px; height: 20px; color: #6366f1; }

        /* ── Card Grid ── */
        .card-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 16px;
        }

        /* ── Store Card ── */
        .store-card {
            background: #fff; border-radius: 14px; padding: 20px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04);
            text-decoration: none; color: inherit;
            display: flex; flex-direction: column; align-items: center;
            text-align: center; gap: 10px;
            transition: all 0.25s cubic-bezier(.4,0,.2,1);
        }
        .store-card:hover {
            transform: translateY(-3px);
            box-shadow: 0 8px 32px rgba(99,102,241,0.1), 0 2px 8px rgba(0,0,0,0.04);
            border-color: #c7d2fe;
        }
        .store-card-avatar {
            width: 56px; height: 56px; border-radius: 16px;
            overflow: hidden; flex-shrink: 0;
            box-shadow: 0 2px 8px rgba(0,0,0,0.08);
        }
        .store-card-avatar img { width: 100%; height: 100%; object-fit: cover; }
        .store-card-avatar-letter {
            width: 100%; height: 100%;
            background: linear-gradient(135deg, #6366f1, #4f46e5);
            display: flex; align-items: center; justify-content: center;
            font-size: 24px; font-weight: 800; color: #fff;
        }
        .store-card-name {
            font-size: 14px; font-weight: 700; color: #0f172a;
            overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
            max-width: 100%;
        }
        .store-card-desc {
            font-size: 12px; color: #64748b; line-height: 1.5;
            overflow: hidden; text-overflow: ellipsis;
            display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
        }

        /* ── Product Card ── */
        .product-card {
            background: #fff; border-radius: 14px; padding: 20px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04);
            text-decoration: none; color: inherit;
            display: flex; flex-direction: column; gap: 8px;
            transition: all 0.25s cubic-bezier(.4,0,.2,1);
        }
        .product-card:hover {
            transform: translateY(-3px);
            box-shadow: 0 8px 32px rgba(99,102,241,0.1), 0 2px 8px rgba(0,0,0,0.04);
            border-color: #c7d2fe;
        }
        .product-card-top {
            display: flex; align-items: center; gap: 10px;
        }
        .product-card-icon {
            width: 32px; height: 32px; border-radius: 8px; flex-shrink: 0;
            background: linear-gradient(135deg, #6366f1, #4f46e5);
            display: flex; align-items: center; justify-content: center;
            box-shadow: 0 2px 6px rgba(99,102,241,0.15);
        }
        .product-card-icon svg { width: 16px; height: 16px; color: #fff; }
        .product-card-title {
            display: flex; align-items: center; gap: 6px; flex: 1; min-width: 0;
        }
        .product-card-name {
            font-size: 14px; font-weight: 700; color: #0f172a;
            overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
        }
        .product-tag {
            flex-shrink: 0; padding: 2px 6px; border-radius: 4px;
            font-size: 10px; font-weight: 600;
        }
        .product-tag.tag-free { background: #dcfce7; color: #16a34a; }
        .product-tag.tag-per-use { background: #e0e7ff; color: #4f46e5; }
        .product-tag.tag-subscription { background: #fef3c7; color: #d97706; }
        .product-card-author {
            font-size: 12px; color: #64748b; font-weight: 500;
        }
        .product-card-desc {
            font-size: 12px; color: #64748b; line-height: 1.4;
            overflow: hidden; text-overflow: ellipsis;
            display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
            margin-top: 4px;
        }
        .product-card-footer {
            display: flex; align-items: center; justify-content: space-between;
            padding-top: 8px; border-top: 1px solid #f1f5f9; margin-top: auto;
        }
        .product-card-price {
            font-size: 13px; font-weight: 800; color: #4f46e5;
        }
        .product-card-price.price-free { color: #16a34a; }
        .product-card-downloads {
            display: flex; align-items: center; gap: 4px;
            font-size: 11px; color: #94a3b8; font-weight: 500;
        }
        .product-card-downloads svg { width: 14px; height: 14px; opacity: 0.6; }

        /* ── Category Card ── */
        .category-card {
            background: #fff; border-radius: 14px; padding: 20px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04);
            text-decoration: none; color: inherit;
            display: flex; align-items: center; gap: 14px;
            transition: all 0.25s cubic-bezier(.4,0,.2,1);
        }
        .category-card:hover {
            transform: translateY(-3px);
            box-shadow: 0 8px 32px rgba(99,102,241,0.1), 0 2px 8px rgba(0,0,0,0.04);
            border-color: #c7d2fe;
        }
        .category-card-icon {
            width: 40px; height: 40px; border-radius: 10px; flex-shrink: 0;
            background: linear-gradient(135deg, #e0e7ff, #c7d2fe);
            display: flex; align-items: center; justify-content: center;
        }
        .category-card-icon svg { width: 20px; height: 20px; color: #4f46e5; }
        .category-card-info { flex: 1; min-width: 0; }
        .category-card-name {
            font-size: 14px; font-weight: 700; color: #0f172a;
            overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
        }
        .category-card-count {
            font-size: 12px; color: #94a3b8; font-weight: 500; margin-top: 2px;
        }

        /* ── Footer ── */
        .footer {
            text-align: center; margin-top: 40px; padding: 24px 0;
            border-top: 1px solid #e2e8f0;
        }
        .footer-text { font-size: 12px; color: #94a3b8; font-weight: 500; }

        /* ── Responsive (7.8) ── */
        @media (max-width: 1023px) {
            .card-grid { grid-template-columns: repeat(2, 1fr); }
        }
        @media (max-width: 767px) {
            .page { padding: 0 16px 36px; }
            .hero { padding: 0 20px; border-radius: 0 0 16px 16px; }
            .nav { flex-wrap: wrap; gap: 10px; justify-content: center; }
            .nav-center { flex-wrap: wrap; justify-content: center; }
            .hero-sep { display: none; }
            .hero-buttons { justify-content: center; }
            .hero-banner { padding: 0 0 16px; }
            .hero-tagline { font-size: 18px; }
            .card-grid { grid-template-columns: 1fr; }
        }
    </style>
</head>
<body>
<div class="page">

    <!-- Hero with integrated Nav -->
    <div class="hero">
        <nav class="nav">
            <a class="logo-link" href="/">
                <span class="logo-mark">
                    <img src="{{logoURL}}" alt="" style="width:100%;height:100%;object-fit:cover;border-radius:inherit;">
                </span>
                <span class="logo-text" data-i18n="site_name">万策分析技能包市场</span>
            </a>
            <div class="nav-center">
                {{if or .DownloadURLWindows .DownloadURLMacOS}}
                <div class="hero-buttons">
                    {{if .DownloadURLWindows}}
                    <a class="dl-btn dl-btn-win" href="{{.DownloadURLWindows}}">
                        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801"/></svg>
                        <span data-i18n="homepage.download_windows">Windows 下载</span>
                    </a>
                    {{end}}
                    {{if .DownloadURLMacOS}}
                    <a class="dl-btn dl-btn-mac" href="{{.DownloadURLMacOS}}">
                        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/></svg>
                        <span data-i18n="homepage.download_macos">macOS 下载</span>
                    </a>
                    {{end}}
                </div>
                {{end}}
            </div>
            <div class="nav-actions">
                {{if .UserID}}
                <a class="nav-link" href="/user/" data-i18n="homepage.user_center">用户中心</a>
                <a class="nav-link" href="/user/storefront" data-i18n="homepage.store_manage">店铺管理</a>
                {{else}}
                <a class="nav-link" href="/user/login" data-i18n="login">登录</a>
                <a class="nav-link" href="/user/register" data-i18n="register">注册</a>
                {{end}}
            </div>
        </nav>
        <div class="hero-banner">
            <span class="hero-tagline" data-i18n="homepage.hero_desc">站在专家肩上，洞察业务秘密</span>
        </div>
    </div>

    <!-- Featured Stores Section (7.3) -->
    {{if .FeaturedStores}}
    <div class="section">
        <h2 class="section-title">
            <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg>
            <span data-i18n="homepage.featured_stores">明星店铺</span>
        </h2>
        <div class="card-grid">
            {{range .FeaturedStores}}
            <a class="store-card" href="/store/{{.PublicID}}">
                <div class="store-card-avatar">
                    {{if .HasLogo}}
                    <img src="/store/{{.PublicID}}/logo" alt="{{.StoreName}}">
                    {{else}}
                    <div class="store-card-avatar-letter">{{firstChar .StoreName}}</div>
                    {{end}}
                </div>
                <div class="store-card-name" title="{{.StoreName}}">{{.StoreName}}</div>
                <div class="store-card-desc">{{truncateDesc .Description 80}}</div>
            </a>
            {{end}}
        </div>
    </div>
    {{end}}

    <!-- Top Sales Stores Section (7.4) -->
    {{if .TopSalesStores}}
    <div class="section">
        <h2 class="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
            <span data-i18n="homepage.top_sales_stores">热销店铺</span>
        </h2>
        <div class="card-grid">
            {{range .TopSalesStores}}
            <a class="store-card" href="/store/{{.PublicID}}">
                <div class="store-card-avatar">
                    {{if .HasLogo}}
                    <img src="/store/{{.PublicID}}/logo" alt="{{.StoreName}}">
                    {{else}}
                    <div class="store-card-avatar-letter">{{firstChar .StoreName}}</div>
                    {{end}}
                </div>
                <div class="store-card-name" title="{{.StoreName}}">{{.StoreName}}</div>
                <div class="store-card-desc">{{truncateDesc .Description 80}}</div>
            </a>
            {{end}}
        </div>
    </div>
    {{end}}

    <!-- Top Downloads Stores Section (7.5) -->
    {{if .TopDownloadsStores}}
    <div class="section">
        <h2 class="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            <span data-i18n="homepage.top_downloads_stores">热门下载店铺</span>
        </h2>
        <div class="card-grid">
            {{range .TopDownloadsStores}}
            <a class="store-card" href="/store/{{.PublicID}}">
                <div class="store-card-avatar">
                    {{if .HasLogo}}
                    <img src="/store/{{.PublicID}}/logo" alt="{{.StoreName}}">
                    {{else}}
                    <div class="store-card-avatar-letter">{{firstChar .StoreName}}</div>
                    {{end}}
                </div>
                <div class="store-card-name" title="{{.StoreName}}">{{.StoreName}}</div>
                <div class="store-card-desc">{{truncateDesc .Description 80}}</div>
            </a>
            {{end}}
        </div>
    </div>
    {{end}}

    <!-- Top Sales Products Section (7.6) -->
    {{if .TopSalesProducts}}
    <div class="section">
        <h2 class="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
            <span data-i18n="homepage.top_sales_products">热销产品</span>
        </h2>
        <div class="card-grid">
            {{range .TopSalesProducts}}
            <a class="product-card" href="/pack/{{.ShareToken}}">
                <div class="product-card-top">
                    <div class="product-card-icon">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                    </div>
                    <div class="product-card-title">
                        <span class="product-card-name" title="{{.PackName}}">{{.PackName}}</span>
                        {{if eq .ShareMode "free"}}<span class="product-tag tag-free" data-i18n="free">免费</span>
                        {{else if eq .ShareMode "per_use"}}<span class="product-tag tag-per-use" data-i18n="per_use">按次</span>
                        {{else if eq .ShareMode "subscription"}}<span class="product-tag tag-subscription" data-i18n="subscription">订阅</span>
                        {{end}}
                    </div>
                </div>
                <div class="product-card-author">{{.AuthorName}}</div>
                {{if .PackDesc}}<div class="product-card-desc">{{.PackDesc}}</div>{{end}}
                <div class="product-card-footer">
                    {{if eq .ShareMode "free"}}
                    <span class="product-card-price price-free" data-i18n="free">免费</span>
                    {{else if eq .ShareMode "per_use"}}
                    <span class="product-card-price">{{.CreditsPrice}} Credits/<span data-i18n="homepage.per_use_unit">次</span></span>
                    {{else if eq .ShareMode "subscription"}}
                    <span class="product-card-price">{{.CreditsPrice}} Credits/<span data-i18n="homepage.monthly_unit">月</span></span>
                    {{end}}
                    <span class="product-card-downloads">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                        {{.DownloadCount}}
                    </span>
                </div>
            </a>
            {{end}}
        </div>
    </div>
    {{end}}

    <!-- Top Downloads Products Section -->
    {{if .TopDownloadsProducts}}
    <div class="section">
        <h2 class="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            <span data-i18n="homepage.top_downloads_products">热门下载产品</span>
        </h2>
        <div class="card-grid">
            {{range .TopDownloadsProducts}}
            <a class="product-card" href="/pack/{{.ShareToken}}">
                <div class="product-card-top">
                    <div class="product-card-icon">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                    </div>
                    <div class="product-card-title">
                        <span class="product-card-name" title="{{.PackName}}">{{.PackName}}</span>
                        {{if eq .ShareMode "free"}}<span class="product-tag tag-free" data-i18n="free">免费</span>
                        {{else if eq .ShareMode "per_use"}}<span class="product-tag tag-per-use" data-i18n="per_use">按次</span>
                        {{else if eq .ShareMode "subscription"}}<span class="product-tag tag-subscription" data-i18n="subscription">订阅</span>
                        {{end}}
                    </div>
                </div>
                <div class="product-card-author">{{.AuthorName}}</div>
                {{if .PackDesc}}<div class="product-card-desc">{{.PackDesc}}</div>{{end}}
                <div class="product-card-footer">
                    {{if eq .ShareMode "free"}}
                    <span class="product-card-price price-free" data-i18n="free">免费</span>
                    {{else if eq .ShareMode "per_use"}}
                    <span class="product-card-price">{{.CreditsPrice}} Credits/<span data-i18n="homepage.per_use_unit">次</span></span>
                    {{else if eq .ShareMode "subscription"}}
                    <span class="product-card-price">{{.CreditsPrice}} Credits/<span data-i18n="homepage.monthly_unit">月</span></span>
                    {{end}}
                    <span class="product-card-downloads">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                        {{.DownloadCount}}
                    </span>
                </div>
            </a>
            {{end}}
        </div>
    </div>
    {{end}}

    <!-- Categories Section -->
    {{if .Categories}}
    <div class="section">
        <h2 class="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
            <span data-i18n="homepage.categories">分类浏览</span>
        </h2>
        <div class="card-grid">
            {{range .Categories}}
            <a class="category-card" href="javascript:void(0)" onclick="loadCategoryPacks({{.ID}}, this)" data-cat-id="{{.ID}}">
                <div class="category-card-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                </div>
                <div class="category-card-info">
                    <div class="category-card-name" title="{{.Name}}">{{.Name}}</div>
                    <div class="category-card-count">{{.PackCount}} <span data-i18n="homepage.packs_unit">个分析包</span></div>
                </div>
            </a>
            {{end}}
        </div>
    </div>
    <div id="category-packs-section" class="section" style="display:none;">
        <h2 class="section-title" id="category-packs-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
            <span id="category-packs-name"></span>
            <a href="javascript:void(0)" onclick="closeCategoryPacks()" style="margin-left:auto;font-size:13px;font-weight:600;color:#94a3b8;text-decoration:none;">✕</a>
        </h2>
        <div class="card-grid" id="category-packs-grid"></div>
    </div>
    {{end}}

    <!-- Newest Products Section -->
    {{if .NewestProducts}}
    <div class="section">
        <h2 class="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            <span data-i18n="homepage.newest_products">最新上架</span>
        </h2>
        <div class="card-grid">
            {{range .NewestProducts}}
            <a class="product-card" href="/pack/{{.ShareToken}}">
                <div class="product-card-top">
                    <div class="product-card-icon">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                    </div>
                    <div class="product-card-title">
                        <span class="product-card-name" title="{{.PackName}}">{{.PackName}}</span>
                        {{if eq .ShareMode "free"}}<span class="product-tag tag-free" data-i18n="free">免费</span>
                        {{else if eq .ShareMode "per_use"}}<span class="product-tag tag-per-use" data-i18n="per_use">按次</span>
                        {{else if eq .ShareMode "subscription"}}<span class="product-tag tag-subscription" data-i18n="subscription">订阅</span>
                        {{end}}
                    </div>
                </div>
                <div class="product-card-author">{{.AuthorName}}</div>
                {{if .PackDesc}}<div class="product-card-desc">{{.PackDesc}}</div>{{end}}
                <div class="product-card-footer">
                    {{if eq .ShareMode "free"}}
                    <span class="product-card-price price-free" data-i18n="free">免费</span>
                    {{else if eq .ShareMode "per_use"}}
                    <span class="product-card-price">{{.CreditsPrice}} Credits/<span data-i18n="homepage.per_use_unit">次</span></span>
                    {{else if eq .ShareMode "subscription"}}
                    <span class="product-card-price">{{.CreditsPrice}} Credits/<span data-i18n="homepage.monthly_unit">月</span></span>
                    {{end}}
                    <span class="product-card-downloads">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                        {{.DownloadCount}}
                    </span>
                </div>
            </a>
            {{end}}
        </div>
    </div>
    {{end}}

    <!-- Floating Customer Support Icon (anonymous) -->
    {{if .ServicePortalURL}}
    <style>
    .hp-support-float {
        position: fixed; bottom: 32px; right: 32px; z-index: 999;
        width: 56px; height: 56px; border-radius: 50%;
        background: linear-gradient(135deg, #6366f1, #4f46e5);
        box-shadow: 0 4px 16px rgba(99,102,241,0.4), 0 2px 6px rgba(0,0,0,0.1);
        display: flex; align-items: center; justify-content: center;
        cursor: pointer; transition: all 0.3s ease; text-decoration: none;
    }
    .hp-support-float:hover {
        transform: translateY(-3px) scale(1.05);
        box-shadow: 0 8px 24px rgba(99,102,241,0.5), 0 4px 12px rgba(0,0,0,0.15);
    }
    .hp-support-float svg { width: 28px; height: 28px; color: #fff; }
    .hp-support-float-label {
        position: absolute; right: 64px; top: 50%; transform: translateY(-50%);
        background: #1e293b; color: #fff; padding: 6px 14px; border-radius: 8px;
        font-size: 13px; font-weight: 600; white-space: nowrap;
        opacity: 0; pointer-events: none; transition: opacity 0.2s;
        box-shadow: 0 2px 8px rgba(0,0,0,0.2);
    }
    .hp-support-float-label::after {
        content: ''; position: absolute; right: -6px; top: 50%; transform: translateY(-50%);
        border: 6px solid transparent; border-left-color: #1e293b; border-right: none;
    }
    .hp-support-float:hover .hp-support-float-label { opacity: 1; }
    .hp-support-overlay {
        display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%;
        background: rgba(0,0,0,0.5); backdrop-filter: blur(4px);
        z-index: 10000; align-items: center; justify-content: center;
    }
    .hp-support-overlay.show { display: flex; }
    .hp-support-dialog {
        position: relative; width: 90%; max-width: 800px; height: 80vh;
        background: #fff; border-radius: 16px; overflow: hidden;
        box-shadow: 0 24px 64px rgba(0,0,0,0.2), 0 8px 24px rgba(0,0,0,0.1);
        display: flex; flex-direction: column;
        animation: hpSupportIn 0.25s ease-out;
    }
    @keyframes hpSupportIn {
        from { opacity: 0; transform: scale(0.95) translateY(10px); }
        to { opacity: 1; transform: scale(1) translateY(0); }
    }
    .hp-support-header {
        display: flex; align-items: center; justify-content: space-between;
        padding: 14px 20px; background: linear-gradient(135deg, #6366f1, #4f46e5);
        color: #fff; flex-shrink: 0;
    }
    .hp-support-title {
        font-size: 15px; font-weight: 700; display: flex; align-items: center; gap: 8px;
    }
    .hp-support-title svg { width: 20px; height: 20px; }
    .hp-support-actions { display: flex; align-items: center; gap: 6px; }
    .hp-support-btn {
        width: 32px; height: 32px; border-radius: 8px; border: none;
        background: rgba(255,255,255,0.2); color: #fff; cursor: pointer;
        display: flex; align-items: center; justify-content: center;
        transition: background 0.2s; font-size: 16px;
    }
    .hp-support-btn:hover { background: rgba(255,255,255,0.35); }
    .hp-support-body { flex: 1; position: relative; background: #f8f9fc; }
    .hp-support-body iframe { width: 100%; height: 100%; border: none; display: block; }
    .hp-support-loading {
        position: absolute; top: 0; left: 0; width: 100%; height: 100%;
        display: flex; align-items: center; justify-content: center;
        background: #f8f9fc; color: #64748b; font-size: 14px; font-weight: 500;
    }
    .hp-support-loading.hidden { display: none; }
    @media (max-width: 640px) {
        .hp-support-float { bottom: 20px; right: 20px; width: 48px; height: 48px; }
        .hp-support-float svg { width: 24px; height: 24px; }
        .hp-support-float-label { display: none; }
        .hp-support-dialog { width: 96%; height: 88vh; border-radius: 12px; }
        .hp-support-header { padding: 12px 16px; }
    }
    </style>
    <div class="hp-support-float" onclick="openHpSupport()" title="客户支持" data-i18n-title="customer_support">
        <span class="hp-support-float-label" data-i18n="customer_support">客户支持</span>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.8" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 12a8.25 8.25 0 1116.5 0v2.25a2.25 2.25 0 01-2.25 2.25h-.75a1.5 1.5 0 01-1.5-1.5v-3a1.5 1.5 0 011.5-1.5h.75c.17 0 .336.019.497.055A6.75 6.75 0 0012 5.25a6.75 6.75 0 00-5.997 5.305c.16-.036.327-.055.497-.055h.75a1.5 1.5 0 011.5 1.5v3a1.5 1.5 0 01-1.5 1.5H6.5a2.25 2.25 0 01-2.25-2.25V12z" />
        </svg>
    </div>
    <div class="hp-support-overlay" id="hpSupportOverlay">
        <div class="hp-support-dialog">
            <div class="hp-support-header">
                <div class="hp-support-title">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.8" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 12a8.25 8.25 0 1116.5 0v2.25a2.25 2.25 0 01-2.25 2.25h-.75a1.5 1.5 0 01-1.5-1.5v-3a1.5 1.5 0 011.5-1.5h.75c.17 0 .336.019.497.055A6.75 6.75 0 0012 5.25a6.75 6.75 0 00-5.997 5.305c.16-.036.327-.055.497-.055h.75a1.5 1.5 0 011.5 1.5v3a1.5 1.5 0 01-1.5 1.5H6.5a2.25 2.25 0 01-2.25-2.25V12z"/></svg>
                    <span data-i18n="customer_support">客户支持</span>
                </div>
                <div class="hp-support-actions">
                    <button class="hp-support-btn" onclick="openHpSupportExternal()" title="在新窗口打开">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                    </button>
                    <button class="hp-support-btn" onclick="closeHpSupport()" title="关闭">&times;</button>
                </div>
            </div>
            <div class="hp-support-body">
                <div class="hp-support-loading" id="hpSupportLoading" data-i18n="loading_support">正在连接客服系统...</div>
                <iframe id="hpSupportIframe" sandbox="allow-same-origin allow-scripts allow-forms allow-popups" allow="clipboard-write"></iframe>
            </div>
        </div>
    </div>
    <script>
    var _hpSupportURL = '{{.ServicePortalURL}}/#anonymous';
    function openHpSupport() {
        var overlay = document.getElementById('hpSupportOverlay');
        var iframe = document.getElementById('hpSupportIframe');
        var loading = document.getElementById('hpSupportLoading');
        overlay.classList.add('show');
        document.body.style.overflow = 'hidden';
        loading.classList.remove('hidden');
        iframe.onload = function(){ loading.classList.add('hidden'); };
        iframe.src = _hpSupportURL;
    }
    function closeHpSupport() {
        var overlay = document.getElementById('hpSupportOverlay');
        var iframe = document.getElementById('hpSupportIframe');
        overlay.classList.remove('show');
        document.body.style.overflow = '';
        iframe.src = '';
    }
    function openHpSupportExternal() {
        window.open(_hpSupportURL, '_blank');
    }
    document.getElementById('hpSupportOverlay').addEventListener('click', function(e) {
        if (e.target === this) closeHpSupport();
    });
    </script>
    {{end}}

    <!-- Footer (7.7) -->
    <footer class="footer">
        <p class="footer-text">&copy; 2026 <a href="https://vantagics.com" target="_blank" rel="noopener" style="color:#6366f1;text-decoration:none;font-weight:600;">Vantagics</a> <span data-i18n="site_name">万策分析技能包市场</span></p>
    </footer>

</div>
<script>
function loadCategoryPacks(catId, el) {
    var section = document.getElementById('category-packs-section');
    var grid = document.getElementById('category-packs-grid');
    var nameEl = document.getElementById('category-packs-name');
    var catName = el.querySelector('.category-card-name').textContent;
    nameEl.textContent = catName;
    grid.innerHTML = '<div style="grid-column:1/-1;text-align:center;color:#94a3b8;padding:20px;" data-i18n="loading">加载中...</div>';
    section.style.display = '';
    section.scrollIntoView({behavior:'smooth', block:'start'});
    // highlight active
    document.querySelectorAll('.category-card').forEach(function(c){c.style.borderColor='';});
    el.style.borderColor = '#6366f1';
    fetch('/api/packs?category_id=' + catId)
        .then(function(r){return r.json();})
        .then(function(data){
            var packs = data.packs || [];
            if (!packs.length) {
                grid.innerHTML = '<div style="grid-column:1/-1;text-align:center;color:#94a3b8;padding:20px;" data-i18n="no_results">没有找到匹配的分析包</div>';
                if(typeof applyI18n==='function') applyI18n();
                return;
            }
            var html = '';
            for (var i = 0; i < packs.length; i++) {
                var p = packs[i];
                var token = p.share_token || '';
                var tag = '', tagClass = '';
                if (p.share_mode === 'free') { tag = '免费'; tagClass = 'tag-free'; }
                else if (p.share_mode === 'per_use') { tag = '按次'; tagClass = 'tag-per-use'; }
                else if (p.share_mode === 'subscription') { tag = '订阅'; tagClass = 'tag-subscription'; }
                var priceHtml = '';
                if (p.share_mode === 'free') priceHtml = '<span class="product-card-price price-free" data-i18n="free">免费</span>';
                else if (p.share_mode === 'per_use') priceHtml = '<span class="product-card-price">' + p.credits_price + ' Credits/<span data-i18n="homepage.per_use_unit">次</span></span>';
                else if (p.share_mode === 'subscription') priceHtml = '<span class="product-card-price">' + p.credits_price + ' Credits/<span data-i18n="homepage.monthly_unit">月</span></span>';
                var desc = p.pack_description || '';
                html += '<a class="product-card" href="/pack/' + token + '">'
                    + '<div class="product-card-top">'
                    + '<div class="product-card-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg></div>'
                    + '<div class="product-card-title"><span class="product-card-name" title="' + p.pack_name + '">' + p.pack_name + '</span>'
                    + (tag ? '<span class="product-tag ' + tagClass + '">' + tag + '</span>' : '')
                    + '</div></div>'
                    + '<div class="product-card-author">' + (p.author_name || '') + '</div>'
                    + (desc ? '<div class="product-card-desc">' + desc + '</div>' : '')
                    + '<div class="product-card-footer">' + priceHtml
                    + '<span class="product-card-downloads"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>'
                    + (p.download_count || 0) + '</span></div></a>';
            }
            grid.innerHTML = html;
            if(typeof applyI18n==='function') applyI18n();
        })
        .catch(function(){
            grid.innerHTML = '<div style="grid-column:1/-1;text-align:center;color:#ef4444;padding:20px;" data-i18n="load_failed">加载失败，请重试</div>';
            if(typeof applyI18n==='function') applyI18n();
        });
}
function closeCategoryPacks() {
    document.getElementById('category-packs-section').style.display = 'none';
    document.querySelectorAll('.category-card').forEach(function(c){c.style.borderColor='';});
}
</script>
` + I18nJS + `
</body>
</html>`

