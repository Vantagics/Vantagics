import React, { useState, useEffect, useRef } from 'react';
import { X, MessageSquare, Plus, Trash2, Send, Loader2, ChevronLeft, ChevronRight, Settings, Upload, Zap, XCircle } from 'lucide-react';
import { GetChatHistory, SaveChatHistory, SendMessage, DeleteThread, ClearHistory, GetDataSources, CreateChatThread, UpdateThreadTitle, ExportSessionHTML, AssetizeSession, OpenSessionResultsDirectory, CancelAnalysis, GetConfig } from '../../wailsjs/go/main/App';
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import MessageBubble from './MessageBubble';
import { useLanguage } from '../i18n';
import DeleteConfirmationModal from './DeleteConfirmationModal';
import ChatThreadContextMenu from './ChatThreadContextMenu';
import MemoryViewModal from './MemoryViewModal';
import CancelConfirmationModal from './CancelConfirmationModal';

// Progress update type from backend
interface ProgressUpdate {
    stage: string;
    progress: number;
    message: string;
    step: number;
    total: number;
    tool_name?: string;
    tool_output?: string;
}

interface ChatSidebarProps {
    isOpen: boolean;
    onClose: () => void;
}

const ChatSidebar: React.FC<ChatSidebarProps> = ({ isOpen, onClose }) => {
    const { t } = useLanguage();
    const [threads, setThreads] = useState<main.ChatThread[]>([]);
    const [activeThreadId, setActiveThreadId] = useState<string | null>(null);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [loadingThreadId, setLoadingThreadId] = useState<string | null>(null); // 跟踪哪个会话正在加载
    const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
    const [showClearConfirm, setShowClearConfirm] = useState(false);
    const [dataSources, setDataSources] = useState<any[]>([]);
    const [deleteThreadTarget, setDeleteThreadTarget] = useState<{ id: string, title: string } | null>(null);
    const [memoryModalTarget, setMemoryModalTarget] = useState<string | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, threadId: string } | null>(null);
    const [blankAreaContextMenu, setBlankAreaContextMenu] = useState<{ x: number, y: number } | null>(null);
    const [progress, setProgress] = useState<ProgressUpdate | null>(null);
    const [showCancelConfirm, setShowCancelConfirm] = useState(false);

    // Resizing State
    const [sidebarWidth, setSidebarWidth] = useState(650);
    const [historyWidth, setHistoryWidth] = useState(208);
    const [isResizingSidebar, setIsResizingSidebar] = useState(false);
    const [isResizingHistory, setIsResizingHistory] = useState(false);

    const messagesEndRef = useRef<HTMLDivElement>(null);
    const blankMenuRef = useRef<HTMLDivElement>(null);

    const activeThread = threads?.find(t => t.id === activeThreadId);
    // Find associated data source name
    const activeDataSource = dataSources?.find(ds => ds.id === activeThread?.data_source_id);

    useEffect(() => {
        if (isOpen) {
            loadThreads();
            loadDataSources();
        }
    }, [isOpen]);

    // Emit session-switched event when active thread changes
    useEffect(() => {
        if (activeThreadId) {
            EventsEmit('session-switched', activeThreadId);
        }
    }, [activeThreadId]);

    // Resizing Effect
    useEffect(() => {
        const handleMouseMove = (e: MouseEvent) => {
            if (isResizingSidebar) {
                const newWidth = window.innerWidth - e.clientX;
                if (newWidth > 400 && newWidth < window.innerWidth - 100) {
                    setSidebarWidth(newWidth);
                }
            } else if (isResizingHistory) {
                const sidebarLeft = window.innerWidth - sidebarWidth;
                const newWidth = e.clientX - sidebarLeft;
                if (newWidth > 150 && newWidth < sidebarWidth - 300) {
                    setHistoryWidth(newWidth);
                }
            }
        };

        const handleMouseUp = () => {
            setIsResizingSidebar(false);
            setIsResizingHistory(false);
            document.body.style.cursor = 'default';
        };

        if (isResizingSidebar || isResizingHistory) {
            window.addEventListener('mousemove', handleMouseMove);
            window.addEventListener('mouseup', handleMouseUp);
        }

        return () => {
            window.removeEventListener('mousemove', handleMouseMove);
            window.removeEventListener('mouseup', handleMouseUp);
        };
    }, [isResizingSidebar, isResizingHistory, sidebarWidth]);

    // Track pending chat creation to prevent duplicates
    const pendingChatRef = useRef<string | null>(null);
    const lastMessageRef = useRef<string | null>(null); // 新增：跟踪最后发送的消息
    const pendingActionRef = useRef<string | null>(null); // 新增：跟踪正在处理的操作

    // Store function refs to use in event handlers without causing re-registration
    // These will be updated after the functions are defined
    const handleCreateThreadRef = useRef<((dataSourceId?: string, title?: string) => Promise<main.ChatThread | null>) | null>(null);
    const handleSendMessageRef = useRef<((text?: string, explicitThreadId?: string, explicitThread?: main.ChatThread) => Promise<void>) | null>(null);

    // Refs to store latest state values for event handlers (avoid closure issues)
    const threadsRef = useRef<main.ChatThread[]>([]);
    const activeThreadIdRef = useRef<string | null>(null);

    // Keep refs in sync with state
    useEffect(() => {
        threadsRef.current = threads || [];
    }, [threads]);

    useEffect(() => {
        activeThreadIdRef.current = activeThreadId;
    }, [activeThreadId]);

    // Listen for new chat creation - separate useEffect with empty deps to prevent duplicate listeners
    useEffect(() => {
        const unsubscribeStart = EventsOn('start-new-chat', async (data: any) => {
            console.log('[ChatSidebar] start-new-chat event received:', data);

            // Use sessionName as a key to prevent duplicate processing
            const chatKey = `${data.dataSourceId}-${data.sessionName}`;
            if (pendingChatRef.current === chatKey) {
                console.log('[ChatSidebar] Ignoring duplicate start-new-chat event:', chatKey);
                return;
            }
            pendingChatRef.current = chatKey;

            try {
                const thread = handleCreateThreadRef.current ? await handleCreateThreadRef.current(data.dataSourceId, data.sessionName) : null;
                if (thread) {
                    console.log('[ChatSidebar] Thread created, preparing to send initial message:', thread.id);

                    // Small delay to ensure state updates (activeThreadId) have propagated
                    setTimeout(async () => {
                        // Get current language from config to ensure consistency
                        let prompt = "Give me some analysis suggestions for this data source.";
                        try {
                            const config = await GetConfig();
                            if (config.language === '简体中文') {
                                prompt = "请给出一些本数据源的分析建议。";
                            }
                        } catch (e) {
                            console.error("Failed to get config for language:", e);
                        }

                        // 防止重复发送相同消息 - 使用更强的检查
                        const messageKey = `${thread.id}-${prompt}`;
                        const currentTime = Date.now();

                        // 检查是否在短时间内发送了相同的消息
                        if (lastMessageRef.current === messageKey) {
                            console.log('[ChatSidebar] Ignoring duplicate message send (exact match):', messageKey);
                            return;
                        }

                        // 额外检查：检查线程中是否已经存在相同的消息
                        const existingMessages = thread.messages || [];
                        const hasDuplicateMessage = existingMessages.some(msg =>
                            msg.role === 'user' &&
                            msg.content === prompt &&
                            (currentTime - (msg.timestamp * 1000)) < 10000 // 10秒内的重复消息
                        );

                        if (hasDuplicateMessage) {
                            console.log('[ChatSidebar] Ignoring duplicate message send (found in thread):', prompt);
                            return;
                        }

                        lastMessageRef.current = messageKey;
                        console.log('[ChatSidebar] Sending initial message:', prompt);

                        if (handleSendMessageRef.current) {
                            handleSendMessageRef.current(prompt, thread.id, thread);
                        }

                        // 清除消息标记
                        setTimeout(() => {
                            if (lastMessageRef.current === messageKey) {
                                lastMessageRef.current = null;
                            }
                        }, 5000); // 增加到5秒
                    }, 100);
                }
            } finally {
                // Clear the pending flag after a delay to allow the operation to complete
                setTimeout(() => {
                    if (pendingChatRef.current === chatKey) {
                        pendingChatRef.current = null;
                    }
                }, 1000); // 增加到1秒
            }
        });

        return () => {
            if (unsubscribeStart) unsubscribeStart();
        };
    }, []); // Empty deps - only register once

    useEffect(() => {
        // Listen for open chat request from Sidebar context menu
        const unsubscribeOpen = EventsOn('open-chat', (thread: main.ChatThread) => {
            // Ensure we have the latest threads
            loadThreads().then(() => {
                setActiveThreadId(thread.id);
            });
        });

        // Listen for thread updates (e.g. background analysis errors)
        const unsubscribeUpdate = EventsOn('thread-updated', (threadId: string) => {
            loadThreads();
        });

        // Listen for loading state from backend (for async tasks like suggestions)
        const unsubscribeLoading = EventsOn('chat-loading', (data: any) => {
            if (typeof data === 'boolean') {
                // 向后兼容：如果是布尔值，应用到当前活动会话
                if (activeThreadId) {
                    setIsLoading(data);
                    if (data) {
                        setLoadingThreadId(activeThreadId);
                    } else {
                        setLoadingThreadId(null);
                    }
                }
            } else if (data && typeof data === 'object') {
                // 新格式：包含threadId的对象
                if (data.threadId === activeThreadId) {
                    setIsLoading(data.loading);
                    if (data.loading) {
                        setLoadingThreadId(data.threadId);
                    } else {
                        setLoadingThreadId(null);
                    }
                }
            }
        });

        // Listen for analysis progress updates
        const unsubscribeProgress = EventsOn('analysis-progress', (update: ProgressUpdate) => {
            // 只有当前活动会话正在加载时才显示进度
            if (loadingThreadId && loadingThreadId === activeThreadId) {
                setProgress(update);
                // Clear progress when complete
                if (update.stage === 'complete') {
                    setTimeout(() => setProgress(null), 1000);
                }
            }
        });

        // Listen for send message request (triggered after sidebar is open)
        const unsubscribeSendMessage = EventsOn('chat-send-message', (text: string) => {
            console.log('[ChatSidebar] chat-send-message event received:', text);
            console.log('[ChatSidebar] isOpen state:', isOpen);
            console.log('[ChatSidebar] activeThreadId:', activeThreadId);
            console.log('[ChatSidebar] threads count:', threads?.length || 0);

            // Only handle if sidebar is open and initialized
            if (isOpen) {
                console.log('[ChatSidebar] Calling handleSendMessage');
                handleSendMessage(text);
            } else {
                console.log('[ChatSidebar] Sidebar not open, ignoring message');
            }
        });

        // Listen for send message request in specific session (for LLM insights)
        const unsubscribeSendMessageInSession = EventsOn('chat-send-message-in-session', (data: any) => {
            console.log('[ChatSidebar] chat-send-message-in-session event received:', data);
            console.log('[ChatSidebar] isOpen state:', isOpen);

            // 使用 refs 获取最新的状态值（避免闭包问题）
            const currentThreads = threadsRef.current;
            const currentActiveThreadId = activeThreadIdRef.current;

            console.log('[ChatSidebar] activeThreadId (from ref):', currentActiveThreadId);
            console.log('[ChatSidebar] threads count (from ref):', currentThreads?.length || 0);
            console.log('[ChatSidebar] threadId:', data.threadId, 'userMessageId:', data.userMessageId);

            // Priority 1: Use directly provided threadId (most reliable)
            if (isOpen && data && data.threadId) {
                const targetThread = currentThreads?.find(t => t.id === data.threadId);
                if (targetThread) {
                    console.log('[ChatSidebar] ✅ Found thread by threadId:', targetThread.id);
                    if (targetThread.id !== currentActiveThreadId) {
                        setActiveThreadId(targetThread.id);
                    }
                    handleSendMessage(data.text, targetThread.id, targetThread);
                    return;
                }
            }

            // Priority 2: Use userMessageId lookup
            if (isOpen && data && data.userMessageId) {
                // 详细调试：检查所有线程和消息
                console.log('[ChatSidebar] Searching for userMessageId in all threads:');
                currentThreads?.forEach((thread, threadIndex) => {
                    console.log(`[ChatSidebar] Thread ${threadIndex}: ${thread.id} (${thread.messages?.length || 0} messages)`);
                    thread.messages?.forEach((msg, msgIndex) => {
                        if (msg.role === 'user') {
                            console.log(`[ChatSidebar]   User message ${msgIndex}: ${msg.id} - "${msg.content?.substring(0, 50)}..."`);
                        }
                    });
                });

                // 找到包含指定用户消息的会话
                const targetThread = currentThreads?.find(thread =>
                    thread.messages?.some(msg =>
                        msg.role === 'user' && msg.id === data.userMessageId
                    )
                );

                if (targetThread) {
                    console.log('[ChatSidebar] ✅ Found target thread for user message:', targetThread.id);
                    console.log('[ChatSidebar] Target thread data source:', targetThread.data_source_id);
                    console.log('[ChatSidebar] Target thread messages count:', targetThread.messages?.length || 0);

                    // 切换到目标会话（如果需要）
                    if (targetThread.id !== currentActiveThreadId) {
                        console.log('[ChatSidebar] Switching to target thread:', targetThread.id);
                        setActiveThreadId(targetThread.id);
                    }

                    // 直接发送消息到目标会话，不依赖状态更新
                    console.log('[ChatSidebar] 🚀 Sending message directly to target thread:', targetThread.id);
                    console.log('[ChatSidebar] Message text:', data.text?.substring(0, 100));
                    handleSendMessage(data.text, targetThread.id, targetThread);
                } else {
                    console.log('[ChatSidebar] ❌ Target thread not found for userMessageId:', data.userMessageId);
                    console.log('[ChatSidebar] Available threads:', currentThreads?.map(t => ({
                        id: t.id,
                        messageCount: t.messages?.length || 0,
                        userMessageIds: t.messages?.filter(m => m.role === 'user').map(m => m.id) || []
                    })));

                    // 回退到当前活动会话，而不是创建新会话
                    // 这确保智能洞察的分析请求在同一个会话中进行
                    if (currentActiveThreadId) {
                        const activeThread = currentThreads?.find(t => t.id === currentActiveThreadId);
                        if (activeThread) {
                            console.log('[ChatSidebar] Falling back to active thread:', currentActiveThreadId);
                            handleSendMessage(data.text, currentActiveThreadId, activeThread);
                        } else {
                            console.log('[ChatSidebar] Active thread not found in threads list, using activeThreadId directly');
                            handleSendMessage(data.text, currentActiveThreadId);
                        }
                    } else {
                        console.log('[ChatSidebar] No active thread, cannot send message in session');
                        // 不创建新会话，只记录错误
                        console.error('[ChatSidebar] Cannot find session for insight analysis');
                    }
                }
            } else {
                console.log('[ChatSidebar] Sidebar not open or invalid data, ignoring message');
                console.log('[ChatSidebar] isOpen:', isOpen, 'data:', data, 'userMessageId:', data?.userMessageId);
            }
        });

        return () => {
            if (unsubscribeOpen) unsubscribeOpen();
            if (unsubscribeUpdate) unsubscribeUpdate();
            if (unsubscribeLoading) unsubscribeLoading();
            if (unsubscribeProgress) unsubscribeProgress();
            if (unsubscribeSendMessage) unsubscribeSendMessage();
            if (unsubscribeSendMessageInSession) unsubscribeSendMessageInSession();
        };
    }, [threads]);

    // 监听活动会话变化，自动显示第一个分析结果
    useEffect(() => {
        if (activeThreadId && threads) {
            const activeThread = threads.find(t => t.id === activeThreadId);
            if (activeThread && activeThread.messages) {
                // 找到第一个有chart_data的用户消息（分析请求）
                const firstAnalysisMessage = activeThread.messages.find(msg =>
                    msg.role === 'user' && msg.chart_data
                );

                if (firstAnalysisMessage) {
                    console.log("[ChatSidebar] Auto-displaying first analysis result for thread:", activeThreadId);
                    console.log("[ChatSidebar] First analysis message:", firstAnalysisMessage.id);

                    // 自动触发显示该消息的分析结果
                    setTimeout(() => {
                        EventsEmit('user-message-clicked', {
                            messageId: firstAnalysisMessage.id,
                            content: firstAnalysisMessage.content,
                            chartData: firstAnalysisMessage.chart_data
                        });
                    }, 100); // 小延迟确保UI更新完成
                } else {
                    console.log("[ChatSidebar] No analysis results found in thread:", activeThreadId);
                    // 如果没有分析结果，清空仪表盘显示系统默认内容
                    EventsEmit('clear-dashboard');
                }
            }
        }
    }, [activeThreadId, threads]);

    useEffect(() => {
        const scrollToBottom = () => {
            messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
        };

        // Use a small timeout to ensure DOM has updated
        const timeoutId = setTimeout(scrollToBottom, 100);
        return () => clearTimeout(timeoutId);
    }, [activeThread?.messages, isLoading]);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (blankMenuRef.current && !blankMenuRef.current.contains(event.target as Node)) {
                setBlankAreaContextMenu(null);
            }
        };
        if (blankAreaContextMenu) {
            document.addEventListener('mousedown', handleClickOutside);
            return () => document.removeEventListener('mousedown', handleClickOutside);
        }
    }, [blankAreaContextMenu]);

    const loadThreads = async () => {
        try {
            const history = await GetChatHistory();
            setThreads(history);
            if (history && history.length > 0 && !activeThreadId) {
                setActiveThreadId(history[0].id);
            }
        } catch (err) {
            console.error('Failed to load chat history:', err);
        }
    };

    const loadDataSources = async () => {
        try {
            const ds = await GetDataSources();
            setDataSources(ds || []);
        } catch (err) {
            console.error('Failed to load data sources:', err);
        }
    };

    const handleCreateThread = async (dataSourceId?: string, title?: string) => {
        try {
            const newThread = await CreateChatThread(dataSourceId || '', title || 'New Chat');
            setThreads(prev => [newThread, ...(prev || [])]);
            setActiveThreadId(newThread.id);
            return newThread;
        } catch (err: any) {
            console.error('Failed to create thread:', err);

            // Check if error is about active session conflict
            const errorMsg = err?.message || String(err);
            if (errorMsg.includes('分析会话进行中') || errorMsg.includes('active analysis')) {
                // Show user-friendly error message via MessageModal
                EventsEmit('show-message-modal', {
                    type: 'warning',
                    title: t('session_conflict_title') || '会话冲突',
                    message: errorMsg
                });
            } else {
                // Generic error
                EventsEmit('show-message-modal', {
                    type: 'error',
                    title: t('create_session_failed') || '创建会话失败',
                    message: errorMsg
                });
            }

            return null;
        }
    };

    const handleDeleteThread = (id: string, e: React.MouseEvent) => {
        e.stopPropagation();
        const thread = threads?.find(t => t.id === id);
        if (thread) {
            setDeleteThreadTarget({ id: thread.id, title: thread.title });
        }
    };

    const confirmDeleteThread = async () => {
        if (!deleteThreadTarget) return;
        try {
            // 如果删除的会话正在进行分析，先取消分析
            if (loadingThreadId === deleteThreadTarget.id) {
                console.log('[DELETE-THREAD] Cancelling ongoing analysis before deletion');
                try {
                    await CancelAnalysis();
                    // 等待取消生效
                    await new Promise(resolve => setTimeout(resolve, 200));
                } catch (cancelErr) {
                    console.error('Failed to cancel analysis:', cancelErr);
                }
                // 重置loading状态
                setIsLoading(false);
                setLoadingThreadId(null);
                setProgress(null);
            }

            await DeleteThread(deleteThreadTarget.id);
            const updatedThreads = threads?.filter(t => t.id !== deleteThreadTarget.id) || [];
            setThreads(updatedThreads);

            // 如果删除的是当前活跃的会话
            if (activeThreadId === deleteThreadTarget.id) {
                if (updatedThreads.length > 0) {
                    // 如果还有其他会话，选择第一个
                    setActiveThreadId(updatedThreads[0].id);
                } else {
                    // 如果没有剩余会话，清空活跃会话ID并通知App清空仪表盘
                    setActiveThreadId(null);
                    EventsEmit('clear-dashboard');
                }
            }
            setDeleteThreadTarget(null);
        } catch (err) {
            console.error('Failed to delete thread:', err);
        }
    };

    const handleContextMenu = (e: React.MouseEvent, threadId: string) => {
        e.preventDefault();
        setContextMenu({ x: e.clientX, y: e.clientY, threadId });
    };

    const handleBlankAreaContextMenu = (e: React.MouseEvent) => {
        // Only show menu if clicking directly on the container (blank area)
        if (e.target === e.currentTarget) {
            e.preventDefault();
            setBlankAreaContextMenu({ x: e.clientX, y: e.clientY });
        }
    };

    const handleImportAction = () => {
        console.log('Import action triggered');
        alert('Import functionality (Not implemented)');
        setBlankAreaContextMenu(null);
    };

    const handleContextAction = async (action: 'export' | 'assetize' | 'view_memory' | 'view_results_directory', threadId: string) => {
        console.log(`Action ${action} on thread ${threadId}`);
        if (action === 'view_memory') {
            setMemoryModalTarget(threadId);
        } else if (action === 'export') {
            try {
                await ExportSessionHTML(threadId);
            } catch (e) {
                console.error("Export failed:", e);
            }
        } else if (action === 'assetize') {
            try {
                await AssetizeSession(threadId);
            } catch (e) {
                console.error("Assetize failed:", e);
            }
        } else if (action === 'view_results_directory') {
            try {
                await OpenSessionResultsDirectory(threadId);
            } catch (e) {
                console.error("Open results directory failed:", e);
                // Show error message to user
                EventsEmit('show-message-modal', {
                    type: 'error',
                    title: '打开目录失败',
                    message: String(e)
                });
            }
        }
    };

    const handleSendMessage = async (text?: string, explicitThreadId?: string, explicitThread?: main.ChatThread) => {
        const msgText = text || input;
        console.log('[ChatSidebar] 🔥 handleSendMessage called with:', {
            text: msgText?.substring(0, 50),
            explicitThreadId,
            hasExplicitThread: !!explicitThread,
            explicitThreadDataSource: explicitThread?.data_source_id,
            explicitThreadMessagesCount: explicitThread?.messages?.length || 0,
            isLoading,
            activeThreadId
        });

        // If explicitThread is passed (auto-start), ignore isLoading check to ensure it fires.
        if (!msgText.trim() || (isLoading && !explicitThread)) {
            console.log('[ChatSidebar] ❌ handleSendMessage early return:', {
                emptyText: !msgText.trim(),
                isLoadingAndNoExplicitThread: isLoading && !explicitThread
            });
            return;
        }

        // 防止重复的操作请求（特别是按钮点击）
        const actionKey = `${explicitThread?.id || activeThreadId || 'no-thread'}-${msgText}`;
        const currentTime = Date.now();

        if (pendingActionRef.current === actionKey) {
            console.log('[ChatSidebar] ⏸️ Ignoring duplicate action (pending):', msgText.substring(0, 50));
            return;
        }
        pendingActionRef.current = actionKey;

        // 设置清除标记的定时器
        const clearActionFlag = () => {
            if (pendingActionRef.current === actionKey) {
                pendingActionRef.current = null;
            }
        };
        const timeoutId = setTimeout(clearActionFlag, 2000); // 增加到2秒

        let currentThreads = [...threads];
        let currentThread = explicitThread;

        console.log('[ChatSidebar] 🧵 Thread selection logic:', {
            hasExplicitThread: !!explicitThread,
            explicitThreadId,
            activeThreadId,
            threadsCount: currentThreads.length,
            explicitThreadIdFromObject: explicitThread?.id
        });

        if (currentThread) {
            console.log('[ChatSidebar] ✅ Using explicit thread:', currentThread.id, 'dataSource:', currentThread.data_source_id);
            console.log('[ChatSidebar] Explicit thread messages count:', currentThread.messages?.length || 0);
            const threadId = currentThread.id;
            const idx = currentThreads.findIndex(t => t.id === threadId);
            if (idx === -1) {
                currentThreads = [currentThread, ...(currentThreads || [])];
                console.log('[ChatSidebar] Added explicit thread to threads list');
            } else {
                currentThreads[idx] = currentThread;
                console.log('[ChatSidebar] Updated explicit thread in threads list');
            }
        } else {
            const targetId = explicitThreadId || activeThreadId;
            console.log('[ChatSidebar] 🔍 Looking for thread with ID:', targetId);
            currentThread = currentThreads?.find(t => t.id === targetId);
            if (currentThread) {
                console.log('[ChatSidebar] ✅ Found target thread:', currentThread.id, 'dataSource:', currentThread.data_source_id);
            } else {
                console.log('[ChatSidebar] ❌ Target thread not found');
            }
        }

        // If no active thread, create one first (only if no explicit thread is provided)
        if (!currentThread && !explicitThread && !explicitThreadId) {
            console.log('[ChatSidebar] 🆕 No current thread and no explicit thread, creating new thread');
            try {
                const title = msgText.slice(0, 30);
                const newThread = await CreateChatThread('', title);
                currentThread = newThread;
                currentThreads = [newThread, ...(currentThreads || [])];
                setThreads(prev => [newThread, ...(prev || [])]);
                setActiveThreadId(newThread.id);
                console.log('[ChatSidebar] ✅ Created new thread:', newThread.id);
            } catch (err) {
                console.error("Failed to create thread on send:", err);
                return;
            }
        } else if (!currentThread && (explicitThreadId || explicitThread)) {
            console.error('[ChatSidebar] 💥 Target thread not found:', explicitThreadId || explicitThread?.id);
            console.error('[ChatSidebar] Available threads:', currentThreads.map(t => t.id));
            return;
        }

        if (!currentThread) {
            console.error('[ChatSidebar] 💥 No current thread available, aborting');
            return;
        }

        console.log('[ChatSidebar] 🎯 Final selected thread:', currentThread.id, 'dataSource:', currentThread.data_source_id);

        // Store thread ID to avoid TypeScript errors after awaits
        const currentThreadId = currentThread.id;

        // 检查是否已经存在相同内容的消息（防止重复发送）
        const existingMessages = currentThread.messages || [];
        const recentMessages = existingMessages.slice(-5); // 检查最近5条消息（增加检查范围）
        const isDuplicate = recentMessages.some(msg =>
            msg.role === 'user' &&
            msg.content === msgText &&
            (currentTime - (msg.timestamp * 1000)) < 10000 // 增加到10秒内的重复消息
        );

        if (isDuplicate) {
            console.log('[ChatSidebar] Ignoring duplicate message (found in recent messages):', msgText.substring(0, 50));
            // 清除操作标记
            if (pendingActionRef.current === actionKey) {
                pendingActionRef.current = null;
            }
            clearTimeout(timeoutId);
            return;
        }

        const userMsg = new main.ChatMessage();
        userMsg.id = Date.now().toString();
        userMsg.role = 'user';
        userMsg.content = msgText;
        userMsg.timestamp = Math.floor(Date.now() / 1000);

        if (!currentThread.messages) currentThread.messages = [];
        // Use immutable update for messages to ensure React detects change
        currentThread.messages = [...(currentThread.messages || []), userMsg];

        if (currentThread.messages && currentThread.messages.length === 1 && currentThread.title === 'New Chat') {
            const newTitle = msgText.slice(0, 30) + (msgText.length > 30 ? '...' : '');
            try {
                const uniqueTitle = await UpdateThreadTitle(currentThreadId, newTitle);
                currentThread.title = uniqueTitle;
            } catch (err) {
                console.error("Failed to rename thread:", err);
            }
        }

        // Optimistic update using functional state logic
        // We must calculate the new state synchronously to ensure SaveChatHistory gets the correct data
        const threadIndex = currentThreads.findIndex(t => t.id === currentThreadId);
        const updatedThreads = [...(currentThreads || [])];

        // Clone the thread with the new user message
        const threadClone = main.ChatThread.createFrom({ ...currentThread!, messages: [...(currentThread!.messages || [])] });

        if (threadIndex !== -1) {
            updatedThreads[threadIndex] = threadClone;
        } else {
            updatedThreads.unshift(threadClone);
        }

        // Update UI immediately
        setThreads(updatedThreads);

        setInput('');
        setIsLoading(true);
        setLoadingThreadId(currentThreadId); // 记录正在加载的会话ID

        try {
            // Await save before sending message to prevent race condition
            // Now passing the explicitly calculated updatedThreads
            await SaveChatHistory(updatedThreads);

            const response = await SendMessage(currentThreadId, msgText, userMsg.id);

            // CRITICAL: Reload threads from backend to get chart_data attached by backend
            // The backend's attachChartToUserMessage modifies the user message after SendMessage
            // If we don't reload, we'll overwrite that modification when saving the assistant response
            const reloadedThreads = await GetChatHistory();

            const assistantMsg = new main.ChatMessage();
            assistantMsg.id = (Date.now() + 1).toString();
            assistantMsg.role = 'assistant';
            assistantMsg.content = response;
            assistantMsg.timestamp = Math.floor(Date.now() / 1000);

            // Find the reloaded thread (includes backend modifications like chart_data)
            const reloadedThread = reloadedThreads?.find(t => t.id === currentThreadId);

            if (reloadedThread) {
                // Add assistant message to reloaded thread (which has chart_data from backend)
                reloadedThread.messages = [...(reloadedThread.messages || []), assistantMsg];

                // Update state with reloaded thread
                setThreads(prevThreads => {
                    const index = (prevThreads || []).findIndex(t => t.id === currentThreadId);
                    const newThreads = [...(prevThreads || [])];
                    if (index !== -1) {
                        newThreads[index] = reloadedThread;
                    } else {
                        newThreads.unshift(reloadedThread);
                    }
                    return newThreads;
                });

                // Save with backend modifications preserved
                const threadsToSaveWithResponse = reloadedThreads.map(t =>
                    t.id === reloadedThread.id ? reloadedThread : t
                );
                await SaveChatHistory(threadsToSaveWithResponse);
            } else {
                // Fallback to old behavior if thread not found (shouldn't happen)
                setThreads(prevThreads => {
                    const index = (prevThreads || []).findIndex(t => t.id === currentThreadId);
                    if (index !== -1) {
                        const newThreads = [...(prevThreads || [])];
                        const updatedThread = main.ChatThread.createFrom({
                            ...newThreads[index],
                            messages: [...(newThreads[index].messages || []), assistantMsg]
                        });
                        newThreads[index] = updatedThread;
                        SaveChatHistory(newThreads).catch(err => console.error("Failed to save assistant response:", err));
                        return newThreads;
                    }
                    return prevThreads || [];
                });
            }

        } catch (error) {
            console.error(error);
            const errorMsg = new main.ChatMessage();
            errorMsg.id = (Date.now() + 1).toString();
            errorMsg.role = 'assistant';

            let errorText = 'Sorry, I encountered an error. Please check your connection and API keys.';
            if (typeof error === 'string') {
                errorText = `Error: ${error}`;
            } else if (error instanceof Error) {
                errorText = `Error: ${error.message}`;
            }
            errorMsg.content = errorText;

            errorMsg.timestamp = Math.floor(Date.now() / 1000);
            currentThread.messages = [...(currentThread.messages || []), errorMsg];

            setThreads(prevThreads => {
                const index = (prevThreads || []).findIndex(t => t.id === currentThread!.id);
                if (index !== -1) {
                    const newThreads = [...(prevThreads || [])];
                    newThreads[index] = currentThread!;
                    return newThreads;
                }
                return prevThreads || [];
            });
        } finally {
            clearTimeout(timeoutId); // 清除定时器
            setIsLoading(false);
            setLoadingThreadId(null); // 清除加载会话ID
            setProgress(null);
            // 清除操作标记
            if (pendingActionRef.current === actionKey) {
                pendingActionRef.current = null;
            }
        }
    };

    // Update refs on every render to ensure they always have the latest function references
    // This is critical for the start-new-chat event listener which has empty dependencies
    handleCreateThreadRef.current = handleCreateThread;
    handleSendMessageRef.current = handleSendMessage;

    // 处理会话切换
    const handleThreadSwitch = (threadId: string) => {
        setActiveThreadId(threadId);
        // 分析结果的显示由useEffect自动处理
    };

    const handleUserMessageClick = (message: main.ChatMessage) => {
        // Debug logging
        console.log("[ChatSidebar] User message clicked:", message.id);
        console.log("[ChatSidebar] Message content:", message.content?.substring(0, 100));
        console.log("[ChatSidebar] Has chart_data:", !!message.chart_data);
        console.log("[ChatSidebar] chart_data object:", message.chart_data);
        if (message.chart_data) {
            console.log("[ChatSidebar] chart_data.charts:", message.chart_data.charts);
            console.log("[ChatSidebar] Number of charts:", message.chart_data.charts?.length || 0);
        }

        // Emit event with message data for dashboard update
        EventsEmit('user-message-clicked', {
            messageId: message.id,
            content: message.content,
            chartData: message.chart_data
        });
    };


    const handleClearHistory = () => {
        setShowClearConfirm(true);
    };

    const confirmClearHistory = async () => {
        try {
            await ClearHistory();
            setThreads([]);
            setActiveThreadId(null);
            setShowClearConfirm(false);
        } catch (err) {
            console.error('Failed to clear history:', err);
            setShowClearConfirm(false);
        }
    };

    const cancelClearHistory = () => {
        setShowClearConfirm(false);
    };

    const handleCancelAnalysis = () => {
        setShowCancelConfirm(true);
    };

    const confirmCancelAnalysis = async () => {
        try {
            await CancelAnalysis();
            setShowCancelConfirm(false);
            setIsLoading(false);
            setLoadingThreadId(null); // 清除加载会话ID
            setProgress(null);
        } catch (err) {
            console.error('Failed to cancel analysis:', err);
            setShowCancelConfirm(false);
        }
    };

    const cancelCancelAnalysis = () => {
        setShowCancelConfirm(false);
    };

    return (
        <>


            <div
                data-testid="chat-sidebar"
                style={{ width: sidebarWidth }}
                className={`fixed inset-y-0 right-0 bg-white shadow-2xl transform transition-transform duration-300 ease-in-out z-50 flex overflow-hidden border-l border-slate-200 ${isOpen ? 'translate-x-0' : 'translate-x-full'}`}
            >
                {/* Sidebar Resizer (Left Edge) */}
                <div
                    className="absolute left-0 top-0 bottom-0 w-1 hover:bg-blue-400 cursor-col-resize z-[60] transition-colors"
                    onMouseDown={() => { setIsResizingSidebar(true); document.body.style.cursor = 'col-resize'; }}
                />

                {/* Thread List Sidebar */}
                <div
                    style={{ width: isSidebarCollapsed ? 0 : historyWidth }}
                    className="bg-slate-50 border-r border-slate-200 flex flex-col transition-all duration-300 overflow-hidden relative flex-shrink-0"
                >
                    <div className="p-4 border-b border-slate-200 flex items-center justify-between bg-white/50 backdrop-blur-sm sticky top-0 z-10"
                    >
                        <span className="font-bold text-slate-900 text-[11px] uppercase tracking-[0.1em]">{t('history')}</span>
                        <div className="w-4" />
                    </div>

                    <div
                        className="flex-1 overflow-y-auto p-2 space-y-1.5 scrollbar-hide"
                        onContextMenu={handleBlankAreaContextMenu}
                    >
                        {threads?.map(thread => {
                            const threadDataSource = dataSources?.find(ds => ds.id === thread.data_source_id);
                            return (
                                <div
                                    key={thread.id}
                                    onClick={() => handleThreadSwitch(thread.id)}
                                    onContextMenu={(e) => handleContextMenu(e, thread.id)}
                                    className={`group flex items-center justify-between p-2.5 rounded-xl cursor-pointer text-xs transition-all border ${activeThreadId === thread.id
                                        ? 'bg-white border-blue-200 text-blue-700 shadow-sm ring-1 ring-blue-100'
                                        : 'text-slate-600 hover:bg-white hover:border-slate-200 border-transparent'
                                        }`}
                                >
                                    <div className="flex items-center gap-2.5 truncate pr-1">
                                        <MessageSquare className={`w-4 h-4 flex-shrink-0 ${activeThreadId === thread.id ? 'text-blue-500' : 'text-slate-400'}`} />
                                        <span className="truncate leading-tight">
                                            {thread.title}
                                            {threadDataSource && (
                                                <span className="text-slate-400 font-normal ml-1">
                                                    ({threadDataSource.name})
                                                </span>
                                            )}
                                        </span>
                                    </div>
                                    <button
                                        onClick={(e) => handleDeleteThread(thread.id, e)}
                                        className="opacity-0 group-hover:opacity-100 p-1.5 hover:text-red-500 transition-all rounded-lg hover:bg-red-50 text-slate-400"
                                    >
                                        <Trash2 className="w-3.5 h-3.5" />
                                    </button>
                                </div>
                            );
                        })}
                        {(!threads || threads.length === 0) && (
                            <div className="text-center py-12 px-4">
                                <div className="w-10 h-10 bg-slate-100 rounded-full flex items-center justify-center mx-auto mb-3">
                                    <MessageSquare className="w-5 h-5 text-slate-300" />
                                </div>
                                <p className="text-[10px] text-slate-400 font-medium">No threads yet</p>
                            </div>
                        )}
                    </div>

                    <div className="p-3 border-t border-slate-200 bg-white/50">
                        <button
                            onClick={handleClearHistory}
                            className="w-full flex items-center justify-center gap-2 py-2.5 text-[10px] font-bold text-slate-500 hover:text-red-600 transition-colors rounded-xl hover:bg-red-50"
                        >
                            <Trash2 className="w-3.5 h-3.5" />
                            {t('clear_history')}
                        </button>
                    </div>

                    {/* History Resizer (Right Edge of History Panel) */}
                    <div
                        className="absolute right-0 top-0 bottom-0 w-1 hover:bg-blue-400 cursor-col-resize z-20 transition-colors"
                        onMouseDown={(e) => { e.preventDefault(); setIsResizingHistory(true); document.body.style.cursor = 'col-resize'; }}
                    />
                </div>

                {/* Main Chat Area */}
                <div className="flex-1 flex flex-col min-w-0 bg-white relative">
                    <button
                        onClick={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
                        className={`absolute left-0 top-1/2 -translate-y-1/2 -translate-x-1/2 z-50 bg-white border border-slate-200 rounded-full p-1.5 shadow-lg hover:bg-slate-50 text-slate-400 hover:text-blue-500 transition-all hover:scale-110 ${isSidebarCollapsed ? 'translate-x-3' : ''}`}
                    >
                        {isSidebarCollapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
                    </button>

                    <div className="h-16 flex items-center justify-between px-6 border-b border-slate-100 bg-white/80 backdrop-blur-md z-10 relative"
                    >
                        <div className="flex items-center gap-3.5">
                            <div className="bg-gradient-to-br from-blue-500 to-indigo-600 p-2 rounded-xl shadow-md shadow-blue-200">
                                <MessageSquare className="w-5 h-5 text-white" />
                            </div>
                            <div>
                                <h3 className="font-bold text-slate-900 text-sm tracking-tight">{t('ai_assistant')}</h3>
                                <div className="flex items-center gap-2 mt-1">
                                    <span className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse" />
                                    <p className="text-[10px] text-slate-500 font-medium truncate max-w-[200px]">
                                        {activeThread?.title || t('ready_to_help')}
                                    </p>
                                    {activeDataSource && (
                                        <span className="text-[9px] px-1.5 py-0.5 bg-slate-100 text-slate-500 rounded-full border border-slate-200">
                                            {activeDataSource.name}
                                        </span>
                                    )}
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center gap-1">
                            <button
                                onClick={(e) => {
                                    console.log('Skills button clicked');
                                    e.preventDefault();
                                    e.stopPropagation();
                                    EventsEmit('open-skills');
                                }}
                                aria-label="Open Skills"
                                className="p-2 hover:bg-slate-100 rounded-full text-slate-400 hover:text-blue-600 transition-all cursor-pointer"
                                title="Skills Plugin"
                            >
                                <Zap className="w-5 h-5 pointer-events-none" />
                            </button>
                            <div
                                onClick={(e) => {
                                    console.log('Close div clicked');
                                    e.preventDefault();
                                    e.stopPropagation();
                                    onClose();
                                }}
                                role="button"
                                tabIndex={0}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' || e.key === ' ') {
                                        e.preventDefault();
                                        onClose();
                                    }
                                }}
                                aria-label="Close sidebar"
                                className="p-2 hover:bg-slate-100 rounded-full text-slate-400 hover:text-slate-600 transition-all cursor-pointer"
                            >
                                <X className="w-5 h-5 pointer-events-none" />
                            </div>
                        </div>
                        {isLoading && loadingThreadId === activeThreadId && (
                            <div className="absolute bottom-0 left-0 right-0 h-1 z-20 overflow-hidden bg-slate-100">
                                {progress ? (
                                    <div
                                        className="h-full bg-gradient-to-r from-blue-500 to-indigo-500 transition-all duration-300 ease-out rounded-full"
                                        style={{ width: `${progress.progress}%` }}
                                    />
                                ) : (
                                    <div className="h-full w-1/3 bg-blue-500 animate-progress-indeterminate rounded-full"></div>
                                )}
                            </div>
                        )}
                    </div>

                    <div className="flex-1 overflow-y-auto p-6 space-y-8 bg-slate-50/10 scrollbar-thin scrollbar-thumb-slate-200 scrollbar-track-transparent">
                        {activeThread?.messages.map((msg, index) => {
                            // 找到对应的用户消息ID（用于assistant消息关联建议）
                            let userMessageId = null;
                            if (msg.role === 'assistant' && index > 0) {
                                // 查找前一条用户消息
                                for (let i = index - 1; i >= 0; i--) {
                                    if (activeThread.messages[i].role === 'user') {
                                        userMessageId = activeThread.messages[i].id;
                                        break;
                                    }
                                }
                            }

                            return (
                                <MessageBubble
                                    key={msg.id || index}
                                    role={msg.role as 'user' | 'assistant'}
                                    content={msg.content}
                                    messageId={msg.id}
                                    userMessageId={userMessageId || undefined}
                                    dataSourceId={activeThread?.data_source_id}
                                    onActionClick={(action) => handleSendMessage(action.value || action.label)}
                                    onClick={msg.role === 'user' ? () => handleUserMessageClick(msg) : undefined}
                                    hasChart={msg.role === 'user' && !!msg.chart_data}
                                />
                            );
                        })}
                        {isLoading && loadingThreadId === activeThreadId && (
                            <div className="flex justify-start animate-in fade-in slide-in-from-bottom-2 duration-300">
                                <div className="bg-white border border-slate-200 rounded-2xl px-5 py-3.5 shadow-sm rounded-bl-none max-w-[90%]">
                                    <div className="flex items-center gap-2 justify-between">
                                        <div className="flex items-center gap-2">
                                            <Loader2 className="w-4 h-4 animate-spin text-blue-600" />
                                            <span className="text-xs text-slate-500 font-medium">
                                                {progress?.message || t('ai_thinking')}
                                            </span>
                                        </div>
                                        <button
                                            onClick={handleCancelAnalysis}
                                            className="flex items-center gap-1 px-2 py-1 text-xs text-red-600 hover:bg-red-50 rounded transition-colors"
                                            title="取消分析"
                                        >
                                            <XCircle className="w-3.5 h-3.5" />
                                            <span>取消</span>
                                        </button>
                                    </div>
                                    {progress && (
                                        <div className="mt-2 flex items-center gap-2">
                                            <div className="flex-1 h-1.5 bg-slate-100 rounded-full overflow-hidden">
                                                <div
                                                    className="h-full bg-gradient-to-r from-blue-500 to-indigo-500 transition-all duration-300"
                                                    style={{ width: `${progress.progress}%` }}
                                                />
                                            </div>
                                            <span className="text-[10px] text-slate-400 font-medium min-w-[40px] text-right">
                                                {progress.step}/{progress.total}
                                            </span>
                                        </div>
                                    )}
                                    {progress?.tool_output && (
                                        <div className="mt-3 p-2 bg-slate-50 rounded-lg border border-slate-100">
                                            <div className="flex items-center gap-1.5 mb-1">
                                                <span className="text-[9px] font-bold text-slate-400 uppercase tracking-wider">
                                                    {progress.tool_name || 'Tool'} Output
                                                </span>
                                            </div>
                                            <pre className="text-[10px] text-slate-600 whitespace-pre-wrap break-words max-h-32 overflow-y-auto font-mono">
                                                {progress.tool_output}
                                            </pre>
                                        </div>
                                    )}
                                </div>
                            </div>
                        )}
                        {!activeThread && !(isLoading && loadingThreadId === activeThreadId) && (
                            <div className="h-full flex flex-col items-center justify-center text-center px-8 animate-in fade-in zoom-in-95 duration-500">
                                <div className="bg-gradient-to-br from-blue-50 to-indigo-50 p-6 rounded-[2.5rem] mb-6 shadow-inner ring-1 ring-white">
                                    <MessageSquare className="w-10 h-10 text-blue-500" />
                                </div>
                                <h4 className="text-slate-900 font-extrabold text-xl tracking-tight mb-3">{t('insights_at_fingertips')}</h4>
                                <p className="text-sm text-slate-500 max-w-[280px] leading-relaxed font-medium">
                                    {t('ask_about_sales')}
                                </p>
                            </div>
                        )}
                        <div ref={messagesEndRef} />
                    </div>

                    <div className="p-6 border-t border-slate-100 bg-white">
                        <div className="flex items-stretch gap-3 max-w-2xl mx-auto w-full">
                            <input
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
                                placeholder={t('what_to_analyze')}
                                className="flex-1 bg-slate-50 border border-slate-200 rounded-2xl px-6 py-1.5 text-sm font-normal text-slate-900 focus:ring-4 focus:ring-blue-100 focus:bg-white focus:border-blue-300 transition-all outline-none shadow-sm hover:border-slate-300"
                            />
                            <button
                                onClick={() => handleSendMessage()}
                                disabled={(isLoading && loadingThreadId === activeThreadId) || !input.trim()}
                                className="aspect-square bg-blue-600 text-white hover:bg-blue-700 rounded-2xl disabled:bg-slate-200 disabled:text-slate-400 transition-all shadow-md active:scale-95 flex items-center justify-center"
                            >
                                <Send className="w-5 h-5" />
                            </button>
                        </div>
                        <div className="flex items-center justify-center gap-4 mt-4">
                            <p className="text-[10px] text-slate-400 font-medium flex items-center gap-1">
                                <span className="w-1 h-1 bg-slate-300 rounded-full" />
                                {t('data_driven_reasoning')}
                            </p>
                            <p className="text-[10px] text-slate-400 font-medium flex items-center gap-1">
                                <span className="w-1 h-1 bg-slate-300 rounded-full" />
                                {t('visualized_summaries')}
                            </p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Confirmation Modal */}
            {showClearConfirm && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
                    <div className="bg-white rounded-xl shadow-2xl p-6 w-[320px] transform transition-all animate-in zoom-in-95 duration-200">
                        <h3 className="text-lg font-bold text-slate-900 mb-2">{t('clear_history_confirm_title')}</h3>
                        <p className="text-sm text-slate-500 mb-6">
                            {t('clear_history_confirm_desc')}
                        </p>
                        <div className="flex justify-end gap-3">
                            <button
                                onClick={cancelClearHistory}
                                className="px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
                            >
                                {t('cancel')}
                            </button>
                            <button
                                onClick={confirmClearHistory}
                                className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-lg shadow-sm transition-colors"
                            >
                                {t('clear')}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            <DeleteConfirmationModal
                isOpen={!!deleteThreadTarget}
                sourceName={deleteThreadTarget?.title || ''}
                onClose={() => setDeleteThreadTarget(null)}
                onConfirm={confirmDeleteThread}
                type="thread"
            />

            <MemoryViewModal
                isOpen={!!memoryModalTarget}
                threadId={memoryModalTarget || ''}
                onClose={() => setMemoryModalTarget(null)}
            />

            {contextMenu && (
                <ChatThreadContextMenu
                    position={{ x: contextMenu.x, y: contextMenu.y }}
                    threadId={contextMenu.threadId}
                    onClose={() => setContextMenu(null)}
                    onAction={handleContextAction}
                />
            )}

            {blankAreaContextMenu && (
                <div
                    ref={blankMenuRef}
                    className="fixed bg-white border border-slate-200 rounded-lg shadow-xl z-[9999] w-40 py-1 overflow-hidden"
                    style={{ top: blankAreaContextMenu.y, left: blankAreaContextMenu.x }}
                    onContextMenu={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                    }}
                >
                    <button
                        onClick={(e) => { e.stopPropagation(); handleImportAction(); }}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
                    >
                        <Upload className="w-4 h-4 text-slate-400" />
                        Import
                    </button>
                </div>
            )}

            <CancelConfirmationModal
                isOpen={showCancelConfirm}
                onClose={cancelCancelAnalysis}
                onConfirm={confirmCancelAnalysis}
            />
        </>
    );
};

export default ChatSidebar;