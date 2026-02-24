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
    <title data-i18n="storefront_manage">小铺管理 - 分析技能包市场</title>
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

        /* Featured pack logo preview */
        .featured-logo-preview {
            width: 36px; height: 36px; border-radius: 8px;
            overflow: hidden; flex-shrink: 0;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            display: flex; align-items: center; justify-content: center;
            box-shadow: 0 2px 8px rgba(99,102,241,0.2);
        }
        .featured-logo-img {
            width: 100%; height: 100%; object-fit: cover; display: block;
        }
        .featured-logo-preview svg { width: 18px; height: 18px; color: #fff; }
        .featured-logo-actions {
            display: flex; gap: 4px; flex-shrink: 0;
        }
        .featured-logo-btn {
            padding: 4px 10px; font-size: 11px; font-weight: 600;
            border-radius: 6px; border: 1px solid #e2e8f0;
            background: #fff; color: #64748b; cursor: pointer;
            transition: all 0.15s; font-family: inherit;
        }
        .featured-logo-btn:hover { background: #f1f5f9; color: #475569; }
        .featured-logo-btn-danger {
            color: #ef4444; border-color: #fecaca;
        }
        .featured-logo-btn-danger:hover { background: #fef2f2; color: #dc2626; }

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

        /* Layout option cards */
        .layout-option {
            padding: 14px; border-radius: 12px; border: 2px solid #e2e8f0;
            background: #fff; transition: all 0.25s ease; text-align: center;
            position: relative;
        }
        .layout-option:hover { border-color: #cbd5e1; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
        input[name="store_layout"]:checked + .layout-option { border-color: #4f46e5; box-shadow: 0 0 0 3px rgba(79,70,229,0.12); }
        .layout-option.switching {
            opacity: 0.6; pointer-events: none;
        }
        .layout-option.switching::after {
            content: ''; position: absolute; top: 50%; left: 50%;
            width: 20px; height: 20px; margin: -10px 0 0 -10px;
            border: 2px solid #e2e8f0; border-top-color: #4f46e5;
            border-radius: 50%; animation: layoutSpin 0.6s linear infinite;
        }
        @keyframes layoutSpin { to { transform: rotate(360deg); } }
        input[name="store_layout"]:checked + .layout-option .layout-name { color: #4f46e5; }
        input[name="store_layout"]:checked + .layout-option .layout-desc { color: #6366f1; }

        /* Decoration confirmation modal */
        .deco-modal-icon { font-size: 40px; text-align: center; margin-bottom: 16px; }
        .deco-modal-title { font-size: 17px; font-weight: 700; color: #1e293b; text-align: center; margin-bottom: 8px; }
        .deco-modal-desc { font-size: 14px; color: #64748b; text-align: center; line-height: 1.7; margin-bottom: 20px; }
        .deco-modal-fee {
            display: flex; align-items: center; justify-content: center; gap: 8px;
            padding: 14px 20px; background: linear-gradient(135deg, #fef3c7, #fde68a);
            border-radius: 10px; border: 1px solid #f59e0b; margin-bottom: 20px;
            font-size: 15px; font-weight: 700; color: #92400e;
        }
        .deco-modal-fee-free {
            background: linear-gradient(135deg, #dcfce7, #bbf7d0);
            border-color: #22c55e; color: #16a34a;
        }
        .deco-modal-actions { display: flex; gap: 10px; justify-content: center; }
        .deco-modal-actions .btn { min-width: 120px; justify-content: center; }
        .layout-preview {
            height: 60px; border-radius: 8px; border: 1px solid #e2e8f0;
            padding: 8px; margin-bottom: 10px; overflow: hidden;
        }
        .layout-name { font-size: 13px; font-weight: 700; color: #1e293b; margin-bottom: 2px; }
        .layout-desc { font-size: 11px; color: #94a3b8; }

        /* Page layout section editor */
        .section-list { display: flex; flex-direction: column; gap: 8px; }

        /* Theme selector */
        .theme-option {
            padding: 14px; border-radius: 12px; border: 2px solid #e2e8f0;
            background: #fff; cursor: pointer; transition: all 0.2s; text-align: center;
        }
        .theme-option:hover { border-color: #cbd5e1; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
        .theme-option-active { border-color: #4f46e5; box-shadow: 0 0 0 3px rgba(79,70,229,0.12); }
        .theme-swatches { display: flex; gap: 8px; justify-content: center; margin-bottom: 10px; }
        .theme-swatch {
            width: 32px; height: 32px; border-radius: 50%;
            box-shadow: 0 1px 3px rgba(0,0,0,0.15);
        }
        .theme-name { font-size: 13px; font-weight: 700; color: #1e293b; }
        .section-item {
            display: flex; align-items: center; gap: 12px;
            padding: 12px 14px; background: #f8fafc;
            border-radius: 10px; border: 1px solid #e2e8f0;
            cursor: grab; transition: box-shadow 0.15s, opacity 0.15s;
            user-select: none;
        }
        .section-item:active { cursor: grabbing; }
        .section-item.dragging { opacity: 0.4; box-shadow: 0 4px 16px rgba(0,0,0,0.1); }
        .section-item .drag-handle {
            color: #94a3b8; font-size: 16px; cursor: grab; user-select: none; flex-shrink: 0;
        }
        .section-item-body { flex: 1; min-width: 0; }
        .section-item-name { font-size: 14px; font-weight: 600; color: #1e293b; }
        .section-item-type { font-size: 11px; color: #94a3b8; margin-top: 1px; }
        .section-item-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
        .section-toggle {
            position: relative; width: 36px; height: 20px;
            background: #cbd5e1; border-radius: 10px;
            cursor: pointer; transition: background 0.2s;
            border: none; padding: 0; flex-shrink: 0;
        }
        .section-toggle.on { background: #4f46e5; }
        .section-toggle.disabled { opacity: 0.5; cursor: not-allowed; }
        .section-toggle::after {
            content: ''; position: absolute;
            top: 2px; left: 2px;
            width: 16px; height: 16px;
            background: #fff; border-radius: 50%;
            transition: transform 0.2s;
            box-shadow: 0 1px 2px rgba(0,0,0,0.15);
        }
        .section-toggle.on::after { transform: translateX(16px); }
        .section-columns-select {
            padding: 4px 8px; border: 1px solid #cbd5e1; border-radius: 6px;
            font-size: 12px; background: #fff; color: #1e293b; font-family: inherit;
        }
        .section-banner-settings {
            margin-top: 8px; padding: 10px 14px;
            background: #fff; border-radius: 8px; border: 1px solid #e2e8f0;
        }
        .section-banner-settings .field-group { margin-bottom: 8px; }
        .section-banner-settings .field-group:last-child { margin-bottom: 0; }
        .section-banner-settings label { font-size: 12px; }
        .section-banner-settings input,
        .section-banner-settings select {
            padding: 6px 10px; font-size: 13px;
        }
        .section-banner-text-counter {
            font-size: 11px; color: #94a3b8; text-align: right; margin-top: 2px;
        }
        .layout-actions {
            display: flex; gap: 10px; margin-top: 16px; align-items: center;
        }

        @media (max-width: 640px) {
            .tabs { overflow-x: auto; }
            .tab-btn { padding: 10px 16px; font-size: 13px; white-space: nowrap; }
            .slug-row { flex-direction: column; align-items: stretch; }
            .logo-upload-area { flex-direction: column; text-align: center; }
            .pack-item { flex-direction: column; align-items: stretch; }
            .pack-item-actions { align-self: flex-end; }
            .cp-item { flex-direction: column; align-items: stretch; }
            .cp-item .pack-item-actions { align-self: flex-end; margin-top: 8px; flex-wrap: wrap; }
        }

        /* Custom product drag list */
        .product-drag-list { display: flex; flex-direction: column; gap: 8px; }
        .cp-item {
            display: flex; align-items: center; gap: 12px;
            padding: 14px 16px; background: #f8fafc;
            border-radius: 10px; border: 1px solid #e2e8f0;
            cursor: grab; transition: box-shadow 0.15s, opacity 0.15s;
            user-select: none;
        }
        .cp-item:active { cursor: grabbing; }
        .cp-item.dragging { opacity: 0.4; box-shadow: 0 4px 16px rgba(0,0,0,0.1); }
    </style>
</head>
<body>
<div class="page">
    <!-- Navigation -->
    <nav class="nav">
        <a class="logo-link" href="/"><span class="logo-mark">📦</span><span class="logo-text" data-i18n="site_name">分析技能包市场</span></a>
        <div style="display:flex;gap:8px;">
            <a class="nav-link" href="/store/{{.Storefront.StoreSlug}}" target="_blank" data-i18n="sm_view_store">🔗 查看小铺</a>
            <a class="nav-link" href="/user/dashboard" data-i18n="personal_center">个人中心</a>
        </div>
    </nav>

    <h1 class="page-title">🏪 <span data-i18n="storefront_manage">小铺管理</span></h1>

    <!-- Messages -->
    <div class="msg msg-ok" id="successMsg"></div>
    <div class="msg msg-err" id="errorMsg"></div>

    <!-- Tabs -->
    <div class="tabs">
        <button class="tab-btn active" onclick="switchTab('settings', this)" data-i18n="sm_tab_settings">⚙️ 小铺设置</button>
        <button class="tab-btn" onclick="switchTab('packs', this)" data-i18n="sm_tab_packs">📦 分析包管理</button>
        <button class="tab-btn" onclick="switchTab('notify', this)" data-i18n="sm_tab_notify">📧 客户通知</button>
        {{if .CustomProductsEnabled}}<button class="tab-btn" onclick="switchTab('custom-products', this)" data-i18n="sm_tab_custom_products">🛍️ 自定义商品</button>{{end}}
    </div>

    <!-- ==================== Tab 1: 小铺设置 ==================== -->
    <div class="tab-content active" id="tab-settings">
        <!-- Store Name & Description -->
        <div class="card">
            <div class="card-title"><span class="icon">📝</span> <span data-i18n="sm_basic_info">基本信息</span></div>
            <div class="field-group">
                <label for="storeName" data-i18n="sm_store_name">小铺名称</label>
                <input type="text" id="storeName" value="{{.Storefront.StoreName}}" maxlength="30" data-i18n-placeholder="sm_store_name_ph" placeholder="输入小铺名称（2-30 字符）">
                <div class="field-hint" data-i18n="sm_store_name_hint">名称长度 2-30 字符</div>
            </div>
            <div class="field-group">
                <label for="storeDesc" data-i18n="sm_store_desc">小铺描述</label>
                <textarea id="storeDesc" rows="3" data-i18n-placeholder="sm_store_desc_ph" placeholder="介绍一下你的小铺...">{{.Storefront.Description}}</textarea>
            </div>
            <button class="btn btn-indigo" onclick="saveSettings()" data-i18n="sm_save_settings">💾 保存设置</button>
        </div>

        <!-- Logo Upload -->
        <div class="card">
            <div class="card-title"><span class="icon">🖼️</span> <span data-i18n="sm_logo_settings">Logo 设置</span></div>
            <div class="logo-upload-area">
                <div class="logo-preview" id="logoPreview">
                    {{if .Storefront.HasLogo}}
                    <img src="/store/{{.Storefront.StoreSlug}}/logo" alt="Logo" id="logoImg">
                    {{else}}
                    <span id="logoLetter">{{if .Storefront.StoreName}}{{slice .Storefront.StoreName 0 1}}{{else}}?{{end}}</span>
                    {{end}}
                </div>
                <div class="logo-upload-info">
                    <p data-i18n="sm_logo_hint">支持 PNG 或 JPEG 格式，文件大小不超过 2MB，也可直接 Ctrl+V 粘贴图片</p>
                    <input type="file" id="logoFile" accept="image/png,image/jpeg" style="display:none;" onchange="uploadLogo()">
                    <button class="btn btn-ghost" onclick="document.getElementById('logoFile').click()" data-i18n="sm_upload_logo">📤 上传 Logo</button>
                </div>
            </div>
        </div>

        <!-- Store Slug -->
        <div class="card">
            <div class="card-title"><span class="icon">🔗</span> <span data-i18n="sm_store_link">小铺链接</span></div>
            <div class="field-group">
                <label for="storeSlug" data-i18n="sm_store_slug">小铺标识（Store Slug）</label>
                <div class="slug-row">
                    <span class="slug-prefix">/store/</span>
                    <input type="text" id="storeSlug" value="{{.Storefront.StoreSlug}}" maxlength="50" placeholder="my-store">
                    <button class="btn btn-indigo btn-sm" onclick="updateSlug()" data-i18n="save">保存</button>
                </div>
                <div class="field-hint" data-i18n="sm_slug_hint">仅允许小写字母、数字和连字符，长度 3-50 字符</div>
            </div>
            <div class="url-display" id="fullUrlDisplay">{{.FullURL}}</div>
            <div style="margin-top:10px;">
                <button class="btn btn-ghost" onclick="copyStoreUrl()" data-i18n="sm_copy_link">📋 复制小铺链接</button>
            </div>
        </div>

        <!-- Layout Switcher -->
        <div class="card">
            <div class="card-title"><span class="icon">🎨</span> <span data-i18n="sm_layout">小铺布局</span></div>
            <div class="field-hint" style="margin-bottom:14px;" data-i18n="sm_layout_hint">选择小铺的展示风格，访客将看到对应的布局效果</div>
            <div style="display:flex;gap:12px;flex-wrap:wrap;" id="layoutOptions">
                <label style="flex:1;min-width:180px;cursor:pointer;">
                    <input type="radio" name="store_layout" value="default" {{if or (eq .Storefront.StoreLayout "default") (eq .Storefront.StoreLayout "")}}checked{{end}} style="display:none;" onchange="saveLayout('default')">
                    <div class="layout-option" id="layout-opt-default">
                        <div class="layout-preview" style="background:linear-gradient(135deg,#eef2ff 0%,#faf5ff 40%,#f0fdf4 100%);border-color:#e0e7ff;">
                            <div style="display:flex;gap:6px;align-items:center;margin-bottom:6px;">
                                <div style="width:20px;height:20px;border-radius:6px;background:linear-gradient(135deg,#6366f1,#8b5cf6);"></div>
                                <div style="flex:1;height:6px;background:#e0e7ff;border-radius:3px;"></div>
                            </div>
                            <div style="display:grid;grid-template-columns:1fr 1fr;gap:4px;">
                                <div style="height:16px;background:#fff;border-radius:4px;border:1px solid #e0e7ff;"></div>
                                <div style="height:16px;background:#fff;border-radius:4px;border:1px solid #e0e7ff;"></div>
                            </div>
                        </div>
                        <div class="layout-name" data-i18n="sm_layout_default">默认布局</div>
                        <div class="layout-desc" data-i18n="sm_layout_default_desc">经典靛蓝风格</div>
                    </div>
                </label>
                <label style="flex:1;min-width:180px;cursor:pointer;">
                    <input type="radio" name="store_layout" value="novelty" {{if eq .Storefront.StoreLayout "novelty"}}checked{{end}} style="display:none;" onchange="saveLayout('novelty')">
                    <div class="layout-option" id="layout-opt-novelty">
                        <div class="layout-preview" style="background:linear-gradient(160deg,#fdf6e3 0%,#fdf8ee 30%,#faf7f0 60%,#fdf6e3 100%);border-color:rgba(212,180,90,0.25);">
                            <div style="display:flex;gap:6px;align-items:center;margin-bottom:6px;">
                                <div style="width:20px;height:20px;border-radius:50%;background:linear-gradient(135deg,#d4b45a,#9a7a2e);"></div>
                                <div style="flex:1;height:6px;background:rgba(212,180,90,0.2);border-radius:3px;"></div>
                            </div>
                            <div style="display:grid;grid-template-columns:1fr 1fr;gap:4px;">
                                <div style="height:16px;background:rgba(255,255,255,0.85);border-radius:4px;border:1px solid rgba(212,180,90,0.25);transform:rotate(-1deg);"></div>
                                <div style="height:16px;background:rgba(255,255,255,0.85);border-radius:4px;border:1px solid rgba(212,180,90,0.25);transform:rotate(1deg);"></div>
                            </div>
                        </div>
                        <div class="layout-name" data-i18n="sm_layout_novelty">新奇布局</div>
                        <div class="layout-desc" data-i18n="sm_layout_novelty_desc">奢华金色风格</div>
                    </div>
                </label>
                <label style="flex:1;min-width:180px;cursor:pointer;">
                    <input type="radio" name="store_layout" value="custom" {{if eq .Storefront.StoreLayout "custom"}}checked{{end}} style="display:none;" onchange="saveLayout('custom')">
                    <div class="layout-option" id="layout-opt-custom">
                        <div class="layout-preview" style="background:#f8fafc;border-color:#e2e8f0;">
                            <div style="display:flex;align-items:center;justify-content:center;height:100%;color:#64748b;font-size:18px;">🎨</div>
                        </div>
                        <div class="layout-name" data-i18n="sm_layout_custom">自定义装修</div>
                        <div class="layout-desc" data-i18n="sm_layout_custom_desc">自由定制风格</div>
                    </div>
                </label>
            </div>
        </div>

        <!-- Page Layout Section Editor -->
        <div class="card">
            <div class="card-title"><span class="icon">📐</span> <span data-i18n="sm_page_layout">页面布局</span></div>
            <div class="field-hint" style="margin-bottom:14px;" data-i18n="sm_page_layout_hint">拖拽调整区块顺序，控制各区块的显示和参数设置</div>
            <div class="section-list" id="sectionList"></div>
            <div class="layout-actions">
                <button class="btn btn-green btn-sm" id="addBannerBtn" onclick="addCustomBanner()" data-i18n="sm_add_banner">+ 添加横幅</button>
                <button class="btn btn-indigo" onclick="savePageLayout()" data-i18n="sm_save_layout">💾 保存布局</button>
                <a class="btn btn-ghost btn-sm" href="/store/{{.Storefront.StoreSlug}}?preview=1" target="_blank" data-i18n="sm_preview">👁️ 预览</a>
            </div>
            <div id="layoutSaveMsg" class="msg" style="margin-top:12px;"></div>
        </div>

        <!-- Customer Support Section -->
        <div class="card">
            <div class="card-title"><span class="icon">🎧</span> 客户支持</div>
            {{if lt .TotalSales .SupportThreshold}}
            <!-- 未达标 -->
            <div style="display:flex;align-items:center;gap:8px;margin-bottom:12px;">
                <span class="tag" style="background:#f1f5f9;color:#64748b;border:1px solid #e2e8f0;">未达标</span>
            </div>
            <div style="font-size:13px;color:#64748b;line-height:1.7;">
                累计销售额达到 {{printf "%.0f" .SupportThreshold}} Credits 后可申请开通客户支持系统
            </div>
            <div style="font-size:12px;color:#94a3b8;margin-top:8px;">
                当前累计销售额：{{printf "%.0f" .TotalSales}} / {{printf "%.0f" .SupportThreshold}} Credits
            </div>
            {{else if eq .SupportStatus "none"}}
            <!-- 未开通 -->
            <div style="display:flex;align-items:center;gap:8px;margin-bottom:12px;">
                <span class="tag" style="background:#f1f5f9;color:#64748b;border:1px solid #e2e8f0;">未开通</span>
            </div>
            <div style="font-size:13px;color:#64748b;margin-bottom:14px;">您的店铺已达到开通门槛，可申请开通客户支持系统</div>
            <button class="btn btn-indigo" id="supportApplyBtn" onclick="applySupportSystem()">🎧 申请开通客户支持</button>
            {{else if eq .SupportStatus "pending"}}
            <!-- 审批中 -->
            <div style="display:flex;align-items:center;gap:8px;margin-bottom:12px;">
                <span class="tag" style="background:#fef3c7;color:#d97706;border:1px solid #fde68a;">审批中</span>
            </div>
            <div style="font-size:13px;color:#64748b;">您的开通请求正在等待管理员审批</div>
            {{else if eq .SupportStatus "approved"}}
            <!-- 已开通 -->
            <div style="display:flex;align-items:center;gap:8px;margin-bottom:12px;">
                <span class="tag" style="background:#dcfce7;color:#16a34a;border:1px solid #bbf7d0;">已开通</span>
            </div>
            <button class="btn btn-green" id="supportLoginBtn" onclick="loginSupportSystem()">🚀 进入客服后台</button>
            {{else if eq .SupportStatus "disabled"}}
            <!-- 已禁用 -->
            <div style="display:flex;align-items:center;gap:8px;margin-bottom:12px;">
                <span class="tag" style="background:#fee2e2;color:#dc2626;border:1px solid #fecaca;">已禁用</span>
            </div>
            <div style="font-size:13px;color:#dc2626;">禁用原因：{{.SupportDisableReason}}</div>
            {{end}}
        </div>
    </div>

    <!-- Theme Selector -->
    <div class="card">
        <div class="card-title"><span class="icon">🎨</span> 主题风格</div>
        <div class="field-hint" style="margin-bottom:14px;">选择小铺的配色主题，访客将看到对应的视觉风格</div>
        <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:12px;" id="themeOptions">
            <div class="theme-option{{if eq .CurrentTheme "default"}} theme-option-active{{end}}" data-theme="default" onclick="selectTheme('default')">
                <div class="theme-swatches">
                    <span class="theme-swatch" style="background:#6366f1;"></span>
                    <span class="theme-swatch" style="background:#8b5cf6;"></span>
                </div>
                <div class="theme-name">默认靛蓝</div>
            </div>
            <div class="theme-option{{if eq .CurrentTheme "ocean"}} theme-option-active{{end}}" data-theme="ocean" onclick="selectTheme('ocean')">
                <div class="theme-swatches">
                    <span class="theme-swatch" style="background:#0891b2;"></span>
                    <span class="theme-swatch" style="background:#06b6d4;"></span>
                </div>
                <div class="theme-name">海洋蓝绿</div>
            </div>
            <div class="theme-option{{if eq .CurrentTheme "sunset"}} theme-option-active{{end}}" data-theme="sunset" onclick="selectTheme('sunset')">
                <div class="theme-swatches">
                    <span class="theme-swatch" style="background:#ea580c;"></span>
                    <span class="theme-swatch" style="background:#f59e0b;"></span>
                </div>
                <div class="theme-name">日落暖橙</div>
            </div>
            <div class="theme-option{{if eq .CurrentTheme "forest"}} theme-option-active{{end}}" data-theme="forest" onclick="selectTheme('forest')">
                <div class="theme-swatches">
                    <span class="theme-swatch" style="background:#16a34a;"></span>
                    <span class="theme-swatch" style="background:#22c55e;"></span>
                </div>
                <div class="theme-name">森林绿</div>
            </div>
            <div class="theme-option{{if eq .CurrentTheme "minimal"}} theme-option-active{{end}}" data-theme="minimal" onclick="selectTheme('minimal')">
                <div class="theme-swatches">
                    <span class="theme-swatch" style="background:#475569;"></span>
                    <span class="theme-swatch" style="background:#64748b;"></span>
                </div>
                <div class="theme-name">极简灰白</div>
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
                    <div class="featured-logo-preview" id="featured-logo-preview-{{.ListingID}}">
                        {{if .HasLogo}}
                        <img class="featured-logo-img" src="/store/{{$.Storefront.StoreSlug}}/featured/{{.ListingID}}/logo" alt="{{.PackName}}">
                        {{else}}
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
                        {{end}}
                    </div>
                    <div class="featured-item-body">
                        <div class="featured-item-name">{{.PackName}}</div>
                        <div class="featured-item-price">
                            {{if eq .ShareMode "free"}}免费{{else}}{{.CreditsPrice}} Credits{{end}}
                        </div>
                    </div>
                    <div class="featured-logo-actions">
                        <button class="featured-logo-btn" onclick="uploadFeaturedLogo({{.ListingID}})">📤 上传 Logo</button>
                        <input type="file" id="featured-logo-input-{{.ListingID}}" accept="image/png,image/jpeg" style="display:none">
                        {{if .HasLogo}}<button class="featured-logo-btn featured-logo-btn-danger" id="featured-logo-delete-{{.ListingID}}" onclick="deleteFeaturedLogo({{.ListingID}})">🗑️ 删除</button>{{end}}
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

            <!-- Macro fields (shown when template selected) -->
            <div id="macroFieldsWrap" style="display:none;">
                <div style="font-weight:600;margin-bottom:8px;color:#4f46e5;">📝 请填写模板变量</div>
                <div id="macroFields"></div>
            </div>

            <!-- Subject -->
            <div class="field-group" id="subjectFieldGroup">
                <label for="notifySubject">邮件主题</label>
                <input type="text" id="notifySubject" placeholder="输入邮件主题">
            </div>

            <!-- Body -->
            <div class="field-group" id="bodyFieldGroup">
                <label for="notifyBody">邮件正文</label>
                <textarea id="notifyBody" rows="8" placeholder="输入邮件正文内容..."></textarea>
            </div>

            <!-- Preview (shown when template selected) -->
            <div id="emailPreviewWrap" style="display:none;">
                <div class="field-group">
                    <label>📧 邮件预览</label>
                    <div id="emailPreviewSubject" style="padding:8px 12px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:6px;margin-bottom:8px;font-weight:600;color:#1e293b;"></div>
                    <div id="emailPreviewBody" style="padding:12px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:6px;white-space:pre-wrap;color:#334155;line-height:1.6;min-height:80px;"></div>
                </div>
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

    {{if .CustomProductsEnabled}}
    <!-- ==================== Tab 4: 自定义商品 ==================== -->
    <div class="tab-content" id="tab-custom-products" data-testid="custom-products-tab">
        <!-- Product list -->
        <div class="card">
            <div class="card-title" style="justify-content:space-between;">
                <span><span class="icon">🛍️</span> 自定义商品列表</span>
                <div style="display:flex;gap:8px;">
                    <a class="btn btn-ghost btn-sm" href="/user/storefront/custom-product-orders" style="text-decoration:none;">📋 订单记录</a>
                    <button class="btn btn-green btn-sm" onclick="showCustomProductForm()">+ 添加商品</button>
                </div>
            </div>
            {{if .CustomProducts}}
            <div class="product-drag-list" id="customProductList">
                {{range .CustomProducts}}
                <div class="cp-item" draggable="true" data-cp-id="{{.ID}}">
                    <span class="drag-handle" style="color:#94a3b8;font-size:16px;cursor:grab;user-select:none;flex-shrink:0;">⠿</span>
                    <div class="pack-item-body" style="flex:1;min-width:0;">
                        <div class="pack-item-name">
                            {{if eq .ProductType "credits"}}<span class="tag" style="background:#dbeafe;color:#2563eb;border:1px solid #bfdbfe;">积分充值</span>{{end}}
                            {{if eq .ProductType "virtual_goods"}}<span class="tag" style="background:#f3e8ff;color:#7c3aed;border:1px solid #ddd6fe;">虚拟商品</span>{{end}}
                            {{.ProductName}}
                        </div>
                        <div class="pack-item-meta">
                            <span>$ {{printf "%.2f" .PriceUSD}}</span>
                            {{if eq .ProductType "credits"}}<span style="margin-left:8px;">{{.CreditsAmount}} 积分</span>{{end}}
                            <span style="margin-left:8px;display:inline-block;padding:2px 8px;border-radius:20px;font-size:11px;font-weight:700;{{if eq .Status "draft"}}background:#f1f5f9;color:#64748b;border:1px solid #e2e8f0;{{end}}{{if eq .Status "pending"}}background:#fef3c7;color:#d97706;border:1px solid #fde68a;{{end}}{{if eq .Status "published"}}background:#dcfce7;color:#16a34a;border:1px solid #bbf7d0;{{end}}{{if eq .Status "rejected"}}background:#fef2f2;color:#dc2626;border:1px solid #fecaca;{{end}}">
                                {{if eq .Status "draft"}}草稿{{end}}{{if eq .Status "pending"}}待审核{{end}}{{if eq .Status "published"}}已上架{{end}}{{if eq .Status "rejected"}}已拒绝{{end}}
                            </span>
                        </div>
                        {{if and (eq .Status "rejected") (ne .RejectReason "")}}
                        <div style="font-size:12px;color:#dc2626;margin-top:4px;">拒绝原因：{{.RejectReason}}</div>
                        {{end}}
                    </div>
                    <div class="pack-item-actions" style="display:flex;gap:6px;flex-shrink:0;">
                        <button class="btn btn-ghost btn-sm" onclick="editCustomProduct({{.ID}}, '{{.ProductName}}', '{{.Description}}', '{{.ProductType}}', {{.PriceUSD}}, {{.CreditsAmount}}, '{{.LicenseAPIEndpoint}}', '{{.LicenseAPIKey}}', '{{.LicenseProductID}}')">编辑</button>
                        {{if or (eq .Status "draft") (eq .Status "rejected")}}
                        <form method="POST" action="/user/storefront/custom-products/submit" style="display:inline;">
                            <input type="hidden" name="product_id" value="{{.ID}}">
                            <button type="submit" class="btn btn-indigo btn-sm">提交审核</button>
                        </form>
                        {{end}}
                        {{if eq .Status "published"}}
                        <form method="POST" action="/user/storefront/custom-products/delist" style="display:inline;">
                            <input type="hidden" name="product_id" value="{{.ID}}">
                            <button type="submit" class="btn btn-ghost btn-sm">下架</button>
                        </form>
                        {{end}}
                        <button class="btn btn-red btn-sm" onclick="deleteCustomProduct({{.ID}}, '{{.ProductName}}')">删除</button>
                    </div>
                </div>
                {{end}}
            </div>
            <div style="margin-top:12px;">
                <button class="btn btn-indigo btn-sm" onclick="saveCustomProductOrder()">💾 保存排序</button>
            </div>
            {{else}}
            <div class="empty-state"><div class="icon">🛍️</div><p>暂无自定义商品，点击上方按钮添加第一个商品</p></div>
            {{end}}
        </div>

        <!-- Create/Edit form -->
        <div class="card" id="cpFormCard" style="display:none;">
            <div class="card-title" id="cpFormTitle"><span class="icon">➕</span> 添加商品</div>
            <form id="cpForm" method="POST" action="/user/storefront/custom-products/create">
                <input type="hidden" id="cpEditId" name="product_id" value="">
                <div class="field-group">
                    <label for="cpName">商品名称 (2-100 字符)</label>
                    <input type="text" id="cpName" name="product_name" required minlength="2" maxlength="100" placeholder="输入商品名称">
                </div>
                <div class="field-group">
                    <label for="cpDesc">商品描述</label>
                    <textarea id="cpDesc" name="description" maxlength="1000" placeholder="输入商品描述（可选）"></textarea>
                </div>
                <div class="field-group">
                    <label for="cpType">商品类型</label>
                    <select id="cpType" name="product_type" onchange="toggleCPTypeFields()" required>
                        <option value="credits">积分充值</option>
                        <option value="virtual_goods">虚拟商品</option>
                    </select>
                </div>
                <div class="field-group">
                    <label for="cpPrice">价格 (USD, 最高 9999.99)</label>
                    <input type="text" id="cpPrice" name="price_usd" required placeholder="0.00">
                </div>
                <div id="cpCreditsFields">
                    <div class="field-group">
                        <label for="cpCreditsAmount">积分数量</label>
                        <input type="text" id="cpCreditsAmount" name="credits_amount" placeholder="购买后充值的积分数量">
                    </div>
                </div>
                <div id="cpVirtualFields" style="display:none;">
                    <div class="field-group">
                        <label for="cpLicenseEndpoint">License API 地址</label>
                        <input type="text" id="cpLicenseEndpoint" name="license_api_endpoint" placeholder="https://license.example.com/api/bind">
                    </div>
                    <div class="field-group">
                        <label for="cpLicenseKey">License API 密钥</label>
                        <input type="text" id="cpLicenseKey" name="license_api_key" placeholder="API 密钥">
                    </div>
                    <div class="field-group">
                        <label for="cpLicenseProductId">License 产品标识</label>
                        <input type="text" id="cpLicenseProductId" name="license_product_id" placeholder="产品标识 ID">
                    </div>
                </div>
                <div style="display:flex;gap:10px;">
                    <button type="submit" class="btn btn-indigo" id="cpSubmitBtn">创建商品</button>
                    <button type="button" class="btn btn-ghost" onclick="hideCPForm()">取消</button>
                </div>
            </form>
        </div>
    </div>
    {{end}}

    <!-- Footer -->
    <div class="foot">
        <p class="foot-text">Vantagics <span data-i18n="site_name">分析技能包市场</span> · <a href="/" data-i18n="browse_more">浏览更多</a></p>
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

<!-- Decoration Confirmation Modal -->
<div class="modal-overlay" id="decoConfirmModal">
    <div class="modal-box" style="max-width:420px;">
        <button class="modal-close" onclick="closeDecoModal()">✕</button>
        <div class="deco-modal-icon" id="decoModalIcon">🎨</div>
        <div class="deco-modal-title" id="decoModalTitle"></div>
        <div class="deco-modal-desc" id="decoModalDesc"></div>
        <div class="deco-modal-fee" id="decoModalFee" style="display:none;"></div>
        <div class="deco-modal-actions">
            <button class="btn btn-ghost" onclick="closeDecoModal()" id="decoModalCancel">取消</button>
            <button class="btn btn-indigo" onclick="confirmDecoAction()" id="decoModalConfirm">确认</button>
        </div>
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

/* ===== Settings: Save Layout ===== */
var _decorationSessionActive = false;
var _decoModalCallback = null;
var _decoModalIsLayoutSwitch = false;
var _previousActiveLayout = '';

function openDecoModal(opts) {
    var modal = document.getElementById('decoConfirmModal');
    document.getElementById('decoModalIcon').textContent = opts.icon || '🎨';
    document.getElementById('decoModalTitle').textContent = opts.title || '';
    document.getElementById('decoModalDesc').textContent = opts.desc || '';
    var feeEl = document.getElementById('decoModalFee');
    if (opts.feeText) {
        feeEl.textContent = opts.feeText;
        feeEl.style.display = '';
        feeEl.className = 'deco-modal-fee' + (opts.isFree ? ' deco-modal-fee-free' : '');
    } else {
        feeEl.style.display = 'none';
    }
    document.getElementById('decoModalConfirm').textContent = opts.confirmText || '确认';
    document.getElementById('decoModalCancel').textContent = opts.cancelText || '取消';
    _decoModalCallback = opts.onConfirm || null;
    _decoModalIsLayoutSwitch = !!opts.isLayoutSwitch;
    modal.classList.add('show');
}

function closeDecoModal() {
    document.getElementById('decoConfirmModal').classList.remove('show');
    _decoModalCallback = null;
    if (_decoModalIsLayoutSwitch) revertLayoutRadio();
    _decoModalIsLayoutSwitch = false;
}

function confirmDecoAction() {
    document.getElementById('decoConfirmModal').classList.remove('show');
    if (_decoModalCallback) { _decoModalCallback(); _decoModalCallback = null; }
}

function revertLayoutRadio() {
    if (_previousActiveLayout) {
        var radio = document.querySelector('input[name="store_layout"][value="' + _previousActiveLayout + '"]');
        if (radio) radio.checked = true;
    }
}

function setLayoutSwitching(layout, on) {
    var opt = document.getElementById('layout-opt-' + layout);
    if (opt) {
        if (on) opt.classList.add('switching');
        else opt.classList.remove('switching');
    }
}

function saveLayout(layout) {
    // Track previous layout for revert on failure
    var prevRadio = document.querySelector('input[name="store_layout"]:checked');
    if (prevRadio && prevRadio.value !== layout) _previousActiveLayout = prevRadio.value;

    if (layout === 'custom') {
        // Fetch decoration fee and show confirmation modal
        setLayoutSwitching('custom', true);
        fetch('/api/decoration-fee')
        .then(function(r) { return r.json(); })
        .then(function(d) {
            setLayoutSwitching('custom', false);
            var fee = parseInt(d.fee || '0', 10);
            var isFree = fee === 0;
            openDecoModal({
                icon: '🎨',
                isLayoutSwitch: true,
                title: isFree
                    ? (T('decoration_free_confirm_title') || '开始自定义装修')
                    : (T('decoration_fee_confirm_title') || '自定义装修需要付费'),
                desc: isFree
                    ? (T('decoration_free_confirm') || '当前自定义装修免费，确定要开始装修吗？')
                    : (T('decoration_fee_confirm') || '启用自定义装修将收取 {fee} Credits 的装修费用，费用将在您发布装修后扣除。确定要开始装修吗？').replace('{fee}', fee),
                feeText: isFree
                    ? (T('decoration_fee_free_label') || '🎉 免费装修')
                    : ('💰 ' + fee + ' Credits'),
                isFree: isFree,
                confirmText: isFree
                    ? (T('decoration_start_btn') || '开始装修')
                    : (T('decoration_start_paid_btn') || '确认并开始装修'),
                cancelText: T('cancel') || '取消',
                onConfirm: function() { doSwitchToCustom(fee); }
            });
        }).catch(function() {
            setLayoutSwitching('custom', false);
            revertLayoutRadio();
            showMsg('err', T('network_error') || '网络错误');
        });
        return;
    }
    // Non-custom layout: save directly with loading state
    _decorationSessionActive = false;
    hidePublishDecorationBtn();
    setLayoutSwitching(layout, true);
    var fd = new FormData();
    fd.append('layout', layout);
    fetch('/user/storefront/layout', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        setLayoutSwitching(layout, false);
        if (d.success) {
            showToast(T('layout_switched') || '布局已切换');
        } else {
            revertLayoutRadio();
            showMsg('err', d.error || T('save_failed') || '保存失败');
        }
    }).catch(function() {
        setLayoutSwitching(layout, false);
        revertLayoutRadio();
        showMsg('err', T('network_error') || '网络错误');
    });
}

function doSwitchToCustom(fee) {
    _decorationSessionActive = true;
    setLayoutSwitching('custom', true);
    var fd = new FormData();
    fd.append('layout', 'custom');
    fetch('/user/storefront/layout', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d2) {
        setLayoutSwitching('custom', false);
        if (d2.success) {
            showToast(T('decoration_started') || '装修模式已开启，完成后请点击发布');
            // Ensure radio is on custom (CSS :checked handles highlight)
            var customRadio = document.querySelector('input[name="store_layout"][value="custom"]');
            if (customRadio) customRadio.checked = true;
            showPublishDecorationBtn(fee);
        } else {
            revertLayoutRadio();
            showMsg('err', d2.error || T('save_failed') || '保存失败');
        }
    }).catch(function() {
        setLayoutSwitching('custom', false);
        revertLayoutRadio();
        showMsg('err', T('network_error') || '网络错误');
    });
}

function showPublishDecorationBtn(fee) {
    var existing = document.getElementById('publishDecorationBar');
    if (existing) { existing.style.display = ''; return; }
    var bar = document.createElement('div');
    bar.id = 'publishDecorationBar';
    bar.style.cssText = 'position:sticky;bottom:0;left:0;right:0;background:linear-gradient(135deg,#fef3c7,#fde68a);border-top:2px solid #f59e0b;padding:16px 24px;display:flex;align-items:center;justify-content:space-between;z-index:100;border-radius:12px 12px 0 0;margin-top:20px;box-shadow:0 -4px 16px rgba(0,0,0,0.1);';
    var feeText = fee > 0
        ? (T('decoration_publish_hint') || '装修完成后点击发布，将扣除 {fee} Credits').replace('{fee}', fee)
        : (T('decoration_publish_hint_free') || '装修完成后点击发布即可生效');
    bar.innerHTML = '<div style="display:flex;align-items:center;gap:8px;"><span style="font-size:20px;">🎨</span><span style="font-size:14px;font-weight:600;color:#92400e;">' + feeText + '</span></div>'
        + '<button onclick="publishDecoration()" style="padding:10px 28px;background:linear-gradient(135deg,#f59e0b,#d97706);color:#fff;border:none;border-radius:8px;font-size:14px;font-weight:700;cursor:pointer;box-shadow:0 2px 8px rgba(245,158,11,0.3);transition:all 0.2s;" onmouseover="this.style.transform=\'translateY(-1px)\'" onmouseout="this.style.transform=\'\'">' + (T('decoration_publish_btn') || '🚀 发布装修') + '</button>';
    document.querySelector('.page') ? document.querySelector('.page').appendChild(bar) : document.body.appendChild(bar);
}

function hidePublishDecorationBtn() {
    var bar = document.getElementById('publishDecorationBar');
    if (bar) bar.style.display = 'none';
}

function publishDecoration() {
    openDecoModal({
        icon: '🚀',
        title: T('decoration_publish_confirm_title') || '确认发布装修',
        desc: T('decoration_publish_confirm') || '一旦发布，此次装修即完成，费用将被扣除。确定发布吗？',
        confirmText: T('decoration_publish_btn') || '🚀 发布装修',
        cancelText: T('cancel') || '取消',
        onConfirm: function() { doPublishDecoration(); }
    });
}

function doPublishDecoration() {
    fetch('/user/storefront/decoration/publish', { method: 'POST' })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.ok) {
            _decorationSessionActive = false;
            hidePublishDecorationBtn();
            var charged = d.fee_charged || 0;
            var successMsg = charged > 0
                ? (T('decoration_published_charged') || '装修已发布，已扣除 {fee} Credits').replace('{fee}', charged)
                : (T('decoration_published_free') || '装修已发布');
            showToast(successMsg);
        } else if (d.error === 'insufficient_balance') {
            showMsg('err', T('decoration_insufficient_balance') || 'Credits 余额不足，无法发布装修');
        } else {
            showMsg('err', d.error || T('save_failed') || '发布失败');
        }
    }).catch(function() { showMsg('err', T('network_error') || '网络错误'); });
}

/* ===== Settings: Select Theme ===== */
function selectTheme(theme) {
    var fd = new FormData();
    fd.append('theme', theme);
    fetch('/user/storefront/theme', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.ok) {
            showToast('主题已切换');
            document.querySelectorAll('.theme-option').forEach(function(el) { el.classList.remove('theme-option-active'); });
            var opt = document.querySelector('.theme-option[data-theme="' + theme + '"]');
            if (opt) opt.classList.add('theme-option-active');
        } else {
            showMsg('err', d.error || '保存失败');
        }
    }).catch(function() { showMsg('err', '网络错误'); });
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
            '<span class="pack-select-item-name">' + escapeAttr(p.name) + '</span>' +
            '<span class="pack-select-item-mode">' + escapeAttr(p.mode) + '</span>';
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

/* ===== Featured: Upload Logo ===== */
function uploadFeaturedLogo(listingId) {
    var input = document.getElementById('featured-logo-input-' + listingId);
    if (!input) return;
    input.onchange = function() {
        if (!input.files.length) return;
        var file = input.files[0];
        var fd = new FormData();
        fd.append('pack_listing_id', listingId);
        fd.append('logo', file);
        fetch('/user/storefront/featured/logo', { method: 'POST', body: fd })
        .then(function(r) { return r.json(); })
        .then(function(d) {
            if (d.success) {
                var preview = document.getElementById('featured-logo-preview-' + listingId);
                if (preview) {
                    preview.innerHTML = '<img class="featured-logo-img" src="/store/{{.Storefront.StoreSlug}}/featured/' + listingId + '/logo?t=' + Date.now() + '" alt="Logo">';
                }
                var delBtn = document.getElementById('featured-logo-delete-' + listingId);
                if (!delBtn) {
                    var actions = input.parentElement;
                    var btn = document.createElement('button');
                    btn.className = 'featured-logo-btn featured-logo-btn-danger';
                    btn.id = 'featured-logo-delete-' + listingId;
                    btn.onclick = function() { deleteFeaturedLogo(listingId); };
                    btn.textContent = '🗑️ 删除';
                    actions.appendChild(btn);
                }
                showToast('Logo 已更新');
            } else {
                alert(d.error || '上传失败');
            }
        }).catch(function() { alert('网络错误'); });
        input.value = '';
    };
    input.click();
}

/* ===== Featured: Delete Logo ===== */
function deleteFeaturedLogo(listingId) {
    var fd = new FormData();
    fd.append('pack_listing_id', listingId);
    fetch('/user/storefront/featured/logo/delete', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            var preview = document.getElementById('featured-logo-preview-' + listingId);
            if (preview) {
                preview.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>';
            }
            var delBtn = document.getElementById('featured-logo-delete-' + listingId);
            if (delBtn) { delBtn.remove(); }
            showToast('Logo 已删除');
        } else {
            alert(d.error || '删除失败');
        }
    }).catch(function() { alert('网络错误'); });
}

/* ===== Featured: Clipboard Paste Upload ===== */
var lastInteractedFeaturedPack = null;

document.addEventListener('DOMContentLoaded', function() {
    var items = document.querySelectorAll('.featured-item');
    items.forEach(function(el) {
        var id = el.getAttribute('data-id');
        el.addEventListener('mouseenter', function() { lastInteractedFeaturedPack = id; });
        el.addEventListener('click', function() { lastInteractedFeaturedPack = id; });
    });
});

document.addEventListener('paste', function(e) {
    if (!lastInteractedFeaturedPack) return;
    var items = e.clipboardData && e.clipboardData.items;
    if (!items) return;
    for (var i = 0; i < items.length; i++) {
        if (items[i].type === 'image/png' || items[i].type === 'image/jpeg') {
            var file = items[i].getAsFile();
            if (!file) continue;
            var fd = new FormData();
            var targetListingId = lastInteractedFeaturedPack;
            fd.append('pack_listing_id', targetListingId);
            fd.append('logo', file);
            fetch('/user/storefront/featured/logo', { method: 'POST', body: fd })
            .then(function(r) { return r.json(); })
            .then(function(d) {
                if (d.success) {
                    var preview = document.getElementById('featured-logo-preview-' + targetListingId);
                    if (preview) {
                        preview.innerHTML = '<img class="featured-logo-img" src="/store/{{.Storefront.StoreSlug}}/featured/' + targetListingId + '/logo?t=' + Date.now() + '" alt="Logo">';
                    }
                    var delBtn = document.getElementById('featured-logo-delete-' + targetListingId);
                    if (!delBtn) {
                        var input = document.getElementById('featured-logo-input-' + targetListingId);
                        if (input) {
                            var actions = input.parentElement;
                            var btn = document.createElement('button');
                            btn.className = 'featured-logo-btn featured-logo-btn-danger';
                            btn.id = 'featured-logo-delete-' + targetListingId;
                            btn.onclick = function() { deleteFeaturedLogo(targetListingId); };
                            btn.textContent = '🗑️ 删除';
                            actions.appendChild(btn);
                        }
                    }
                    showToast('Logo 已通过粘贴更新');
                } else {
                    alert(d.error || '上传失败');
                }
            }).catch(function() { alert('网络错误'); });
            e.preventDefault();
            return;
        }
    }
});

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
            info.textContent = '📬 收件人：' + d.count + ' 人（需消耗 ' + d.count + ' credits）';
        }
    }).catch(function() {});
}
// Attach change listeners to partial pack checkboxes
document.querySelectorAll('.notify-pack-cb').forEach(function(cb) {
    cb.addEventListener('change', loadRecipientCount);
});

/* ===== Notifications: Macro label map ===== */
var macroLabels = {
    'Version': '版本号',
    'UpdateContent': '更新内容',
    'PromoInfo': '促销信息',
    'HolidayName': '节日名称',
    'PromoTime': '活动时间',
    'PromoContent': '优惠内容',
    'PromoReason': '促销原因'
};

/* ===== Notifications: Apply template ===== */
var _tplSubject = '';
var _tplBody = '';
var _tplMacros = [];

function applyTemplate() {
    var sel = document.getElementById('notifyTemplate');
    var opt = sel.options[sel.selectedIndex];
    var macroWrap = document.getElementById('macroFieldsWrap');
    var macroContainer = document.getElementById('macroFields');
    var previewWrap = document.getElementById('emailPreviewWrap');
    var subjectEl = document.getElementById('notifySubject');
    var bodyEl = document.getElementById('notifyBody');

    if (opt.value === '') {
        _tplSubject = ''; _tplBody = ''; _tplMacros = [];
        macroWrap.style.display = 'none';
        macroContainer.innerHTML = '';
        previewWrap.style.display = 'none';
        subjectEl.value = ''; subjectEl.readOnly = false;
        bodyEl.value = ''; bodyEl.readOnly = false;
        document.getElementById('subjectFieldGroup').style.display = '';
        document.getElementById('bodyFieldGroup').style.display = '';
        return;
    }
    var subject = opt.getAttribute('data-subject') || '';
    var body = opt.getAttribute('data-body') || '';
    var storeName = document.getElementById('storeName').value || '我的小铺';
    subject = subject.replace(/\{\{\.StoreName\}\}/g, storeName);
    body = body.replace(/\{\{\.StoreName\}\}/g, storeName);
    _tplSubject = subject;
    _tplBody = body;

    // Extract macros (excluding StoreName which is already replaced)
    var allText = subject + ' ' + body;
    var re = /\{\{\.([\w]+)\}\}/g;
    var found = {};
    _tplMacros = [];
    var m;
    while ((m = re.exec(allText)) !== null) {
        if (!found[m[1]]) {
            found[m[1]] = true;
            _tplMacros.push(m[1]);
        }
    }

    // Build macro input fields
    macroContainer.innerHTML = '';
    if (_tplMacros.length > 0) {
        _tplMacros.forEach(function(macro) {
            var label = macroLabels[macro] || macro;
            var div = document.createElement('div');
            div.className = 'field-group';
            div.innerHTML = '<label>' + label + '</label>' +
                '<input type="text" id="macro_' + macro + '" placeholder="请输入' + label + '" oninput="updateEmailPreview()">';
            macroContainer.appendChild(div);
        });
        macroWrap.style.display = '';
    } else {
        macroWrap.style.display = 'none';
    }

    // Hide direct subject/body editing, show preview
    document.getElementById('subjectFieldGroup').style.display = 'none';
    document.getElementById('bodyFieldGroup').style.display = 'none';
    previewWrap.style.display = '';
    updateEmailPreview();
}

function updateEmailPreview() {
    var subject = _tplSubject;
    var body = _tplBody;
    _tplMacros.forEach(function(macro) {
        var input = document.getElementById('macro_' + macro);
        var val = input ? input.value : '';
        var display = val || ('{' + '{.' + macro + '}' + '}');
        var re = new RegExp('\\{\\{\\.' + macro + '\\}\\}', 'g');
        subject = subject.replace(re, display);
        body = body.replace(re, display);
    });
    document.getElementById('emailPreviewSubject').textContent = subject;
    document.getElementById('emailPreviewBody').textContent = body;
}

/* ===== Notifications: Send ===== */
function sendNotification() {
    var templateType = document.getElementById('notifyTemplate').value;
    var subject, body;

    if (templateType && _tplSubject) {
        // Template mode: replace macros with actual values
        subject = _tplSubject;
        body = _tplBody;
        var missing = [];
        _tplMacros.forEach(function(macro) {
            var input = document.getElementById('macro_' + macro);
            var val = input ? input.value.trim() : '';
            if (!val) {
                missing.push(macroLabels[macro] || macro);
            }
            var re = new RegExp('\\{\\{\\.' + macro + '\\}\\}', 'g');
            subject = subject.replace(re, val);
            body = body.replace(re, val);
        });
        if (missing.length > 0) {
            showMsg('err', '请填写以下模板变量：' + missing.join('、'));
            return;
        }
    } else {
        // Free-form mode
        subject = document.getElementById('notifySubject').value.trim();
        body = document.getElementById('notifyBody').value.trim();
    }

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
    .then(function(r) { return r.json().then(function(d) { return {ok: r.ok, data: d}; }); })
    .then(function(res) {
        if (res.ok && res.data.success) {
            showMsg('ok', res.data.message || '邮件已发送');
            document.getElementById('emailEditor').classList.remove('show');
            document.getElementById('notifySubject').value = '';
            document.getElementById('notifyBody').value = '';
            setTimeout(function() { location.reload(); }, 1200);
        } else {
            showMsg('err', res.data.error || '发送失败');
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

/* ===== Page Layout Section Editor ===== */
var _sectionTypeNames = {
    'hero': '店铺头部',
    'featured': '推荐分析包',
    'filter_bar': '筛选栏',
    'pack_grid': '分析包网格',
    'custom_banner': '自定义横幅'
};
var _sectionTypeIcons = {
    'hero': '🏠',
    'featured': '⭐',
    'filter_bar': '🔍',
    'pack_grid': '📦',
    'custom_banner': '📢'
};
var _defaultSections = [
    { type: 'hero', visible: true, settings: {} },
    { type: 'featured', visible: true, settings: {} },
    { type: 'filter_bar', visible: true, settings: {} },
    { type: 'pack_grid', visible: true, settings: { columns: 2 } }
];

var _layoutSections = [];

function initLayoutSections() {
    var raw = '{{.LayoutSectionsJSON}}';
    if (raw && raw !== '' && raw !== '&lt;no value&gt;') {
        try {
            // Decode HTML entities that Go template may produce
            var txt = raw.replace(/&amp;/g,'&').replace(/&lt;/g,'<').replace(/&gt;/g,'>').replace(/&#34;/g,'"').replace(/&#39;/g,"'").replace(/&quot;/g,'"');
            var parsed = JSON.parse(txt);
            if (parsed && parsed.sections && parsed.sections.length > 0) {
                _layoutSections = parsed.sections.map(function(s) {
                    var settings = s.settings || {};
                    if (typeof settings === 'string') {
                        try { settings = JSON.parse(settings); } catch(e) { settings = {}; }
                    }
                    return { type: s.type, visible: s.visible, settings: settings };
                });
            } else {
                _layoutSections = JSON.parse(JSON.stringify(_defaultSections));
            }
        } catch(e) {
            _layoutSections = JSON.parse(JSON.stringify(_defaultSections));
        }
    } else {
        _layoutSections = JSON.parse(JSON.stringify(_defaultSections));
    }
    renderSectionList();
}

function renderSectionList() {
    var list = document.getElementById('sectionList');
    if (!list) return;
    list.innerHTML = '';
    var bannerCount = 0;
    _layoutSections.forEach(function(sec) {
        if (sec.type === 'custom_banner') bannerCount++;
    });
    _layoutSections.forEach(function(sec, idx) {
        var item = document.createElement('div');
        item.className = 'section-item';
        item.draggable = true;
        item.setAttribute('data-idx', idx);

        var isRequired = (sec.type === 'hero' || sec.type === 'pack_grid');
        var typeName = _sectionTypeNames[sec.type] || sec.type;
        var typeIcon = _sectionTypeIcons[sec.type] || '📄';

        var html = '<span class="drag-handle">⠿</span>';
        html += '<div class="section-item-body">';
        html += '<div class="section-item-name">' + typeIcon + ' ' + typeName + '</div>';
        html += '<div class="section-item-type">' + sec.type + '</div>';

        // Hero layout setting
        if (sec.type === 'hero') {
            var heroLayout = (sec.settings && sec.settings.hero_layout) || 'default';
            html += '<div style="margin-top:6px;display:flex;align-items:center;gap:6px;">';
            html += '<span style="font-size:12px;color:#64748b;">布局:</span>';
            html += '<select class="section-columns-select" onchange="updateHeroLayout(' + idx + ', this.value)">';
            html += '<option value="default"' + (heroLayout === 'default' ? ' selected' : '') + '>Logo 在左，推荐在右</option>';
            html += '<option value="reversed"' + (heroLayout === 'reversed' ? ' selected' : '') + '>推荐在左，Logo 在右</option>';
            html += '</select></div>';
        }

        // Pack grid columns setting
        if (sec.type === 'pack_grid') {
            var cols = (sec.settings && sec.settings.columns) || 2;
            html += '<div style="margin-top:6px;display:flex;align-items:center;gap:6px;">';
            html += '<span style="font-size:12px;color:#64748b;">列数:</span>';
            html += '<select class="section-columns-select" onchange="updatePackGridColumns(' + idx + ', this.value)">';
            html += '<option value="1"' + (cols === 1 ? ' selected' : '') + '>1 列</option>';
            html += '<option value="2"' + (cols === 2 ? ' selected' : '') + '>2 列</option>';
            html += '<option value="3"' + (cols === 3 ? ' selected' : '') + '>3 列</option>';
            html += '</select></div>';
        }

        // Custom banner settings
        if (sec.type === 'custom_banner') {
            var text = (sec.settings && sec.settings.text) || '';
            var style = (sec.settings && sec.settings.style) || 'info';
            html += '<div class="section-banner-settings">';
            html += '<div class="field-group"><label>横幅文本</label>';
            html += '<input type="text" maxlength="200" value="' + escapeAttr(text) + '" oninput="updateBannerText(' + idx + ', this.value)" placeholder="输入横幅文本（最多 200 字符）">';
            html += '<div class="section-banner-text-counter"><span id="bannerCounter' + idx + '">' + text.length + '</span>/200</div>';
            html += '</div>';
            html += '<div class="field-group"><label>样式</label>';
            html += '<select onchange="updateBannerStyle(' + idx + ', this.value)">';
            html += '<option value="info"' + (style === 'info' ? ' selected' : '') + '>信息（蓝色）</option>';
            html += '<option value="success"' + (style === 'success' ? ' selected' : '') + '>成功（绿色）</option>';
            html += '<option value="warning"' + (style === 'warning' ? ' selected' : '') + '>警告（橙色）</option>';
            html += '</select></div>';
            html += '</div>';
        }

        html += '</div>'; // close section-item-body

        // Actions: visibility toggle + delete for banners
        html += '<div class="section-item-actions">';
        var toggleClass = 'section-toggle' + (sec.visible ? ' on' : '') + (isRequired ? ' disabled' : '');
        html += '<button class="' + toggleClass + '" title="' + (sec.visible ? '显示中' : '已隐藏') + '"' +
            (isRequired ? ' disabled' : ' onclick="toggleSectionVisibility(' + idx + ')"') + '></button>';
        if (sec.type === 'custom_banner') {
            html += '<button class="btn btn-red btn-sm" onclick="removeCustomBanner(' + idx + ')">删除</button>';
        }
        html += '</div>';

        item.innerHTML = html;
        list.appendChild(item);
    });

    // Update add banner button state
    var addBtn = document.getElementById('addBannerBtn');
    if (addBtn) {
        if (bannerCount >= 3) {
            addBtn.disabled = true;
            addBtn.title = '最多添加 3 个自定义横幅';
        } else {
            addBtn.disabled = false;
            addBtn.title = '';
        }
    }

    // Re-attach drag events
    initSectionDragDrop();
}

function escapeAttr(str) {
    return str.replace(/&/g,'&amp;').replace(/"/g,'&quot;').replace(/'/g,'&#39;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

function toggleSectionVisibility(idx) {
    if (idx < 0 || idx >= _layoutSections.length) return;
    var sec = _layoutSections[idx];
    if (sec.type === 'hero' || sec.type === 'pack_grid') return;
    sec.visible = !sec.visible;
    renderSectionList();
}

function updatePackGridColumns(idx, val) {
    if (idx < 0 || idx >= _layoutSections.length) return;
    var cols = parseInt(val);
    if (cols < 1 || cols > 3) cols = 2;
    if (!_layoutSections[idx].settings) _layoutSections[idx].settings = {};
    _layoutSections[idx].settings.columns = cols;
}

function updateHeroLayout(idx, val) {
    if (idx < 0 || idx >= _layoutSections.length) return;
    if (!_layoutSections[idx].settings) _layoutSections[idx].settings = {};
    _layoutSections[idx].settings.hero_layout = val;
}

function updateBannerText(idx, val) {
    if (idx < 0 || idx >= _layoutSections.length) return;
    if (val.length > 200) val = val.substring(0, 200);
    if (!_layoutSections[idx].settings) _layoutSections[idx].settings = {};
    _layoutSections[idx].settings.text = val;
    var counter = document.getElementById('bannerCounter' + idx);
    if (counter) counter.textContent = val.length;
}

function updateBannerStyle(idx, val) {
    if (idx < 0 || idx >= _layoutSections.length) return;
    if (!_layoutSections[idx].settings) _layoutSections[idx].settings = {};
    _layoutSections[idx].settings.style = val;
}

function addCustomBanner() {
    var bannerCount = 0;
    _layoutSections.forEach(function(s) { if (s.type === 'custom_banner') bannerCount++; });
    if (bannerCount >= 3) {
        showToast('最多添加 3 个自定义横幅');
        return;
    }
    _layoutSections.push({
        type: 'custom_banner',
        visible: true,
        settings: { text: '', style: 'info' }
    });
    renderSectionList();
}

function removeCustomBanner(idx) {
    if (idx < 0 || idx >= _layoutSections.length) return;
    if (_layoutSections[idx].type !== 'custom_banner') return;
    _layoutSections.splice(idx, 1);
    renderSectionList();
}

function savePageLayout() {
    // Build the layout config JSON
    var sections = _layoutSections.map(function(sec) {
        var s = { type: sec.type, visible: sec.visible, settings: sec.settings || {} };
        return s;
    });
    var config = JSON.stringify({ sections: sections });

    var fd = new FormData();
    fd.append('layout_config', config);
    fetch('/user/storefront/layout', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.ok) {
            showToast('布局已保存');
            // Server auto-switches to custom layout when saving layout_config
            if (d.layout_switched) {
                var customRadio = document.querySelector('input[name="store_layout"][value="custom"]');
                if (customRadio) customRadio.checked = true;
            }
        } else {
            showMsg('err', d.error || '保存失败');
        }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Section Drag and Drop ===== */
function initSectionDragDrop() {
    var list = document.getElementById('sectionList');
    if (!list) return;
    var dragItem = null;
    var dragIdx = -1;

    list.addEventListener('dragstart', function(e) {
        dragItem = e.target.closest('.section-item');
        if (dragItem) {
            dragIdx = parseInt(dragItem.getAttribute('data-idx'));
            dragItem.classList.add('dragging');
            e.dataTransfer.effectAllowed = 'move';
            e.dataTransfer.setData('text/plain', dragIdx);
        }
    });
    list.addEventListener('dragend', function(e) {
        if (dragItem) {
            dragItem.classList.remove('dragging');
            dragItem = null;
            dragIdx = -1;
        }
    });
    list.addEventListener('dragover', function(e) {
        e.preventDefault();
        if (!dragItem) return;
        var afterEl = getSectionDragAfter(list, e.clientY);
        if (afterEl == null) {
            list.appendChild(dragItem);
        } else {
            list.insertBefore(dragItem, afterEl);
        }
    });
    list.addEventListener('drop', function(e) {
        e.preventDefault();
        if (dragItem == null || dragIdx < 0) return;
        // Recompute order from DOM
        var items = list.querySelectorAll('.section-item');
        var newSections = [];
        items.forEach(function(el) {
            var i = parseInt(el.getAttribute('data-idx'));
            if (i >= 0 && i < _layoutSections.length) {
                newSections.push(_layoutSections[i]);
            }
        });
        if (newSections.length === _layoutSections.length) {
            _layoutSections = newSections;
            renderSectionList();
        }
    });
}

function getSectionDragAfter(container, y) {
    var els = Array.from(container.querySelectorAll('.section-item:not(.dragging)'));
    var closest = null;
    var closestOffset = Number.NEGATIVE_INFINITY;
    els.forEach(function(el) {
        var box = el.getBoundingClientRect();
        var offset = y - box.top - box.height / 2;
        if (offset < 0 && offset > closestOffset) {
            closestOffset = offset;
            closest = el;
        }
    });
    return closest;
}

// Initialize layout sections on page load
document.addEventListener('DOMContentLoaded', function() {
    initLayoutSections();
    initCustomProductDragDrop();
    // If already in custom layout, show publish decoration bar
    var currentLayout = '{{.Storefront.StoreLayout}}';
    if (currentLayout === 'custom') {
        var fee = parseInt('{{.DecorationFee}}' || '0', 10);
        _decorationSessionActive = true;
        showPublishDecorationBtn(fee);
    }
});

/* ===== Custom Products: Form management ===== */
function showCustomProductForm() {
    var card = document.getElementById('cpFormCard');
    card.style.display = 'block';
    document.getElementById('cpFormTitle').innerHTML = '<span class="icon">➕</span> 添加商品';
    document.getElementById('cpForm').action = '/user/storefront/custom-products/create';
    document.getElementById('cpEditId').value = '';
    document.getElementById('cpName').value = '';
    document.getElementById('cpDesc').value = '';
    document.getElementById('cpType').value = 'credits';
    document.getElementById('cpPrice').value = '';
    document.getElementById('cpCreditsAmount').value = '';
    document.getElementById('cpLicenseEndpoint').value = '';
    document.getElementById('cpLicenseKey').value = '';
    document.getElementById('cpLicenseProductId').value = '';
    document.getElementById('cpSubmitBtn').textContent = '创建商品';
    toggleCPTypeFields();
    card.scrollIntoView({ behavior: 'smooth' });
}

function editCustomProduct(id, name, desc, ptype, price, credits, endpoint, key, pid) {
    var card = document.getElementById('cpFormCard');
    card.style.display = 'block';
    document.getElementById('cpFormTitle').innerHTML = '<span class="icon">✏️</span> 编辑商品';
    document.getElementById('cpForm').action = '/user/storefront/custom-products/update';
    document.getElementById('cpEditId').value = id;
    document.getElementById('cpName').value = name;
    document.getElementById('cpDesc').value = desc;
    document.getElementById('cpType').value = ptype;
    document.getElementById('cpPrice').value = price;
    document.getElementById('cpCreditsAmount').value = credits;
    document.getElementById('cpLicenseEndpoint').value = endpoint;
    document.getElementById('cpLicenseKey').value = key;
    document.getElementById('cpLicenseProductId').value = pid;
    document.getElementById('cpSubmitBtn').textContent = '保存修改';
    toggleCPTypeFields();
    card.scrollIntoView({ behavior: 'smooth' });
}

function hideCPForm() {
    document.getElementById('cpFormCard').style.display = 'none';
}

function toggleCPTypeFields() {
    var t = document.getElementById('cpType').value;
    document.getElementById('cpCreditsFields').style.display = t === 'credits' ? 'block' : 'none';
    document.getElementById('cpVirtualFields').style.display = t === 'virtual_goods' ? 'block' : 'none';
}

/* ===== Custom Products: Delete with confirmation ===== */
function deleteCustomProduct(id, name) {
    if (!confirm('确定删除"' + name + '"？删除后无法恢复，已有订单记录将保留。')) return;
    var form = document.createElement('form');
    form.method = 'POST';
    form.action = '/user/storefront/custom-products/delete';
    var input = document.createElement('input');
    input.type = 'hidden';
    input.name = 'product_id';
    input.value = id;
    form.appendChild(input);
    document.body.appendChild(form);
    form.submit();
}

/* ===== Custom Products: Drag-and-drop reorder ===== */
function initCustomProductDragDrop() {
    var list = document.getElementById('customProductList');
    if (!list) return;
    var dragItem = null;
    list.addEventListener('dragstart', function(e) {
        dragItem = e.target.closest('.cp-item');
        if (dragItem) { dragItem.classList.add('dragging'); e.dataTransfer.effectAllowed = 'move'; }
    });
    list.addEventListener('dragend', function(e) {
        if (dragItem) { dragItem.classList.remove('dragging'); dragItem = null; }
    });
    list.addEventListener('dragover', function(e) {
        e.preventDefault();
        var afterEl = getCPDragAfter(list, e.clientY);
        if (dragItem) {
            if (afterEl == null) { list.appendChild(dragItem); }
            else { list.insertBefore(dragItem, afterEl); }
        }
    });
}

function getCPDragAfter(container, y) {
    var els = Array.from(container.querySelectorAll('.cp-item:not(.dragging)'));
    var closest = null; var closestOffset = Number.NEGATIVE_INFINITY;
    els.forEach(function(el) {
        var box = el.getBoundingClientRect();
        var offset = y - box.top - box.height / 2;
        if (offset < 0 && offset > closestOffset) { closestOffset = offset; closest = el; }
    });
    return closest;
}

function saveCustomProductOrder() {
    var items = document.querySelectorAll('#customProductList .cp-item');
    var ids = [];
    items.forEach(function(el) { ids.push(el.getAttribute('data-cp-id')); });
    var fd = new FormData();
    fd.append('ids', ids.join(','));
    fetch('/user/storefront/custom-products/reorder', { method: 'POST', body: fd })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.ok) { showMsg('ok', '排序已保存'); }
        else { showMsg('err', d.error || '保存失败'); }
    }).catch(function() { showMsg('err', '网络错误'); });
}

/* ===== Customer Support: Apply ===== */
function applySupportSystem() {
    var btn = document.getElementById('supportApplyBtn');
    if (btn) btn.disabled = true;
    fetch('/user/storefront/support/apply', { method: 'POST' })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success) {
            showMsg('ok', '开通请求已提交，等待管理员审批');
            setTimeout(function() { location.reload(); }, 1500);
        } else {
            showMsg('err', d.error || '申请失败');
            if (btn) btn.disabled = false;
        }
    }).catch(function() {
        showMsg('err', '网络错误');
        if (btn) btn.disabled = false;
    });
}

/* ===== Customer Support: Login ===== */
function loginSupportSystem() {
    var btn = document.getElementById('supportLoginBtn');
    if (btn) btn.disabled = true;
    fetch('/user/storefront/support/login', { method: 'POST' })
    .then(function(r) { return r.json(); })
    .then(function(d) {
        if (d.success && d.login_url) {
            window.open(d.login_url, '_blank');
        } else {
            showMsg('err', d.error || '登录失败');
        }
        if (btn) btn.disabled = false;
    }).catch(function() {
        showMsg('err', '网络错误');
        if (btn) btn.disabled = false;
    });
}
</script>
` + I18nJS + `
</body>
</html>`

// StorefrontCustomProductOrdersTmpl is the parsed custom product orders page template.
var StorefrontCustomProductOrdersTmpl = template.Must(template.New("storefront_custom_product_orders").Parse(storefrontCustomProductOrdersHTML))

const storefrontCustomProductOrdersHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>订单记录 - 自定义商品 - 分析技能包市场</title>
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
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            font-size: 18px; box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .logo-text { font-size: 15px; font-weight: 700; color: #1e293b; }
        .nav-link {
            padding: 7px 16px; font-size: 13px; font-weight: 500; color: #64748b;
            background: #fff; border: 1px solid #e2e8f0; border-radius: 8px;
            text-decoration: none; transition: all .2s;
        }
        .nav-link:hover { color: #1e293b; border-color: #cbd5e1; }
        .page-title { font-size: 22px; font-weight: 800; color: #0f172a; margin-bottom: 20px; display: flex; align-items: center; gap: 10px; }
        .card {
            background: #fff; border-radius: 12px; padding: 24px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06), 0 4px 16px rgba(0,0,0,0.04);
            margin-bottom: 20px; border: 1px solid #e2e8f0;
        }
        .card-title { font-size: 15px; font-weight: 700; color: #1e293b; margin-bottom: 16px; display: flex; align-items: center; gap: 8px; }
        .filter-bar {
            display: flex; gap: 12px; margin-bottom: 20px; flex-wrap: wrap; align-items: flex-end;
        }
        .filter-group { display: flex; flex-direction: column; gap: 4px; }
        .filter-group label { font-size: 12px; font-weight: 600; color: #64748b; }
        .filter-group input, .filter-group select {
            padding: 8px 12px; border: 1px solid #cbd5e1; border-radius: 8px;
            font-size: 13px; background: #fff; font-family: inherit;
        }
        .filter-group input:focus, .filter-group select:focus {
            outline: none; border-color: #4f46e5; box-shadow: 0 0 0 3px rgba(79,70,229,0.12);
        }
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
        .btn-ghost {
            padding: 8px 18px; font-size: 13px; border-radius: 8px;
            background: #f8fafc; color: #64748b; border: 1px solid #e2e8f0;
            cursor: pointer; transition: all .2s; font-family: inherit; font-weight: 600;
        }
        .btn-ghost:hover { background: #f1f5f9; color: #475569; }
        .order-table { width: 100%; border-collapse: collapse; }
        .order-table th {
            text-align: left; padding: 10px 12px; font-size: 12px; font-weight: 700;
            color: #64748b; border-bottom: 2px solid #e2e8f0; white-space: nowrap;
        }
        .order-table td {
            padding: 12px; font-size: 13px; color: #1e293b;
            border-bottom: 1px solid #f1f5f9; vertical-align: top;
        }
        .order-table tr:hover td { background: #f8fafc; }
        .status-badge {
            display: inline-block; padding: 2px 10px; border-radius: 20px;
            font-size: 11px; font-weight: 700;
        }
        .status-pending { background: #fef3c7; color: #d97706; border: 1px solid #fde68a; }
        .status-paid { background: #dbeafe; color: #2563eb; border: 1px solid #bfdbfe; }
        .status-fulfilled { background: #dcfce7; color: #16a34a; border: 1px solid #bbf7d0; }
        .status-failed { background: #fef2f2; color: #dc2626; border: 1px solid #fecaca; }
        .sn-info { font-size: 12px; color: #6366f1; margin-top: 4px; word-break: break-all; }
        .empty-state { text-align: center; padding: 40px 20px; color: #94a3b8; font-size: 13px; }
        .empty-state .icon { font-size: 28px; margin-bottom: 8px; opacity: 0.7; }
        .foot { text-align: center; margin-top: 28px; padding-top: 16px; border-top: 1px solid #e2e8f0; }
        .foot-text { font-size: 11px; color: #94a3b8; }
        .foot-text a { color: #6366f1; text-decoration: none; }
        @media (max-width: 640px) {
            .filter-bar { flex-direction: column; }
            .order-table { font-size: 12px; }
            .order-table th, .order-table td { padding: 8px 6px; }
        }
    </style>
</head>
<body>
<div class="page">
    <nav class="nav">
        <a class="logo-link" href="/"><span class="logo-mark">📦</span><span class="logo-text">分析技能包市场</span></a>
        <a class="nav-link" href="/user/storefront/">← 返回小铺管理</a>
    </nav>

    <h1 class="page-title">📋 订单记录</h1>

    <!-- Filter bar -->
    <div class="card">
        <form method="GET" action="/user/storefront/custom-product-orders" class="filter-bar">
            <div class="filter-group">
                <label>商品名称</label>
                <input type="text" name="product_name" value="{{.FilterProductName}}" placeholder="搜索商品名称...">
            </div>
            <div class="filter-group">
                <label>订单状态</label>
                <select name="status">
                    <option value="">全部</option>
                    <option value="pending"{{if eq .FilterStatus "pending"}} selected{{end}}>待支付</option>
                    <option value="paid"{{if eq .FilterStatus "paid"}} selected{{end}}>已支付</option>
                    <option value="fulfilled"{{if eq .FilterStatus "fulfilled"}} selected{{end}}>已履约</option>
                    <option value="failed"{{if eq .FilterStatus "failed"}} selected{{end}}>失败</option>
                </select>
            </div>
            <button type="submit" class="btn btn-indigo">🔍 筛选</button>
            {{if or .FilterProductName .FilterStatus}}
            <a href="/user/storefront/custom-product-orders" class="btn btn-ghost">清除筛选</a>
            {{end}}
        </form>
    </div>

    <!-- Orders list -->
    <div class="card">
        <div class="card-title"><span>🛒</span> 自定义商品订单</div>
        {{if .Orders}}
        <div style="overflow-x:auto;">
            <table class="order-table">
                <thead>
                    <tr>
                        <th>订单号</th>
                        <th>商品名称</th>
                        <th>买家邮箱</th>
                        <th>支付金额</th>
                        <th>状态</th>
                        <th>创建时间</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Orders}}
                    <tr>
                        <td>#{{.ID}}</td>
                        <td>{{.ProductName}}</td>
                        <td>{{.BuyerEmail}}</td>
                        <td>$ {{printf "%.2f" .AmountUSD}}</td>
                        <td>
                            <span class="status-badge status-{{.Status}}">
                                {{if eq .Status "pending"}}待支付{{end}}
                                {{if eq .Status "paid"}}已支付{{end}}
                                {{if eq .Status "fulfilled"}}已履约{{end}}
                                {{if eq .Status "failed"}}失败{{end}}
                            </span>
                            {{if and (eq .ProductType "virtual_goods") (eq .Status "fulfilled") (ne .LicenseSN "")}}
                            <div class="sn-info">🔑 SN: {{.LicenseSN}}</div>
                            {{end}}
                        </td>
                        <td>{{.CreatedAt}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{else}}
        <div class="empty-state">
            <div class="icon">📋</div>
            <p>暂无订单记录</p>
        </div>
        {{end}}
    </div>

    <div class="foot">
        <p class="foot-text">Vantagics 分析技能包市场 · <a href="/">浏览更多</a></p>
    </div>
</div>
</body>
</html>`
