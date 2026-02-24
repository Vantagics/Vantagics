package templates

import "html/template"

var StorefrontNoveltyTmpl = template.Must(
template.New("storefront_novelty").Funcs(storefrontFuncMap).Parse(novP1 + novP2 + novP3 + novP4 + "\n" + I18nJS + "\n</body>\n</html>"),
)

const novP1 = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="default-lang" content="{{.DefaultLang}}">
<title>{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}} - 分析技能包市场</title>
<meta property="og:type" content="website" />
<meta property="og:title" content="{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}}" />
<meta property="og:description" content="{{if .Storefront.Description}}{{truncateDesc .Storefront.Description 200}}{{else}}该作者暂未设置小铺描述{{end}}" />
{{if .Storefront.HasLogo}}<meta property="og:image" content="/store/{{.Storefront.StoreSlug}}/logo" />{{end}}
<meta name="twitter:card" content="summary" />
<meta name="twitter:title" content="{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}}" />
<meta name="twitter:description" content="{{if .Storefront.Description}}{{truncateDesc .Storefront.Description 200}}{{else}}该作者暂未设置小铺描述{{end}}" />
{{if .Storefront.HasLogo}}<meta name="twitter:image" content="/store/{{.Storefront.StoreSlug}}/logo" />{{end}}
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap" rel="stylesheet">
<style>
*,*::before,*::after{margin:0;padding:0;box-sizing:border-box;}
:root{--g100:#fdf6e3;--g200:#f5e6b8;--g300:#e8d08a;--g400:#d4b45a;--g500:#b8943a;--g600:#9a7a2e;--g700:#7c6124;--cream:#faf7f0;--tp:#3d3425;--ts:#7a6f5d;--tm:#a89f8b;--cbg:rgba(255,255,255,0.85);--cb:rgba(212,180,90,0.25);--cs:0 4px 24px rgba(184,148,58,0.08);}
body{font-family:'Inter',-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:var(--cream);min-height:100vh;color:var(--tp);line-height:1.6;-webkit-font-smoothing:antialiased;position:relative;overflow-x:hidden;}
body::before{content:'';position:fixed;top:-120px;right:-80px;width:500px;height:500px;border-radius:50%;background:radial-gradient(ellipse,rgba(212,180,90,0.12) 0%,transparent 70%);pointer-events:none;z-index:0;}
body::after{content:'';position:fixed;bottom:-100px;left:-60px;width:400px;height:400px;border-radius:50%;background:radial-gradient(ellipse,rgba(184,148,58,0.08) 0%,transparent 70%);pointer-events:none;z-index:0;}
.page{max-width:1000px;margin:0 auto;padding:20px 24px 48px;position:relative;z-index:1;}
.nav{display:flex;align-items:center;justify-content:space-between;margin-bottom:28px;padding:0 2px;}
.logo-link{display:flex;align-items:center;gap:10px;text-decoration:none;}
.logo-mark{width:36px;height:36px;border-radius:10px;display:flex;align-items:center;justify-content:center;background:linear-gradient(135deg,var(--g400),var(--g600));box-shadow:0 2px 8px rgba(184,148,58,0.3);}
.logo-mark svg{width:20px;height:20px;}.logo-text{font-size:15px;font-weight:700;color:var(--tp);letter-spacing:-0.3px;}
.nav-actions{display:flex;align-items:center;gap:8px;}
.nav-link{padding:8px 18px;font-size:13px;font-weight:600;color:var(--g600);background:var(--cbg);border:1px solid var(--cb);border-radius:10px;text-decoration:none;transition:all .2s;backdrop-filter:blur(8px);}
.nav-link:hover{background:#fff;border-color:var(--g300);box-shadow:0 2px 8px rgba(184,148,58,0.1);}
.store-hero{position:relative;overflow:hidden;background:linear-gradient(160deg,var(--g100) 0%,#fdf8ee 30%,var(--cream) 60%,var(--g100) 100%);border:1px solid var(--cb);border-radius:24px;padding:40px 36px 36px;margin-bottom:28px;}
.store-hero::before{content:'';position:absolute;top:-80px;right:-40px;width:300px;height:300px;border-radius:50%;border:2px solid rgba(212,180,90,0.15);pointer-events:none;}
.store-hero::after{content:'';position:absolute;bottom:-60px;left:20%;width:200px;height:200px;border-radius:50%;border:1.5px solid rgba(212,180,90,0.1);pointer-events:none;}
.hero-glow{position:absolute;top:20px;right:60px;width:180px;height:180px;border-radius:50%;background:radial-gradient(circle,rgba(212,180,90,0.15) 0%,transparent 70%);pointer-events:none;}
.store-hero-inner{position:relative;z-index:1;display:flex;gap:36px;align-items:stretch;}
.store-hero-inner.hero-reversed{flex-direction:row-reverse;}
.store-profile{display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;min-width:220px;flex-shrink:0;}
.store-avatar-ring{width:120px;height:120px;border-radius:50%;padding:4px;margin-bottom:16px;background:linear-gradient(135deg,var(--g300),var(--g500),var(--g300));box-shadow:0 4px 20px rgba(184,148,58,0.2);}
.store-avatar{width:100%;height:100%;border-radius:50%;overflow:hidden;background:#fff;}
.store-avatar img{width:100%;height:100%;object-fit:cover;}
.store-avatar-letter{width:100%;height:100%;background:linear-gradient(135deg,var(--g400),var(--g600));display:flex;align-items:center;justify-content:center;font-size:42px;font-weight:800;color:#fff;}
.store-name{font-size:22px;font-weight:800;color:var(--tp);margin-bottom:8px;letter-spacing:-0.4px;}
.store-desc{font-size:13px;color:var(--ts);line-height:1.7;max-width:220px;}
.store-stats{display:flex;gap:16px;margin-top:14px;}
.store-stat{display:flex;flex-direction:column;align-items:center;padding:8px 16px;background:rgba(255,255,255,0.7);border-radius:12px;border:1px solid var(--cb);}
.store-stat-val{font-size:18px;font-weight:800;color:var(--g600);}
.store-stat-label{font-size:10px;color:var(--tm);font-weight:600;text-transform:uppercase;letter-spacing:0.5px;}
`
const novP2 = `.store-featured{flex:1;min-width:0;display:flex;flex-direction:column;}
.store-featured-header{display:flex;align-items:center;justify-content:space-between;margin-bottom:12px;}
.store-featured-title{font-size:11px;font-weight:700;color:var(--g500);margin-bottom:0;display:flex;align-items:center;gap:6px;letter-spacing:0.8px;text-transform:uppercase;}
.store-featured-title svg{width:14px;height:14px;color:var(--g400);}
.featured-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:12px;flex:1;}
.featured-card{background:var(--cbg);border-radius:16px;padding:18px 16px 14px;border:1px solid var(--cb);cursor:pointer;text-decoration:none;display:flex;flex-direction:column;align-items:flex-start;color:inherit;transition:all 0.3s cubic-bezier(.4,0,.2,1);backdrop-filter:blur(8px);position:relative;overflow:hidden;box-shadow:var(--cs);}
.featured-card:nth-child(1){transform:rotate(-1.5deg);}.featured-card:nth-child(2){transform:rotate(1deg) translateY(8px);}
.featured-card:nth-child(3){transform:rotate(0.5deg) translateY(-4px);}.featured-card:nth-child(4){transform:rotate(-0.8deg) translateY(4px);}
.featured-card:hover{transform:rotate(0deg) translateY(-4px) !important;background:#fff;box-shadow:0 12px 40px rgba(184,148,58,0.15),0 2px 8px rgba(0,0,0,0.04);border-color:var(--g300);}
.featured-card-top{display:flex;align-items:center;gap:10px;width:100%;margin-bottom:8px;}
.featured-icon{width:36px;height:36px;border-radius:10px;flex-shrink:0;background:linear-gradient(135deg,var(--g400),var(--g600));display:flex;align-items:center;justify-content:center;box-shadow:0 2px 8px rgba(184,148,58,0.25);}
.featured-icon svg{width:18px;height:18px;color:#fff;}.featured-icon-img{width:36px;height:36px;border-radius:10px;object-fit:cover;flex-shrink:0;box-shadow:0 2px 8px rgba(184,148,58,0.2);}
.featured-card-title{flex:1;min-width:0;}.featured-name{font-size:13px;font-weight:700;color:var(--tp);line-height:1.3;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:100%;}
.featured-tag{display:inline-block;padding:1px 7px;border-radius:10px;font-size:10px;font-weight:600;margin-top:2px;}
.featured-tag-free{background:#f0f5e8;color:#5a7a2e;}.featured-tag-per_use{background:var(--g100);color:var(--g700);}.featured-tag-subscription{background:#f5f0e0;color:#8a6d2e;}
.featured-desc{font-size:11px;color:var(--ts);line-height:1.5;margin-bottom:10px;flex:1;overflow:hidden;text-overflow:ellipsis;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;}
.featured-footer{display:flex;align-items:center;justify-content:space-between;width:100%;padding-top:8px;border-top:1px solid rgba(212,180,90,0.15);}
.featured-price{font-size:12px;font-weight:800;}.featured-price.price-free{color:#5a7a2e;}.featured-price.price-paid{color:var(--g600);}
.featured-downloads{display:flex;align-items:center;gap:3px;font-size:11px;color:var(--tm);font-weight:500;}.featured-downloads svg{width:12px;height:12px;opacity:0.6;}
.featured-empty-slot{background:rgba(255,255,255,0.4);border-radius:14px;padding:16px;border:1px dashed rgba(212,180,90,0.3);display:flex;align-items:center;justify-content:center;color:var(--tm);font-size:20px;}
.sf-dl-btn{display:inline-flex;align-items:center;gap:6px;padding:7px 16px;border-radius:10px;font-size:12px;font-weight:600;text-decoration:none;transition:all .25s;border:1px solid var(--cb);background:var(--cbg);color:var(--g600);backdrop-filter:blur(8px);}
.sf-dl-btn:hover{background:rgba(255,255,255,0.85);border-color:var(--g300);box-shadow:0 4px 16px rgba(184,148,58,0.15);transform:translateY(-1px);color:var(--g600);}
.sf-dl-btn-primary{background:linear-gradient(135deg,var(--g400),var(--g600));color:#fff;border-color:transparent;box-shadow:0 2px 8px rgba(184,148,58,0.25);}
.sf-dl-btn-primary:hover{background:linear-gradient(135deg,var(--g500),var(--g700));box-shadow:0 4px 16px rgba(184,148,58,0.35);color:#fff;}.sf-dl-btn svg{width:16px;height:16px;flex-shrink:0;}
.filter-bar{display:flex;align-items:center;gap:12px;margin-bottom:20px;flex-wrap:wrap;}
.filter-group{display:flex;gap:4px;background:var(--cbg);border:1px solid var(--cb);border-radius:12px;padding:3px;box-shadow:var(--cs);}
.filter-btn{padding:7px 16px;border:none;border-radius:8px;font-size:12px;font-weight:600;cursor:pointer;background:transparent;color:var(--ts);transition:all 0.2s;text-decoration:none;display:inline-block;}
.filter-btn:hover{color:var(--tp);background:rgba(212,180,90,0.08);}
.filter-btn.active{background:linear-gradient(135deg,var(--g400),var(--g600));color:#fff;box-shadow:0 2px 8px rgba(184,148,58,0.3);}
.search-input{padding:8px 16px;border:1px solid var(--cb);border-radius:12px;font-size:13px;background:var(--cbg);min-width:200px;transition:all 0.2s;color:var(--tp);box-shadow:var(--cs);}
.search-input:focus{outline:none;border-color:var(--g400);box-shadow:0 0 0 3px rgba(212,180,90,0.12);}.search-input::placeholder{color:var(--tm);}
.sort-select{padding:8px 16px;border:1px solid var(--cb);border-radius:12px;font-size:13px;background:var(--cbg);color:var(--tp);cursor:pointer;box-shadow:var(--cs);}.sort-select:focus{outline:none;border-color:var(--g400);}
.pack-list{display:grid;grid-template-columns:repeat(2,1fr);gap:16px;}
.pack-item{background:var(--cbg);border-radius:18px;padding:22px 24px;border:1px solid var(--cb);box-shadow:var(--cs);display:flex;flex-direction:column;gap:12px;transition:all 0.3s;position:relative;backdrop-filter:blur(8px);}
.pack-item:hover{transform:translateY(-3px);box-shadow:0 12px 40px rgba(184,148,58,0.12);border-color:var(--g300);}
.pack-item-body{flex:1;min-width:0;}.pack-item-header{display:flex;align-items:center;gap:10px;margin-bottom:8px;flex-wrap:wrap;}
.pack-item-name{font-size:15px;font-weight:700;color:var(--tp);letter-spacing:-0.2px;}
.tag{display:inline-flex;align-items:center;padding:3px 10px;border-radius:20px;font-size:10px;font-weight:700;letter-spacing:0.3px;text-transform:uppercase;}
.tag-free{background:#f0f5e8;color:#5a7a2e;border:1px solid #d4e4b8;}.tag-per-use{background:var(--g100);color:var(--g700);border:1px solid var(--g200);}
.tag-subscription{background:#f5f0e0;color:#8a6d2e;border:1px solid #e8d8a8;}.tag-category{background:#f0ece0;color:#6a5d3e;border:1px solid #ddd4b8;}
.pack-item-desc{font-size:13px;color:var(--ts);line-height:1.7;margin-bottom:12px;overflow:hidden;text-overflow:ellipsis;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;}
.pack-item-footer{display:flex;align-items:center;justify-content:space-between;padding-top:12px;border-top:1px solid rgba(212,180,90,0.12);}
.pack-item-meta{display:flex;align-items:center;gap:14px;font-size:12px;color:var(--tm);}.pack-item-meta .meta-item{display:flex;align-items:center;gap:4px;}.pack-item-meta .meta-item svg{width:14px;height:14px;opacity:0.6;}
.pack-item-price{font-weight:800;color:var(--g600);font-size:14px;letter-spacing:-0.2px;}.pack-item-price.price-free{color:#5a7a2e;}
.btn{padding:9px 20px;border:none;border-radius:10px;font-size:13px;font-weight:600;cursor:pointer;display:inline-flex;align-items:center;gap:6px;text-decoration:none;transition:all 0.25s;font-family:inherit;}
.btn-green{background:linear-gradient(135deg,#7a9e3a,#5a7a2e);color:#fff;box-shadow:0 2px 8px rgba(90,122,46,0.25);}.btn-green:hover{box-shadow:0 4px 16px rgba(90,122,46,0.3);transform:translateY(-1px);}
.btn-indigo{background:linear-gradient(135deg,var(--g400),var(--g600));color:#fff;box-shadow:0 2px 8px rgba(184,148,58,0.25);}.btn-indigo:hover{box-shadow:0 4px 16px rgba(184,148,58,0.3);transform:translateY(-1px);}
.btn:disabled{opacity:0.6;cursor:not-allowed;transform:none !important;}
.badge-owned{display:inline-flex;align-items:center;gap:6px;padding:8px 16px;background:#f0f5e8;color:#5a7a2e;border:1px solid #d4e4b8;border-radius:10px;font-size:12px;font-weight:700;}.badge-owned svg{width:14px;height:14px;}
.btn-ghost{padding:9px 20px;font-size:13px;border-radius:10px;background:rgba(255,255,255,0.6);color:var(--ts);border:1px solid var(--cb);cursor:pointer;transition:all .2s;font-family:inherit;font-weight:600;}.btn-ghost:hover{background:#fff;color:var(--tp);}
.empty-state{text-align:center;padding:56px 24px;color:var(--ts);background:var(--cbg);border-radius:18px;border:1px dashed var(--cb);}.empty-state .icon{font-size:40px;margin-bottom:14px;opacity:0.5;}.empty-state p{font-size:14px;font-weight:500;}
.modal-overlay{display:none;position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(61,52,37,0.4);backdrop-filter:blur(6px);z-index:1000;align-items:center;justify-content:center;}.modal-overlay.show{display:flex;}
.modal-box{background:#fff;border-radius:20px;padding:32px;max-width:420px;width:90%;box-shadow:0 24px 64px rgba(184,148,58,0.15),0 8px 24px rgba(0,0,0,0.08);position:relative;border:1px solid var(--cb);}
.modal-close{position:absolute;top:16px;right:18px;background:none;border:none;font-size:18px;cursor:pointer;color:var(--tm);width:32px;height:32px;border-radius:8px;display:flex;align-items:center;justify-content:center;}.modal-close:hover{background:var(--g100);color:var(--tp);}
.modal-title{font-size:17px;font-weight:700;color:var(--tp);margin-bottom:22px;}.modal-actions{display:flex;gap:10px;justify-content:flex-end;margin-top:22px;}
.field-group{margin-bottom:16px;}.field-group label{font-size:12px;color:var(--ts);display:block;margin-bottom:6px;font-weight:600;}
.field-group input,.field-group select{width:100%;padding:10px 14px;border:1px solid var(--cb);border-radius:10px;font-size:14px;background:#fefdf8;transition:all 0.2s;color:var(--tp);font-family:inherit;}
.field-group input:focus,.field-group select:focus{outline:none;border-color:var(--g400);background:#fff;box-shadow:0 0 0 3px rgba(212,180,90,0.12);}
.total-price{font-size:18px;font-weight:800;color:var(--g600);margin-bottom:4px;}
.msg{display:none;padding:14px 18px;border-radius:12px;font-size:13px;margin-bottom:16px;font-weight:600;}
.msg-ok{background:#f0f5e8;color:#5a7a2e;border:1px solid #d4e4b8;}.msg-err{background:#fef2f2;color:#dc2626;border:1px solid #fecaca;}
.foot{text-align:center;margin-top:36px;padding-top:20px;border-top:1px solid rgba(212,180,90,0.15);}
.foot-text{font-size:12px;color:var(--tm);font-weight:500;}.foot-text a{color:var(--g600);text-decoration:none;font-weight:600;}.foot-text a:hover{text-decoration:underline;}
.powered-by{margin-top:10px;font-size:11px;color:var(--tm);font-weight:500;display:flex;align-items:center;justify-content:center;gap:5px;}
.powered-by a{color:var(--g600);text-decoration:none;font-weight:600;display:inline-flex;align-items:center;gap:4px;}.powered-by a:hover{text-decoration:underline;}.powered-by svg{width:14px;height:14px;flex-shrink:0;}
.toast{position:fixed;bottom:32px;left:50%;transform:translateX(-50%) translateY(20px);background:var(--tp);color:#fff;padding:12px 28px;border-radius:12px;font-size:13px;font-weight:600;opacity:0;transition:all .3s;pointer-events:none;z-index:9999;box-shadow:0 8px 24px rgba(0,0,0,0.2);}.toast.show{opacity:1;transform:translateX(-50%) translateY(0);}
@media(max-width:640px){.page{padding:16px 16px 36px;}.store-hero{padding:24px;border-radius:18px;}.store-hero-inner{flex-direction:column;gap:24px;}.store-profile{min-width:auto;}.store-stats{justify-content:center;}.filter-bar{flex-direction:column;align-items:stretch;}.search-input{min-width:auto;}.pack-list{grid-template-columns:1fr;}.featured-grid{grid-template-columns:repeat(2,1fr);}.featured-card:nth-child(1),.featured-card:nth-child(2),.featured-card:nth-child(3),.featured-card:nth-child(4){transform:none;}}
</style></head><body>
`
const novP3 = `<div class="page">
<nav class="nav"><a class="logo-link" href="/"><span class="logo-mark"><svg viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg></span><span class="logo-text" data-i18n="site_name">分析技能包市场</span></a>
<div class="nav-actions">{{if .IsLoggedIn}}<a class="nav-link" href="/user/dashboard" data-i18n="personal_center">个人中心</a>{{else}}<a class="nav-link" href="/user/login" data-i18n="login">登录</a>{{end}}</div></nav>
<div class="store-hero"><div class="hero-glow"></div><div class="store-hero-inner{{if eq .HeroLayout "reversed"}} hero-reversed{{end}}">
<div class="store-profile"><div class="store-avatar-ring"><div class="store-avatar">{{if .Storefront.HasLogo}}<img src="/store/{{.Storefront.StoreSlug}}/logo" alt="{{.Storefront.StoreName}}">{{else}}<div class="store-avatar-letter">{{firstChar .Storefront.StoreName}}</div>{{end}}</div></div>
<h1 class="store-name">{{if .Storefront.StoreName}}{{.Storefront.StoreName}}{{else}}小铺{{end}}</h1>
<p class="store-desc">{{if .Storefront.Description}}{{.Storefront.Description}}{{else}}该作者暂未设置小铺描述{{end}}</p>
<div class="store-stats"><div class="store-stat"><span class="store-stat-val">{{len .Packs}}</span><span class="store-stat-label" data-i18n="stat_packs">分析包</span></div>{{if and .FeaturedPacks .FeaturedVisible}}<div class="store-stat"><span class="store-stat-val">{{len .FeaturedPacks}}</span><span class="store-stat-label" data-i18n="stat_featured">推荐</span></div>{{end}}</div></div>
{{if and .FeaturedPacks .FeaturedVisible}}<div class="store-featured"><div class="store-featured-header"><div class="store-featured-title"><svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg><span data-i18n="featured_packs">店主推荐</span></div>{{if or .DownloadURLWindows .DownloadURLMacOS}}<span id="sfDlBtn"></span>{{end}}</div>
<div class="featured-grid">{{range .FeaturedPacks}}<a class="featured-card" href="/pack/{{.ShareToken}}" target="_blank" rel="noopener"><div class="featured-card-top">{{if .HasLogo}}<img class="featured-icon-img" src="/store/{{$.Storefront.StoreSlug}}/featured/{{.ListingID}}/logo" alt="{{.PackName}}">{{else}}<div class="featured-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg></div>{{end}}<div class="featured-card-title"><div class="featured-name" title="{{.PackName}}">{{.PackName}}</div>{{if eq .ShareMode "free"}}<span class="featured-tag featured-tag-free" data-i18n="free">免费</span>{{else if eq .ShareMode "per_use"}}<span class="featured-tag featured-tag-per_use" data-i18n="per_use">按次收费</span>{{else if eq .ShareMode "subscription"}}<span class="featured-tag featured-tag-subscription" data-i18n="subscription">订阅制</span>{{end}}</div></div>{{if .PackDesc}}<div class="featured-desc">{{.PackDesc}}</div>{{else}}<div class="featured-desc" style="color:var(--tm);" data-i18n="no_description">暂无描述</div>{{end}}<div class="featured-footer">{{if eq .ShareMode "free"}}<span class="featured-price price-free" data-i18n="free">免费</span>{{else}}<span class="featured-price price-paid">{{.CreditsPrice}} Credits</span>{{end}}<span class="featured-downloads"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>{{.DownloadCount}}</span></div></a>{{end}}</div></div>{{end}}
{{if or (not .FeaturedPacks) (not .FeaturedVisible)}}{{if or .DownloadURLWindows .DownloadURLMacOS}}<div class="store-featured" style="justify-content:flex-start;"><span id="sfDlBtn"></span></div>{{end}}{{end}}
</div></div>
<div class="msg msg-ok" id="successMsg"></div><div class="msg msg-err" id="errorMsg"></div>
<div class="filter-bar"><div class="filter-group"><a class="filter-btn{{if eq .Filter ""}} active{{end}}" href="?filter=&sort={{.Sort}}&q={{.SearchQuery}}&cat={{.CategoryFilter}}" data-i18n="filter_all">全部</a><a class="filter-btn{{if eq .Filter "free"}} active{{end}}" href="?filter=free&sort={{.Sort}}&q={{.SearchQuery}}&cat={{.CategoryFilter}}" data-i18n="free">免费</a><a class="filter-btn{{if eq .Filter "per_use"}} active{{end}}" href="?filter=per_use&sort={{.Sort}}&q={{.SearchQuery}}&cat={{.CategoryFilter}}" data-i18n="per_use">按次收费</a><a class="filter-btn{{if eq .Filter "subscription"}} active{{end}}" href="?filter=subscription&sort={{.Sort}}&q={{.SearchQuery}}&cat={{.CategoryFilter}}" data-i18n="subscription">订阅制</a></div>
{{if .Categories}}<select class="sort-select" id="catSelect" onchange="changeCat(this.value)"><option value=""{{if eq .CategoryFilter ""}} selected{{end}} data-i18n="all_categories">全部类别</option>{{range .Categories}}<option value="{{.}}"{{if eq $.CategoryFilter .}} selected{{end}}>{{.}}</option>{{end}}</select>{{end}}
<form id="searchForm" method="GET" style="display:flex;gap:8px;align-items:center;"><input type="hidden" name="filter" value="{{.Filter}}"><input type="hidden" name="sort" value="{{.Sort}}"><input type="hidden" name="cat" value="{{.CategoryFilter}}"><input class="search-input" type="text" name="q" value="{{.SearchQuery}}" placeholder="搜索分析包..." data-i18n-placeholder="search_packs"></form>
<select class="sort-select" id="sortSelect" onchange="changeSort(this.value)"><option value="revenue"{{if eq .Sort "revenue"}} selected{{end}} data-i18n="sort_revenue">按销售金额</option><option value="downloads"{{if eq .Sort "downloads"}} selected{{end}} data-i18n="sort_downloads">按下载量</option><option value="orders"{{if eq .Sort "orders"}} selected{{end}} data-i18n="sort_orders">按订单数</option></select></div>
{{if .Packs}}<div class="pack-list">{{range .Packs}}<div class="pack-item"><div class="pack-item-body"><div class="pack-item-header"><span class="pack-item-name">{{.PackName}}</span>{{if eq .ShareMode "free"}}<span class="tag tag-free" data-i18n="free">免费</span>{{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use" data-i18n="per_use">按次收费</span>{{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription" data-i18n="subscription">订阅制</span>{{end}}{{if .CategoryName}}<span class="tag tag-category">{{.CategoryName}}</span>{{end}}</div>{{if .PackDesc}}<div class="pack-item-desc">{{.PackDesc}}</div>{{end}}</div>
<div class="pack-item-footer"><div class="pack-item-meta">{{if eq .ShareMode "free"}}<span class="meta-item"><span class="pack-item-price price-free" data-i18n="free">免费</span></span>{{else}}<span class="meta-item"><span class="pack-item-price">{{.CreditsPrice}} Credits</span></span>{{end}}<span class="meta-item"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>{{.DownloadCount}}</span></div>
<div class="pack-item-actions">{{if $.IsLoggedIn}}{{if index $.PurchasedIDs .ListingID}}<span class="badge-owned"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg><span data-i18n="already_purchased">已购买</span></span>{{else if eq .ShareMode "free"}}<button class="btn btn-green" onclick="claimPack('{{.ShareToken}}')" data-i18n="claim_free">免费领取</button>{{else}}<button class="btn btn-indigo" onclick="showPurchaseDialog('{{.ShareToken}}', '{{.ShareMode}}', {{.CreditsPrice}}, '{{.PackName}}')" data-i18n="purchase">购买</button>{{end}}{{else}}{{if eq .ShareMode "free"}}<a class="btn btn-green" href="/user/login?redirect=/store/{{$.Storefront.StoreSlug}}" data-i18n="login_to_claim">登录后领取</a>{{else}}<a class="btn btn-indigo" href="/user/login?redirect=/store/{{$.Storefront.StoreSlug}}" data-i18n="login_to_buy">登录后购买</a>{{end}}{{end}}</div></div></div>{{end}}</div>
{{else}}<div class="empty-state"><div class="icon">📭</div><p data-i18n="storefront_empty">该小铺暂无分析包</p></div>{{end}}
<div class="foot"><p class="foot-text">Vantagics <span data-i18n="site_name">分析技能包市场</span> &middot; <a href="/" data-i18n="browse_more">浏览更多</a></p><div class="powered-by">Powered by <a href="https://vantagics.com" target="_blank" rel="noopener"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>Vantagics</a></div></div></div>
<div class="modal-overlay" id="purchaseModal"><div class="modal-box">
<button class="modal-close" onclick="closePurchaseDialog()">&times;</button>
<div class="modal-title" id="purchaseModalTitle" data-i18n="purchase">购买</div>
<div id="perUseFields" style="display:none;"><div class="field-group"><label for="purchaseQuantity" data-i18n="buy_count_label">购买次数</label><input type="number" id="purchaseQuantity" min="1" value="1" onchange="updatePurchaseTotal()" oninput="updatePurchaseTotal()"></div></div>
<div id="subscriptionFields" style="display:none;"><div class="field-group"><label for="purchaseDuration" data-i18n="sub_duration">订阅时长</label><select id="purchaseDuration" onchange="updatePurchaseTotal()"><optgroup label="按月"><option value="1">1 个月</option><option value="2">2 个月</option><option value="3">3 个月</option><option value="4">4 个月</option><option value="5">5 个月</option><option value="6">6 个月</option><option value="7">7 个月</option><option value="8">8 个月</option><option value="9">9 个月</option><option value="10">10 个月</option><option value="11">11 个月</option><option value="12">12 个月</option></optgroup><optgroup label="按年"><option value="12">1 年</option><option value="24">2 年</option><option value="36">3 年</option></optgroup></select></div></div>
<div class="total-price" id="purchaseTotal"></div>
<div class="modal-actions"><button class="btn-ghost" onclick="closePurchaseDialog()" data-i18n="cancel">取消</button><button class="btn btn-indigo" id="confirmPurchaseBtn" onclick="confirmPurchase()" data-i18n="confirm_purchase">确认购买</button></div>
</div></div>
<div class="toast" id="toast"></div>
`
const novP4 = `<script>
var _currentShareToken='';var _currentShareMode='';var _currentCreditsPrice=0;
var _storeSlug='{{.Storefront.StoreSlug}}';
var _dlURLWindows="{{.DownloadURLWindows}}";var _dlURLMacOS="{{.DownloadURLMacOS}}";
(function(){var c=document.getElementById('sfDlBtn');if(!c)return;if(!_dlURLWindows&&!_dlURLMacOS)return;
function esc(s){var d=document.createElement('div');d.appendChild(document.createTextNode(s));return d.innerHTML;}
var ua=navigator.userAgent||navigator.platform||'';var isWin=/Win/.test(ua),isMac=/Mac/.test(ua);
var winSVG='<svg viewBox="0 0 24 24" fill="currentColor"><path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801"/></svg>';
var macSVG='<svg viewBox="0 0 24 24" fill="currentColor"><path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/></svg>';
function mkBtn(url,svg,i18nKey,label,primary){return '<a class="sf-dl-btn'+(primary?' sf-dl-btn-primary':'')+'" href="'+esc(url)+'" target="_blank" rel="noopener">'+svg+' <span data-i18n="'+i18nKey+'">'+label+'</span></a>';}
var html='';
if(isWin){if(_dlURLWindows)html=mkBtn(_dlURLWindows,winSVG,'download_vantagics_windows','下载 Windows 版',true);}
else if(isMac){if(_dlURLMacOS)html=mkBtn(_dlURLMacOS,macSVG,'download_vantagics_macos','下载 macOS 版',true);}
else{if(_dlURLWindows)html+=mkBtn(_dlURLWindows,winSVG,'download_vantagics_windows','下载 Windows 版',false);if(_dlURLMacOS)html+=mkBtn(_dlURLMacOS,macSVG,'download_vantagics_macos','下载 macOS 版',false);}
c.innerHTML=html;})();
function showToast(msg){var t=document.getElementById('toast');t.textContent=msg;t.classList.add('show');setTimeout(function(){t.classList.remove('show');},2500);}
function showMsg(type,msg){var s=document.getElementById('successMsg');var e=document.getElementById('errorMsg');if(s)s.style.display='none';if(e)e.style.display='none';if(type==='success'&&s){s.textContent=msg;s.style.display='block';}else if(e){e.textContent=msg;e.style.display='block';}}
function changeSort(val){var p=new URLSearchParams(window.location.search);p.set('sort',val);window.location.search=p.toString();}
function changeCat(val){var p=new URLSearchParams(window.location.search);p.set('cat',val);window.location.search=p.toString();}
function claimPack(shareToken){if(!confirm(window._i18n('add_to_purchased_confirm','是否将此分析包添加到您的已购分析技能包中？')))return;fetch('/pack/'+shareToken+'/claim',{method:'POST',headers:{'Content-Type':'application/json'}}).then(function(r){return r.json();}).then(function(d){if(d.success){showMsg('success',window._i18n('claim_success','领取成功！'));setTimeout(function(){location.reload();},1000);}else{showMsg('error',d.error||window._i18n('claim_failed','领取失败'));}}).catch(function(){showMsg('error',window._i18n('network_error','网络错误'));});}
function showPurchaseDialog(shareToken,shareMode,creditsPrice,packName){_currentShareToken=shareToken;_currentShareMode=shareMode;_currentCreditsPrice=creditsPrice;document.getElementById('purchaseModalTitle').textContent=window._i18n('purchase','购买')+' - '+packName;var pu=document.getElementById('perUseFields');var su=document.getElementById('subscriptionFields');pu.style.display='none';su.style.display='none';if(shareMode==='per_use'){pu.style.display='block';document.getElementById('purchaseQuantity').value=1;}else if(shareMode==='subscription'){su.style.display='block';document.getElementById('purchaseDuration').selectedIndex=0;}updatePurchaseTotal();document.getElementById('purchaseModal').classList.add('show');}
function closePurchaseDialog(){document.getElementById('purchaseModal').classList.remove('show');}
function updatePurchaseTotal(){var total=0;if(_currentShareMode==='per_use'){var q=parseInt(document.getElementById('purchaseQuantity').value)||1;if(q<1)q=1;total=_currentCreditsPrice*q;}else if(_currentShareMode==='subscription'){var m=parseInt(document.getElementById('purchaseDuration').value)||1;total=_currentCreditsPrice*m;}var el=document.getElementById('purchaseTotal');if(el)el.textContent=window._i18n('total','合计')+'：'+total+' Credits';}
function confirmPurchase(){var body={};if(_currentShareMode==='per_use'){var q=parseInt(document.getElementById('purchaseQuantity').value)||1;if(q<1){showMsg('error',window._i18n('min_1_count','购买次数至少为 1'));return;}body.quantity=q;}else if(_currentShareMode==='subscription'){body.months=parseInt(document.getElementById('purchaseDuration').value)||1;}var btn=document.getElementById('confirmPurchaseBtn');if(btn){btn.disabled=true;btn.textContent=window._i18n('processing','处理中...');}fetch('/pack/'+_currentShareToken+'/purchase',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)}).then(function(r){return r.json();}).then(function(d){if(d.success){closePurchaseDialog();showMsg('success',window._i18n('purchase_success','购买成功！'));setTimeout(function(){location.reload();},1000);}else if(d.insufficient_balance){closePurchaseDialog();var errEl=document.getElementById('errorMsg');if(errEl){errEl.innerHTML=window._i18n('insufficient_balance','余额不足，当前余额')+' '+(d.balance||0)+' Credits。<a href="/user/dashboard" style="color:var(--g600);text-decoration:underline;font-weight:600;">'+window._i18n('go_topup','前往充值')+'</a>';errEl.style.display='block';}if(btn){btn.disabled=false;btn.textContent=window._i18n('confirm_purchase','确认购买');}}else{showMsg('error',d.error||window._i18n('purchase_failed','购买失败'));if(btn){btn.disabled=false;btn.textContent=window._i18n('confirm_purchase','确认购买');}}}).catch(function(){showMsg('error',window._i18n('network_error','网络错误'));if(btn){btn.disabled=false;btn.textContent=window._i18n('confirm_purchase','确认购买');}});}
</script>
`
