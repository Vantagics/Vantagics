import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { GetDataSources, DeleteDataSource, RenameDataSource } from '../../wailsjs/go/main/App';
import { EventsEmit, EventsOn } from '../../wailsjs/runtime/runtime';
import AddDataSourceModal from './AddDataSourceModal';
import DeleteConfirmationModal from './DeleteConfirmationModal';
import ConfirmationModal from './ConfirmationModal';
import NewChatModal from './NewChatModal';
import SourceContextMenu from './SourceContextMenu';
import ExportDataSourceModal from './ExportDataSourceModal';
import DataSourcePropertiesModal from './DataSourcePropertiesModal';
import DataSourceOptimizeModal from './DataSourceOptimizeModal';
import RenameDataSourceModal from './RenameDataSourceModal';
import SemanticOptimizeModal from './SemanticOptimizeModal';
import { Trash2, Plus } from 'lucide-react';

interface SidebarProps {
    onOpenSettings: () => void;
    onToggleChat: () => void;
    width: number;
    isChatOpen: boolean; // 添加当前会话区状态
    isAnalysisLoading?: boolean; // 分析进行中状态
}

const Sidebar: React.FC<SidebarProps> = ({ onOpenSettings, onToggleChat, width, isChatOpen, isAnalysisLoading }) => {
    const { t } = useLanguage();
    const [sources, setSources] = useState<any[]>([]);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);
    const [isNewChatModalOpen, setIsNewChatModalOpen] = useState(false);
    const [deleteTarget, setDeleteTarget] = useState<{ id: string, name: string } | null>(null);
    const [exportTarget, setExportTarget] = useState<any | null>(null);
    const [propertiesTarget, setPropertiesTarget] = useState<any | null>(null);
    const [optimizeTarget, setOptimizeTarget] = useState<{ id: string, name: string } | null>(null);
    const [semanticOptimizeTarget, setSemanticOptimizeTarget] = useState<{ id: string, name: string } | null>(null);
    const [renameTarget, setRenameTarget] = useState<{ id: string, name: string } | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, sourceId: string, sourceName: string, hasLocalDB: boolean, isOptimized: boolean } | null>(null);
    const [isReplayLoading, setIsReplayLoading] = useState(false);
    const [isDataSourceExpanded, setIsDataSourceExpanded] = useState(true); // 数据源区域展开/折叠状态
    const [showNoDataSourcePrompt, setShowNoDataSourcePrompt] = useState(false); // 无数据源提示对话框

    const fetchSources = async () => {
        try {
            const data = await GetDataSources();
            setSources(data || []);
        } catch (err) {
            console.error('Failed to fetch data sources:', err);
        }
    };

    useEffect(() => {
        fetchSources();
        const unsubscribeDeleted = EventsOn('data-source-deleted', (id) => {
            fetchSources();
            if (selectedId === id) setSelectedId(null);
        });
        const unsubscribeOptimized = EventsOn('data-source-optimized', () => {
            fetchSources();
        });
        const unsubscribeColumnRenamed = EventsOn('column-renamed', async (data: { dataSourceId: string }) => {
            // Refresh sources list
            const updatedSources = await GetDataSources();
            setSources(updatedSources || []);
            
            // If properties modal is open for this data source, update it
            if (propertiesTarget && propertiesTarget.id === data.dataSourceId) {
                const updatedSource = updatedSources?.find((s: any) => s.id === data.dataSourceId);
                if (updatedSource) {
                    setPropertiesTarget(updatedSource);
                }
            }
        });
        return () => {
            if (unsubscribeDeleted) unsubscribeDeleted();
            if (unsubscribeOptimized) unsubscribeOptimized();
            if (unsubscribeColumnRenamed) unsubscribeColumnRenamed();
        };
    }, [selectedId, propertiesTarget]);

    const handleDelete = (source: any, e: React.MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();
        setDeleteTarget(source);
    };

    const confirmDelete = async () => {
        if (!deleteTarget) return;
        try {
            await DeleteDataSource(deleteTarget.id);
            EventsEmit('data-source-deleted', deleteTarget.id);
            fetchSources();
            if (selectedId === deleteTarget.id) setSelectedId(null);
            setDeleteTarget(null);
        } catch (err) {
            console.error('Failed to delete data source:', err);
            alert('Failed to delete: ' + err);
        }
    };

    const handleRename = async (newName: string) => {
        if (!renameTarget) return;
        try {
            await RenameDataSource(renameTarget.id, newName);
            fetchSources();
            EventsEmit('data-source-renamed', { id: renameTarget.id, newName });
        } catch (err) {
            throw err; // Re-throw to let modal handle the error
        }
    };

    const handleSourceClick = (source: any) => {
        setSelectedId(source.id);
        EventsEmit('data-source-selected', source);
    };

    const handleContextMenu = (e: React.MouseEvent, sourceId: string) => {
        e.preventDefault();
        const source = sources?.find(s => s.id === sourceId);
        if (source) {
            // Check if this is a local SQLite database
            // A data source has a local DB if it has a db_path in config
            // This includes Excel, CSV, JSON imports that are stored locally
            const hasLocalDB = !!(source.config && source.config.db_path);
            // Check if the data source has been optimized (for index optimization menu)
            const isOptimized = !!(source.config && source.config.optimized);
            console.log('[DEBUG] Context menu for source:', {
                sourceId,
                sourceName: source.name,
                sourceType: source.type,
                hasConfig: !!source.config,
                hasDbPath: !!(source.config && source.config.db_path),
                dbPath: source.config?.db_path,
                optimized: source.config?.optimized,
                hasLocalDB,
                isOptimized
            });
            setContextMenu({ 
                x: e.clientX, 
                y: e.clientY, 
                sourceId,
                sourceName: source.name,
                hasLocalDB,
                isOptimized
            });
        }
    };

    const handleStartChatAnalysis = () => {
        if (!selectedId) {
            // 显示确认对话框，让用户选择是否进入随意聊模式
            setShowNoDataSourcePrompt(true);
            return;
        }
        setIsNewChatModalOpen(true);
    };

    const handleStartFreeChat = () => {
        // 用户确认进入随意聊模式
        setShowNoDataSourcePrompt(false);
        EventsEmit('start-free-chat', {
            sessionName: t('free_chat'),
            keepChatOpen: true
        });
        // 只在会话区关闭时才打开它
        if (!isChatOpen) {
            onToggleChat();
        }
    };

    const handleNewChatSubmit = (sessionName: string) => {
        const source = sources?.find(s => s.id === selectedId);
        if (source) {
            // Trigger chat open with new session details
            // We need to pass data to ChatSidebar.
            // Using EventsEmit is convenient.
            EventsEmit('start-new-chat', {
                dataSourceId: source.id,
                dataSourceName: source.name,
                sessionName: sessionName,
                keepChatOpen: true // 确保创建新会话后保持会话区展开
            });
            // 只在会话区关闭时才打开它，避免切换状态
            if (!isChatOpen) {
                onToggleChat();
            }
        }
    };

    return (
        <div
            className="bg-slate-100 border-r border-slate-200 flex flex-col h-full flex-shrink-0"
            style={{ width: width }}
        >
            <div
                className="p-4 pt-8 border-b border-slate-200 bg-slate-50 flex items-center justify-between"
            >
                <h2 className="text-lg font-semibold text-slate-700">{t('data_sources')}</h2>
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => setIsDataSourceExpanded(!isDataSourceExpanded)}
                        className="p-1 hover:bg-slate-200 rounded-md text-slate-500 hover:text-blue-600 transition-colors"
                        title={isDataSourceExpanded ? "折叠" : "展开"}
                    >
                        {isDataSourceExpanded ? '<<' : '>>'}
                    </button>
                    <button
                        onClick={() => setIsAddModalOpen(true)}
                        className="p-1 hover:bg-slate-200 rounded-md text-slate-500 hover:text-blue-600 transition-colors"
                        title={t('add_source')}
                    >
                        <Plus className="w-5 h-5" />
                    </button>
                </div>
            </div>
            {isDataSourceExpanded && (
                <>
                    <div className="flex-1 overflow-y-auto p-2">
                        {!sources || sources.length === 0 ? (
                            <div className="p-4 text-center text-xs text-slate-400 italic">
                                {t('no_data_sources_yet')}
                            </div>
                        ) : (
                            <ul className="space-y-1">
                                {sources.map((source) => (
                                    <li
                                        key={source.id}
                                        className={`group p-2 rounded-md text-sm flex items-center justify-between transition-colors relative ${selectedId === source.id ? 'bg-blue-200 text-blue-800' : 'hover:bg-blue-100 text-slate-600'}`}
                                        onContextMenu={(e) => handleContextMenu(e, source.id)}
                                    >
                                        <div
                                            className="flex items-center gap-2 overflow-hidden flex-1 cursor-pointer"
                                            onClick={() => handleSourceClick(source)}
                                        >
                                            <span className={`flex-shrink-0 w-2 h-2 rounded-full ${source.type === 'excel' ? 'bg-green-500' :
                                                    ['mysql', 'postgresql', 'doris'].includes(source.type) ? 'bg-blue-500' :
                                                        'bg-gray-400'
                                                }`}></span>
                                            <span className="truncate" title={source.name}>{source.name}</span>
                                        </div>
                                        <button
                                            onClick={(e) => handleDelete(source, e)}
                                            className={`p-1 hover:text-red-600 transition-opacity relative z-10 ${selectedId === source.id ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'}`}
                                            title={t('delete_source')}
                                        >
                                            <Trash2 className="w-3 h-3" />
                                        </button>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                    <div className="p-4 border-t border-slate-200 flex flex-col gap-2">
                        <button
                            onClick={handleStartChatAnalysis}
                            aria-label={t('chat_analysis')}
                            className="w-full py-2 px-4 rounded-md text-sm font-medium transition-colors flex items-center justify-center gap-2 bg-blue-100 hover:bg-blue-200 text-blue-700"
                        >
                            <span>💬</span> {t('chat_analysis')}
                        </button>
                    </div>
                </>
            )}

            <AddDataSourceModal
                isOpen={isAddModalOpen}
                onClose={() => setIsAddModalOpen(false)}
                onSuccess={(newDataSource) => {
                    fetchSources();
                    // Check if we should auto-open optimize modal
                    if (newDataSource && newDataSource.config?.db_path && !newDataSource.config?.optimized) {
                        // Auto-open optimize modal for new local databases
                        setOptimizeTarget({ id: newDataSource.id, name: newDataSource.name });
                    }
                }}
            />

            <DeleteConfirmationModal
                isOpen={!!deleteTarget}
                sourceName={deleteTarget?.name || ''}
                onClose={() => setDeleteTarget(null)}
                onConfirm={confirmDelete}
            />

            <NewChatModal
                isOpen={isNewChatModalOpen}
                dataSourceId={selectedId || ''}
                onClose={() => setIsNewChatModalOpen(false)}
                onSubmit={handleNewChatSubmit}
                onStartFreeChat={() => {
                    // Emit event to start free chat mode with localized title
                    EventsEmit('start-free-chat', { 
                        sessionName: t('free_chat'),
                        openChat: true  // Signal to open chat panel
                    });
                }}
            />

            <ExportDataSourceModal
                isOpen={!!exportTarget}
                sourceId={exportTarget?.id || ''}
                sourceName={exportTarget?.name || ''}
                onClose={() => setExportTarget(null)}
                dataSource={exportTarget}
            />

            <DataSourcePropertiesModal
                isOpen={!!propertiesTarget}
                dataSource={propertiesTarget}
                onClose={() => setPropertiesTarget(null)}
            />
            
            <DataSourceOptimizeModal
                isOpen={!!optimizeTarget}
                dataSourceId={optimizeTarget?.id || ''}
                dataSourceName={optimizeTarget?.name || ''}
                onClose={() => setOptimizeTarget(null)}
            />

            <SemanticOptimizeModal
                isOpen={!!semanticOptimizeTarget}
                dataSourceId={semanticOptimizeTarget?.id || ''}
                dataSourceName={semanticOptimizeTarget?.name || ''}
                onClose={() => setSemanticOptimizeTarget(null)}
                onSuccess={() => {
                    fetchSources();
                    setSemanticOptimizeTarget(null);
                }}
            />

            <RenameDataSourceModal
                isOpen={!!renameTarget}
                currentName={renameTarget?.name || ''}
                onClose={() => setRenameTarget(null)}
                onRename={handleRename}
            />

            <ConfirmationModal
                isOpen={showNoDataSourcePrompt}
                title={t('no_data_source_prompt_title')}
                message={t('no_data_source_prompt_message')}
                confirmText={t('start_free_chat')}
                cancelText={t('cancel')}
                onClose={() => setShowNoDataSourcePrompt(false)}
                onConfirm={handleStartFreeChat}
            />

            {contextMenu && (
                <SourceContextMenu
                    position={{ x: contextMenu.x, y: contextMenu.y }}
                    sourceId={contextMenu.sourceId}
                    sourceName={contextMenu.sourceName}
                    hasLocalDB={contextMenu.hasLocalDB}
                    isOptimized={contextMenu.isOptimized}
                    onClose={() => setContextMenu(null)}
                    onSelectThread={(thread) => {
                        EventsEmit('open-chat', thread);
                        // 只在会话区关闭时才打开它，避免切换状态
                        if (!isChatOpen) {
                            onToggleChat();
                        }
                    }}
                    onExport={() => {
                        const source = sources?.find(s => s.id === contextMenu.sourceId);
                        if (source) setExportTarget(source);
                    }}
                    onProperties={() => {
                        const source = sources?.find(s => s.id === contextMenu.sourceId);
                        if (source) setPropertiesTarget(source);
                    }}
                    onRename={() => {
                        setRenameTarget({ id: contextMenu.sourceId, name: contextMenu.sourceName });
                    }}
                    onOptimize={() => {
                        setOptimizeTarget({ id: contextMenu.sourceId, name: contextMenu.sourceName });
                    }}
                    onSemanticOptimize={() => {
                        setSemanticOptimizeTarget({ id: contextMenu.sourceId, name: contextMenu.sourceName });
                    }}
                    onStartAnalysis={() => {
                        const source = sources?.find(s => s.id === contextMenu.sourceId);
                        if (source) {
                            setSelectedId(source.id);
                            setIsNewChatModalOpen(true);
                        }
                    }}
                />
            )}
        </div>
    );
};

export default Sidebar;