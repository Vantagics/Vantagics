import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { GetDataSources, DeleteDataSource, ReplayAnalysis } from '../../wailsjs/go/main/App';
import { EventsEmit, EventsOn } from '../../wailsjs/runtime/runtime';
import AddDataSourceModal from './AddDataSourceModal';
import DeleteConfirmationModal from './DeleteConfirmationModal';
import NewChatModal from './NewChatModal';
import SourceContextMenu from './SourceContextMenu';
import ExportDataSourceModal from './ExportDataSourceModal';
import DataSourcePropertiesModal from './DataSourcePropertiesModal';
import { Trash2, Plus } from 'lucide-react';
import { main } from '../../wailsjs/go/models';

interface SidebarProps {
    onOpenSettings: () => void;
    onToggleChat: () => void;
    onToggleSkills: () => void;
    width: number;
}

const Sidebar: React.FC<SidebarProps> = ({ onOpenSettings, onToggleChat, onToggleSkills, width }) => {
    const { t } = useLanguage();
    const [sources, setSources] = useState<any[]>([]);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);
    const [isNewChatModalOpen, setIsNewChatModalOpen] = useState(false);
    const [deleteTarget, setDeleteTarget] = useState<{ id: string, name: string } | null>(null);
    const [exportTarget, setExportTarget] = useState<any | null>(null);
    const [propertiesTarget, setPropertiesTarget] = useState<any | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, sourceId: string } | null>(null);
    const [isReplayLoading, setIsReplayLoading] = useState(false);

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
        const unsubscribe = EventsOn('data-source-deleted', (id) => {
            fetchSources();
            if (selectedId === id) setSelectedId(null);
        });
        return () => {
            if (unsubscribe) unsubscribe();
        };
    }, [selectedId]);

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

    const handleSourceClick = (source: any) => {
        setSelectedId(source.id);
        EventsEmit('data-source-selected', source);
    };

    const handleContextMenu = (e: React.MouseEvent, sourceId: string) => {
        e.preventDefault();
        setContextMenu({ x: e.clientX, y: e.clientY, sourceId });
    };

    const handleStartChatAnalysis = () => {
        if (!selectedId) {
            alert("Please select a data source first.");
            return;
        }
        setIsNewChatModalOpen(true);
    };

    const handleNewChatSubmit = (sessionName: string) => {
        const source = sources.find(s => s.id === selectedId);
        if (source) {
            // Trigger chat open with new session details
            // We need to pass data to ChatSidebar.
            // Using EventsEmit is convenient.
            EventsEmit('start-new-chat', {
                dataSourceId: source.id,
                dataSourceName: source.name,
                sessionName: sessionName
            });
            onToggleChat(); // Ensure chat is open
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
                <button 
                    onClick={() => setIsAddModalOpen(true)}
                    className="p-1 hover:bg-slate-200 rounded-md text-slate-500 hover:text-blue-600 transition-colors"
                    title={t('add_source')}
                >
                    <Plus className="w-5 h-5" />
                </button>
            </div>
            <div className="flex-1 overflow-y-auto p-2">
                {sources.length === 0 ? (
                    <div className="p-4 text-center text-xs text-slate-400 italic">
                        No data sources added yet.
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
                                    <span className={`flex-shrink-0 w-2 h-2 rounded-full ${
                                        source.type === 'excel' ? 'bg-green-500' : 
                                        ['mysql', 'postgresql', 'doris'].includes(source.type) ? 'bg-blue-500' : 
                                        'bg-gray-400'
                                    }`}></span>
                                    <span className="truncate" title={source.name}>{source.name}</span>
                                </div>
                                <button 
                                    onClick={(e) => handleDelete(source, e)}
                                    className={`p-1 hover:text-red-600 transition-opacity relative z-10 ${selectedId === source.id ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'}`}
                                    title="Delete source"
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
                    aria-label="Toggle chat"
                    className={`w-full py-2 px-4 rounded-md text-sm font-medium transition-colors flex items-center justify-center gap-2 ${selectedId ? 'bg-blue-100 hover:bg-blue-200 text-blue-700' : 'bg-slate-100 text-slate-400 cursor-not-allowed'}`}
                >
                    <span>💬</span> {t('chat_analysis')}
                </button>
                <button
                    onClick={onToggleSkills}
                    aria-label="Skills"
                    className="w-full py-2 px-4 bg-white border border-slate-300 hover:bg-slate-50 text-slate-700 rounded-md text-sm font-medium transition-colors flex items-center justify-center gap-2"
                >
                    <span>⚡</span> {t('skills') || 'Skills'}
                </button>
                <button
                    onClick={async () => {
                        if (isReplayLoading) return;
                        
                        setIsReplayLoading(true);
                        try {
                            await ReplayAnalysis();
                        } catch (err) {
                            console.error('Replay analysis error:', err);
                            if (err && typeof err === 'object' && 'message' in err) {
                                alert(`分析重放失败: ${err.message}`);
                            } else if (typeof err === 'string') {
                                alert(`分析重放失败: ${err}`);
                            } else {
                                alert('分析重放失败: 未知错误');
                            }
                        } finally {
                            setIsReplayLoading(false);
                        }
                    }}
                    disabled={isReplayLoading}
                    aria-label={t('replay_analysis')}
                    className={`w-full py-2 px-4 border border-slate-300 rounded-md text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
                        isReplayLoading 
                            ? 'bg-slate-100 text-slate-400 cursor-not-allowed' 
                            : 'bg-white hover:bg-slate-50 text-slate-700'
                    }`}
                >
                    {isReplayLoading ? (
                        <>
                            <div className="w-4 h-4 border-2 border-slate-300 border-t-slate-600 rounded-full animate-spin"></div>
                            <span>处理中...</span>
                        </>
                    ) : (
                        <>
                            <span>▶️</span> {t('replay_analysis')}
                        </>
                    )}
                </button>
                <button
                    onClick={onOpenSettings}
                    aria-label="Settings"
                    className="w-full py-2 px-4 bg-white border border-slate-300 hover:bg-slate-50 text-slate-700 rounded-md text-sm font-medium transition-colors flex items-center justify-center gap-2"
                >
                    <span>⚙️</span> {t('settings')}
                </button>
            </div>

            <AddDataSourceModal 
                isOpen={isAddModalOpen} 
                onClose={() => setIsAddModalOpen(false)} 
                onSuccess={fetchSources}
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

            {contextMenu && (
                <SourceContextMenu
                    position={{ x: contextMenu.x, y: contextMenu.y }}
                    sourceId={contextMenu.sourceId}
                    onClose={() => setContextMenu(null)}
                    onSelectThread={(thread) => {
                        EventsEmit('open-chat', thread);
                        onToggleChat();
                    }}
                    onExport={() => {
                        const source = sources.find(s => s.id === contextMenu.sourceId);
                        if (source) setExportTarget(source);
                    }}
                    onProperties={() => {
                        const source = sources.find(s => s.id === contextMenu.sourceId);
                        if (source) setPropertiesTarget(source);
                    }}
                    onStartAnalysis={() => {
                        const source = sources.find(s => s.id === contextMenu.sourceId);
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