package templates

import "html/template"

// StorefrontManageTmpl is the parsed storefront management page template.
var StorefrontManageTmpl = template.Must(template.New("storefront_manage").Parse(storefrontManageHTML))

const storefrontManageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="default-lang" content="{{.DefaultLang}}">
    <title>小铺管理 - 快捷分析包市场</title>
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

        /* Page title */
        .page-title {
            font-size: 22px; font-weight: 800; color: #0f172a;
            margin-bottom: 24px; letter-spacing: -0.3px;
            display: flex; align-items: center; gap: 10px;
        }

        /* Tabs */
        .tabs {
            display: flex; gap: 0; margin-bottom: 24px;
            border-bottom: 2px solid #e2e8f0;
        }
        .tab-btn {
            padding: 12px 24px; font-size: 14px; font-weight: 600;
            color: #64748b; background: none; border: none;
            border-bottom: 2px solid transparent; margin-bottom: -2px;
            cursor: pointer; transition: all 0.15s; font-family: inherit;
        }
        .tab-btn:hover { color: #4f46e5; }
        .tab-btn.active { color: #4f46e5; border-bottom-color: #4f46e5; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }

        /* Card */
        .card {
            background: #fff; border-radius: 12px; padding: 24px;
            margin-bottom: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 4px 16px rgba(0,0,0,0.04);
            border: 1px solid #e2e8f0;
        }
        .card-title {
            font-size: 15px; font-weight: 700; color: #1e293b;
            margin-bottom: 16px; display: flex; align-items: center; gap: 8px;
        }
        .card-title .icon { font-size: 16px; }

        /* Form fields */
        .field-group { margin-bottom: 16px; }
        .field-group label {
            font-size: 13px; color: #334155; display: block;
            margin-bottom: 6px; font-weight: 600;
        }
        .field-group input[type="text"],
        .field-group textarea,
        .field-group select {
            width: 100%; padding: 9px 14px;
            border: 1px solid #cbd5e1; border-radius: 8px;
            font-size: 14px; background: #fff;
            transition: border-color 0.15s, box-shadow 0.15s;
            color: #1e293b; font-family: inherit;
        }
        .field-group input[type="text"]:focus,
        .field-group textarea:focus,
        .field-group select:focus {
            outline: none; border-color: #4f46e5;
            box-shadow: 0 0 0 3px rgba(79,70,229,0.12);
        }
        .field-group textarea { resize: vertical; min-height: 80px; }
        .field-hint { font-size: 12px; color: #94a3b8; margin-top: 4px; }

        /* Slug row */
        .slug-row {
            display: flex; align-items: center; gap: 8px;
        }
        .slug-row input { flex: 1; }
        .slug-prefix {
            font-size: 13px; color: #64748b; white-space: nowrap;
            font-weight: 500;
        }
        .url-display {
            font-size: 13px; color: #6366f1; word-break: break-all;
            margin-top: 6px; padding: 8px 12px;
            background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;
        }

        /* Buttons */
        .btn {
            padding: 8px 18px; border: none; border-radius: 8px;
            font-size: 13px; font-weight: 600; cursor: pointer;
            display: inline-flex; align-items: center; gap: 5px;
            text-decoration: none; transition: all 0.2s; font-family: inherit;
        }
        .btn-indigo {
            background: linear-gradient(135deg, #6366f1, #4f46e5); color: #fff;
            box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .btn-indigo:hover { box-shadow: 0 4px 16px rgba(99,102,241,0.3); transform: translateY(-1px); }
        .btn-green {
            background: linear-gradient(135deg, #22c55e, #16a34a); color: #fff;
            box-shadow: 0 2px 8px rgba(34,197,94,0.25);
        }
        .btn-green:hover { box-shadow: 0 4px 16px rgba(34,197,94,0.3); transform: translateY(-1px); }
        .btn-red {
            background: linear-gradient(135deg, #f87171, #ef4444); color: #fff;
            box-shadow: 0 2px 8px rgba(239,68,68,0.25);
        }
        .btn-red:hover { box-shadow: 0 4px 16px rgba(239,68,68,0.3); transform: translateY(-1px); }
        .btn-ghost {
            padding: 8px 18px; font-size: 13px; border-radius: 8px;
            background: #f8fafc; color: #64748b; border: 1px solid #e2e8f0;
            cursor: pointer; transition: all .2s; font-family: inherit; font-weight: 600;
        }
        .btn-ghost:hover { background: #f1f5f9; color: #475569; }
        .btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none !important; }
        .btn-sm { padding: 6px 14px; font-size: 12px; }

        /* Logo upload */
        .logo-upload-area {
            display: flex; align-items: center; gap: 20px;
            padding: 16px; background: #f8fafc; border-radius: 10px;
            border: 1px dashed #cbd5e1;
        }
        .logo-preview {
            width: 80px; height: 80px; border-radius: 16px;
            overflow: hidden; flex-shrink: 0;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            display: flex; align-items: center; justify-content: center;
            font-size: 32px; font-weight: 800; color: #fff;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .logo-preview img { width: 100%; height: 100%; object-fit: cover; }
        .logo-upload-info { flex: 1; }
        .logo-upload-info p { font-size: 13px; color: #64748b; margin-bottom: 8px; }

        /* Toggle switch */
        .toggle-row {
            display: flex; align-items: center; justify-content: space-between;
            padding: 14px 0; border-bottom: 1px solid #f1f5f9;
        }
        .toggle-row:last-child { border-bottom: none; }
        .toggle-label { font-size: 14px; font-weight: 600; color: #1e293b; }
        .toggle-desc { font-size: 12px; color: #94a3b8; margin-top: 2px; }
        .toggle-switch {
            position: relative; width: 44px; height: 24px;
            background: #cbd5e1; border-radius: 12px;
            cursor: pointer; transition: background 0.2s;
            border: none; padding: 0;
        }
        .toggle-switch.on { background: #4f46e5; }
        .toggle-switch::after {
            content: ''; position: absolute;
            top: 2px; left: 2px;
            width: 20px; height: 20px;
            background: #fff; border-radius: 50%;
            transition: transform 0.2s;
            box-shadow: 0 1px 3px rgba(0,0,0,0.15);
        }
        .toggle-switch.on::after { transform: translateX(20px); }

        /* Pack list items */
        .pack-list { display: flex; flex-direction: column; gap: 10px; }
        .pack-item {
            display: flex; align-items: center; gap: 14px;
            padding: 14px 16px; background: #f8fafc;
            border-radius: 10px; border: 1px solid #e2e8f0;
            transition: background 0.15s;
        }
        .pack-item:hover { background: #f1f5f9; }
        .pack-item-body { flex: 1; min-width: 0; }
        .pack-item-name { font-size: 14px; font-weight: 600; color: #1e293b; }
        .pack-item-meta { font-size: 12px; color: #94a3b8; margin-top: 2px; }
        .pack-item-actions { display: flex; gap: 6px; flex-shrink: 0; }
        .tag {
            display: inline-flex; align-items: center;
            padding: 2px 8px; border-radius: 20px;
            font-size: 11px; font-weight: 700; letter-spacing: 0.2px;
        }
        .tag-free { background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; }
        .tag-per-use { background: #eef2ff; color: #4338ca; border: 1px solid #c7d2fe; }
        .tag-subscription { background: #f5f3ff; color: #7c3aed; border: 1px solid #ddd6fe; }
        .tag-featured { background: #fef3c7; color: #b45309; border: 1px solid #fde68a; }

        /* Featured drag list */
        .featured-list { display: flex; flex-direction: column; gap: 8px; }
        .featured-item {
            display: flex; align-items: center; gap: 12px;
            padding: 12px 14px; background: #fffbeb;
            border-radius: 10px; border: 1px solid #fde68a;
            cursor: grab; transition: box-shadow 0.15s;
        }
        .featured-item:active { cursor: grabbing; }
        .featured-item.dragging { opacity: 0.5; box-shadow: 0 4px 16px rgba(0,0,0,0.1); }
        .featured-item .drag-handle {
            color: #d97706; font-size: 16px; cursor: grab; user-select: none;
        }
        .featured-item-body { flex: 1; min-width: 0; }
        .featured-item-name { font-size: 13px; font-weight: 600; color: #92400e; }
        .featured-item-price { font-size: 12px; color: #b45309; }

        /* Notification list */
        .notify-list { display: flex; flex-direction: column; gap: 10px; }
        .notify-item {
            display: flex; align-items: center; gap: 14px;
            padding: 14px 16px; background: #f8fafc;
            border-radius: 10px; border: 1px solid #e2e8f0;
            cursor: pointer; transition: background 0.15s;
        }
        .notify-item:hover { background: #f1f5f9; }
        .notify-item-body { flex: 1; min-width: 0; }
        .notify-item-subject { font-size: 14px; font-weight: 600; color: #1e293b; }
        .notify-item-meta { font-size: 12px; color: #94a3b8; margin-top: 2px; }
        .notify-status {
            padding: 3px 10px; border-radius: 20px;
            font-size: 11px; font-weight: 700;
        }
        .notify-status-sent { background: #dcfce7; color: #16a34a; border: 1px solid #bbf7d0; }
        .notify-status-failed { background: #fee2e2; color: #dc2626; border: 1px solid #fecaca; }

        /* Email editor */
        .email-editor {
            display: none; background: #fff; border-radius: 12px;
            padding: 24px; margin-bottom: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 4px 16px rgba(0,0,0,0.04);
            border: 1px solid #e2e8f0;
        }
        .email-editor.show { display: block; }
        .recipient-info {
            display: inline-flex; align-items: center; gap: 6px;
            padding: 6px 14px; background: #eef2ff; color: #4338ca;
            border-radius: 8px; font-size: 13px; font-weight: 600;
            margin-bottom: 16px;
        }

        /* Messages */
        .msg { display: none; padding: 12px 16px; border-radius: 10px; font-size: 13px; margin-bottom: 14px; font-weight: 500; }
        .msg-ok { background: #dcfce7; color: #16a34a; border: 1px solid #bbf7d0; }
        .msg-err { background: #fee2e2; color: #dc2626; border: 1px solid #fecaca; }

        /* Empty state */
        .empty-state {
            text-align: center; padding: 32px 20px; color: #94a3b8;
            font-size: 13px;
        }
        .empty-state .icon { font-size: 28px; margin-bottom: 8px; opacity: 0.7; }

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
            max-width: 480px; width: 90%;
            box-shadow: 0 20px 60px rgba(0,0,0,0.15);
            position: relative; border: 1px solid #e2e8f0;
            max-height: 80vh; overflow-y: auto;
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

        /* Pack select list in modal */
        .pack-select-list { max-height: 300px; overflow-y: auto; }
        .pack-select-item {
            display: flex; align-items: center; gap: 10px;
            padding: 10px 12px; border-radius: 8px;
            cursor: pointer; transition: background 0.15s;
        }
        .pack-select-item:hover { background: #f8fafc; }
        .pack-select-item input[type="checkbox"] { flex-shrink: 0; }
        .pack-select-item-name { font-size: 13px; font-weight: 600; color: #1e293b; }
        .pack-select-item-mode { font-size: 11px; color: #94a3b8; }

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

        /* Footer */
        .foot { text-align: center; margin-top: 28px; padding-top: 16px; border-top: 1px solid #e2e8f0; }
        .foot-text { font-size: 11px; color: #94a3b8; }
        .foot-text a { color: #6366f1; text-decoration: none; }
        .foot-text a:hover { text-decoration: underline; }

        /* Notification detail modal */
        .notify-detail-subject { font-size: 16px; font-weight: 700; color: #1e293b; margin-bottom: 12px; }
        .notify-detail-body {
            font-size: 14px; color: #475569; line-height: 1.8;
            white-space: pre-wrap; padding: 16px;
            background: #f8fafc; border-radius: 8px; border: 1px solid #e2e8f0;
        }

        @media (max-width: 640px) {
            .tabs { overflow-x: auto; }
            .tab-btn { padding: 10px 16px; font-size: 13px; white-space: nowrap; }
            .slug-row { flex-direction: column; align-items: stretch; }
            .logo-upload-area { flex-direction: column; text-align: center; }
            .pack-item { flex-direction: column; align-items: stretch; }
            .pack-item-actions { align-self: flex-end; }
        }
    </style>
</head>
<body>
<div class="page">
    <!-- Navigation -->
    <nav class="nav">
        <a class="logo-link" href="/"><span class="logo-mark">📦</span><span class="logo-text" data-i18n="site_name">快捷分析包市场</span></a>
        <div style="display:flex;gap:8px;">
            <a class="nav-link" href="/store/{{.Storefront.StoreSlug}}" target="_blank">🔗 查看小铺</a>
            <a class="nav-link" href="/user/dashboard" data-i18n="personal_center">个人中心</a>
        </div>
    </nav>

    <h1 class="page-title">🏪 小铺管理</h1>

    <!-- Messages -->
    <div class="msg msg-ok" id="successMsg"></div>
    <div class="msg msg-err" id="errorMsg"></div>

    <!-- Tabs -->
    <div class="tabs">
        <button class="tab-btn active" onclick="switchTab('settings', this)">⚙️ 小铺设置</button>
        <button class="tab-btn" onclick="switchTab('packs', this)">📦 分析包管理</button>
        <button class="tab-btn" onclick="switchTab('notify', this)">📧 客户通知</button>
    </div>

    <!-- ==================== Tab 1: 小铺设置 ==================== -->
    <div class="tab-content active" id="tab-settings">
        <!-- Store Name & Description -->
        <div class="card">
            <div class="card-title"><span class="icon">📝</span> 基本信息</div>
            <div class="field-group">
                <label for="storeName">小铺名称</label>
                <input type="text" id="storeName" value="{{.Storefront.StoreName}}" maxlength="30" placeholder="输入小铺名称（2-30 字符）">
                <div class="field-hint">名称长度 2-30 字符</div>
            </div>
            <div class="field-group">
                <label for="storeDesc">小铺描述</label>
                <textarea id="storeDesc" rows="3" placeholder="介绍一下你的小铺...">{{.Storefront.Description}}</textarea>
            </div>
            <button class="btn btn-indigo" onclick="saveSettings()">💾 保存设置</button>
        </div>

        <!-- Logo Upload -->
        <div class="card">
            <div class="card-title"><span class="icon">🖼️</span> Logo 设置</div>
            <div class="logo-upload-area">
                <div class="logo-preview" id="logoPreview">
                    {{if .Storefront.HasLogo}}
                    <img src="/store/{{.Storefront.StoreSlug}}/logo" alt="Logo" id="logoImg">
                    {{else}}
                    <span id="logoLetter">{{if .Storefront.StoreName}}{{slice .Storefront.StoreName 0 1}}{{else}}?{{end}}</span>
                    {{end}}
                </div>
                <div class="logo-upload-info">
                    <p>支持 PNG 或 JPEG 格式，文件大小不超过 2MB，也可直接 Ctrl+V 粘贴图片</p>
                    <input type="file" id="logoFile" accept="image/png,image/jpeg" style="display:none;" onchange="uploadLogo()">
                    <button class="btn btn-ghost" onclick="document.getElementById('logoFile').click()">📤 上传 Logo</button>
                </div>
            </div>
        </div>

        <!-- Store Slug -->
        <div class="card">
            <div class="card-title"><span class="icon">🔗</span> 小铺链接</div>
            <div class="field-group">
                <label for="storeSlug">小铺标识（Store Slug）</label>
                <div class="slug-row">
                    <span class="slug-prefix">/store/</span>
                    <input type="text" id="storeSlug" value="{{.Storefront.StoreSlug}}" maxlength="50" placeholder="my-store">
                    <button class="btn btn-indigo btn-sm" onclick="updateSlug()">保存</button>
                </div>
                <div class="field-hint">仅允许小写字母、数字和连字符，长度 3-50 字符</div>
            </div>
            <div class="url-display" id="fullUrlDisplay">{{.FullURL}}</div>
            <div style="margin-top:10px;">
                <button class="btn btn-ghost" onclick="copyStoreUrl()">📋 复制小铺链接</button>
            </div>
        </div>
    </div>

    <!-- ==================== Tab 2: 分析包管理 ==================== -->
    <div class="tab-content" id="tab-packs">
        <!-- Auto-add toggle -->
        <div class="card">
            <div class="card-title"><span class="icon">🔄</span> 入铺模式</div>
            <div class="toggle-row">
                <div>
                    <div class="toggle-label">自动入铺模式</div>
                    <div class="toggle-desc">开启后，所有在售分析包自动加入小铺，无需手动添加</div>
                </div>
                <button class="toggle-switch{{if .Storefront.AutoAddEnabled}} on{{end}}" id="autoAddToggle" onclick="toggleAutoAdd()"></button>
            </div>
        </div>

        <!-- Pack list -->
        <div class="card">
            <div class="card-title" style="justify-content:space-between;">
                <span><span class="icon">📦</span> 小铺分析包</span>
                <span id="addPackBtnWrap"{{if .Storefront.AutoAddEnabled}} style="display:none;"{{end}}>
                    <button class="btn btn-green btn-sm" onclick="showAddPackModal()">+ 添加分析包</button>
                </span>
            </div>
            <div id="packListArea">
            {{if .Storefront.AutoAddEnabled}}
                {{if .AuthorPacks}}
                <div class="pack-list">
                    {{range .AuthorPacks}}
                    <div class="pack-item">
                        <div class="pack-item-body">
                            <div class="pack-item-name">
                                {{.PackName}}
                                {{if eq .ShareMode "free"}}<span class="tag tag-free">免费</span>
                                {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use">按次</span>
                                {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription">订阅</span>
                                {{end}}
                            </div>
                            <div class="pack-item-meta">{{.CreditsPrice}} Credits · 自动入铺</div>
                        </div>
                    </div>
                    {{end}}
                </div>
                {{else}}
                <div class="empty-state"><div class="icon">📭</div><p>暂无在售分析包</p></div>
                {{end}}
            {{else}}
                {{if .StorefrontPacks}}
                <div class="pack-list">
                    {{range .StorefrontPacks}}
                    <div class="pack-item" id="pack-item-{{.ListingID}}">
                        <div class="pack-item-body">
                            <div class="pack-item-name">
                                {{.PackName}}
                                {{if eq .ShareMode "free"}}<span class="tag tag-free">免费</span>
                                {{else if eq .ShareMode "per_use"}}<span class="tag tag-per-use">按次</span>
                                {{else if eq .ShareMode "subscription"}}<span class="tag tag-subscription">订阅</span>
                                {{end}}
                                {{if .IsFeatured}}<span class="tag tag-featured">⭐ 推荐</span>{{end}}
                            </div>
                            <div class="pack-item-meta">{{.CreditsPrice}} Credits</div>
                        </div>
                        <div class="pack-item-actions">
                            <button class="btn btn-red btn-sm" onclick="removePack({{.ListingID}}, '{{.PackName}}')">移除</button>
                        </div>
                    </div>
                    {{end}}
                </div>
                {{else}}
                <div class="empty-state"><div class="icon">📭</div><p>小铺中暂无分析包，点击上方按钮添加</p></div>
                {{end}}
            {{end}}
            </div>
        </div>

        <!-- Featured packs -->
        <div class="card">
            <div class="card-title" style="justify-content:space-between;">
                <span><span class="icon">⭐</span> 店主推荐（最多 4 个）</span>
                <button class="btn btn-ghost btn-sm" onclick="showFeaturedSelectModal()">+ 设置推荐</button>
            </div>
            {{if .FeaturedPacks}}
            <div class="featured-list" id="featuredList">
                {{range .FeaturedPacks}}
                <div class="featured-item" draggable="true" data-id="{{.ListingID}}">
                    <span class="drag-handle">⠿</span>
                    <div class="featured-item-body">
                        <div class="featured-item-name">{{.PackName}}</div>
                        <div class="featured-item-price">
                            {{if eq .ShareMode "free"}}免费{{else}}{{.CreditsPrice}} Credits{{end}}
                        </div>
                    </div>
                    <button class="btn btn-ghost btn-sm" onclick="removeFeatured({{.ListingID}})">取消推荐</button>
                </div>
                {{end}}
            </div>
            <div style="margin-top:12px;">
                <button class="btn btn-indigo btn-sm" onclick="saveFeaturedOrder()">💾 保存排序</button>
            </div>
            {{else}}
            <div class="empty-state" id="featuredEmpty"><div class="icon">⭐</div><p>暂未设置推荐分析包</p></div>
            {{end}}
        </div>
    </div>

    <!-- ==================== Tab 3: 客户通知 ==================== -->
    <div class="tab-content" id="tab-notify">
        <!-- Send new notification button -->
        <div style="margin-bottom:16px;">
            <button class="btn btn-indigo" onclick="toggleEmailEditor()">✉️ 发送新通知</button>
        </div>

        <!-- Email editor -->
        <div class="email-editor" id="emailEditor">
            <div class="card-title"><span class="icon">✏️</span> 编辑邮件</div>

            <!-- Recipient scope -->
            <div class="field-group">
                <label>发送范围</label>
                <select id="recipientScope" onchange="onScopeChange()">
                    <option value="all">全部客户</option>
                    <option value="partial">部分客户（按分析包选择）</option>
                </select>
            </div>

            <!-- Partial: pack selection -->
            <div class="field-group" id="partialPacksWrap" style="display:none;">
                <label>选择分析包（勾选后，购买过这些分析包的客户将收到邮件）</label>
                <div class="pack-select-list">
                    {{range .AuthorPacks}}
                    <label class="pack-select-item">
                        <input type="checkbox" class="notify-pack-cb" value="{{.ListingID}}">
                        <span class="pack-select-item-name">{{.PackName}}</span>
                        <span class="pack-select-item-mode">
                            {{if eq .ShareMode "free"}}免费{{else if eq .ShareMode "per_use"}}按次{{else}}订阅{{end}}
                        </span>
                    </label>
                    {{end}}
                </div>
            </div>

            <div class="recipient-info" id="recipientInfo">📬 收件人数量加载中...</div>

            <!-- Template selection -->
            <div class="field-group">
                <label for="notifyTemplate">邮件模板</label>
                <select id="notifyTemplate" onchange="applyTemplate()">
                    <option value="">不使用模板</option>
                    {{range .Templates}}
                    <option value="{{.Type}}" data-subject="{{.Subject}}" data-body="{{.Body}}">{{.Name}}</option>
                    {{end}}
                </select>
            </div>

            <!-- Subject -->
            <div class="field-group">
                <label for="notifySubject">邮件主题</label>
                <input type="text" id="notifySubject" placeholder="输入邮件主题">
            </div>

            <!-- Body -->
            <div class="field-group">
                <label for="notifyBody">邮件正文</label>
                <textarea id="notifyBody" rows="8" placeholder="输入邮件正文内容..."></textarea>
            </div>

            <div style="display:flex;gap:10px;">
                <button class="btn btn-indigo" onclick="sendNotification()">📤 发送邮件</button>
                <button class="btn btn-ghost" onclick="toggleEmailEditor()">取消</button>
            </div>
        </div>

        <!-- Notification history -->
        <div class="card">
            <div class="card-title"><span class="icon">📋</span> 发送记录</div>
            {{if .Notifications}}
            <div class="notify-list">
                {{range .Notifications}}
                <div class="notify-item" onclick="showNotifyDetail({{.ID}})">
                    <div class="notify-item-body">
                        <div class="notify-item-subject">{{.Subject}}</div>
                        <div class="notify-item-meta">{{.CreatedAt}} · 收件人 {{.RecipientCount}} 人</div>
                    </div>
                    <span class="notify-status {{if eq .Status "sent"}}notify-status-sent{{else}}notify-status-failed{{end}}">
                        {{if eq .Status "sent"}}已发送{{else}}失败{{end}}
                    </span>
                </div>
                {{end}}
            </div>
            {{else}}
            <div class="empty-state"><div class="icon">📭</div><p>暂无发送记录</p></div>
            {{end}}
        </div>
    </div>

    <!-- Footer -->
    <div class="foot">
        <p class="foot-text">Vantagics <span data-i18n="site_name">快捷分析包市场</span> · <a href="/" data-i18n="browse_more">浏览更多</a></p>
    </div>
</div>

<!-- Add Pack Modal -->
<div class="modal-overlay" id="addPackModal">
    <div class="modal-box">
        <button class="modal-close" onclick="closeAddPackModal()">✕</button>
        <div class="modal-title">添加分析包到小铺</div>
        <div class="pack-select-list" id="addPackList">
            {{range .AuthorPacks}}
            <label class="pack-select-item">
                <input type="checkbox" class="add-pack-cb" value="{{.ListingID}}" data-name="{{.PackName}}">
                <span class="pack-select-item-name">{{.PackName}}</span>
                <span class="pack-select-item-mode">
                    {{if eq .ShareMode "free"}}免费{{else if eq .ShareMode "per_use"}}按次{{else}}订阅{{end}} · {{.CreditsPrice}} Credits
                </span>
            </label>
            {{end}}
        </div>
        <div class="modal-actions">
            <button class="btn-ghost" onclick="closeAddPackModal()">取消</button>
            <button class="btn btn-green" onclick="confirmAddPacks()">确认添加</button>
        </div>
    </div>
</div>

<!-- Featured Select Modal -->
<div class="modal-overlay" id="featuredSelectModal">
    <div class="modal-box">
        <button class="modal-close" onclick="closeFeaturedSelectModal()">✕</button>
        <div class="modal-title">选择推荐分析包（最多 4 个）</div>
        <div class="pack-select-list" id="featuredSelectList">
        </div>
        <div class="modal-actions">
            <button class="btn-ghost" onclick="closeFeaturedSelectModal()">取消</button>
            <button class="btn btn-indigo" onclick="confirmSetFeatured()">确认设置</button>
        </div>
    </div>
</div>

<!-- Notification Detail Modal -->
<div class="modal-overlay" id="notifyDetailModal">
    <div class="modal-box">
        <button class="modal-close" onclick="closeNotifyDetailModal()">✕</button>
        <div class="modal-title">通知详情</div>
        <div class="notify-detail-subject" id="notifyDetailSubject"></div>
        <div class="notify-detail-body" id="notifyDetailBody"></div>
    </div>
</div>

<!-- Toast -->
<div class="toast" id="toast"></div>

<script>
/* ===== Utility functions ===== */
function showToast(msg) {
    var t = document.getElementById('toast');
    t.textContent = msg; t.classList.add('show');
    setTimeout(function() { t.classList.remove('show'); }, 2500);
}
function showMsg(type, msg) {
    var s = document.getElementById('successMsg');
    var e = document.getElementById('errorMsg');
    if (s) s.style.display = 'none';
    if (e) e.style.display = 'none';
    if (type === 'ok' && s) { s.textContent = msg; s.style.display = 'block'; }
    else if (e) { e.textContent = msg; e.style.display = 'block'; }
    window.scrollTo({ top: 0, behavior: 'smooth' });
}
function clearMsg() {
    var s = document.getElementById('successMsg');
    var e = document.getElementById('errorMsg');
    if (s) s.style.display = 'none';
    if (e) e.style.display = 'none';
}

/* ===== Tab switching ===== */
function switchTab(tabId, btn) {
    document.querySelectorAll('.tab-content').forEach(function(el) { el.classList.remove('active'); });
    document.querySelectorAll('.tab-btn').forEach(function(el) { el.classList.remove('active'); });
    document.getElementById('tab-' + tabId).classList.add('active');
    btn.classList.add('active');
    clearMsg();
}

/* ===== Settings: Save name & description ===== */
function saveSettings() {
    var name = document.getElementById('storeName').value.trim();
    var desc = document.getElementById('storeDesc').value.trim();
    if (name.length < 2 || name.length > 30) {
        showMsg('err', '小铺名称长度需在 2-30 字符之间');
        return;
    }
    var fd = new FormData();
    fd.append('store_name', name);
    fd.append('description', desc);
    fetch('/user/storefront/settings', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) { showMsg('ok', '设置已保存'); }
        else { showMsg('err', d.error || '保存失败'); }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Settings: Upload Logo ===== */
function doUploadLogo(file) {
    if (file.size > 2 * 1024 * 1024) {
        showMsg('err', '图片大小不能超过 2MB');
        return;
    }
    if (file.type !== 'image/png' && file.type !== 'image/jpeg') {
        showMsg('err', '仅支持 PNG 或 JPEG 格式');
        return;
    }
    var fd = new FormData();
    fd.append('logo', file);
    fetch('/user/storefront/logo', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            showMsg('ok', 'Logo 已更新');
            var preview = document.getElementById('logoPreview');
            preview.innerHTML = '<img src="/store/{{.Storefront.StoreSlug}}/logo?t=' + Date.now() + '" alt="Logo">';
        } else {
            showMsg('err', d.error || '上传失败');
        }
    }).catch(function() { showMsg('err', '网络错误'); });
}
function uploadLogo() {
    var fileInput = document.getElementById('logoFile');
    if (!fileInput.files.length) return;
    doUploadLogo(fileInput.files[0]);
    fileInput.value = '';
}
/* Paste image from clipboard */
document.addEventListener('paste', function(e) {
    var activeTab = document.getElementById('tab-settings');
    if (!activeTab || !activeTab.classList.contains('active')) return;
    var items = (e.clipboardData || e.originalEvent.clipboardData).items;
    for (var i = 0; i < items.length; i++) {
        if (items[i].type === 'image/png' || items[i].type === 'image/jpeg') {
            e.preventDefault();
            doUploadLogo(items[i].getAsFile());
            return;
        }
    }
});

/* ===== Settings: Update Slug ===== */
function updateSlug() {
    var slug = document.getElementById('storeSlug').value.trim();
    if (!/^[a-z0-9-]+$/.test(slug)) {
        showMsg('err', '小铺标识仅允许小写字母、数字和连字符');
        return;
    }
    if (slug.length < 3 || slug.length > 50) {
        showMsg('err', '小铺标识长度需在 3-50 字符之间');
        return;
    }
    var fd = new FormData();
    fd.append('slug', slug);
    fetch('/user/storefront/slug', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            showMsg('ok', '小铺标识已更新');
            var base = window.location.protocol + '//' + window.location.host;
            document.getElementById('fullUrlDisplay').textContent = base + '/store/' + slug;
        } else {
            showMsg('err', d.error || '更新失败');
        }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Settings: Copy URL ===== */
function copyStoreUrl() {
    var url = document.getElementById('fullUrlDisplay').textContent.trim();
    if (navigator.clipboard) {
        navigator.clipboard.writeText(url).then(function() {
            showToast('小铺链接已复制');
        });
    } else {
        var ta = document.createElement('textarea');
        ta.value = url; document.body.appendChild(ta);
        ta.select(); document.execCommand('copy');
        document.body.removeChild(ta);
        showToast('小铺链接已复制');
    }
}

/* ===== Packs: Toggle auto-add ===== */
function toggleAutoAdd() {
    var btn = document.getElementById('autoAddToggle');
    var enabling = !btn.classList.contains('on');
    var fd = new FormData();
    fd.append('enabled', enabling ? '1' : '0');
    fetch('/user/storefront/auto-add', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            if (enabling) { btn.classList.add('on'); } else { btn.classList.remove('on'); }
            showMsg('ok', enabling ? '已开启自动入铺模式' : '已关闭自动入铺模式');
            // Reload to refresh pack list display
            setTimeout(function() { location.reload(); }, 800);
        } else {
            showMsg('err', d.error || '操作失败');
        }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Packs: Add pack modal ===== */
function showAddPackModal() {
    document.getElementById('addPackModal').classList.add('show');
}
function closeAddPackModal() {
    document.getElementById('addPackModal').classList.remove('show');
}
function confirmAddPacks() {
    var cbs = document.querySelectorAll('.add-pack-cb:checked');
    if (cbs.length === 0) { showToast('请选择至少一个分析包'); return; }
    var promises = [];
    cbs.forEach(function(cb) {
        var fd = new FormData();
        fd.append('pack_listing_id', cb.value);
        promises.push(
            fetch('/user/storefront/packs', { method: 'POST', body: fd })
            .then(function(r) { return r.json(); })
        );
    });
    Promise.all(promises).then(function(results) {
        var errors = results.filter(function(d) { return !d.success; });
        closeAddPackModal();
        if (errors.length === 0) {
            showMsg('ok', '分析包已添加');
        } else {
            showMsg('ok', '部分添加成功，' + errors.length + ' 个失败');
        }
        setTimeout(function() { location.reload(); }, 800);
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Packs: Remove pack ===== */
function removePack(listingId, packName) {
    if (!confirm('确定从小铺中移除"' + packName + '"？')) return;
    var fd = new FormData();
    fd.append('pack_listing_id', listingId);
    fetch('/user/storefront/packs/remove', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            showMsg('ok', '已移除');
            var el = document.getElementById('pack-item-' + listingId);
            if (el) el.remove();
        } else {
            showMsg('err', d.error || '移除失败');
        }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Featured: Select modal ===== */
function showFeaturedSelectModal() {
    var list = document.getElementById('featuredSelectList');
    list.innerHTML = '';
    // Determine source: if auto-add is on, use all author packs; otherwise use storefront packs
    var autoOn = document.getElementById('autoAddToggle').classList.contains('on');
    var packs = autoOn ? _authorPacks : _storefrontPacks;
    packs.forEach(function(p) {
        var isFeat = _featuredIds.indexOf(p.id) >= 0;
        var label = document.createElement('label');
        label.className = 'pack-select-item';
        label.innerHTML = '<input type="checkbox" class="feat-select-cb" value="' + p.id + '"' +
            (isFeat ? ' checked' : '') + '>' +
            '<span class="pack-select-item-name">' + p.name + '</span>' +
            '<span class="pack-select-item-mode">' + p.mode + '</span>';
        list.appendChild(label);
    });
    document.getElementById('featuredSelectModal').classList.add('show');
}
function closeFeaturedSelectModal() {
    document.getElementById('featuredSelectModal').classList.remove('show');
}
function confirmSetFeatured() {
    var cbs = document.querySelectorAll('.feat-select-cb:checked');
    if (cbs.length > 4) { showToast('最多设置 4 个推荐分析包'); return; }
    // First, remove all current featured, then set new ones
    var removePromises = _featuredIds.map(function(id) {
        var fd = new FormData();
        fd.append('pack_listing_id', id);
        fd.append('featured', '0');
        return fetch('/user/storefront/featured', { method: 'POST', body: fd })
            .then(function(r) { return r.json(); });
    });
    Promise.all(removePromises).then(function() {
        var setPromises = [];
        cbs.forEach(function(cb, idx) {
            var fd = new FormData();
            fd.append('pack_listing_id', cb.value);
            fd.append('featured', '1');
            fd.append('sort_order', idx + 1);
            setPromises.push(
                fetch('/user/storefront/featured', { method: 'POST', body: fd })
                .then(function(r) { return r.json(); })
            );
        });
        return Promise.all(setPromises);
    }).then(function(results) {
        var errors = results.filter(function(d) { return !d.success; });
        closeFeaturedSelectModal();
        if (errors.length === 0) {
            showMsg('ok', '推荐设置已更新');
        } else {
            showMsg('err', errors[0].error || '部分设置失败');
        }
        setTimeout(function() { location.reload(); }, 800);
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Featured: Remove single ===== */
function removeFeatured(listingId) {
    var fd = new FormData();
    fd.append('pack_listing_id', listingId);
    fd.append('featured', '0');
    fetch('/user/storefront/featured', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            showMsg('ok', '已取消推荐');
            setTimeout(function() { location.reload(); }, 800);
        } else {
            showMsg('err', d.error || '操作失败');
        }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Featured: Drag-and-drop reorder ===== */
(function() {
    var list = document.getElementById('featuredList');
    if (!list) return;
    var dragItem = null;
    list.addEventListener('dragstart', function(e) {
        dragItem = e.target.closest('.featured-item');
        if (dragItem) { dragItem.classList.add('dragging'); e.dataTransfer.effectAllowed = 'move'; }
    });
    list.addEventListener('dragend', function(e) {
        if (dragItem) { dragItem.classList.remove('dragging'); dragItem = null; }
    });
    list.addEventListener('dragover', function(e) {
        e.preventDefault();
        var afterEl = getDragAfterElement(list, e.clientY);
        if (dragItem) {
            if (afterEl == null) { list.appendChild(dragItem); }
            else { list.insertBefore(dragItem, afterEl); }
        }
    });
    function getDragAfterElement(container, y) {
        var els = Array.from(container.querySelectorAll('.featured-item:not(.dragging)'));
        var closest = null; var closestOffset = Number.NEGATIVE_INFINITY;
        els.forEach(function(el) {
            var box = el.getBoundingClientRect();
            var offset = y - box.top - box.height / 2;
            if (offset < 0 && offset > closestOffset) { closestOffset = offset; closest = el; }
        });
        return closest;
    }
})();

function saveFeaturedOrder() {
    var items = document.querySelectorAll('#featuredList .featured-item');
    var ids = [];
    items.forEach(function(el) { ids.push(parseInt(el.getAttribute('data-id'))); });
    fetch('/user/storefront/featured/reorder', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids: ids })
    })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) { showMsg('ok', '排序已保存'); }
        else { showMsg('err', d.error || '保存失败'); }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Notifications: Email editor toggle ===== */
function toggleEmailEditor() {
    var editor = document.getElementById('emailEditor');
    if (editor.classList.contains('show')) {
        editor.classList.remove('show');
    } else {
        editor.classList.add('show');
        loadRecipientCount();
    }
}

/* ===== Notifications: Scope change ===== */
function onScopeChange() {
    var scope = document.getElementById('recipientScope').value;
    var wrap = document.getElementById('partialPacksWrap');
    wrap.style.display = scope === 'partial' ? 'block' : 'none';
    loadRecipientCount();
}

/* ===== Notifications: Load recipient count ===== */
function loadRecipientCount() {
    var scope = document.getElementById('recipientScope').value;
    var url = '/user/storefront/notify/recipients?scope=' + scope;
    if (scope === 'partial') {
        var cbs = document.querySelectorAll('.notify-pack-cb:checked');
        var ids = [];
        cbs.forEach(function(cb) { ids.push(cb.value); });
        if (ids.length > 0) { url += '&listing_ids=' + ids.join(','); }
    }
    fetch(url)
    .then(function(r) { return r.json(); })
    .then(function(d) {
        var info = document.getElementById('recipientInfo');
        if (d.count !== undefined) {
            info.textContent = '📬 收件人：' + d.count + ' 人';
        }
    }).catch(function() {});
}
// Attach change listeners to partial pack checkboxes
document.querySelectorAll('.notify-pack-cb').forEach(function(cb) {
    cb.addEventListener('change', loadRecipientCount);
});

/* ===== Notifications: Apply template ===== */
function applyTemplate() {
    var sel = document.getElementById('notifyTemplate');
    var opt = sel.options[sel.selectedIndex];
    if (opt.value === '') {
        document.getElementById('notifySubject').value = '';
        document.getElementById('notifyBody').value = '';
        return;
    }
    var subject = opt.getAttribute('data-subject') || '';
    var body = opt.getAttribute('data-body') || '';
    // Replace store name placeholder
    var storeName = document.getElementById('storeName').value || '我的小铺';
    subject = subject.replace(/\{\{\.StoreName\}\}/g, storeName);
    body = body.replace(/\{\{\.StoreName\}\}/g, storeName);
    document.getElementById('notifySubject').value = subject;
    document.getElementById('notifyBody').value = body;
}

/* ===== Notifications: Send ===== */
function sendNotification() {
    var subject = document.getElementById('notifySubject').value.trim();
    var body = document.getElementById('notifyBody').value.trim();
    if (!subject || !body) {
        showMsg('err', '邮件主题和正文不能为空');
        return;
    }
    var scope = document.getElementById('recipientScope').value;
    var listingIds = [];
    if (scope === 'partial') {
        document.querySelectorAll('.notify-pack-cb:checked').forEach(function(cb) {
            listingIds.push(parseInt(cb.value));
        });
        if (listingIds.length === 0) {
            showMsg('err', '请选择至少一个分析包');
            return;
        }
    }
    var templateType = document.getElementById('notifyTemplate').value;
    if (!confirm('确定发送此邮件通知？')) return;
    var fd = new FormData();
    fd.append('subject', subject);
    fd.append('body', body);
    fd.append('scope', scope);
    fd.append('template_type', templateType);
    if (listingIds.length > 0) {
        fd.append('listing_ids', listingIds.join(','));
    }
    fetch('/user/storefront/notify', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            showMsg('ok', d.message || '邮件已发送');
            document.getElementById('emailEditor').classList.remove('show');
            document.getElementById('notifySubject').value = '';
            document.getElementById('notifyBody').value = '';
            setTimeout(function() { location.reload(); }, 1200);
        } else {
            showMsg('err', d.error || '发送失败');
        }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Notifications: Show detail ===== */
function showNotifyDetail(notifyId) {
    fetch('/user/storefront/notify/detail?id=' + notifyId)
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.subject) {
            document.getElementById('notifyDetailSubject').textContent = d.subject;
            document.getElementById('notifyDetailBody').textContent = d.body || '';
            document.getElementById('notifyDetailModal').classList.add('show');
        }
    }).catch(function() { showMsg('err', '加载失败'); });
}
function closeNotifyDetailModal() {
    document.getElementById('notifyDetailModal').classList.remove('show');
}

/* ===== Data for JS ===== */
var _authorPacks = [
    {{range .AuthorPacks}}
    { id: {{.ListingID}}, name: '{{.PackName}}', mode: '{{if eq .ShareMode "free"}}免费{{else if eq .ShareMode "per_use"}}按次{{else}}订阅{{end}}' },
    {{end}}
];
var _storefrontPacks = [
    {{range .StorefrontPacks}}
    { id: {{.ListingID}}, name: '{{.PackName}}', mode: '{{if eq .ShareMode "free"}}免费{{else if eq .ShareMode "per_use"}}按次{{else}}订阅{{end}}' },
    {{end}}
];
var _featuredIds = [
    {{range .FeaturedPacks}}{{.ListingID}},{{end}}
];
</script>
` + I18nJS + `
</body>
</html>`
