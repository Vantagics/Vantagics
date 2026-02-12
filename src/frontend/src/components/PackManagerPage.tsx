import React, { useState, useEffect, useRef, useCallback } from 'react';
import ReactDOM from 'react-dom';
import { useLanguage } from '../i18n';
import { Loader2, Package, X, Share2 } from 'lucide-react';
import { ListLocalQuickAnalysisPacks } from '../../wailsjs/go/main/App';

interface LocalPackInfo {
    thread_id: string;
    pack_name: string;
    description: string;
    source_name: string;
    author: string;
    qap_file_path: string;
    created_at: string;
}

interface PackManagerPageProps {
    isOpen: boolean;
    onClose: () => void;
    onSharePack?: (pack: LocalPackInfo) => void;
}

const PackManagerPage: React.FC<PackManagerPageProps> = ({ isOpen, onClose, onSharePack }) => {
    const { t } = useLanguage();
    const [packs, setPacks] = useState<LocalPackInfo[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; pack: LocalPackInfo } | null>(null);
    const contextMenuRef = useRef<HTMLDivElement>(null);

    const loadPacks = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const result = await ListLocalQuickAnalysisPacks();
            setPacks(result || []);
        } catch (err: any) {
            setError(err?.message || err?.toString() || 'Failed to load packs');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        if (isOpen) {
            loadPacks();
            setContextMenu(null);
        }
    }, [isOpen, loadPacks]);

    // Close context menu on outside click
    useEffect(() => {
        if (!contextMenu) return;
        const handleClick = (e: MouseEvent) => {
            if (contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
                setContextMenu(null);
            }
        };
        document.addEventListener('mousedown', handleClick);
        return () => document.removeEventListener('mousedown', handleClick);
    }, [contextMenu]);

    // Close on Escape
    useEffect(() => {
        if (!isOpen) return;
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                if (contextMenu) {
                    setContextMenu(null);
                } else {
                    onClose();
                }
            }
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [isOpen, contextMenu, onClose]);

    const handleContextMenu = (e: React.MouseEvent, pack: LocalPackInfo) => {
        e.preventDefault();
        e.stopPropagation();
        setContextMenu({ x: e.clientX, y: e.clientY, pack });
    };

    const handleShareToMarket = () => {
        if (contextMenu && onSharePack) {
            onSharePack(contextMenu.pack);
        }
        setContextMenu(null);
    };

    if (!isOpen) return null;

    return ReactDOM.createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm"
            onClick={onClose}
        >
            <div
                className="bg-white dark:bg-[#252526] w-[560px] max-h-[70vh] rounded-xl shadow-2xl overflow-hidden text-slate-900 dark:text-[#d4d4d4] flex flex-col"
                onClick={e => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-[#3e3e42]">
                    <h3 className="text-lg font-bold text-slate-800 dark:text-[#d4d4d4]">
                        {t('pack_manager_title')}
                    </h3>
                    <button
                        onClick={onClose}
                        className="p-1 rounded-lg hover:bg-slate-100 dark:hover:bg-[#2d2d30] transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-400 dark:text-[#808080]" />
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6">
                    {loading && (
                        <div className="flex items-center justify-center py-12 gap-3 text-sm text-slate-500 dark:text-[#8e8e8e]">
                            <Loader2 className="w-5 h-5 animate-spin" />
                            {t('pack_manager_loading')}
                        </div>
                    )}

                    {error && !loading && (
                        <div className="text-center py-12">
                            <p className="text-sm text-red-500">{error}</p>
                            <button
                                onClick={loadPacks}
                                className="mt-3 px-4 py-2 text-sm font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                            >
                                {t('retry')}
                            </button>
                        </div>
                    )}

                    {!loading && !error && packs.length === 0 && (
                        <div className="flex flex-col items-center justify-center py-12 text-slate-400 dark:text-[#6e6e6e]">
                            <Package className="w-12 h-12 mb-3 opacity-50" />
                            <p className="text-sm">{t('pack_manager_empty')}</p>
                        </div>
                    )}

                    {!loading && !error && packs.length > 0 && (
                        <div className="space-y-2">
                            {packs.map(pack => (
                                <div
                                    key={pack.thread_id}
                                    className="p-3 rounded-lg border border-slate-200 dark:border-[#3e3e42] hover:bg-slate-50 dark:hover:bg-[#2d2d30] cursor-context-menu transition-colors"
                                    onContextMenu={e => handleContextMenu(e, pack)}
                                >
                                    <div className="flex items-start justify-between">
                                        <div className="flex-1 min-w-0">
                                            <p className="text-sm font-medium text-slate-800 dark:text-[#d4d4d4] truncate">
                                                {pack.pack_name}
                                            </p>
                                            {pack.description && (
                                                <p className="text-xs text-slate-500 dark:text-[#8e8e8e] mt-0.5 truncate">
                                                    {pack.description}
                                                </p>
                                            )}
                                            <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-400 dark:text-[#6e6e6e]">
                                                <span>{t('pack_manager_source')}: {pack.source_name}</span>
                                                {pack.author && <span>{t('pack_manager_author')}: {pack.author}</span>}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Hint */}
                {!loading && packs.length > 0 && (
                    <div className="px-6 py-3 border-t border-slate-200 dark:border-[#3e3e42] text-xs text-slate-400 dark:text-[#6e6e6e] text-center">
                        {t('pack_manager_hint')}
                    </div>
                )}
            </div>

            {/* Context Menu */}
            {contextMenu && (
                <div
                    ref={contextMenuRef}
                    className="fixed bg-white dark:bg-[#252526] border border-slate-200 dark:border-[#3c3c3c] rounded-lg shadow-xl z-[9999] w-48 py-1 overflow-hidden"
                    style={{ top: contextMenu.y, left: contextMenu.x }}
                    onContextMenu={e => { e.preventDefault(); e.stopPropagation(); }}
                >
                    <button
                        onClick={handleShareToMarket}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                    >
                        <Share2 className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                        {t('pack_manager_share_to_market')}
                    </button>
                </div>
            )}
        </div>,
        document.body
    );
};

export default PackManagerPage;
