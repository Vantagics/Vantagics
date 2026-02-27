import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { GetDataSources, DeleteDataSource, RenameDataSource, GetChatHistory, DeleteThread, OpenSessionResultsDirectory, GetConfig, SaveConfig, ClearThreadMessages, CreateChatThread, UpdateThreadTitle, ServicePortalLogin, MarketplacePortalLogin } from '../../wailsjs/go/main/App';
import { EventsEmit, EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import AddDataSourceModal from './AddDataSourceModal';
import DeleteConfirmationModal from './DeleteConfirmationModal';
import ConfirmationModal from './ConfirmationModal';
import NewChatModal from './NewChatModal';
import SourceContextMenu from './SourceContextMenu';
import ExportDataSourceModal from './ExportDataSourceModal';
import DataSourcePropertiesModal from './DataSourcePropertiesModal';
import RenameDataSourceModal from './RenameDataSourceModal';
import SemanticOptimizeModal from './SemanticOptimizeModal';
import OnboardingWizard from './OnboardingWizard';
import HistoricalSessionsSection from './HistoricalSessionsSection';
import ChatThreadContextMenu from './ChatThreadContextMenu';
import MemoryViewModal from './MemoryViewModal';
import ExportPackDialog from './ExportPackDialog';
import ImportPackDialog from './ImportPackDialog';
import RenameSessionModal from './RenameSessionModal';
import ServiceAuthErrorDialog from './ServiceAuthErrorDialog';
import { Trash2, Plus, Database, FileSpreadsheet, MessageCircle, BarChart3, History, Play } from 'lucide-react';
import { useLoadingState } from '../hooks/useLoadingState';
import './LeftPanel.css';

interface SidebarProps {
    onOpenSettings: () => void;
    onToggleChat: () => void;
    width: number;
    isChatOpen: boolean; // 添加当前会话区状态
    isAnalysisLoading?: boolean; // 分析进行中状态
    isPermanentFree?: boolean; // 永久免费模式状态
    onSessionSelect: (sessionId: string) => void;
    selectedSessionId: string | null;
    onOpenPackManager?: (dataSourceId?: string) => void; // 打开分析包管理
}

const Sidebar: React.FC<SidebarProps> = ({ onOpenSettings, onToggleChat, width, isChatOpen, isAnalysisLoading, isPermanentFree, onSessionSelect, selectedSessionId, onOpenPackManager }) => {
    const { t } = useLanguage();
    const { isLoading: isSessionAnalysisLoading } = useLoadingState();
    const [sources, setSources] = useState<any[]>([]);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);
    const [isNewChatModalOpen, setIsNewChatModalOpen] = useState(false);
    const [deleteTarget, setDeleteTarget] = useState<{ id: string, name: string } | null>(null);
    const [exportTarget, setExportTarget] = useState<any | null>(null);
    const [propertiesTarget, setPropertiesTarget] = useState<any | null>(null);
    const [semanticOptimizeTarget, setSemanticOptimizeTarget] = useState<{ id: string, name: string } | null>(null);
    const [renameTarget, setRenameTarget] = useState<{ id: string, name: string } | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, sourceId: string, sourceName: string, sourceType: string, hasLocalDB: boolean, isOptimized: boolean } | null>(null);
    const [isReplayLoading, setIsReplayLoading] = useState(false);
    const [isDataSourceExpanded, setIsDataSourceExpanded] = useState(true); // 数据源区域展开/折叠状态
    const [showNoDataSourcePrompt, setShowNoDataSourcePrompt] = useState(false); // 无数据源提示对话框
    const [showFreeNoDataSourcePrompt, setShowFreeNoDataSourcePrompt] = useState(false); // 免费模式无数据源提示
    const [showOnboardingWizard, setShowOnboardingWizard] = useState(false); // 新手向导
    const [preSelectedDriverType, setPreSelectedDriverType] = useState<string | null>(null); // 预选的数据源类型
    const [hasShownOnboarding, setHasShownOnboarding] = useState(false); // 是否已显示过新手向导

    // Historical sessions state
    const [sessions, setSessions] = useState<main.ChatThread[]>([]);
    const [isLoadingSessions, setIsLoadingSessions] = useState(false);
    const [sessionContextMenu, setSessionContextMenu] = useState<{ x: number; y: number; sessionId: string; isReplaySession?: boolean } | null>(null);
    const [deleteSessionTarget, setDeleteSessionTarget] = useState<{ id: string; title: string } | null>(null);
    const [memoryModalTarget, setMemoryModalTarget] = useState<string | null>(null);
    const [exportPackThreadId, setExportPackThreadId] = useState<string | null>(null);
    const [importPackDataSourceId, setImportPackDataSourceId] = useState<string | null>(null);
    const [autoIntentUnderstanding, setAutoIntentUnderstanding] = useState<boolean>(true);
    const [freeChatThreadId, setFreeChatThreadId] = useState<string | null>(null);
    const [isGeneratingReport, setIsGeneratingReport] = useState(false);
    const [renameSessionTarget, setRenameSessionTarget] = useState<{ id: string; title: string; dataSourceId: string; dataSourceName?: string } | null>(null);

    // Service portal SSO login state
    const [serviceLoginLoading, setServiceLoginLoading] = useState(false);
    const [showServiceError, setShowServiceError] = useState(false);
    const [serviceErrorMessage, setServiceErrorMessage] = useState('');

    // Shared SSO login handler for both button click and menu event
    const handleServicePortalLogin = async () => {
        if (serviceLoginLoading) return;
        setServiceLoginLoading(true);
        setShowServiceError(false);
        setServiceErrorMessage('');
        try {
            const url = await ServicePortalLogin();
            BrowserOpenURL(url);
        } catch (err: any) {
            const msg = typeof err === 'string' ? err : err?.message || 'Unknown error';
            setServiceErrorMessage(msg);
            setShowServiceError(true);
        } finally {
            setServiceLoginLoading(false);
        }
    };

    // Listen for "service-portal-login" event from menu item (Task 4.1)
    useEffect(() => {
        const cleanup = EventsOn("service-portal-login", () => {
            handleServicePortalLogin();
        });
        return () => { cleanup(); };
    }, []);

    // Marketplace portal SSO login handler
    const handleMarketplacePortalLogin = async () => {
        try {
            const url = await MarketplacePortalLogin();
            BrowserOpenURL(url);
        } catch (err: any) {
            const msg = typeof err === 'string' ? err : err?.message || 'Unknown error';
            setServiceErrorMessage(msg);
            setShowServiceError(true);
        }
    };

    // Listen for "marketplace-portal-login" event from menu item
    useEffect(() => {
        const cleanup = EventsOn("marketplace-portal-login", () => {
            handleMarketplacePortalLogin();
        });
        return () => { cleanup(); };
    }, []);

    const fetchSources = async () => {
        try {
            const data = await GetDataSources();
            setSources(data || []);
            
            // Show onboarding wizard if no data sources and haven't shown before
            if ((!data || data.length === 0) && !hasShownOnboarding) {
                setShowOnboardingWizard(true);
                setHasShownOnboarding(true);
            }
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
        const unsubscribeAnalysisComplete = EventsOn('data-source-analysis-complete', async (dataSourceId: string) => {
            // Refresh sources list when background analysis completes
            const updatedSources = await GetDataSources();
            setSources(updatedSources || []);

            // If properties modal is open for this data source, update it with the new analysis
            if (propertiesTarget && propertiesTarget.id === dataSourceId) {
                const updatedSource = updatedSources?.find((s: any) => s.id === dataSourceId);
                if (updatedSource) {
                    setPropertiesTarget(updatedSource);
                }
            }
        });
        return () => {
            if (unsubscribeDeleted) unsubscribeDeleted();
            if (unsubscribeColumnRenamed) unsubscribeColumnRenamed();
            if (unsubscribeAnalysisComplete) unsubscribeAnalysisComplete();
        };
    }, [selectedId, propertiesTarget]);

    // Fetch historical sessions
    const fetchSessions = async () => {
        setIsLoadingSessions(true);
        try {
            const history = await GetChatHistory();
            const allSessions = history || [];
            
            // Find or create the free chat thread (no data_source_id)
            let freeChat = allSessions.find((s: main.ChatThread) => !s.data_source_id || s.data_source_id === '');
            
            if (!freeChat) {
                // Create a free chat thread if none exists
                try {
                    freeChat = await CreateChatThread('', t('free_chat'));
                    allSessions.unshift(freeChat);
                    EventsEmit('chat-thread-created', freeChat.id);
                } catch (e) {
                    console.error('Failed to create free chat thread:', e);
                }
            }
            
            if (freeChat) {
                setFreeChatThreadId(freeChat.id);
            }
            
            const sortedSessions = allSessions.sort((a: main.ChatThread, b: main.ChatThread) => b.created_at - a.created_at);
            setSessions(sortedSessions);
        } catch (error) {
            console.error('Failed to fetch sessions:', error);
            setSessions([]);
        } finally {
            setIsLoadingSessions(false);
        }
    };

    // Load sessions on mount
    useEffect(() => {
        fetchSessions();
        // Load autoIntentUnderstanding setting
        GetConfig().then(cfg => {
            setAutoIntentUnderstanding(cfg.autoIntentUnderstanding !== false);
        }).catch(() => {});
    }, []);

    // Listen for session events
    useEffect(() => {
        const unsubCreated = EventsOn('chat-thread-created', () => fetchSessions());
        const unsubDeleted = EventsOn('chat-thread-deleted', () => fetchSessions());
        const unsubUpdated = EventsOn('chat-thread-updated', () => fetchSessions());
        const unsubQapCreated = EventsOn('qap-session-created', () => fetchSessions());
        const unsubReportStatus = EventsOn('comprehensive-report-status', (data: any) => {
            setIsGeneratingReport(!!data?.generating);
        });
        return () => {
            if (unsubCreated) unsubCreated();
            if (unsubDeleted) unsubDeleted();
            if (unsubUpdated) unsubUpdated();
            if (unsubQapCreated) unsubQapCreated();
            if (unsubReportStatus) unsubReportStatus();
        };
    }, []);

    const handleSessionContextMenu = (e: React.MouseEvent, sessionId: string) => {
        e.preventDefault();
        e.stopPropagation();
        const session = sessions.find(s => s.id === sessionId);
        setSessionContextMenu({ x: e.clientX, y: e.clientY, sessionId, isReplaySession: session?.is_replay_session });
    };

    // Close session context menu on outside click
    useEffect(() => {
        const handleClick = () => { if (sessionContextMenu) setSessionContextMenu(null); };
        if (sessionContextMenu) {
            document.addEventListener('click', handleClick);
            return () => document.removeEventListener('click', handleClick);
        }
    }, [sessionContextMenu]);

    const handleDeleteSession = (id: string, title: string) => {
        // Don't allow deleting the free chat thread
        if (id === freeChatThreadId) return;
        setDeleteSessionTarget({ id, title });
    };

    const confirmDeleteSession = async () => {
        if (!deleteSessionTarget) return;
        try {
            await DeleteThread(deleteSessionTarget.id);
            EventsEmit('chat-thread-deleted', deleteSessionTarget.id);
            await fetchSessions();
            
            // If the deleted session was selected, switch to free chat
            if (selectedSessionId === deleteSessionTarget.id && freeChatThreadId) {
                onSessionSelect(freeChatThreadId);
            }
            
            setDeleteSessionTarget(null);
        } catch (err) {
            console.error('Failed to delete session:', err);
            setDeleteSessionTarget(null);
        }
    };

    const handleSessionContextAction = async (action: 'view_memory' | 'view_results_directory' | 'toggle_intent_understanding' | 'clear_messages' | 'comprehensive_report' | 'export_quick_analysis_pack' | 'rename', threadId: string) => {
        if (action === 'rename') {
            const session = sessions.find(s => s.id === threadId);
            if (session) {
                const dsName = sources?.find((s: any) => s.id === session.data_source_id)?.name;
                setRenameSessionTarget({
                    id: session.id,
                    title: session.title,
                    dataSourceId: session.data_source_id,
                    dataSourceName: dsName,
                });
            }
        } else if (action === 'view_memory') {
            setMemoryModalTarget(threadId);
        } else if (action === 'view_results_directory') {
            try {
                await OpenSessionResultsDirectory(threadId);
            } catch (e) {
                console.error('Open results directory failed:', e);
            }
        } else if (action === 'toggle_intent_understanding') {
            try {
                const newValue = !autoIntentUnderstanding;
                const config = await GetConfig();
                config.autoIntentUnderstanding = newValue;
                await SaveConfig(config);
                setAutoIntentUnderstanding(newValue);
            } catch (e) {
                console.error('Toggle intent understanding failed:', e);
            }
        } else if (action === 'clear_messages') {
            try {
                await ClearThreadMessages(threadId);
                EventsEmit('chat-thread-updated', threadId);
                // If this is the active session, clear the chat display
                if (selectedSessionId === threadId) {
                    EventsEmit('clear-dashboard');
                    // Re-select to refresh the chat view
                    onSessionSelect(threadId);
                }
            } catch (e) {
                console.error('Clear thread messages failed:', e);
            }
        } else if (action === 'comprehensive_report') {
            // First switch to the session, then trigger report generation
            onSessionSelect(threadId);
            EventsEmit('generate-comprehensive-report', { threadId });
        } else if (action === 'export_quick_analysis_pack') {
            setExportPackThreadId(threadId);
        }
        setSessionContextMenu(null);
    };

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

    const handleRenameSession = async (newTitle: string) => {
        if (!renameSessionTarget) return;
        const targetId = renameSessionTarget.id;
        setRenameSessionTarget(null);
        try {
            await UpdateThreadTitle(targetId, newTitle);
            await fetchSessions();
            EventsEmit('chat-thread-updated', targetId);
        } catch (err) {
            console.error('Failed to rename session:', err);
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
            console.log('[DEBUG] Context menu for source:', {
                sourceId,
                sourceName: source.name,
                sourceType: source.type,
                hasConfig: !!source.config,
                hasDbPath: !!(source.config && source.config.db_path),
                dbPath: source.config?.db_path,
                hasLocalDB
            });
            setContextMenu({ 
                x: e.clientX, 
                y: e.clientY, 
                sourceId,
                sourceName: source.name,
                sourceType: source.type || '',
                hasLocalDB,
                isOptimized: false
            });
        }
    };

    const handleStartChatAnalysis = () => {
        if (!selectedId) {
            // 如果只有一个数据源，自动选中它并打开新会话对话框
            if (sources && sources.length === 1) {
                const source = sources[0];
                setSelectedId(source.id);
                EventsEmit('data-source-selected', source);
                setIsNewChatModalOpen(true);
                return;
            }
            // 显示确认对话框，让用户选择是否进入系统助手模式
            setShowNoDataSourcePrompt(true);
            return;
        }
        setIsNewChatModalOpen(true);
    };

    const handleStartFreeChat = () => {
        // 用户确认进入系统助手模式 - 切换到已有的系统助手会话，而不是创建新的
        setShowNoDataSourcePrompt(false);
        
        // 直接选中已有的系统助手会话
        if (freeChatThreadId) {
            onSessionSelect(freeChatThreadId);
        }
        
        EventsEmit('start-free-chat', {
            sessionName: t('free_chat'),
            keepChatOpen: true
        });
        // 只在会话区关闭时才打开它
        if (!isChatOpen) {
            onToggleChat();
        }
    };

    // 免费模式下"开始分析"按钮点击处理
    const handleFreeStartAnalysis = () => {
        if (!selectedId) {
            // 如果只有一个数据源，自动选中它
            if (sources && sources.length === 1) {
                const source = sources[0];
                setSelectedId(source.id);
                EventsEmit('data-source-selected', source);
                if (onOpenPackManager) {
                    onOpenPackManager(source.id);
                }
                return;
            }
            setShowFreeNoDataSourcePrompt(true);
            return;
        }
        // 有数据源时直接打开分析包管理，传递已选数据源ID
        if (onOpenPackManager) {
            onOpenPackManager(selectedId);
        }
    };

    const handleFreeNoDataSourceConfirm = () => {
        setShowFreeNoDataSourcePrompt(false);
        // 确认后打开分析包管理（无预选数据源）
        if (onOpenPackManager) {
            onOpenPackManager();
        }
    };

    const handleNewChatSubmit = (sessionName: string) => {
        // 优先用 selectedId 查找，如果为空且只有一个数据源则自动使用
        const source = sources?.find(s => s.id === selectedId)
            || (sources && sources.length === 1 ? sources[0] : undefined);
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
            className="bg-gradient-to-b from-slate-50 to-slate-100/80 dark:from-[#1e1e1e] dark:to-[#1e1e1e] border-r border-slate-200/70 dark:border-[#3c3c3c] flex flex-col h-full flex-shrink-0"
            style={{ width: width }}
        >
            <div
                className="px-5 py-4 pt-8 border-b border-slate-200/60 dark:border-[#3c3c3c] bg-white/60 dark:bg-[#252526] flex items-center justify-between backdrop-blur-sm"
            >
                <h2 className="text-[15px] font-semibold text-slate-600 dark:text-[#d4d4d4] flex items-center gap-2 tracking-wide"><Database className="w-4 h-4 text-slate-400 dark:text-slate-500" />{t('data_sources')}</h2>
                <div className="flex items-center gap-1">
                    <button
                        onClick={() => setIsDataSourceExpanded(!isDataSourceExpanded)}
                        className="p-1.5 hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg text-slate-400 dark:text-[#808080] hover:text-slate-600 transition-all duration-200"
                        title={isDataSourceExpanded ? "折叠" : "展开"}
                    >
                        {isDataSourceExpanded ? '<<' : '>>'}
                    </button>
                    <button
                        onClick={() => {
                            setShowOnboardingWizard(true);
                        }}
                        className="p-1.5 hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg text-slate-400 dark:text-[#808080] hover:text-slate-600 transition-all duration-200"
                        title={t('add_source')}
                    >
                        <Plus className="w-4 h-4" />
                    </button>
                </div>
            </div>
            {isDataSourceExpanded && (
                <>
                    <div className="overflow-y-auto px-3 py-2" style={{ maxHeight: '30vh' }}>
                        {!sources || sources.length === 0 ? (
                            <div className="p-4 text-center text-xs text-slate-400 dark:text-[#808080]">
                                {t('no_data_sources_yet')}
                            </div>
                        ) : (
                            <ul className="space-y-0.5">
                                {sources.map((source) => (
                                    <li
                                        key={source.id}
                                        className={`group px-3 py-2 rounded-lg text-[13px] flex items-center justify-between transition-all duration-150 relative ${selectedId === source.id ? 'bg-slate-200/80 dark:bg-[#264f78] text-slate-700 dark:text-[#569cd6]' : 'hover:bg-slate-100 dark:hover:bg-[#2d2d30] text-slate-500 dark:text-[#d4d4d4]'}`}
                                        onContextMenu={(e) => handleContextMenu(e, source.id)}
                                    >
                                        <div
                                            className="flex items-center gap-2.5 overflow-hidden flex-1 cursor-pointer"
                                            onClick={() => handleSourceClick(source)}
                                        >
                                            {source.type === 'excel' ? (
                                                <FileSpreadsheet className="flex-shrink-0 w-3.5 h-3.5 text-emerald-400" />
                                            ) : (
                                                <Database className="flex-shrink-0 w-3.5 h-3.5 text-slate-400" />
                                            )}
                                            <span className="truncate" title={source.name}>{source.name}</span>
                                        </div>
                                        <div className={`flex items-center gap-3 transition-opacity ${selectedId === source.id ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'}`}>
                                            {isPermanentFree && (
                                                <button
                                                    onClick={(e) => {
                                                        e.preventDefault();
                                                        e.stopPropagation();
                                                        handleSourceClick(source);
                                                        if (onOpenPackManager) {
                                                            onOpenPackManager(source.id);
                                                        }
                                                    }}
                                                    className="p-1 hover:text-blue-500 dark:hover:text-[#569cd6] relative z-10"
                                                    title={t('start_new_analysis')}
                                                >
                                                    <Play className="w-3 h-3" />
                                                </button>
                                            )}
                                            <button
                                                onClick={(e) => handleDelete(source, e)}
                                                className="p-1 hover:text-red-400 relative z-10"
                                                title={t('delete_source')}
                                            >
                                                <Trash2 className="w-3 h-3" />
                                            </button>
                                        </div>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                    <div className="px-4 py-3 border-t border-slate-200/60 dark:border-[#3c3c3c] flex flex-col gap-2">
                        {/* Analysis Session button - hidden in permanent free mode (Req 4.4) */}
                        {!isPermanentFree && (
                        <button
                            onClick={handleStartChatAnalysis}
                            aria-label={t('chat_analysis')}
                            className="w-full py-2.5 px-4 rounded-lg text-[13px] font-medium transition-all duration-200 flex items-center justify-center gap-2 bg-slate-700 dark:bg-[#1e3a5f] hover:bg-slate-800 dark:hover:bg-[#264f78] text-white dark:text-[#cdd6e4] shadow-sm"
                        >
                            <span>💬</span> {t('chat_analysis')}
                        </button>
                        )}
                        {/* System Assistant and Customer Service - row layout */}
                        <div className="flex gap-2">
                            {/* Free mode: "开始分析" button - left of customer service */}
                            {isPermanentFree && (
                                <button
                                    onClick={handleFreeStartAnalysis}
                                    aria-label={t('chat_analysis')}
                                    className="flex-1 py-2 px-3 rounded-lg text-[12px] font-medium transition-all duration-200 flex items-center justify-center gap-1.5 border bg-slate-700 dark:bg-[#1e3a5f] hover:bg-slate-800 dark:hover:bg-[#264f78] text-white dark:text-[#cdd6e4] border-slate-600 dark:border-[#2e4f6e] shadow-sm"
                                >
                                    <span>💬</span> {t('chat_analysis')}
                                </button>
                            )}
                            {/* System Assistant button - hidden in permanent free mode (Req 4.3) */}
                            {freeChatThreadId && !isPermanentFree && (
                                <button
                                    className={`flex-1 py-2 px-3 rounded-lg text-[12px] font-medium transition-all duration-200 flex items-center justify-center gap-1.5 border ${selectedSessionId === freeChatThreadId ? 'bg-slate-200 dark:bg-[#5f1e1e] text-slate-700 dark:text-[#d69656] border-slate-300 dark:border-[#783026]' : 'bg-white/70 dark:bg-[#3a1f1f] hover:bg-slate-50 dark:hover:bg-[#4a2626] text-slate-500 dark:text-[#d69656] border-slate-200 dark:border-[#5f2e2e]'}`}
                                    onClick={() => {
                                        onSessionSelect(freeChatThreadId);
                                        if (!isChatOpen) {
                                            onToggleChat();
                                        }
                                    }}
                                    onContextMenu={(e) => {
                                        e.preventDefault();
                                        e.stopPropagation();
                                        handleSessionContextMenu(e, freeChatThreadId);
                                    }}
                                >
                                    <span>🤖</span> {t('free_chat')}
                                </button>
                            )}
                            <button
                                className="flex-1 py-2 px-3 rounded-lg text-[12px] font-medium transition-all duration-200 flex items-center justify-center gap-1.5 border bg-white/70 dark:bg-[#1f3a2a] hover:bg-slate-50 dark:hover:bg-[#264f3a] text-slate-500 dark:text-[#56d6a0] border-slate-200 dark:border-[#2e5f3e]"
                                onClick={handleServicePortalLogin}
                                disabled={serviceLoginLoading}
                                aria-label={t('customer_service')}
                            >
                                {serviceLoginLoading ? (
                                    <svg className="animate-spin w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                                    </svg>
                                ) : (
                                    <span>🎧</span>
                                )} {t('customer_service')}
                            </button>
                        </div>
                    </div>

                    {/* Historical Sessions - below Chat Analysis button */}
                    {isLoadingSessions ? (
                        <div className="historical-sessions-section" style={{ flex: 1, minHeight: 0 }}>
                            <div className="section-header">
                                <h3>{t('historical_sessions')}</h3>
                            </div>
                            <div className="loading">{t('loading_sessions')}</div>
                        </div>
                    ) : (
                        <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
                            <HistoricalSessionsSection
                                sessions={sessions.map(session => ({
                                    id: session.id,
                                    title: session.title,
                                    data_source_id: session.data_source_id,
                                    created_at: session.created_at,
                                    dataSourceName: sources?.find(s => s.id === session.data_source_id)?.name,
                                    is_replay_session: session.is_replay_session,
                                }))}
                                selectedId={selectedSessionId}
                                onSelect={onSessionSelect}
                                onContextMenu={handleSessionContextMenu}
                                onDelete={handleDeleteSession}
                                freeChatThreadId={freeChatThreadId}
                                isSessionLoading={isSessionAnalysisLoading}
                            />
                        </div>
                    )}

                    {/* Session Context Menu */}
                    {sessionContextMenu && (
                        <ChatThreadContextMenu
                            position={{ x: sessionContextMenu.x, y: sessionContextMenu.y }}
                            threadId={sessionContextMenu.sessionId}
                            onClose={() => setSessionContextMenu(null)}
                            onAction={handleSessionContextAction}
                            autoIntentUnderstanding={autoIntentUnderstanding}
                            isFreeChatThread={sessionContextMenu.sessionId === freeChatThreadId}
                            isReplaySession={sessionContextMenu.isReplaySession}
                            isGeneratingComprehensiveReport={isGeneratingReport}
                        />
                    )}

                    {/* Export Quick Analysis Pack Dialog */}
                    {exportPackThreadId && (
                        <ExportPackDialog
                            isOpen={true}
                            onClose={() => setExportPackThreadId(null)}
                            onConfirm={() => setExportPackThreadId(null)}
                            threadId={exportPackThreadId}
                        />
                    )}

                    {/* Import Quick Analysis Pack Dialog */}
                    {importPackDataSourceId && (
                        <ImportPackDialog
                            isOpen={true}
                            onClose={() => setImportPackDataSourceId(null)}
                            onConfirm={() => setImportPackDataSourceId(null)}
                            dataSourceId={importPackDataSourceId}
                        />
                    )}
                </>
            )}

            <AddDataSourceModal
                isOpen={isAddModalOpen}
                onClose={() => {
                    setIsAddModalOpen(false);
                    setPreSelectedDriverType(null);
                }}
                onSuccess={(newDataSource) => {
                    fetchSources();
                }}
                preSelectedDriverType={preSelectedDriverType}
            />

            <DeleteConfirmationModal
                isOpen={!!deleteTarget}
                sourceName={deleteTarget?.name || ''}
                onClose={() => setDeleteTarget(null)}
                onConfirm={confirmDelete}
            />

            <DeleteConfirmationModal
                isOpen={!!deleteSessionTarget}
                sourceName={deleteSessionTarget?.title || ''}
                onClose={() => setDeleteSessionTarget(null)}
                onConfirm={confirmDeleteSession}
                type="thread"
            />

            <NewChatModal
                isOpen={isNewChatModalOpen}
                dataSourceId={selectedId || (sources && sources.length === 1 ? sources[0].id : '') || ''}
                onClose={() => setIsNewChatModalOpen(false)}
                onSubmit={handleNewChatSubmit}
                onStartFreeChat={() => {
                    // 切换到已有的系统助手会话
                    if (freeChatThreadId) {
                        onSessionSelect(freeChatThreadId);
                    }
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

            <ConfirmationModal
                isOpen={showFreeNoDataSourcePrompt}
                title={t('no_data_source_free_prompt_title')}
                message={t('no_data_source_free_prompt_message')}
                confirmText={t('confirm')}
                cancelText={t('cancel')}
                onClose={() => setShowFreeNoDataSourcePrompt(false)}
                onConfirm={handleFreeNoDataSourceConfirm}
            />

            {contextMenu && (
                <SourceContextMenu
                    position={{ x: contextMenu.x, y: contextMenu.y }}
                    sourceId={contextMenu.sourceId}
                    sourceName={contextMenu.sourceName}
                    sourceType={contextMenu.sourceType}
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
                    onExploreData={() => {
                        EventsEmit('open-data-browser', {
                            sourceId: contextMenu.sourceId,
                            sourceName: contextMenu.sourceName
                        });
                    }}
                    onLoadPack={(dataSourceId) => {
                        setImportPackDataSourceId(dataSourceId);
                    }}
                    isFreeMode={isPermanentFree}
                />
            )}

            <OnboardingWizard
                isOpen={showOnboardingWizard}
                onClose={() => setShowOnboardingWizard(false)}
                isPermanentFree={isPermanentFree}
                onSelectPlatform={(platform) => {
                    setShowOnboardingWizard(false);
                    setPreSelectedDriverType(platform);
                    setIsAddModalOpen(true);
                }}
                onSelectSelfImport={() => {
                    setShowOnboardingWizard(false);
                    setPreSelectedDriverType(null);
                    setIsAddModalOpen(true);
                }}
            />


            <MemoryViewModal
                isOpen={!!memoryModalTarget}
                threadId={memoryModalTarget || ''}
                onClose={() => setMemoryModalTarget(null)}
            />

            <RenameSessionModal
                isOpen={!!renameSessionTarget}
                currentTitle={renameSessionTarget?.title || ''}
                threadId={renameSessionTarget?.id || ''}
                dataSourceId={renameSessionTarget?.dataSourceId || ''}
                dataSourceName={renameSessionTarget?.dataSourceName}
                onClose={() => setRenameSessionTarget(null)}
                onConfirm={handleRenameSession}
            />

            <ServiceAuthErrorDialog
                show={showServiceError}
                errorMessage={serviceErrorMessage}
                onRetry={() => {
                    setShowServiceError(false);
                    handleServicePortalLogin();
                }}
                onClose={() => setShowServiceError(false)}
            />
        </div>
    );
};

export default Sidebar;