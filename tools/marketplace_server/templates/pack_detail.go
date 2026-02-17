package templates

import "html/template"

// PackDetailTmpl is the parsed pack detail page template.
var PackDetailTmpl = template.Must(template.New("pack_detail").Parse(packDetailHTML))

const packDetailHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.PackName}} - 快捷分析包市场</title>
    <meta property="og:title" content="{{.PackName}} - 快捷分析包市场" />
    <meta property="og:description" content="{{.PackDescription}}" />
    <meta property="og:type" content="product" />
    <meta name="twitter:card" content="summary_large_image" />
    <meta name="twitter:title" content="{{.PackName}}" />
    <meta name="twitter:description" content="{{.PackDescription}}" />
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap');
        *,*::before,*::after{margin:0;padding:0;box-sizing:border-box}
        body{font-family:'Inter',-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f8f9fc;min-height:100vh;color:#1e293b;-webkit-font-smoothing:antialiased}
        .page{max-width:720px;margin:0 auto;padding:40px 20px 60px}
        .nav{display:flex;align-items:center;justify-content:space-between;margin-bottom:40px}
        .logo{display:flex;align-items:center;gap:10px;text-decoration:none}
        .logo-mark{width:36px;height:36px;border-radius:10px;display:flex;align-items:center;justify-content:center;background:linear-gradient(135deg,#6366f1,#8b5cf6);font-size:18px;box-shadow:0 2px 8px rgba(99,102,241,0.25)}
        .logo-text{font-size:15px;font-weight:700;color:#1e293b;letter-spacing:-0.2px}
        .nav-link{padding:7px 16px;font-size:13px;font-weight:500;color:#64748b;background:#fff;border:1px solid #e2e8f0;border-radius:8px;text-decoration:none;transition:all .2s}
        .nav-link:hover{color:#1e293b;border-color:#cbd5e1;box-shadow:0 1px 3px rgba(0,0,0,0.06)}
        .hero{position:relative;overflow:hidden;background:linear-gradient(135deg,#eef2ff 0%,#faf5ff 50%,#f0fdf4 100%);border:1px solid #e0e7ff;border-radius:20px;padding:36px 32px 32px;margin-bottom:20px}
        .hero-inner{position:relative;z-index:1}
        .hero-meta{display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-bottom:20px}
        .tag{padding:4px 12px;border-radius:20px;font-size:11px;font-weight:600;letter-spacing:0.3px}
        .tag-free{background:#dcfce7;color:#16a34a;border:1px solid #bbf7d0}
        .tag-peruse{background:#e0e7ff;color:#4f46e5;border:1px solid #c7d2fe}
        .tag-sub{background:#f3e8ff;color:#7c3aed;border:1px solid #e9d5ff}
        .tag-cat{background:#f1f5f9;color:#64748b;border:1px solid #e2e8f0;font-weight:500}
        .pack-title{font-size:28px;font-weight:800;line-height:1.25;letter-spacing:-0.5px;margin-bottom:10px;color:#0f172a}
        .pack-author{display:flex;align-items:center;gap:6px;font-size:13px;color:#64748b;font-weight:500}
        .pack-author svg{opacity:.5}
        .stats{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-bottom:20px}
        @media(max-width:480px){.stats{grid-template-columns:1fr}}
        .stat{background:#fff;border:1px solid #e2e8f0;border-radius:14px;padding:16px 18px;transition:all .25s}
        .stat:hover{border-color:#cbd5e1;box-shadow:0 2px 8px rgba(0,0,0,0.04);transform:translateY(-1px)}
        .stat-label{font-size:10px;text-transform:uppercase;letter-spacing:0.8px;color:#94a3b8;font-weight:600;margin-bottom:6px}
        .stat-val{font-size:14px;color:#1e293b;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
        .desc{background:#fff;border:1px solid #e2e8f0;border-radius:16px;padding:24px 28px;margin-bottom:20px}
        .desc-heading{font-size:13px;font-weight:600;color:#6366f1;margin-bottom:12px;letter-spacing:0.2px}
        .desc-text{font-size:14px;color:#475569;line-height:1.85;white-space:pre-wrap}
        .action-bar{background:#fff;border:1px solid #e2e8f0;border-radius:16px;padding:24px 28px;display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:16px;margin-bottom:16px;box-shadow:0 1px 3px rgba(0,0,0,0.04)}
        .price{font-size:32px;font-weight:800;letter-spacing:-0.5px;color:#6366f1}
        .price-free{color:#16a34a}
        .price-unit{font-size:14px;font-weight:600}
        .price-sub{font-size:12px;color:#94a3b8;margin-top:2px}
        .btn{padding:11px 24px;border:none;border-radius:12px;font-size:14px;font-weight:600;cursor:pointer;display:inline-flex;align-items:center;gap:7px;text-decoration:none;transition:all .25s cubic-bezier(.4,0,.2,1);font-family:inherit}
        .btn-green{background:linear-gradient(135deg,#22c55e,#16a34a);color:#fff;box-shadow:0 2px 8px rgba(34,197,94,0.25)}
        .btn-green:hover{box-shadow:0 4px 16px rgba(34,197,94,0.3);transform:translateY(-1px)}
        .btn-indigo{background:linear-gradient(135deg,#6366f1,#4f46e5);color:#fff;box-shadow:0 2px 8px rgba(99,102,241,0.25)}
        .btn-indigo:hover{box-shadow:0 4px 16px rgba(99,102,241,0.3);transform:translateY(-1px)}
        .btn:disabled{opacity:.6;cursor:not-allowed;transform:none!important}
        .badge-owned{display:inline-flex;align-items:center;gap:7px;padding:10px 22px;background:#dcfce7;color:#16a34a;border:1px solid #bbf7d0;border-radius:12px;font-size:14px;font-weight:600}
        .share-bar{display:flex;align-items:center;gap:8px;margin-bottom:20px}
        .share-label{font-size:12px;color:#94a3b8;font-weight:500}
        .share-btn{width:34px;height:34px;border-radius:8px;border:1px solid #e2e8f0;background:#fff;display:flex;align-items:center;justify-content:center;cursor:pointer;transition:all .2s;color:#94a3b8;text-decoration:none}
        .share-btn:hover{background:#f8fafc;color:#475569;border-color:#cbd5e1;box-shadow:0 1px 3px rgba(0,0,0,0.06)}
        .share-btn svg{width:16px;height:16px}
        .copy-toast{position:fixed;bottom:32px;left:50%;transform:translateX(-50%) translateY(20px);background:#6366f1;color:#fff;padding:10px 24px;border-radius:10px;font-size:13px;font-weight:500;opacity:0;transition:all .3s;pointer-events:none;z-index:99;box-shadow:0 4px 12px rgba(99,102,241,0.3)}
        .copy-toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
        .dialog{display:none;margin-top:14px;background:#fff;border:1px solid #e2e8f0;border-radius:14px;padding:22px 24px}
        .dialog-title{font-size:14px;font-weight:600;color:#1e293b;margin-bottom:14px}
        .field{margin-bottom:12px}
        .field label{font-size:12px;color:#64748b;display:block;margin-bottom:5px;font-weight:500}
        .field input,.field select{width:100%;padding:9px 14px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:8px;font-size:14px;color:#1e293b;transition:border-color .2s,box-shadow .2s;font-family:inherit}
        .field input:focus,.field select:focus{outline:none;border-color:#6366f1;box-shadow:0 0 0 3px rgba(99,102,241,0.1)}
        .dialog-total{font-size:16px;font-weight:700;color:#6366f1;margin-bottom:14px}
        .dialog-btns{display:flex;gap:8px}
        .btn-sm{padding:9px 18px;font-size:13px;border-radius:8px}
        .btn-ghost{padding:9px 18px;font-size:13px;border-radius:8px;background:#f8fafc;color:#64748b;border:1px solid #e2e8f0;cursor:pointer;transition:all .2s;font-family:inherit}
        .btn-ghost:hover{background:#f1f5f9;color:#475569}
        .msg{display:none;padding:12px 16px;border-radius:10px;font-size:13px;margin-top:14px}
        .msg-ok{background:#dcfce7;color:#16a34a;border:1px solid #bbf7d0}
        .msg-err{background:#fee2e2;color:#dc2626;border:1px solid #fecaca}
        .err-card{background:#fff;border:1px solid #e2e8f0;border-radius:20px;padding:72px 28px;text-align:center}
        .err-icon{font-size:48px;margin-bottom:16px}
        .err-text{font-size:15px;color:#64748b;line-height:1.6}
        .foot{text-align:center;margin-top:48px;padding-top:20px;border-top:1px solid #e2e8f0}
        .foot-text{font-size:11px;color:#94a3b8}
        .foot-text a{color:#6366f1;text-decoration:none}
        .foot-text a:hover{text-decoration:underline}
    </style>
</head>
<body>
<div class="page">
    <nav class="nav">
        <a class="logo" href="/"><span class="logo-mark">📦</span><span class="logo-text" data-i18n="site_name">快捷分析包市场</span></a>
        <div>{{if .IsLoggedIn}}<a class="nav-link" href="/user/dashboard" data-i18n="personal_center">个人中心</a>{{else}}<a class="nav-link" href="/user/login" data-i18n="login">登录</a>{{end}}</div>
    </nav>
    {{if .Error}}
    <div class="err-card"><div class="err-icon">😔</div><p class="err-text">{{.Error}}</p><a class="nav-link" href="/" style="margin-top:16px;display:inline-block" data-i18n="back_to_home">返回首页</a></div>
    {{else}}
    <div class="hero"><div class="hero-inner">
        <div class="hero-meta">
            {{if eq .ShareMode "free"}}<span class="tag tag-free" data-i18n="free">免费</span>{{else if eq .ShareMode "per_use"}}<span class="tag tag-peruse" data-i18n="per_use">按次付费</span>{{else if eq .ShareMode "subscription"}}<span class="tag tag-sub" data-i18n="subscription_mode">订阅制</span>{{end}}
            <span class="tag tag-cat">{{.CategoryName}}</span>
        </div>
        <h1 class="pack-title">{{.PackName}}</h1>
        <p class="pack-author"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg> {{.AuthorName}}</p>
    </div></div>
    <div class="stats">
        <div class="stat"><div class="stat-label" data-i18n="data_source">数据源</div><div class="stat-val">{{.SourceName}}</div></div>
        <div class="stat"><div class="stat-label" data-i18n="category">分类</div><div class="stat-val">{{.CategoryName}}</div></div>
        <div class="stat"><div class="stat-label" data-i18n="downloads">下载</div><div class="stat-val">{{.DownloadCount}}</div></div>
    </div>
    <div class="share-bar">
        <span class="share-label" data-i18n="share">分享</span>
        <button class="share-btn" onclick="copyLink()" title="复制链接"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg></button>
        <a class="share-btn" id="shareX" href="#" target="_blank" rel="noopener" title="X"><svg viewBox="0 0 24 24" fill="currentColor"><path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/></svg></a>
        <a class="share-btn" id="shareLI" href="#" target="_blank" rel="noopener" title="LinkedIn"><svg viewBox="0 0 24 24" fill="currentColor"><path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433a2.062 2.062 0 01-2.063-2.065 2.064 2.064 0 112.063 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/></svg></a>
    </div>
    {{if .PackDescription}}<div class="desc"><h3 class="desc-heading" data-i18n="pack_intro">分析包介绍</h3><p class="desc-text">{{.PackDescription}}</p></div>{{end}}
    <div class="action-bar">
        <div>
            {{if eq .ShareMode "free"}}<div class="price price-free" data-i18n="free">免费</div><div class="price-sub" data-i18n="no_credits_free">无需 Credits，直接领取</div>
            {{else}}<div class="price">{{.CreditsPrice}} <span class="price-unit">Credits</span></div><div class="price-sub">{{if eq .ShareMode "per_use"}}<span data-i18n="per_use_label">每次使用</span>{{else}}<span data-i18n="monthly_sub">每月订阅</span>{{end}}</div>{{end}}
        </div>
        <div>
            {{if not .IsLoggedIn}}
                {{if eq .ShareMode "free"}}<a class="btn btn-green" href="/user/login?redirect=/pack/{{.ListingID}}" data-i18n="login_to_claim">登录后领取</a>
                {{else}}<a class="btn btn-indigo" href="/user/login?redirect=/pack/{{.ListingID}}" data-i18n="login_to_buy">登录后购买</a>{{end}}
            {{else if .HasPurchased}}
                <div class="badge-owned"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg> <span data-i18n="already_purchased">已购买</span></div>
            {{else}}
                {{if eq .ShareMode "free"}}<button class="btn btn-green" id="claimBtn" onclick="claimPack()" data-i18n="claim_free">免费领取</button>
                {{else}}<button class="btn btn-indigo" id="purchaseBtn" onclick="showPurchaseDialog()" data-i18n="purchase">购买</button>{{end}}
            {{end}}
        </div>
    </div>
    {{if and .IsLoggedIn (not .HasPurchased)}}
    {{if eq .ShareMode "per_use"}}
    <div class="dialog" id="purchaseDialog"><div class="dialog-title" data-i18n="select_quantity">选择购买数量</div><div class="field"><label for="quantity" data-i18n="buy_count_label">购买次数</label><input type="number" id="quantity" min="1" value="1" onchange="updateTotal()" oninput="updateTotal()" /></div><div class="dialog-total" id="totalPrice"></div><div class="dialog-btns"><button class="btn btn-indigo btn-sm" onclick="confirmPurchase()" data-i18n="confirm_purchase">确认购买</button><button class="btn-ghost" onclick="hidePurchaseDialog()" data-i18n="cancel">取消</button></div></div>
    {{else if eq .ShareMode "subscription"}}
    <div class="dialog" id="purchaseDialog"><div class="dialog-title" data-i18n="select_sub_duration">选择订阅时长</div><div class="field"><label for="months" data-i18n="sub_months">订阅月数</label><select id="months" onchange="updateTotal()">{{range $i := .MonthOptions}}<option value="{{$i}}">{{$i}} <span data-i18n="months_unit">个月</span></option>{{end}}</select></div><div class="dialog-total" id="totalPrice"></div><div class="dialog-btns"><button class="btn btn-indigo btn-sm" onclick="confirmPurchase()" data-i18n="confirm_purchase">确认购买</button><button class="btn-ghost" onclick="hidePurchaseDialog()" data-i18n="cancel">取消</button></div></div>
    {{end}}
    {{end}}
    <div class="msg msg-ok" id="successMsg"></div>
    <div class="msg msg-err" id="errorMsg"></div>
    {{end}}
    <div class="foot"><p class="foot-text">VantageData <span data-i18n="site_name">快捷分析包市场</span> · <a href="/" data-i18n="browse_more">浏览更多</a></p></div>
</div>
<div class="copy-toast" id="copyToast" data-i18n="link_copied">链接已复制</div>
<script>
var listingID={{.ListingID}},creditsPrice={{.CreditsPrice}},shareMode="{{.ShareMode}}";
(function(){var u=encodeURIComponent(location.href),t=encodeURIComponent(document.title),x=document.getElementById("shareX"),l=document.getElementById("shareLI");if(x)x.href="https://twitter.com/intent/tweet?text="+t+"&url="+u;if(l)l.href="https://www.linkedin.com/sharing/share-offsite/?url="+u})();
function showMsg(a,b){var s=document.getElementById("successMsg"),e=document.getElementById("errorMsg");if(s)s.style.display="none";if(e)e.style.display="none";if(a==="success"&&s){s.textContent=b;s.style.display="block"}else if(e){e.textContent=b;e.style.display="block"}}
function copyLink(){navigator.clipboard.writeText(location.href).then(function(){var t=document.getElementById("copyToast");t.classList.add("show");setTimeout(function(){t.classList.remove("show")},2e3)})}
function claimPack(){if(!confirm(window._i18n("add_to_purchased_confirm","是否将此分析包添加到您的已购快捷分析包中？")))return;var b=document.getElementById("claimBtn");b.disabled=!0;b.innerHTML=window._i18n("claiming","领取中...");fetch("/pack/"+listingID+"/claim",{method:"POST",headers:{"Content-Type":"application/json"}}).then(function(r){return r.json()}).then(function(d){if(d.success){showMsg("success",window._i18n("claim_success","领取成功！"));b.outerHTML='<div class="badge-owned">'+window._i18n("claimed","已领取")+'</div>'}else{showMsg("error",d.error||window._i18n("claim_failed","领取失败"));b.disabled=!1;b.innerHTML=window._i18n("claim_free","免费领取")}}).catch(function(){showMsg("error",window._i18n("network_error","网络错误"));b.disabled=!1;b.innerHTML=window._i18n("claim_free","免费领取")})}
function showPurchaseDialog(){var d=document.getElementById("purchaseDialog");if(d)d.style.display="block";updateTotal()}
function hidePurchaseDialog(){var d=document.getElementById("purchaseDialog");if(d)d.style.display="none"}
function updateTotal(){var a=0;if(shareMode==="per_use"){var q=parseInt(document.getElementById("quantity").value)||1;if(q<1)q=1;a=creditsPrice*q}else if(shareMode==="subscription"){a=creditsPrice*(parseInt(document.getElementById("months").value)||1)}var el=document.getElementById("totalPrice");if(el)el.textContent=window._i18n("total","合计")+"："+a+" Credits"}
function confirmPurchase(){var body={};if(shareMode==="per_use"){var q=parseInt(document.getElementById("quantity").value)||1;if(q<1){showMsg("error",window._i18n("min_1_count","购买次数至少为 1"));return}body.quantity=q}else if(shareMode==="subscription"){body.months=parseInt(document.getElementById("months").value)||1}var b=document.querySelectorAll("#purchaseDialog .btn-indigo")[0];if(b){b.disabled=!0;b.textContent=window._i18n("processing","处理中...")}fetch("/pack/"+listingID+"/purchase",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(body)}).then(function(r){return r.json()}).then(function(d){if(d.success){hidePurchaseDialog();alert(window._i18n("purchase_success","购买成功！"));location.href="/user/dashboard"}else if(d.insufficient_balance){showMsg("error",window._i18n("insufficient_balance","余额不足，当前余额")+" "+(d.balance||0)+" Credits");if(b){b.disabled=!1;b.textContent=window._i18n("confirm_purchase","确认购买")}}else{showMsg("error",d.error||window._i18n("purchase_failed","购买失败"));if(b){b.disabled=!1;b.textContent=window._i18n("confirm_purchase","确认购买")}}}).catch(function(){showMsg("error",window._i18n("network_error","网络错误"));if(b){b.disabled=!1;b.textContent=window._i18n("confirm_purchase","确认购买")}})}
</script>
` + I18nJS + `
</body>
</html>`
