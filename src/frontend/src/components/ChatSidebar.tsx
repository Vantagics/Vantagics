import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import ReactDOM from 'react-dom';
import { MessageSquare, Plus, Trash2, Send, ChevronLeft, ChevronRight, Settings, Upload, Zap, XCircle, MessageCircle, Loader2, Database, FileText, FileChartColumn, Play, BarChart3 } from 'lucide-react';
import { GetChatHistory, SaveChatHistory, SendMessage, SendFreeChatMessage, DeleteThread, ClearHistory, ClearThreadMessages, GetDataSources, CreateChatThread, UpdateThreadTitle, OpenSessionResultsDirectory, CancelAnalysis, GetConfig, SaveConfig, GenerateIntentSuggestions, GenerateIntentSuggestionsWithExclusions, RecordIntentSelection, GetActiveSearchAPIInfo, GetMessageAnalysisData, PrepareComprehensiveReport, ExportComprehensiveReport, ExecuteQuickAnalysisPack, ShowStepResultOnDashboard, ShowAllSessionResults, ReExecuteQuickAnalysisPack, GetPackLicenseInfo, GetActivationStatus } from '../../wailsjs/go/main/App';
import { EventsOn, EventsEmit, BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import * as echarts from 'echarts';
import MessageBubble from './MessageBubble';
import { useLanguage } from '../i18n';
import { getDataSourceIcon } from './DataSourcesSection';
import DeleteConfirmationModal from './DeleteConfirmationModal';
import ChatThreadContextMenu from './ChatThreadContextMenu';
import MemoryViewModal from './MemoryViewModal';
import CancelConfirmationModal from './CancelConfirmationModal';
import ExportPackDialog from './ExportPackDialog';
import RenameSessionModal from './RenameSessionModal';
import Toast, { ToastType } from './Toast';
import { createLogger } from '../utils/systemLog';
import { loadingStateManager } from '../managers/LoadingStateManager';
import { getAnalysisResultManager } from '../managers/AnalysisResultManager';
import { AnalysisStatusIndicator } from './AnalysisStatusIndicator';
import { useSessionStatus } from '../hooks/useSessionStatus';
import { useLoadingState } from '../hooks/useLoadingState';

const systemLog = createLogger('ChatSidebar');

// Intent suggestion type
interface IntentSuggestion {
    id: string;
    title: string;
    description: string;
    icon: string;
    query: string;
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
    const isLoadingRef = useRef<boolean>(false); // Ref to track loading state for event handlers
    const loadingThreadIdRef = useRef<string | null>(null); // Ref to track loading thread ID
    const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(true);
    const [showClearConfirm, setShowClearConfirm] = useState(false);
    const [showClearConversationConfirm, setShowClearConversationConfirm] = useState<string | null>(null);
    const [dataSources, setDataSources] = useState<any[]>([]);
    const [deleteThreadTarget, setDeleteThreadTarget] = useState<{ id: string, title: string } | null>(null);
    const [memoryModalTarget, setMemoryModalTarget] = useState<string | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, threadId: string, isReplaySession?: boolean } | null>(null);
    const [blankAreaContextMenu, setBlankAreaContextMenu] = useState<{ x: number, y: number } | null>(null);
    const [showCancelConfirm, setShowCancelConfirm] = useState(false);
    const [toast, setToast] = useState<{ message: string; type: ToastType } | null>(null);
    const [exportPackThreadId, setExportPackThreadId] = useState<string | null>(null);
    const [renameSessionTarget, setRenameSessionTarget] = useState<{ id: string; title: string; dataSourceId: string; dataSourceName?: string } | null>(null);
    const [suggestionButtonSessions, setSuggestionButtonSessions] = useState<Set<string>>(new Set()); // 跟踪哪些会话需要显示建议按钮
    const [isSearching, setIsSearching] = useState(false); // 跟踪是否正在进行网络搜索
    const [isStreaming, setIsStreaming] = useState(false); // 跟踪Free Chat是否正在流式响应
    const [streamingThreadId, setStreamingThreadId] = useState<string | null>(null); // 跟踪哪个会话正在流式响应

    // Intent Understanding State
    const [intentSuggestions, setIntentSuggestions] = useState<IntentSuggestion[]>([]);
    const [excludedIntentSuggestions, setExcludedIntentSuggestions] = useState<IntentSuggestion[]>([]); // 累积所有被拒绝的意图建议
    const [isGeneratingIntent, setIsGeneratingIntent] = useState(false);
    const [pendingMessage, setPendingMessage] = useState<string>('');
    const [pendingThreadId, setPendingThreadId] = useState<string>('');
    const [intentMessageId, setIntentMessageId] = useState<string>(''); // 意图消息的ID
    const [autoIntentUnderstanding, setAutoIntentUnderstanding] = useState<boolean>(true); // 自动意图理解开关

    // Refs to track intent state for event handlers (avoid closure issues)
    const pendingMessageRef = useRef<string>('');
    const pendingThreadIdRef = useRef<string>('');
    const intentMessageIdRef = useRef<string>('');
    const intentSuggestionsRef = useRef<IntentSuggestion[]>([]);
    const excludedIntentSuggestionsRef = useRef<IntentSuggestion[]>([]);

    // Resizing State
    const [sidebarWidth, setSidebarWidth] = useState(650);
    const [historyWidth, setHistoryWidth] = useState(208);
    const [isResizingSidebar, setIsResizingSidebar] = useState(false);
    const [isResizingHistory, setIsResizingHistory] = useState(false);

    // Comprehensive Report State
    const [isGeneratingComprehensiveReport, setIsGeneratingComprehensiveReport] = useState(false);
    const [preparedComprehensiveReportId, setPreparedComprehensiveReportId] = useState<string | null>(null);
    const [comprehensiveReportExportDropdownOpen, setComprehensiveReportExportDropdownOpen] = useState(false);
    const [comprehensiveReportCached, setComprehensiveReportCached] = useState(false);
    const [comprehensiveReportError, setComprehensiveReportError] = useState<string | null>(null);
    const [showFreeModReportDialog, setShowFreeModReportDialog] = useState(false);

    // QAP Replay Session State (Requirements: 5.7, 6.2, 6.3, 6.4)
    const [qapProgress, setQapProgress] = useState<{ threadId: string; currentStep: number; totalSteps: number; description: string } | null>(null);
    const [qapCompleteThreads, setQapCompleteThreads] = useState<Set<string>>(new Set());
    const [qapReExecuting, setQapReExecuting] = useState(false);
    const [qapShowingResults, setQapShowingResults] = useState(false);
    const [packLicense, setPackLicense] = useState<main.UsageLicense | null>(null);

    // Broadcast comprehensive report generation state to other components (e.g. Sidebar context menu)
    useEffect(() => {
        EventsEmit('comprehensive-report-status', { generating: isGeneratingComprehensiveReport });
    }, [isGeneratingComprehensiveReport]);

    const messagesEndRef = useRef<HTMLDivElement>(null);
    const blankMenuRef = useRef<HTMLDivElement>(null);

    const activeThread = threads?.find(t => t.id === activeThreadId);
    
    // Use useSessionStatus hook to get the current session's loading status
    // Requirements: 1.1, 1.2, 1.3 - Display loading state in chat area
    const sessionStatus = useSessionStatus(activeThreadId);
    
    // Use useLoadingState hook to get all loading sessions for history list
    // Requirements: 2.1, 2.2, 2.3 - Display loading indicator in history list
    const { isLoading: isThreadLoading } = useLoadingState();
    
    // Find associated data source name
    const activeDataSource = dataSources?.find(ds => ds.id === activeThread?.data_source_id);

    // Fetch pack license info when active thread changes
    useEffect(() => {
        const listingId = activeThread?.pack_metadata?.listing_id;
        if (activeThread?.is_replay_session && listingId && listingId > 0) {
            GetPackLicenseInfo(listingId).then(lic => {
                setPackLicense(lic || null);
            }).catch(() => setPackLicense(null));
        } else {
            setPackLicense(null);
        }
    }, [activeThreadId, activeThread?.pack_metadata?.listing_id]);

    useEffect(() => {
        if (isOpen) {
            loadThreads();
            loadDataSources();
            // Load autoIntentUnderstanding setting from config
            GetConfig().then(cfg => {
                setAutoIntentUnderstanding(cfg.autoIntentUnderstanding !== false);
            }).catch(err => {
                console.error('Failed to load config:', err);
            });
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
                const newWidth = e.clientX;
                if (newWidth > 400 && newWidth < window.innerWidth - 100) {
                    setSidebarWidth(newWidth);
                }
            } else if (isResizingHistory) {
                const newWidth = e.clientX;
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
    // Task 3.1: Updated type to include requestId parameter
    const handleSendMessageRef = useRef<((text?: string, explicitThreadId?: string, explicitThread?: main.ChatThread, requestId?: string) => Promise<void>) | null>(null);

    // Pending error message ref: protects error messages from being overwritten
    // by loadThreads() race conditions. When the catch block creates an error message,
    // it stores it here. loadThreads() will merge it into the loaded data.
    const pendingErrorRef = useRef<{ threadId: string; errorMsg: main.ChatMessage } | null>(null);

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

    useEffect(() => {
        isLoadingRef.current = isLoading;
    }, [isLoading]);

    useEffect(() => {
        loadingThreadIdRef.current = loadingThreadId;
    }, [loadingThreadId]);

    // Keep intent-related refs in sync with state (avoid closure issues in event handlers)
    useEffect(() => {
        pendingMessageRef.current = pendingMessage;
    }, [pendingMessage]);

    useEffect(() => {
        pendingThreadIdRef.current = pendingThreadId;
    }, [pendingThreadId]);

    useEffect(() => {
        intentMessageIdRef.current = intentMessageId;
    }, [intentMessageId]);

    useEffect(() => {
        intentSuggestionsRef.current = intentSuggestions;
    }, [intentSuggestions]);

    useEffect(() => {
        excludedIntentSuggestionsRef.current = excludedIntentSuggestions;
    }, [excludedIntentSuggestions]);

    // Listen for new chat creation - separate useEffect with empty deps to prevent duplicate listeners
    useEffect(() => {
        const unsubscribeStart = EventsOn('start-new-chat', async (data: any) => {
            systemLog.info(`start-new-chat event received: ${JSON.stringify(data)}`);

            // Use sessionName as a key to prevent duplicate processing
            const chatKey = `${data.dataSourceId}-${data.sessionName}`;
            if (pendingChatRef.current === chatKey) {
                systemLog.info(`Ignoring duplicate start-new-chat event: ${chatKey}`);
                return;
            }
            pendingChatRef.current = chatKey;

            try {
                // 直接调用后端 API 创建线程，绕过 ref
                systemLog.info(`Creating thread directly via backend API`);
                systemLog.info(`dataSourceId: ${data.dataSourceId}, sessionName: ${data.sessionName}`);

                const thread = await CreateChatThread(data.dataSourceId || '', data.sessionName || 'New Chat');

                if (thread) {
                    systemLog.info(`Thread created successfully: ${thread.id}`);

                    // 更新线程列表
                    setThreads(prev => [thread, ...(prev || [])]);
                    setActiveThreadId(thread.id);
                    EventsEmit('chat-thread-created', thread.id);

                    // 发送 session-switched 事件，确保 App.tsx 更新 activeSessionId
                    EventsEmit('session-switched', thread.id);

                    // Small delay to ensure state updates have propagated
                    setTimeout(async () => {
                        systemLog.info('Timeout callback executing');

                        // Check if there's an explicit initial message (e.g., from insight click)
                        if (data.initialMessage) {
                            // Use the provided initial message directly (insight text)
                            systemLog.info(`Has initialMessage, sending it directly via backend`);
                            systemLog.info(`initialMessage: ${data.initialMessage}`);
                            systemLog.info(`thread.id: ${thread.id}`);

                            try {
                                // 直接调用后端发送消息，绕过前端的 handleSendMessage
                                // 注意：SendMessage 已经在文件顶部导入，不需要动态导入

                                // 生成唯一的消息ID和请求ID
                                const userMessageId = `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
                                const requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

                                systemLog.info(`Calling SendMessage with userMessageId: ${userMessageId}`);

                                // 设置加载状态
                                setIsLoading(true);
                                setLoadingThreadId(thread.id);

                                // 调用后端发送消息
                                await SendMessage(thread.id, data.initialMessage, userMessageId, requestId);

                                systemLog.info('SendMessage completed successfully');

                                // 确保聊天区域保持打开状态
                                systemLog.info('Emitting ensure-chat-open event');
                                EventsEmit('ensure-chat-open');

                                // 清除 pending 标记
                                pendingChatRef.current = null;
                            } catch (error) {
                                systemLog.error(`Failed to send initial message: ${error}`);
                                setIsLoading(false);
                                setLoadingThreadId(null);
                                // 通知 LoadingStateManager
                                loadingStateManager.setLoading(thread.id, false);
                            }
                            return;
                        }

                        systemLog.info('[ChatSidebar] No initialMessage, checking auto analysis settings');

                        // Get current config to check autoAnalysisSuggestions setting
                        let autoAnalysis = true; // Default to true
                        let prompt = "Give me some analysis suggestions for this data source.";
                        try {
                            const config = await GetConfig();
                            // Check if auto analysis suggestions is enabled (default true)
                            autoAnalysis = config.autoAnalysisSuggestions !== false;
                            if (config.language === '简体中文') {
                                prompt = "请给出一些本数据源的分析建议。";
                            }
                        } catch (e) {
                            console.error("Failed to get config:", e);
                        }

                        // If auto analysis is disabled, show the suggestion button instead
                        if (!autoAnalysis) {
                            console.log('[ChatSidebar] Auto analysis suggestions disabled, showing suggestion button');
                            setSuggestionButtonSessions(prev => new Set(prev).add(thread.id));
                            return;
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
                        systemLog.info(`[LOADING-DEBUG] start-new-chat: Sending initial message via handleSendMessageRef: ${prompt.substring(0, 50)}`);
                        console.log('[ChatSidebar] Sending initial message:', prompt);

                        if (handleSendMessageRef.current) {
                            systemLog.info(`[LOADING-DEBUG] start-new-chat: handleSendMessageRef.current exists, calling it`);
                            handleSendMessageRef.current(prompt, thread.id, thread);
                        } else {
                            systemLog.info(`[LOADING-DEBUG] start-new-chat: handleSendMessageRef.current is NULL!`);
                        }

                        // 清除消息标记
                        setTimeout(() => {
                            if (lastMessageRef.current === messageKey) {
                                lastMessageRef.current = null;
                            }
                        }, 5000); // 增加到5秒
                    }, 100);
                }
            } catch (err: any) {
                // Handle error when creating thread (e.g., active analysis in progress)
                systemLog.error(`Failed to create thread: ${err}`);

                const errorMsg = err?.message || String(err);

                // Show user-friendly error message
                EventsEmit('show-message-modal', {
                    type: 'warning',
                    title: t('create_session_failed'),
                    message: errorMsg
                });

                // Clear pending flag immediately on error
                pendingChatRef.current = null;
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

    // Listen for start-free-chat event from Sidebar (when no data source is selected)
    // Instead of creating a new free chat thread, switch to the existing one (system has only one free chat)
    useEffect(() => {
        const unsubscribeStartFreeChat = EventsOn('start-free-chat', async (data: any) => {
            systemLog.info(`start-free-chat event received: ${JSON.stringify(data)}`);

            try {
                // Find the existing free chat thread (no data_source_id)
                const currentThreads = threadsRef.current;
                let freeChatThread = currentThreads?.find(t => !t.data_source_id || t.data_source_id === '');

                // If no free chat thread exists yet, create one
                if (!freeChatThread) {
                    systemLog.info('No existing free chat thread found, creating one');
                    const title = data.sessionName || t('free_chat');
                    freeChatThread = await CreateChatThread('', title);
                    if (freeChatThread) {
                        setThreads(prev => [freeChatThread!, ...(prev || [])]);
                        EventsEmit('chat-thread-created', freeChatThread.id);
                    }
                }

                if (freeChatThread) {
                    // Switch to the existing free chat thread
                    systemLog.info(`Switching to existing free chat thread: ${freeChatThread.id}`);
                    setActiveThreadId(freeChatThread.id);
                    EventsEmit('session-switched', freeChatThread.id);

                    // Open chat panel if requested
                    if (data.openChat) {
                        EventsEmit('ensure-chat-open');
                    }

                    // Show toast with search API info
                    try {
                        const [apiName, apiId, isEnabled] = await GetActiveSearchAPIInfo();
                        let toastMessage = t('free_chat_started');
                        if (apiName && isEnabled) {
                            toastMessage = `${t('free_chat_started')} (${t('search_api')}: ${apiName})`;
                        } else {
                            toastMessage = `${t('free_chat_started')} (${t('no_search_api')})`;
                        }
                        setToast({
                            message: toastMessage,
                            type: 'info'
                        });
                    } catch (apiErr) {
                        setToast({
                            message: t('free_chat_started'),
                            type: 'info'
                        });
                    }
                }
            } catch (e) {
                console.error("Start free chat failed:", e);
                EventsEmit('show-message-modal', {
                    type: 'error',
                    title: t('start_free_chat_failed'),
                    message: String(e)
                });
            }
        });

        return () => {
            if (unsubscribeStartFreeChat) unsubscribeStartFreeChat();
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

        // Listen for switch-to-session event (from data source insight click)
        const unsubscribeSwitchToSession = EventsOn('switch-to-session', async (data: any) => {
            console.log('[ChatSidebar] switch-to-session event received:', data);
            const { threadId, openChat } = data;
            if (threadId) {
                // 重新加载线程列表以包含新创建的会话
                await loadThreads();
                setActiveThreadId(threadId);
                console.log('[ChatSidebar] Switched to new session:', threadId);
            }
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

        // Listen for messages sent via insights/dashboard (chat-send-message event)
        const unsubscribeChatMessage = EventsOn('chat-send-message', (message: string) => {
            console.log('[ChatSidebar] Received chat-send-message event:', message);
            // Send the message if there's an active thread, otherwise create new thread
            if (activeThread) {
                console.log('[ChatSidebar] Sending to active thread:', activeThread.id);
                handleSendMessage(message, activeThread.id, activeThread);
            } else {
                console.log('[ChatSidebar] No active thread, skipping message');
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
                    // Task 3.1: Pass requestId from event data to handleSendMessage
                    handleSendMessage(data.text, targetThread.id, targetThread, data.requestId);
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
                    // Task 3.1: Pass requestId from event data to handleSendMessage
                    handleSendMessage(data.text, targetThread.id, targetThread, data.requestId);
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
                            // Task 3.1: Pass requestId from event data to handleSendMessage
                            handleSendMessage(data.text, currentActiveThreadId, activeThread, data.requestId);
                        } else {
                            console.log('[ChatSidebar] Active thread not found in threads list, using activeThreadId directly');
                            // Task 3.1: Pass requestId from event data to handleSendMessage
                            handleSendMessage(data.text, currentActiveThreadId, undefined, data.requestId);
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

        // Listen for load-message-data event (from analysis-completed)
        const unsubscribeLoadMessageData = EventsOn('load-message-data', async (data: any) => {
            systemLog.warn(`[load-message-data] event received: ${JSON.stringify(data)}`);

            const { messageId, threadId } = data;
            if (!messageId) {
                systemLog.warn('[load-message-data] No messageId provided, ignoring');
                return;
            }

            // Find the thread to get the thread ID
            // 优先使用事件中提供的 threadId，避免依赖可能过时的 threads 状态
            const targetThreadId = threadId || activeThreadId;
            if (!targetThreadId) {
                systemLog.warn('[load-message-data] No threadId available');
                return;
            }
            systemLog.warn(`[load-message-data] Loading message data on-demand: threadId=${targetThreadId}, messageId=${messageId}`);

            // 直接使用 AnalysisResultManager 恢复数据
            const manager = getAnalysisResultManager();

            // Load analysis data on-demand from backend (heavy data is stripped from in-memory messages)
            try {
                const analysisData = await GetMessageAnalysisData(targetThreadId, messageId);
                const analysisResults = analysisData?.analysisResults;

                if (analysisResults && analysisResults.length > 0) {
                    systemLog.warn(`[load-message-data] Loaded ${analysisResults.length} analysis_results`);
                    const stats = manager.restoreResults(targetThreadId, messageId, analysisResults);
                    systemLog.warn(`[load-message-data] restoreResults: valid=${stats.validItems}, invalid=${stats.invalidItems}`);
                } else {
                    // Fallback to legacy chart_data format
                    const chartDataToUse = analysisData?.chartData;
                    if (chartDataToUse && chartDataToUse.charts && chartDataToUse.charts.length > 0) {
                        const convertedItems = chartDataToUse.charts.map((chart: any, index: number) => ({
                            id: `legacy_${messageId}_${index}`,
                            type: chart.type || 'echarts',
                            data: chart.data,
                            metadata: {
                                sessionId: targetThreadId,
                                messageId: messageId,
                                timestamp: Date.now()
                            },
                            source: 'restored'
                        }));

                        systemLog.warn(`[load-message-data] Converted ${convertedItems.length} legacy chart_data items`);
                        const stats = manager.restoreResults(targetThreadId, messageId, convertedItems);
                        systemLog.warn(`[load-message-data] legacy restoreResults: valid=${stats.validItems}, invalid=${stats.invalidItems}`);
                    } else {
                        systemLog.warn('[load-message-data] No analysis results or chart data found');
                        manager.restoreResults(targetThreadId, messageId, []);
                    }
                }
            } catch (err) {
                systemLog.error(`[load-message-data] Failed to load message analysis data: ${err}`);
            }

            // Emit user-message-clicked for UI state update
            EventsEmit('user-message-clicked', {
                threadId: targetThreadId,
                messageId: messageId,
                content: ''
            });
        });

        // Listen for analysis cancellation event
        const unsubscribeCancelled = EventsOn('analysis-cancelled', (data: any) => {
            console.log('[ChatSidebar] Analysis cancelled event received:', data);

            setShowCancelConfirm(false);
            const cancelledThreadId = data?.threadId || activeThreadIdRef.current;

            // Clear loading state
            setIsLoading(false);
            setLoadingThreadId(null);
            if (cancelledThreadId) {
                loadingStateManager.setLoading(cancelledThreadId, false);
            }

            // Reload threads to pick up the backend-saved cancellation message
            loadThreads();

            console.log('[ChatSidebar] Loading state cleared after cancellation');
        });

        // Listen for analysis error event
        // NOTE: The error message display is handled by the catch block in
        // handleSendMessage. This handler only clears loading state for cases
        // where the error occurs outside handleSendMessage (e.g., background).
        // The App.tsx handler shows the toast notification.
        const unsubscribeAnalysisError = EventsOn('analysis-error', (data: any) => {
            console.log('[ChatSidebar] Analysis error event received:', data);

            const errorThreadId = data?.threadId || data?.sessionId || activeThreadIdRef.current;

            // Clear loading state
            setIsLoading(false);
            setLoadingThreadId(null);
            if (errorThreadId) {
                loadingStateManager.setLoading(errorThreadId, false);
            }
        });

        // Listen for free chat streaming events
        const unsubscribeStreamStart = EventsOn('free-chat-stream-start', (data: any) => {
            console.log('[ChatSidebar] Free chat stream started:', data);
            const { threadId, messageId } = data;

            // 设置流式响应状态，禁止用户输入
            setIsStreaming(true);
            setStreamingThreadId(threadId);

            // Add empty assistant message to thread for streaming
            setThreads(prevThreads => {
                const newThreads = [...(prevThreads || [])];
                const threadIndex = newThreads.findIndex(t => t.id === threadId);
                if (threadIndex !== -1) {
                    const thread = { ...newThreads[threadIndex] };
                    const streamingMsg = new main.ChatMessage();
                    streamingMsg.id = messageId;
                    streamingMsg.role = 'assistant';
                    streamingMsg.content = '';
                    streamingMsg.timestamp = Math.floor(Date.now() / 1000);
                    thread.messages = [...(thread.messages || []), streamingMsg];
                    newThreads[threadIndex] = thread;
                }
                return newThreads;
            });
        });

        const unsubscribeStreamChunk = EventsOn('free-chat-stream-chunk', (data: any) => {
            const { threadId, messageId, content } = data;

            // Append content to the streaming message
            setThreads(prevThreads => {
                const newThreads = [...(prevThreads || [])];
                const threadIndex = newThreads.findIndex(t => t.id === threadId);
                if (threadIndex !== -1) {
                    const thread = { ...newThreads[threadIndex] };
                    const messages = [...(thread.messages || [])];
                    const msgIndex = messages.findIndex(m => m.id === messageId);
                    if (msgIndex !== -1) {
                        const msg = { ...messages[msgIndex] };
                        msg.content = (msg.content || '') + content;
                        messages[msgIndex] = msg;
                        thread.messages = messages;
                        newThreads[threadIndex] = thread;
                    }
                }
                return newThreads;
            });
        });

        const unsubscribeStreamEnd = EventsOn('free-chat-stream-end', (data: any) => {
            console.log('[ChatSidebar] Free chat stream ended:', data);
            // Stream complete - the final message is already in state
            // Just reload threads to ensure persistence is synced
            loadThreads();
            // Clear searching state
            setIsSearching(false);
            // 清除流式响应状态，允许用户再次输入
            setIsStreaming(false);
            setStreamingThreadId(null);
        });

        // Listen for free chat search status events
        const unsubscribeSearchStatus = EventsOn('free-chat-search-status', (data: any) => {
            console.log('[ChatSidebar] Free chat search status:', data);
            const { searching } = data;
            setIsSearching(searching);
        });

        // Listen for comprehensive report generation request from Sidebar context menu
        const unsubscribeComprehensiveReport = EventsOn('generate-comprehensive-report', (data: any) => {
            console.log('[ChatSidebar] Generate comprehensive report event received:', data);
            const { threadId } = data;
            if (threadId) {
                handleGenerateComprehensiveReport(threadId);
            }
        });

        // QAP Replay Session events (Requirements: 5.7, 6.4)
        const unsubscribeQapProgress = EventsOn('qap-progress', (data: any) => {
            if (data && data.threadId) {
                setQapProgress({
                    threadId: data.threadId,
                    currentStep: data.currentStep,
                    totalSteps: data.totalSteps,
                    description: data.description || '',
                });
            }
        });

        const unsubscribeQapComplete = EventsOn('qap-complete', async (data: any) => {
            if (data && data.threadId) {
                setQapProgress(null);
                setQapCompleteThreads(prev => new Set(prev).add(data.threadId));
                setQapReExecuting(false);

                // Refresh pack license info after execution (usage may have been consumed)
                const thread = threads?.find(t => t.id === data.threadId);
                const listingId = thread?.pack_metadata?.listing_id;
                if (listingId && listingId > 0) {
                    GetPackLicenseInfo(listingId).then(lic => setPackLicense(lic || null)).catch(() => {});
                }

                // Call ShowAllSessionResults from the frontend context.
                // Calling from within the backend RPC doesn't work reliably because
                // Wails events emitted during a long-running RPC may not be processed
                // by the frontend until after the RPC returns and the dialog closes.
                try {
                    await ShowAllSessionResults(data.threadId);
                } catch (err) {
                    console.error('[ChatSidebar] qap-complete ShowAllSessionResults failed:', err);
                }
            }
        });

        // QAP session created by backend — add to thread list and switch to it
        const unsubscribeQapSessionCreated = EventsOn('qap-session-created', (data: any) => {
            if (data && data.threadId) {
                systemLog.info(`qap-session-created: threadId=${data.threadId}, title=${data.title}`);
                const newThread = new main.ChatThread({
                    id: data.threadId,
                    title: data.title || '',
                    data_source_id: data.dataSourceId || '',
                    created_at: Math.floor(Date.now() / 1000),
                    messages: [],
                    is_replay_session: true,
                });
                setThreads(prev => [newThread, ...(prev || [])]);
                setActiveThreadId(data.threadId);
                EventsEmit('session-switched', data.threadId);
            }
        });

        return () => {
            if (unsubscribeOpen) unsubscribeOpen();
            if (unsubscribeSwitchToSession) unsubscribeSwitchToSession();
            if (unsubscribeUpdate) unsubscribeUpdate();
            if (unsubscribeLoading) unsubscribeLoading();
            if (unsubscribeChatMessage) unsubscribeChatMessage();
            if (unsubscribeSendMessageInSession) unsubscribeSendMessageInSession();
            if (unsubscribeLoadMessageData) unsubscribeLoadMessageData();
            if (unsubscribeCancelled) unsubscribeCancelled();
            if (unsubscribeAnalysisError) unsubscribeAnalysisError();
            if (unsubscribeStreamStart) unsubscribeStreamStart();
            if (unsubscribeStreamChunk) unsubscribeStreamChunk();
            if (unsubscribeStreamEnd) unsubscribeStreamEnd();
            if (unsubscribeSearchStatus) unsubscribeSearchStatus();
            if (unsubscribeComprehensiveReport) unsubscribeComprehensiveReport();
            if (unsubscribeQapProgress) unsubscribeQapProgress();
            if (unsubscribeQapComplete) unsubscribeQapComplete();
            if (unsubscribeQapSessionCreated) unsubscribeQapSessionCreated();
        };
    }, [threads]);

    // 监听活动会话变化，自动显示第一个分析结果
    // 注意：仅在会话切换时加载历史数据，不在实时分析完成后重复加载
    const prevAutoLoadThreadIdRef = useRef<string | null>(null);
    useEffect(() => {
        if (activeThreadId && threads) {
            // 仅在 activeThreadId 真正变化时触发自动加载
            // threads 变化（如分析完成后 GetChatHistory 重新加载）不应触发重新加载，
            // 否则会覆盖 AnalysisResultManager 中的实时数据
            if (prevAutoLoadThreadIdRef.current === activeThreadId) {
                return;
            }
            prevAutoLoadThreadIdRef.current = activeThreadId;

            // 检查 AnalysisResultManager 是否已有当前会话的数据
            // 如果有，说明是实时分析刚完成（数据已通过 analysis-result-update 流式传入），
            // 不需要从磁盘重新加载，否则会覆盖实时数据导致表格等丢失
            const manager = getAnalysisResultManager();
            const currentSession = manager.getCurrentSession();
            const currentMessage = manager.getCurrentMessage();
            if (currentSession === activeThreadId && currentMessage && manager.hasCurrentData()) {
                console.log("[ChatSidebar] AnalysisResultManager already has data for this session, skipping auto-load to preserve live data");
                return;
            }

            const activeThread = threads.find(t => t.id === activeThreadId);

            // Replay_Session 检测逻辑 (Requirements: 3.1, 3.2, 3.3, 3.4)
            if (activeThread?.is_replay_session) {
                // 已完成的 Replay_Session：调用 ShowAllSessionResults 合并显示所有结果
                if (qapCompleteThreads.has(activeThreadId)) {
                    ShowAllSessionResults(activeThreadId).catch(err => {
                        console.error('[ChatSidebar] Auto load replay session results failed:', err);
                    });
                    return;
                }

                // 正在执行中的 Replay_Session：保留 AnalysisResultManager 中的实时数据（由上方检查处理）
                // 尚未执行或无成功步骤的 Replay_Session：清空 Dashboard
                const hasSuccessfulSteps = activeThread.messages?.some(
                    msg => msg.role === 'assistant' && msg.content?.includes('✅')
                );
                if (!hasSuccessfulSteps) {
                    EventsEmit('clear-dashboard');
                    return;
                }
            }

            if (activeThread && activeThread.messages) {
                // 找到第一个有分析结果的用户消息
                // 判断标准：用户消息后有助手回复，或者有分析数据
                let firstAnalysisMessage: main.ChatMessage | null = null;

                for (let i = 0; i < activeThread.messages.length; i++) {
                    const msg = activeThread.messages[i];

                    // 必须是用户消息
                    if (msg.role !== 'user') continue;

                    // 检查是否有分析数据（chart_data已被strip，使用has_analysis_data标志）
                    if (msg.chart_data || (msg as any).has_analysis_data) {
                        firstAnalysisMessage = msg;
                        break;
                    }

                    // 检查下一条消息是否是助手回复
                    if (i < activeThread.messages.length - 1) {
                        const nextMsg = activeThread.messages[i + 1];
                        if (nextMsg.role === 'assistant') {
                            firstAnalysisMessage = msg;
                            break;
                        }
                    }
                }

                if (firstAnalysisMessage) {
                    console.log("[ChatSidebar] Auto-displaying first analysis result for thread:", activeThreadId);
                    console.log("[ChatSidebar] First analysis message:", firstAnalysisMessage.id);

                    // 自动触发显示该消息的分析结果（通过handleUserMessageClick按需加载数据）
                    setTimeout(() => {
                        handleUserMessageClick(firstAnalysisMessage!);
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

            // Merge any pending error message that hasn't been persisted yet.
            // This prevents the race condition where loadThreads() is called
            // (e.g., from analysis-cancelled event) before SaveChatHistory
            // in the catch block has completed writing the error message.
            const pending = pendingErrorRef.current;
            if (pending && history) {
                const threadIndex = history.findIndex(t => t.id === pending.threadId);
                if (threadIndex !== -1) {
                    const thread = history[threadIndex];
                    const messages = thread.messages || [];
                    // Check if the backend already saved an error/assistant message
                    // (either our exact message by ID, or any assistant error message)
                    const alreadyHasThisError = messages.some(m => m.id === pending.errorMsg.id);
                    const alreadyHasBackendError = messages.some(m =>
                        m.role === 'assistant' && m.id?.startsWith('error_')
                    );
                    if (!alreadyHasThisError && !alreadyHasBackendError) {
                        console.log('[ChatSidebar] loadThreads: merging pending error message into loaded data');
                        thread.messages = [...messages, pending.errorMsg];
                    } else {
                        console.log('[ChatSidebar] loadThreads: error message already present, skipping merge');
                        // Error is already persisted, clear the ref
                        pendingErrorRef.current = null;
                    }
                }
            }

            setThreads(history);
            if (history && history.length > 0 && !activeThreadId) {
                setActiveThreadId(history[0].id);
            }

            // Restore qapCompleteThreads for replay sessions that were completed
            // before the app was closed. Without this, the re-execute and show-all-results
            // buttons won't appear for historical replay sessions after restart.
            if (history) {
                const completedIds: string[] = [];
                for (const t of history) {
                    if (t.is_replay_session && t.messages?.some(
                        m => m.role === 'assistant' && m.content?.includes('执行完成')
                    )) {
                        completedIds.push(t.id);
                    }
                }
                if (completedIds.length > 0) {
                    setQapCompleteThreads(prev => {
                        const next = new Set(prev);
                        for (const id of completedIds) {
                            next.add(id);
                        }
                        return next;
                    });
                }
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
            EventsEmit('chat-thread-created', newThread.id);
            return newThread;
        } catch (err: any) {
            console.error('Failed to create thread:', err);

            // Check if error is about active session conflict
            const errorMsg = err?.message || String(err);
            if (errorMsg.includes('分析会话进行中') || errorMsg.includes('active analysis')) {
                // Show user-friendly error message via MessageModal
                EventsEmit('show-message-modal', {
                    type: 'warning',
                    title: t('session_conflict_title'),
                    message: errorMsg
                });
            } else {
                // Generic error
                EventsEmit('show-message-modal', {
                    type: 'error',
                    title: t('create_session_failed'),
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

        console.log('[DELETE-THREAD] Starting deletion for thread:', deleteThreadTarget.id);

        try {
            // 如果删除的会话正在进行分析，先取消分析
            if (loadingThreadId === deleteThreadTarget.id) {
                console.log('[DELETE-THREAD] Cancelling ongoing analysis before deletion');
                try {
                    await CancelAnalysis();
                    // 等待取消生效
                    await new Promise(resolve => setTimeout(resolve, 200));
                } catch (cancelErr) {
                    console.error('[DELETE-THREAD] Failed to cancel analysis:', cancelErr);
                }
                // 重置loading状态
                setIsLoading(false);
                setLoadingThreadId(null);
            }
            
            // 清理 LoadingStateManager 中的会话状态 - Requirements: 4.5
            loadingStateManager.clearSession(deleteThreadTarget.id);

            console.log('[DELETE-THREAD] Calling DeleteThread API...');
            await DeleteThread(deleteThreadTarget.id);
            console.log('[DELETE-THREAD] DeleteThread API completed successfully');

            const updatedThreads = threads?.filter(t => t.id !== deleteThreadTarget.id) || [];
            setThreads(updatedThreads);
            console.log('[DELETE-THREAD] Updated threads list, remaining:', updatedThreads.length);

            // 如果删除的是当前活跃的会话
            if (activeThreadId === deleteThreadTarget.id) {
                if (updatedThreads.length > 0) {
                    // 如果还有其他会话，选择第一个并加载其数据
                    console.log('[DELETE-THREAD] Switching to first remaining thread');
                    const newActiveThread = updatedThreads[0];
                    setActiveThreadId(newActiveThread.id);

                    // 清空当前仪表盘，准备显示新会话的数据
                    console.log('[DELETE-THREAD] Clearing dashboard before loading new thread data');
                    EventsEmit('clear-dashboard');

                    // 加载新会话的最后一条用户消息的分析结果
                    if (newActiveThread.messages && newActiveThread.messages.length > 0) {
                        // 从后往前找最后一条用户消息
                        for (let i = newActiveThread.messages.length - 1; i >= 0; i--) {
                            const msg = newActiveThread.messages[i];
                            if (msg.role === 'user' && msg.id) {
                                console.log('[DELETE-THREAD] Loading analysis results from new active thread, message:', msg.id);
                                // 触发仪表盘更新（chart_data已被strip，通过user-message-clicked触发按需加载）
                                EventsEmit('user-message-clicked', {
                                    threadId: newActiveThread.id,
                                    messageId: msg.id
                                });
                                break;
                            }
                        }
                    }
                } else {
                    // 如果没有剩余会话，清空活跃会话ID并通知App清空仪表盘
                    console.log('[DELETE-THREAD] No remaining threads, clearing dashboard');
                    setActiveThreadId(null);
                    EventsEmit('clear-dashboard');
                }
            } else {
                // 如果删除的不是当前活跃会话，仪表盘保持不变
                console.log('[DELETE-THREAD] Deleted non-active thread, dashboard unchanged');
            }

            // 关闭删除确认模态框
            console.log('[DELETE-THREAD] Closing delete confirmation modal');
            setDeleteThreadTarget(null);
            console.log('[DELETE-THREAD] Deletion completed successfully');
        } catch (err) {
            console.error('[DELETE-THREAD] Failed to delete thread:', err);
            // 即使失败也关闭模态框，并显示错误消息
            setDeleteThreadTarget(null);

            // 显示错误消息给用户
            EventsEmit('show-message-modal', {
                type: 'error',
                title: t('delete_failed'),
                message: `${t('cannot_delete_session')}: ${err}`
            });
        }
    };

    const handleContextMenu = (e: React.MouseEvent, threadId: string) => {
        e.preventDefault();
        const thread = threads.find(t => t.id === threadId);
        setContextMenu({ x: e.clientX, y: e.clientY, threadId, isReplaySession: thread?.is_replay_session });
    };

    const handleBlankAreaContextMenu = (e: React.MouseEvent) => {
        // Only show menu if clicking directly on the container (blank area)
        if (e.target === e.currentTarget) {
            e.preventDefault();
            setBlankAreaContextMenu({ x: e.clientX, y: e.clientY });
        }
    };

    const handleImportAction = async () => {
        try {
            // Import the function dynamically to avoid build errors
            const { ImportAnalysisProcess } = await import('../../wailsjs/go/main/App');
            await ImportAnalysisProcess();
            // Reload threads after import to show the new imported session
            await loadThreads();
        } catch (err) {
            console.error('Import analysis failed:', err);
        }
        setBlankAreaContextMenu(null);
    };

    const handleStartFreeChat = async () => {
        try {
            // Create a new thread without data source for free chat
            // Use translated title for proper localization
            const title = t('free_chat');
            const thread = await CreateChatThread('', title);

            if (thread) {
                // Add the new thread to the list and set it as active
                setThreads(prev => [thread, ...(prev || [])]);
                setActiveThreadId(thread.id);
                EventsEmit('chat-thread-created', thread.id);

                // Emit session-switched event
                EventsEmit('session-switched', thread.id);

                // Show toast notification
                setToast({
                    message: t('free_chat_started'),
                    type: 'info'
                });
            }
        } catch (e) {
            console.error("Start free chat failed:", e);
            EventsEmit('show-message-modal', {
                type: 'error',
                title: t('start_free_chat_failed'),
                message: String(e)
            });
        }
        setBlankAreaContextMenu(null);
    };

    const handleContextAction = async (action: 'view_memory' | 'view_results_directory' | 'toggle_intent_understanding' | 'clear_messages' | 'start_free_chat' | 'comprehensive_report' | 'export_quick_analysis_pack' | 'rename', threadId: string) => {
        console.log(`Action ${action} on thread ${threadId}`);
        if (action === 'rename') {
            const thread = threads.find(t => t.id === threadId);
            if (thread) {
                const dsName = dataSources?.find((s: any) => s.id === thread.data_source_id)?.name;
                setRenameSessionTarget({
                    id: thread.id,
                    title: thread.title,
                    dataSourceId: thread.data_source_id,
                    dataSourceName: dsName,
                });
            }
            setContextMenu(null);
        } else if (action === 'view_memory') {
            setMemoryModalTarget(threadId);
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
        } else if (action === 'toggle_intent_understanding') {
            try {
                const newValue = !autoIntentUnderstanding;
                const config = await GetConfig();
                config.autoIntentUnderstanding = newValue;
                await SaveConfig(config);
                setAutoIntentUnderstanding(newValue);
                // Close context menu after toggle
                setContextMenu(null);
                // Show toast notification
                setToast({
                    message: newValue ? t('auto_intent_understanding_enabled') : t('auto_intent_understanding_disabled'),
                    type: 'info'
                });
            } catch (e) {
                console.error("Toggle intent understanding failed:", e);
                EventsEmit('show-message-modal', {
                    type: 'error',
                    title: '设置失败',
                    message: String(e)
                });
            }
        } else if (action === 'comprehensive_report') {
            // Generate comprehensive report for the session
            await handleGenerateComprehensiveReport(threadId);
        } else if (action === 'export_quick_analysis_pack') {
            setExportPackThreadId(threadId);
        } else if (action === 'start_free_chat') {
            try {
                // Create a new thread without data source for free chat
                // Use translated title for proper localization
                const title = t('free_chat');
                const thread = await CreateChatThread('', title);

                if (thread) {
                    // Add the new thread to the list and set it as active
                    setThreads(prev => [thread, ...(prev || [])]);
                    setActiveThreadId(thread.id);
                    EventsEmit('chat-thread-created', thread.id);

                    // Emit session-switched event
                    EventsEmit('session-switched', thread.id);

                    // Show toast notification
                    setToast({
                        message: t('free_chat_started'),
                        type: 'info'
                    });
                }
            } catch (e) {
                console.error("Start free chat failed:", e);
                EventsEmit('show-message-modal', {
                    type: 'error',
                    title: t('start_free_chat_failed'),
                    message: String(e)
                });
            }
        }
    };

    const handleRenameSession = async (newTitle: string) => {
        if (!renameSessionTarget) return;
        const targetId = renameSessionTarget.id;
        setRenameSessionTarget(null);
        try {
            await UpdateThreadTitle(targetId, newTitle);
            const history = await GetChatHistory();
            setThreads(history || []);
            EventsEmit('chat-thread-updated', targetId);
        } catch (err) {
            console.error('Failed to rename session:', err);
        }
    };

    // 处理"生成建议分析"按钮点击
    const handleGenerateSuggestions = async () => {
        systemLog.info(`[LOADING-DEBUG] handleGenerateSuggestions ENTRY: activeThreadId=${activeThreadId}, hasActiveThread=${!!activeThread}`);

        if (!activeThreadId || !activeThread) {
            systemLog.info(`[LOADING-DEBUG] handleGenerateSuggestions EARLY RETURN: no activeThreadId or activeThread`);
            return;
        }

        systemLog.info(`[LOADING-DEBUG] handleGenerateSuggestions called: threadId=${activeThreadId}, dataSourceId=${activeThread.data_source_id}`);

        // 清除当前会话的按钮显示状态
        setSuggestionButtonSessions(prev => {
            const newSet = new Set(prev);
            newSet.delete(activeThreadId);
            return newSet;
        });

        // 获取当前语言设置
        let prompt = "Give me some analysis suggestions for this data source.";
        try {
            const config = await GetConfig();
            if (config.language === '简体中文') {
                prompt = "请给出一些本数据源的分析建议。";
            }
        } catch (e) {
            console.error("Failed to get config for language:", e);
        }

        // 发送分析建议请求
        handleSendMessage(prompt, activeThreadId, activeThread);
    };

    // 处理"显示步骤结果到仪表盘"按钮点击
    const handleShowStepResult = async (messageId: string) => {
        if (!activeThreadId) return;
        try {
            await ShowStepResultOnDashboard(activeThreadId, messageId);
            setToast({ message: t('show_result_success'), type: 'success' });
        } catch (err: any) {
            setToast({ message: (t('show_result_failed')) + ': ' + (err?.message || err), type: 'error' });
        }
    };

    // 处理"生成综合报告"按钮点击
    const handleGenerateComprehensiveReport = async (threadId?: string) => {
        // Check if in permanent free mode - block report generation
        try {
            const status = await GetActivationStatus();
            if (status.is_permanent_free === true) {
                setShowFreeModReportDialog(true);
                return;
            }
        } catch (_) { /* proceed if check fails */ }

        const targetThreadId = threadId || activeThreadId;
        if (!targetThreadId) {
            setToast({
                message: t('comprehensive_report_no_analysis'),
                type: 'error'
            });
            return;
        }

        // Find the thread
        const targetThread = threads?.find(t => t.id === targetThreadId);
        if (!targetThread) {
            setToast({
                message: t('comprehensive_report_no_analysis'),
                type: 'error'
            });
            return;
        }

        // Find the data source name
        const dataSource = dataSources?.find(ds => ds.id === targetThread.data_source_id);
        const dataSourceName = dataSource?.name || 'Unknown';
        const sessionName = targetThread.title || 'Session';

        // Close any existing dropdown first
        setComprehensiveReportExportDropdownOpen(false);
        setComprehensiveReportError(null);
        
        // Show progress indicator
        setIsGeneratingComprehensiveReport(true);
        setPreparedComprehensiveReportId(null);
        setComprehensiveReportCached(false);

        console.log("[COMPREHENSIVE-REPORT] Starting report generation:", {
            threadId: targetThreadId,
            dataSourceName: dataSourceName,
            sessionName: sessionName
        });

        // Shared offscreen container for ECharts rendering (declared outside try for cleanup in catch)
        let offscreenContainer: HTMLDivElement | null = null;

        try {
            // Collect ECharts images from all analysis results in this thread
            const chartImages: string[] = [];
            const collectedImageHashes = new Set<string>(); // deduplicate by first 200 chars
            
            const addChartImage = (dataURL: string) => {
                if (!dataURL || dataURL.length < 500) return; // skip trivially small images
                const hash = dataURL.substring(0, 200);
                if (collectedImageHashes.has(hash)) return;
                collectedImageHashes.add(hash);
                chartImages.push(dataURL);
            };
            
            // Create a single offscreen container for all ECharts rendering.
            // Reusing one container avoids repeated DOM add/remove cycles that cause UI flashing.
            offscreenContainer = document.createElement('div');
            offscreenContainer.style.cssText = 'width:800px;height:500px;position:fixed;left:-9999px;top:-9999px;z-index:-9999;pointer-events:none;overflow:hidden;';
            document.body.appendChild(offscreenContainer);
            
            // Helper: render ECharts options to a base64 PNG image (reuses offscreenContainer)
            const renderEChartsToImage = (options: any): Promise<string | null> => {
                return new Promise((resolve) => {
                    let chart: any = null;
                    let resolved = false;
                    let timeoutId: any = null;
                    
                    const cleanup = () => {
                        if (timeoutId) clearTimeout(timeoutId);
                        try { if (chart) chart.dispose(); } catch (_) {}
                    };
                    
                    const captureImage = () => {
                        if (resolved) return;
                        resolved = true;
                        try {
                            const dataURL = chart.getDataURL({
                                type: 'png',
                                pixelRatio: 3,
                                backgroundColor: '#fff'
                            });
                            cleanup();
                            resolve(dataURL && dataURL.length > 500 ? dataURL : null);
                        } catch (e) {
                            console.warn("[COMPREHENSIVE-REPORT] getDataURL failed:", e);
                            cleanup();
                            resolve(null);
                        }
                    };
                    
                    try {
                        chart = echarts.init(offscreenContainer, null, { width: 800, height: 500 });
                        
                        // Listen for 'finished' event before setOption, in case it fires synchronously
                        chart.on('finished', () => {
                            if (timeoutId) clearTimeout(timeoutId);
                            // Small delay after 'finished' to ensure canvas is fully painted
                            setTimeout(captureImage, 50);
                        });
                        
                        // Disable animation for instant rendering
                        const renderOptions = { ...options, animation: false };
                        chart.setOption(renderOptions);
                        
                        // Fallback: if 'finished' doesn't fire within 2s, capture anyway
                        timeoutId = setTimeout(() => {
                            console.warn("[COMPREHENSIVE-REPORT] ECharts 'finished' event timeout, capturing anyway");
                            captureImage();
                        }, 2000);
                    } catch (e) {
                        console.warn("[COMPREHENSIVE-REPORT] ECharts init/setOption failed:", e);
                        cleanup();
                        if (!resolved) { resolved = true; resolve(null); }
                    }
                });
            };
            
            // Helper: parse ECharts options from raw data (string or object)
            const parseEChartsOptions = (rawData: any): any | null => {
                if (!rawData) return null;
                if (typeof rawData === 'object' && rawData !== null) {
                    // Already an object (Wails may auto-deserialize JSON)
                    return rawData;
                }
                if (typeof rawData === 'string') {
                    // Try direct JSON parse first
                    try { return JSON.parse(rawData); } catch (_) {}
                    // Try cleaning JS function expressions that break JSON parsing
                    try {
                        const cleaned = rawData
                            .replace(/:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, ': null')
                            .replace(/:\s*\([^)]*\)\s*=>\s*\{[^}]*\}/g, ': null')
                            .replace(/:\s*\([^)]*\)\s*=>\s*[^,}\]]+/g, ': null');
                        return JSON.parse(cleaned);
                    } catch (_) {}
                }
                return null;
            };
            
            // First, collect from currently rendered ECharts in the DOM
            const echartsComponents = document.querySelectorAll('.echarts-for-react');
            console.log(`[COMPREHENSIVE-REPORT] Found ${echartsComponents.length} DOM ECharts components`);
            for (let i = 0; i < echartsComponents.length; i++) {
                try {
                    const component = echartsComponents[i] as any;
                    if (component?.getEchartsInstance) {
                        const instance = component.getEchartsInstance();
                        if (instance) {
                            const dataURL = instance.getDataURL({
                                type: 'png',
                                pixelRatio: 4,
                                backgroundColor: '#fff'
                            });
                            addChartImage(dataURL);
                        }
                    }
                } catch (e) { /* skip */ }
            }
            
            // Canvas fallback for currently rendered charts
            if (chartImages.length === 0) {
                const canvasElements = document.querySelectorAll('canvas');
                for (let i = 0; i < canvasElements.length; i++) {
                    const canvas = canvasElements[i];
                    const parent = canvas.parentElement;
                    if (parent && (parent.classList.contains('echarts-for-react') || canvas.width > 200)) {
                        try { addChartImage(canvas.toDataURL('image/png')); } catch (e) { /* skip */ }
                    }
                }
            }
            
            const domChartCount = chartImages.length;
            console.log(`[COMPREHENSIVE-REPORT] Collected ${domChartCount} charts from DOM`);
            
            // Then, load analysis data for ALL user messages and render ECharts offscreen
            if (targetThread.messages) {
                for (const msg of targetThread.messages) {
                    if (msg.role !== 'user') continue;
                    try {
                        const analysisData = await GetMessageAnalysisData(targetThreadId, msg.id);
                        const items = analysisData?.analysisResults;
                        if (!Array.isArray(items) || items.length === 0) continue;
                        
                        console.log(`[COMPREHENSIVE-REPORT] Message ${msg.id}: ${items.length} items, types: ${items.map((i: any) => i.type).join(',')}`);
                        
                        for (const item of items) {
                            if (item.type === 'echarts' && item.data) {
                                const options = parseEChartsOptions(item.data);
                                if (!options || typeof options !== 'object') {
                                    console.warn(`[COMPREHENSIVE-REPORT] Could not parse ECharts options for item ${item.id}`);
                                    continue;
                                }
                                
                                try {
                                    const dataURL = await renderEChartsToImage(options);
                                    if (dataURL) {
                                        addChartImage(dataURL);
                                        console.log(`[COMPREHENSIVE-REPORT] Rendered ECharts image for ${item.id}: ${dataURL.length} chars`);
                                    } else {
                                        console.warn(`[COMPREHENSIVE-REPORT] ECharts rendered empty/null for ${item.id}`);
                                    }
                                } catch (e) {
                                    console.warn(`[COMPREHENSIVE-REPORT] Failed to render ECharts ${item.id}:`, e);
                                }
                            } else if (item.type === 'image' && typeof item.data === 'string' && item.data.startsWith('data:image')) {
                                addChartImage(item.data);
                            }
                        }
                    } catch (e) {
                        console.log(`[COMPREHENSIVE-REPORT] No analysis data for message ${msg.id}`);
                    }
                }
            }
            
            console.log(`[COMPREHENSIVE-REPORT] Total: ${chartImages.length} chart images (${domChartCount} from DOM, ${chartImages.length - domChartCount} from offscreen rendering)`);

            // Clean up the shared offscreen container now that all charts are rendered
            try { document.body.removeChild(offscreenContainer); } catch (_) {}

            const result = await PrepareComprehensiveReport({
                threadId: targetThreadId,
                dataSourceName: dataSourceName,
                sessionName: sessionName,
                chartImages: chartImages
            });
            
            console.log("[COMPREHENSIVE-REPORT] PrepareComprehensiveReport result:", JSON.stringify(result));
            
            if (!result || !result.reportId) {
                console.error("[COMPREHENSIVE-REPORT] Invalid result: missing reportId", result);
                setIsGeneratingComprehensiveReport(false);
                const errMsg = '返回结果无效 (reportId is empty)';
                setComprehensiveReportError(errMsg);
                setToast({
                    message: (t('comprehensive_report_failed')) + errMsg,
                    type: 'error'
                });
                return;
            }
            
            // Set report ID and cached state, stop progress
            setPreparedComprehensiveReportId(result.reportId);
            setComprehensiveReportCached(result.cached);
            setIsGeneratingComprehensiveReport(false);
            
            // Use a longer delay to ensure React has rendered the button before opening dropdown
            // This prevents the dropdown from being immediately closed by stale click events
            setTimeout(() => {
                dropdownJustOpenedRef.current = true;
                setComprehensiveReportExportDropdownOpen(true);
                setTimeout(() => { dropdownJustOpenedRef.current = false; }, 300);
            }, 250);
            
            if (result.cached) {
                setToast({
                    message: t('comprehensive_report_cached'),
                    type: 'success'
                });
            } else {
                setToast({
                    message: t('comprehensive_report_ready'),
                    type: 'success'
                });
            }
        } catch (e) {
            // Ensure offscreen container is cleaned up on error
            try { if (offscreenContainer?.parentNode) offscreenContainer.parentNode.removeChild(offscreenContainer); } catch (_) {}
            setIsGeneratingComprehensiveReport(false);
            const errMsg = String(e);
            setComprehensiveReportError(errMsg);
            console.error("[COMPREHENSIVE-REPORT] Prepare comprehensive report failed:", e);
            setToast({
                message: (t('comprehensive_report_failed')) + errMsg,
                type: 'error'
            });
        }
    };

    // Export comprehensive report in specified format
    const exportComprehensiveReportAs = async (format: 'word' | 'pdf') => {
        if (!preparedComprehensiveReportId) return;
        try {
            await ExportComprehensiveReport(preparedComprehensiveReportId, format);
            setToast({
                message: t('comprehensive_report_success'),
                type: 'success'
            });
        } catch (e) {
            console.error("Export comprehensive report failed:", e);
            setToast({
                message: (t('comprehensive_report_export_failed')) + String(e),
                type: 'error'
            });
        }
    };

    // Close comprehensive report dropdown when clicking outside
    const dropdownJustOpenedRef = useRef(false);
    useEffect(() => {
        if (comprehensiveReportExportDropdownOpen) {
            const handleClickOutside = (event: MouseEvent) => {
                if (dropdownJustOpenedRef.current) return;
                const target = event.target as HTMLElement;
                if (!target.closest('.comprehensive-report-dropdown-container')) {
                    setComprehensiveReportExportDropdownOpen(false);
                }
            };
            // Use mouseup (not mousedown) to avoid catching the originating click
            document.addEventListener('mouseup', handleClickOutside);
            return () => {
                document.removeEventListener('mouseup', handleClickOutside);
            };
        }
    }, [comprehensiveReportExportDropdownOpen]);

    /**
     * 格式化意图建议为 Markdown 显示
     * 关键改进：将所有需要的数据直接嵌入到按钮标签中，避免闭包问题
     * 
     * @param suggestions 意图建议列表
     * @param excludedCount 已排除的选项数量
     * @param originalRequest 用户原始请求
     * @param threadId 当前线程ID
     * @param intentMessageId 意图消息ID
     * @returns 格式化后的 Markdown 字符串
     * 
     * Requirements: 1.2, 3.1, 3.4, 5.1, 5.2
     */
    const formatIntentSuggestions = (
        suggestions: IntentSuggestion[],
        excludedCount: number,
        originalRequest: string,
        threadId?: string,
        intentMsgId?: string
    ): string => {
        systemLog.info(`[formatIntentSuggestions] Called with: suggestionsCount=${suggestions.length}, excludedCount=${excludedCount}, threadId=${threadId}, intentMsgId=${intentMsgId}`);
        
        // 添加隐藏标记，用于标识这是意图理解消息（多语言兼容）
        let content = `[INTENT_SUGGESTIONS]\n\n`;
        
        // Header
        const header = t('select_your_intent');
        const desc = t('intent_selection_desc');

        content += `**${header}**\n\n${desc}\n\n`;

        // Show excluded count if > 0 (Requirement 5.2)
        if (excludedCount > 0) {
            const excludedText = t('excluded_count') || `已排除 ${excludedCount} 个选项`;
            // Replace {count} placeholder if present
            const formattedExcludedText = excludedText.replace('{count}', String(excludedCount));
            content += `*${formattedExcludedText}*\n\n`;
        }

        // Format each intent suggestion (Requirement 1.2)
        // 将建议数据编码到标签中，格式: [INTENT_SELECT:index:query_base64]
        // 注意：数据标记必须放在同一行，否则 MessageBubble 的按钮提取逻辑无法正确提取
        suggestions.forEach((suggestion: IntentSuggestion, index: number) => {
            const queryBase64 = btoa(encodeURIComponent(suggestion.query));
            content += `${index + 1}. ${suggestion.icon} **${suggestion.title}** - ${suggestion.description} [INTENT_SELECT:${index}:${queryBase64}]\n\n`;
        });

        // 将重试所需的数据编码到标签中
        // 格式: [INTENT_RETRY_DATA:threadId:originalRequest_base64:suggestions_json_base64]
        const suggestionsJson = JSON.stringify(suggestions);
        const originalRequestBase64 = btoa(encodeURIComponent(originalRequest));
        const suggestionsBase64 = btoa(encodeURIComponent(suggestionsJson));
        const retryData = `[INTENT_RETRY_DATA:${threadId || ''}:${originalRequestBase64}:${suggestionsBase64}:${intentMsgId || ''}]`;

        // Add "重新理解" button (Requirement 5.1 - order: options, retry, stick)
        const retryIndex = suggestions.length + 1;
        const retryText = t('retry_intent_understanding');
        content += `${retryIndex}. 🔄 **${retryText}** ${retryData}\n\n`;
        
        systemLog.info(`[formatIntentSuggestions] Retry button line: ${retryIndex}. 🔄 **${retryText}** ${retryData.substring(0, 50)}...`);

        // 将坚持原始请求所需的数据编码到标签中
        // 格式: [INTENT_STICK_DATA:threadId:originalRequest_base64:intentMsgId]
        const stickData = `[INTENT_STICK_DATA:${threadId || ''}:${originalRequestBase64}:${intentMsgId || ''}]`;

        // Add "坚持我的请求" button with original request preview (Requirements 3.1, 3.4, 5.1)
        const stickIndex = suggestions.length + 2;
        const stickText = t('stick_to_original');
        // Truncate original request to ~30 chars for preview (Requirement 3.4)
        const truncatedRequest = originalRequest.length > 30
            ? originalRequest.substring(0, 30) + '...'
            : originalRequest;
        content += `${stickIndex}. 📝 **${stickText}**: "${truncatedRequest}" ${stickData}\n\n`;

        // Footer hint
        content += `\n*${t('click_suggestion_to_continue')}*`;

        return content;
    };

    const handleSendMessage = async (text?: string, explicitThreadId?: string, explicitThread?: main.ChatThread, requestId?: string, skipIntentUnderstanding?: boolean) => {
        const msgText = text || input;

        // 使用 refs 获取最新的状态值（避免闭包问题）
        const currentIsLoading = isLoadingRef.current;
        const currentLoadingThreadId = loadingThreadIdRef.current;

        // CRITICAL DEBUG: Log at the very start of handleSendMessage
        systemLog.info(`[LOADING-DEBUG] handleSendMessage ENTRY: text=${msgText?.substring(0, 50)}, explicitThreadId=${explicitThreadId}, hasExplicitThread=${!!explicitThread}, requestId=${requestId}, skipIntentUnderstanding=${skipIntentUnderstanding}`);

        console.log('[ChatSidebar] 🔥 handleSendMessage called with:', {
            text: msgText?.substring(0, 50),
            explicitThreadId,
            hasExplicitThread: !!explicitThread,
            explicitThreadDataSource: explicitThread?.data_source_id,
            explicitThreadMessagesCount: explicitThread?.messages?.length || 0,
            currentIsLoading,
            currentLoadingThreadId,
            activeThreadId,
            requestId,
            skipIntentUnderstanding
        });

        // 确定目标会话ID（提前计算，用于后续的加载状态检查）
        const targetThreadId = explicitThread?.id || explicitThreadId || activeThreadId;

        console.log('[ChatSidebar] 🔍 Loading state check:', {
            currentIsLoading,
            currentLoadingThreadId,
            targetThreadId,
            matches: currentLoadingThreadId === targetThreadId,
            willBlock: currentIsLoading && currentLoadingThreadId === targetThreadId
        });

        // 只有当目标会话正在加载时才阻止发送消息
        // 这允许用户在一个会话加载时，在其他会话中发送消息（并行会话支持）
        if (!msgText.trim()) {
            systemLog.info('[LOADING-DEBUG] handleSendMessage EARLY RETURN: empty text');
            console.log('[ChatSidebar] ❌ handleSendMessage early return: empty text');
            return;
        }

        if (currentIsLoading && currentLoadingThreadId === targetThreadId) {
            systemLog.info(`[LOADING-DEBUG] handleSendMessage EARLY RETURN: analysis in progress for target thread ${targetThreadId}`);
            console.log('[ChatSidebar] ⚠️ Analysis in progress for target thread, blocking new request');
            // 显示Toast提示
            setToast({
                message: t('analysis_in_progress'),
                type: 'warning'
            });
            return;
        }

        // 防止重复的操作请求（特别是按钮点击）
        const actionKey = `${explicitThread?.id || activeThreadId || 'no-thread'}-${msgText}`;
        const currentTime = Date.now();

        if (pendingActionRef.current === actionKey) {
            systemLog.info(`[LOADING-DEBUG] handleSendMessage EARLY RETURN: duplicate action pending, actionKey=${actionKey}`);
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

        // Track if user message was already added (to prevent duplication)
        let userMessageAlreadyAdded = false;

        // Check if we need intent understanding
        // Only for user-initiated messages (not system-generated or action clicks)
        // Also skip for free chat sessions (no data source)
        const isUserInitiated = !explicitThread && !requestId && !skipIntentUnderstanding && !text; // text参数表示是通过代码调用的（如点击建议）

        // Check if this is a free chat session (no data source) - skip intent understanding
        const targetThread = threads?.find(t => t.id === (explicitThreadId || activeThreadId));
        const isFreeChatSession = !targetThread?.data_source_id || targetThread?.data_source_id === '';

        if (isUserInitiated && !isFreeChatSession) {
            try {
                const cfg = await GetConfig();

                if (cfg.autoIntentUnderstanding !== false) {
                    // Add user message first
                    setInput('');

                    // Generate intent suggestions
                    setIsGeneratingIntent(true);
                    setPendingMessage(msgText);
                    pendingMessageRef.current = msgText; // 立即同步 ref，避免闭包问题
                    setPendingThreadId(targetThreadId || '');
                    pendingThreadIdRef.current = targetThreadId || ''; // 立即同步 ref
                    // 清理之前累积的排除项（新的意图理解流程开始）
                    setExcludedIntentSuggestions([]);
                    excludedIntentSuggestionsRef.current = []; // 立即同步 ref

                    // 通知 LoadingStateManager 意图理解开始，并设置进度消息
                    if (targetThreadId) {
                        loadingStateManager.setLoading(targetThreadId, true);
                        loadingStateManager.updateProgress(targetThreadId, {
                            stage: 'initializing',
                            progress: 0,
                            message: t('generating_intent'),
                            step: 1,
                            total: 2
                        });
                    }

                    // Create a temporary message ID for the intent message
                    const tempIntentMsgId = `intent_${Date.now()}`;
                    setIntentMessageId(tempIntentMsgId);
                    intentMessageIdRef.current = tempIntentMsgId; // 立即同步 ref

                    // Add "generating intent" message to thread
                    let intentThread = threads.find(t => t.id === targetThreadId);
                    let actualThreadId = targetThreadId;
                    if (!intentThread) {
                        // Create new thread if needed
                        const title = msgText.slice(0, 30);
                        intentThread = await CreateChatThread('', title);
                        setThreads([intentThread, ...threads]);
                        setActiveThreadId(intentThread.id);
                        EventsEmit('chat-thread-created', intentThread.id);
                        actualThreadId = intentThread.id;
                        setPendingThreadId(actualThreadId);
                        pendingThreadIdRef.current = actualThreadId; // 立即同步 ref
                        // 更新 LoadingStateManager 的 threadId
                        if (actualThreadId) {
                            loadingStateManager.setLoading(actualThreadId, true);
                            loadingStateManager.updateProgress(actualThreadId, {
                                stage: 'initializing',
                                progress: 0,
                                message: t('generating_intent'),
                                step: 1,
                                total: 2
                            });
                        }
                    }

                    // Add user message (不添加进度消息到历史，使用 AnalysisStatusIndicator 显示)
                    const userMsg: main.ChatMessage = {
                        id: `user_${Date.now()}`,
                        role: 'user',
                        content: msgText,
                        timestamp: Date.now()
                    };

                    intentThread.messages = [...(intentThread.messages || []), userMsg];
                    let currentThreads = threads.map(t => t.id === intentThread!.id ? intentThread! : t);
                    setThreads(currentThreads);
                    await SaveChatHistory(currentThreads);

                    // Mark that user message was added
                    userMessageAlreadyAdded = true;

                    try {
                        const suggestions = await GenerateIntentSuggestions(actualThreadId || '', msgText);

                        if (suggestions && suggestions.length > 0) {
                            // Add intent suggestions message directly
                            // Use the new formatIntentSuggestions function (Task 5.1)
                            // excludedCount is 0 for initial generation
                            // 传入 threadId 和 intentMsgId 以便嵌入到按钮数据中
                            const intentContent = formatIntentSuggestions(suggestions, 0, msgText, actualThreadId, tempIntentMsgId);

                            const intentSuggestionsMsg: main.ChatMessage = {
                                id: tempIntentMsgId,
                                role: 'assistant',
                                content: intentContent,
                                timestamp: Date.now()
                            };

                            // 添加意图建议消息
                            intentThread.messages = [...intentThread.messages, intentSuggestionsMsg];
                            currentThreads = currentThreads.map(t => t.id === intentThread!.id ? intentThread! : t);
                            setThreads(currentThreads);
                            await SaveChatHistory(currentThreads);

                            setIntentSuggestions(suggestions);
                            intentSuggestionsRef.current = suggestions; // 立即同步 ref
                            setPendingThreadId(actualThreadId || '');
                            pendingThreadIdRef.current = actualThreadId || ''; // 立即同步 ref
                            // 注意：不要清空 intentMessageId，因为重新理解流程需要它
                            setIsGeneratingIntent(false);
                            // 清除加载状态，等待用户选择
                            if (actualThreadId) {
                                loadingStateManager.setLoading(actualThreadId, false);
                            }
                            clearTimeout(timeoutId);
                            if (pendingActionRef.current === actionKey) {
                                pendingActionRef.current = null;
                            }
                            return; // Wait for user to click a suggestion
                        } else {
                            // No suggestions generated, clear intent state
                            setIsGeneratingIntent(false);
                            setIntentMessageId('');
                            // 清除加载状态
                            if (actualThreadId) {
                                loadingStateManager.setLoading(actualThreadId, false);
                            }
                        }
                    } catch (error) {
                        console.error('[Intent] Failed to generate suggestions:', error);
                        
                        // User message is already added, so we should NOT add it again below
                        // Clear intent state on error
                        setIsGeneratingIntent(false);
                        setIntentMessageId('');
                        // 清除加载状态
                        if (actualThreadId) {
                            loadingStateManager.setLoading(actualThreadId, false);
                        }
                    }
                }
            } catch (error) {
                console.error('[Intent] Error checking config:', error);
                // 清除加载状态
                if (targetThreadId) {
                    loadingStateManager.setLoading(targetThreadId, false);
                }
            }
        }

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
                EventsEmit('chat-thread-created', newThread.id);
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
        // 但如果是 skipIntentUnderstanding=true（来自"坚持我的请求"），则跳过此检查
        // 因为用户消息已经在意图理解阶段添加，我们需要发送它进行分析
        const existingMessages = currentThread.messages || [];
        const recentMessages = existingMessages.slice(-5); // 检查最近5条消息（增加检查范围）
        const isDuplicate = recentMessages.some(msg =>
            msg.role === 'user' &&
            msg.content === msgText &&
            (currentTime - (msg.timestamp * 1000)) < 10000 // 增加到10秒内的重复消息
        );

        if (isDuplicate && !skipIntentUnderstanding) {
            console.log('[ChatSidebar] Ignoring duplicate message (found in recent messages):', msgText.substring(0, 50));
            // 清除操作标记
            if (pendingActionRef.current === actionKey) {
                pendingActionRef.current = null;
            }
            clearTimeout(timeoutId);
            return;
        }

        if (isDuplicate && skipIntentUnderstanding) {
            systemLog.info('[ChatSidebar] Duplicate message found but skipIntentUnderstanding=true, proceeding with analysis');
        }

        // Only add user message if it wasn't already added during intent understanding
        let userMsg: main.ChatMessage;

        // When skipIntentUnderstanding is true and message is duplicate, use existing message
        if (skipIntentUnderstanding && isDuplicate) {
            systemLog.info('[ChatSidebar] Using existing user message from intent understanding phase');
            const existingUserMsg = currentThread.messages?.find(m =>
                m.role === 'user' && m.content === msgText
            );
            if (existingUserMsg) {
                userMsg = existingUserMsg;
                console.log('[ChatSidebar] Found existing user message:', userMsg.id);
            } else {
                // Fallback: create new message (shouldn't happen)
                console.log('[ChatSidebar] Warning: Could not find existing user message, creating new one');
                userMsg = new main.ChatMessage();
                userMsg.id = Date.now().toString();
                userMsg.role = 'user';
                userMsg.content = msgText;
                userMsg.timestamp = Math.floor(Date.now() / 1000);
            }
        } else if (!userMessageAlreadyAdded) {
            userMsg = new main.ChatMessage();
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

            // Set loading state BEFORE await to prevent flash of "cancelled" state
            // (await breaks React batching, so setThreads renders before setIsLoading if we wait)
            const isFreeChatEarly = !currentThread.data_source_id || currentThread.data_source_id === '';
            if (!isFreeChatEarly) {
                setIsLoading(true);
                setLoadingThreadId(currentThreadId);
            }

            setInput('');

            // Await save before sending message to prevent race condition
            // IMPORTANT: Reload from backend first to avoid overwriting backend-saved
            // messages (e.g., error messages from previous failed analyses).
            // The frontend state may be stale if the backend added messages via AddMessage.
            try {
                const freshThreads = await GetChatHistory();
                const freshThread = freshThreads?.find(t => t.id === currentThreadId);
                if (freshThread) {
                    // Check if user message already exists in fresh data
                    const userMsgExists = freshThread.messages?.some(m => m.id === userMsg.id);
                    if (!userMsgExists) {
                        freshThread.messages = [...(freshThread.messages || []), userMsg];
                    }
                    const mergedThreads = freshThreads.map(t =>
                        t.id === currentThreadId ? freshThread : t
                    );
                    await SaveChatHistory(mergedThreads);
                } else {
                    // Thread not found in backend, save our version
                    await SaveChatHistory(updatedThreads);
                }
            } catch (saveErr) {
                console.error('[ChatSidebar] Failed to merge and save, falling back:', saveErr);
                await SaveChatHistory(updatedThreads);
            }
        } else {
            systemLog.info('[ChatSidebar] User message already added during intent understanding, skipping duplicate');
            // Find the existing user message that was added during intent understanding
            const existingUserMsg = currentThread.messages?.find(m =>
                m.role === 'user' && m.content === msgText
            );
            if (existingUserMsg) {
                userMsg = existingUserMsg;
            } else {
                // Fallback: create a placeholder (shouldn't happen)
                userMsg = new main.ChatMessage();
                userMsg.id = Date.now().toString();
                userMsg.role = 'user';
                userMsg.content = msgText;
                userMsg.timestamp = Math.floor(Date.now() / 1000);
            }
        }

        // Check if this is a free chat session (no data source)
        // Note: We use currentThread here which is the resolved thread after all the logic above
        const isFreeChatSessionForLoading = !currentThread.data_source_id || currentThread.data_source_id === '';

        systemLog.info(`[LOADING-DEBUG] Loading state decision: threadId=${currentThreadId}, dataSourceId=${currentThread.data_source_id}, isFreeChatSession=${isFreeChatSessionForLoading}`);

        // Only set loading state for non-free-chat sessions
        // Free chat uses streaming which provides its own visual feedback
        if (!isFreeChatSessionForLoading) {
            systemLog.info(`Setting loading state for threadId=${currentThreadId}`);
            setIsLoading(true);
            setLoadingThreadId(currentThreadId); // 记录正在加载的会话ID

            // 通知 LoadingStateManager
            loadingStateManager.setLoading(currentThreadId, true);
            loadingStateManager.updateProgress(currentThreadId, {
                stage: 'initializing',
                progress: 5,
                message: t('stage_initializing'),
                step: 0,
                total: 0
            });
        } else {
            systemLog.info(`Skipping loading state - free chat session`);
        }

        try {
            let response: string;
            if (isFreeChatSessionForLoading) {
                // Use free chat mode - direct LLM conversation without data analysis
                console.log('[ChatSidebar] 💬 Free chat mode - using SendFreeChatMessage');
                response = await SendFreeChatMessage(currentThreadId, msgText, userMsg.id);
            } else {
                // Task 3.1: Pass requestId to backend for request tracking (Requirements 1.3, 4.3, 4.4)
                response = await SendMessage(currentThreadId, msgText, userMsg.id, requestId || '');
            }

            // CRITICAL: Reload threads from backend to get chart_data attached by backend
            // The backend's attachChartToUserMessage modifies the user message after SendMessage
            // Backend also creates the assistant message with chart_data
            const reloadedThreads = await GetChatHistory();

            // Create assistant message (may be used in fallback)
            const assistantMsg = new main.ChatMessage();
            assistantMsg.id = (Date.now() + 1).toString();
            assistantMsg.role = 'assistant';
            assistantMsg.content = response;
            assistantMsg.timestamp = Math.floor(Date.now() / 1000);

            // Find the reloaded thread (includes backend modifications like chart_data)
            const reloadedThread = reloadedThreads?.find(t => t.id === currentThreadId);

            if (reloadedThread) {
                // Ensure user message is preserved (in case backend hasn't saved it yet)
                const userMessageExists = reloadedThread.messages.some(m => m.id === userMsg.id);
                if (!userMessageExists) {
                    console.log("[ChatSidebar] User message not found in reloaded thread, preserving it");
                    // Find the position to insert the user message (should be before assistant message)
                    const assistantIndex = reloadedThread.messages.findIndex(m => m.role === 'assistant' && m.content === response);
                    if (assistantIndex !== -1) {
                        // Insert user message before assistant message
                        reloadedThread.messages = [
                            ...reloadedThread.messages.slice(0, assistantIndex),
                            userMsg,
                            ...reloadedThread.messages.slice(assistantIndex)
                        ];
                    } else {
                        // Append user message at the end
                        reloadedThread.messages = [...reloadedThread.messages, userMsg];
                    }
                }

                // Check if backend already added the assistant message
                const lastMessage = reloadedThread.messages[reloadedThread.messages.length - 1];
                const backendAddedAssistant = lastMessage && lastMessage.role === 'assistant' && lastMessage.content === response;

                if (!backendAddedAssistant) {
                    // Backend didn't add assistant message, add it ourselves (backward compatibility)
                    console.log("[ChatSidebar] Backend didn't add assistant message, adding it manually");
                    reloadedThread.messages = [...(reloadedThread.messages || []), assistantMsg];
                } else {
                    console.log("[ChatSidebar] Backend already added assistant message");
                }

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
            console.error('[ChatSidebar] Analysis error:', error);

            // Create an error message directly in the frontend and add it to the thread.
            const errorMsg = new main.ChatMessage();
            errorMsg.id = `error_${Date.now()}`;
            errorMsg.role = 'assistant';
            errorMsg.timestamp = Math.floor(Date.now() / 1000);

            // Extract meaningful error text from the error object
            let errorText: string;
            if (typeof error === 'string') {
                errorText = error;
            } else if (error instanceof Error) {
                errorText = error.message;
            } else {
                errorText = 'Unknown error';
            }
            errorMsg.content = `❌ **分析出错**\n\n${errorText}`;

            // CRITICAL: Store the error message in pendingErrorRef BEFORE updating state.
            // This ensures that if any event handler (analysis-cancelled, thread-updated, etc.)
            // calls loadThreads() during or after our setThreads, the error message will be
            // merged back into the loaded data and won't be lost.
            pendingErrorRef.current = { threadId: currentThreadId, errorMsg };

            // Build the updated threads with the error message for both state and persistence
            const currentThreads = threadsRef.current || [];
            const newThreads = [...currentThreads];
            const threadIndex = newThreads.findIndex(t => t.id === currentThreadId);
            if (threadIndex !== -1) {
                const thread = newThreads[threadIndex];
                newThreads[threadIndex] = main.ChatThread.createFrom({
                    ...thread,
                    messages: [...(thread.messages || []), errorMsg]
                });
            }

            // Update state immediately so the user sees the error
            setThreads(newThreads);

            // Await SaveChatHistory to persist the error message before any loadThreads() can read stale data
            try {
                await SaveChatHistory(newThreads);
                console.log('[ChatSidebar] Error message persisted successfully');
            } catch (saveErr) {
                console.error('[ChatSidebar] Failed to save error message:', saveErr);
            }

            // Clear the pending error ref now that it's persisted
            // Use a small delay to handle any in-flight loadThreads() calls
            setTimeout(() => {
                pendingErrorRef.current = null;
            }, 2000);
        } finally {
            clearTimeout(timeoutId); // 清除定时器
            console.log('[ChatSidebar] 🧹 Clearing loading state:', {
                wasLoading: isLoading,
                wasLoadingThreadId: loadingThreadId,
                currentThreadId
            });
            setIsLoading(false);
            setLoadingThreadId(null); // 清除加载会话ID
            // 通知 LoadingStateManager
            loadingStateManager.setLoading(currentThreadId, false);
            // 清除操作标记
            if (pendingActionRef.current === actionKey) {
                pendingActionRef.current = null;
            }

            // Auto-update dashboard after analysis completes
            // Find the user message that was just sent and trigger dashboard update
            try {
                const updatedThread = threadsRef.current.find(t => t.id === currentThreadId);
                if (updatedThread && updatedThread.messages) {
                    // Find the user message we just sent (by matching the message text)
                    const userMessage = updatedThread.messages.find(msg =>
                        msg.role === 'user' &&
                        msg.content === msgText &&
                        msg.id === userMsg.id
                    );

                    if (userMessage) {
                        console.log('[ChatSidebar] 🎯 Auto-updating dashboard after analysis completion');

                        // Emit event to update dashboard UI state (chart data was already displayed in real-time during analysis)
                        EventsEmit('user-message-clicked', {
                            threadId: updatedThread.id,
                            messageId: userMessage.id,
                            content: userMessage.content
                        });
                    }
                }
            } catch (autoUpdateError) {
                console.error('[ChatSidebar] Failed to auto-update dashboard:', autoUpdateError);
                // Don't throw - this is a nice-to-have feature
            }
        }
    };

    // Handle intent selection
    // Requirements: 4.1, 4.2, 4.3
    const handleIntentSelect = async (suggestionIndex: number) => {
        // Use refs to get the latest state values (avoid closure issues)
        const currentIntentSuggestions = intentSuggestionsRef.current;
        const currentPendingThreadId = pendingThreadIdRef.current;
        const currentIntentMessageId = intentMessageIdRef.current;
        const currentPendingMessage = pendingMessageRef.current;

        systemLog.info(`[handleIntentSelect] called: suggestionIndex=${suggestionIndex}, intentSuggestionsLength=${currentIntentSuggestions.length}`);

        // 检查是否是"重新理解"选项（index === suggestions.length）
        if (suggestionIndex === currentIntentSuggestions.length) {
            systemLog.info('[handleIntentSelect] Retry intent understanding triggered');
            // Call the handleRetryIntentUnderstandingWithData function (Task 6.1)
            await handleRetryIntentUnderstandingWithData({
                threadId: currentPendingThreadId,
                originalRequest: currentPendingMessage,
                currentSuggestions: currentIntentSuggestions,
                intentMessageId: currentIntentMessageId
            });
            return;
        }

        // 检查是否是"坚持我的请求"选项（index === suggestions.length + 1）
        if (suggestionIndex === currentIntentSuggestions.length + 1) {
            systemLog.info('[handleIntentSelect] Stick to original request triggered');
            // Call the handleStickToOriginalWithData function (Task 7.1)
            await handleStickToOriginalWithData({
                threadId: currentPendingThreadId,
                originalRequest: currentPendingMessage,
                intentMessageId: currentIntentMessageId
            });
            return;
        }

        if (suggestionIndex < 0 || suggestionIndex >= currentIntentSuggestions.length) {
            systemLog.warn('[handleIntentSelect] Invalid suggestion index');
            return;
        }

        systemLog.info('[handleIntentSelect] Normal intent selection, continuing with analysis');
        const suggestion = currentIntentSuggestions[suggestionIndex];
        const selectedQuery = suggestion.query;

        // Record the intent selection for preference learning (Requirement 4.3)
        try {
            // Convert to the format expected by the backend
            const intentToRecord = {
                id: suggestion.id,
                title: suggestion.title,
                description: suggestion.description,
                icon: suggestion.icon,
                query: suggestion.query
            };
            await RecordIntentSelection(currentPendingThreadId, intentToRecord);
            systemLog.info(`[handleIntentSelect] Intent selection recorded successfully: ${suggestion.title}`);
        } catch (error) {
            // Don't block the flow if recording fails
            systemLog.error(`[handleIntentSelect] Failed to record intent selection: ${error}`);
        }

        // Remove the intent message from thread
        if (currentIntentMessageId && currentPendingThreadId) {
            const thread = threadsRef.current.find(t => t.id === currentPendingThreadId);
            if (thread) {
                const filteredMessages = thread.messages.filter(m => m.id !== currentIntentMessageId);
                const updatedThreads = threadsRef.current.map(t => {
                    if (t.id === thread.id) {
                        return { ...t, messages: filteredMessages } as main.ChatThread;
                    }
                    return t;
                });
                setThreads(updatedThreads);
                await SaveChatHistory(updatedThreads);
            }
        }

        // Continue with normal message sending flow using the refined query (Requirement 4.1)
        // Skip intent understanding since this is already a refined query
        // Note: handleSendMessage will set loading state internally, no need to set it here
        await handleSendMessage(selectedQuery, currentPendingThreadId, undefined, undefined, true);

        // Clear all intent-related state using clearIntentState (Requirement 4.2)
        clearIntentState();
    };

    /**
     * 数据驱动的重试意图理解接口
     * 所有数据从按钮标签中解析，完全避免闭包问题
     */
    interface RetryIntentData {
        threadId: string;
        originalRequest: string;
        currentSuggestions: IntentSuggestion[];
        intentMessageId: string;
    }

    /**
     * 处理"重新理解"按钮点击 - 数据驱动版本
     * 所有需要的数据都从参数传入，不依赖任何闭包或 ref
     * 
     * Requirements: 2.1, 2.4, 2.5
     */
    const handleRetryIntentUnderstandingWithData = async (data: RetryIntentData): Promise<void> => {
        systemLog.info(`[handleRetryIntentUnderstandingWithData] 🚀 Called with data: threadId=${data.threadId}, originalRequest=${data.originalRequest?.substring(0, 50)}, suggestionsCount=${data.currentSuggestions?.length}, intentMessageId=${data.intentMessageId}`);

        const { threadId, originalRequest, currentSuggestions, intentMessageId } = data;

        // Validate data
        if (!originalRequest || !threadId) {
            systemLog.error(`[handleRetryIntentUnderstandingWithData] ❌ Missing required data: hasOriginalRequest=${!!originalRequest}, hasThreadId=${!!threadId}`);
            setToast({
                message: t('intent_retry_error'),
                type: 'error'
            });
            return;
        }

        // Get current excluded suggestions from ref (this is cumulative)
        const currentExcludedSuggestions = excludedIntentSuggestionsRef.current;

        // Step 1: Add current suggestions to excluded list
        const newExcludedSuggestions = [...currentExcludedSuggestions, ...currentSuggestions];
        setExcludedIntentSuggestions(newExcludedSuggestions);
        excludedIntentSuggestionsRef.current = newExcludedSuggestions;

        systemLog.info(`[handleRetryIntentUnderstandingWithData] Updated exclusions: newExcludedCount=${newExcludedSuggestions.length}`);

        // Check if exclusions exceed threshold (just log, no toast)
        if (newExcludedSuggestions.length > 15) {
            systemLog.warn(`[handleRetryIntentUnderstandingWithData] Too many exclusions: ${newExcludedSuggestions.length}`);
        }

        // Step 2: Set loading state
        setIsGeneratingIntent(true);

        // Find the thread
        let currentThreads = threadsRef.current;
        let intentThread = currentThreads.find(t => t.id === threadId);

        // Step 2.5: 设置 LoadingStateManager 进度状态（使用 AnalysisStatusIndicator 显示）
        loadingStateManager.setLoading(threadId, true);
        loadingStateManager.updateProgress(threadId, {
            stage: 'initializing',
            progress: 0,
            message: t('regenerating_intent'),
            step: 1,
            total: 2
        });

        try {
            // Step 3: Call backend API with exclusions
            systemLog.info('[handleRetryIntentUnderstandingWithData] Calling GenerateIntentSuggestionsWithExclusions');
            
            let newSuggestions: IntentSuggestion[] = [];
            try {
                newSuggestions = await GenerateIntentSuggestionsWithExclusions(
                    threadId,
                    originalRequest,
                    newExcludedSuggestions
                );
            } catch (apiError) {
                systemLog.error(`[handleRetryIntentUnderstandingWithData] API call failed: ${apiError}`);
                setToast({
                    message: `${t('api_call_failed')}: ${apiError}`,
                    type: 'error'
                });
                throw apiError;
            }

            systemLog.info(`[handleRetryIntentUnderstandingWithData] Received new suggestions: count=${newSuggestions?.length || 0}`);

            // Step 4: Update UI with new suggestions - 添加新消息而不是更新原消息
            if (newSuggestions && newSuggestions.length > 0) {
                // 生成新的消息ID
                const newIntentMsgId = `intent-${Date.now()}`;
                const intentContent = formatIntentSuggestions(
                    newSuggestions,
                    newExcludedSuggestions.length,
                    originalRequest,
                    threadId,
                    newIntentMsgId  // 使用新的消息ID
                );

                systemLog.info(`[handleRetryIntentUnderstandingWithData] intentThread found: ${!!intentThread}`);

                if (intentThread) {
                    const newIntentMsg: main.ChatMessage = {
                        id: newIntentMsgId,
                        role: 'assistant',
                        content: intentContent,
                        timestamp: Date.now()
                    };
                    // 添加新的意图建议消息
                    const updatedMessages = [...intentThread.messages, newIntentMsg];
                    
                    const updatedThread = {
                        ...intentThread,
                        messages: updatedMessages
                    };
                    const finalThreads = currentThreads.map(t => 
                        t.id === intentThread.id ? updatedThread : t
                    );
                    
                    systemLog.info(`[handleRetryIntentUnderstandingWithData] Adding new message, finalThreads.length=${finalThreads.length}`);
                    threadsRef.current = finalThreads;
                    setThreads(finalThreads);
                    await SaveChatHistory(finalThreads);
                    
                    // 更新 intentMessageId 为新的消息ID
                    setIntentMessageId(newIntentMsgId);
                    intentMessageIdRef.current = newIntentMsgId;
                } else {
                    systemLog.error(`[handleRetryIntentUnderstandingWithData] Cannot update UI: intentThread not found`);
                    setToast({
                        message: t('intent_retry_error'),
                        type: 'error'
                    });
                }

                // Update state
                setIntentSuggestions(newSuggestions);
                intentSuggestionsRef.current = newSuggestions;
                setPendingMessage(originalRequest);
                pendingMessageRef.current = originalRequest;
                setPendingThreadId(threadId);
                pendingThreadIdRef.current = threadId;
                setIntentMessageId(intentMessageId);
                intentMessageIdRef.current = intentMessageId;

                systemLog.info('[handleRetryIntentUnderstandingWithData] Successfully updated with new suggestions');
            } else {
                // No more suggestions available
                systemLog.info('[handleRetryIntentUnderstandingWithData] No more suggestions available');

                const newNoMoreMsgId = `no-more-${Date.now()}`;
                const originalRequestBase64 = btoa(encodeURIComponent(originalRequest));
                const stickData = `[INTENT_STICK_DATA:${threadId}:${originalRequestBase64}:${newNoMoreMsgId}]`;

                const noMoreContent = `**${t('no_more_suggestions')}**\n\n` +
                    `${t('no_more_suggestions_desc')}\n\n` +
                    `*${t('excluded_count')?.replace('{count}', String(newExcludedSuggestions.length)) || `已排除 ${newExcludedSuggestions.length} 个选项`}*\n\n` +
                    `1. 📝 **${t('stick_to_original')}**: "${originalRequest.length > 30 ? originalRequest.substring(0, 30) + '...' : originalRequest}" ${stickData}\n\n` +
                    `\n*${t('click_to_use_original')}*`;

                if (intentThread) {
                    const noMoreMsg: main.ChatMessage = {
                        id: newNoMoreMsgId,
                        role: 'assistant',
                        content: noMoreContent,
                        timestamp: Date.now()
                    };
                    // 添加新消息
                    const updatedMessages = [...intentThread.messages, noMoreMsg];
                    
                    const updatedThread = {
                        ...intentThread,
                        messages: updatedMessages
                    };
                    const finalThreads = currentThreads.map(t => 
                        t.id === intentThread.id ? updatedThread : t
                    );
                    threadsRef.current = finalThreads;
                    setThreads(finalThreads);
                    await SaveChatHistory(finalThreads);
                }

                setIntentSuggestions([]);
                intentSuggestionsRef.current = [];
            }
        } catch (error) {
            systemLog.error(`[handleRetryIntentUnderstandingWithData] Error: ${error}`);

            setToast({
                message: t('intent_generation_failed'),
                type: 'error'
            });

            // Revert excluded suggestions
            setExcludedIntentSuggestions(currentExcludedSuggestions);
            excludedIntentSuggestionsRef.current = currentExcludedSuggestions;
        } finally {
            setIsGeneratingIntent(false);
            // 清除加载状态
            loadingStateManager.setLoading(threadId, false);
        }
    };

    /**
     * 数据驱动的坚持原始请求接口
     */
    interface StickToOriginalData {
        threadId: string;
        originalRequest: string;
        intentMessageId: string;
    }

    /**
     * 处理"坚持我的请求"按钮点击 - 数据驱动版本
     * 所有需要的数据都从参数传入，不依赖任何闭包或 ref
     */
    const handleStickToOriginalWithData = async (data: StickToOriginalData): Promise<void> => {
        systemLog.info(`[handleStickToOriginalWithData] 🚀 Called with data: threadId=${data.threadId}, originalRequest=${data.originalRequest?.substring(0, 50)}, intentMessageId=${data.intentMessageId}`);

        const { threadId, originalRequest, intentMessageId } = data;

        // Validate data
        if (!originalRequest || !threadId) {
            systemLog.error(`[handleStickToOriginalWithData] ❌ Missing required data`);
            setToast({
                message: t('stick_to_original_error'),
                type: 'error'
            });
            return;
        }

        // Step 1: Remove the intent message from thread
        const currentThreads = threadsRef.current;
        const intentThread = currentThreads.find(t => t.id === threadId);

        if (intentThread && intentMessageId) {
            const filteredMessages = intentThread.messages.filter(m => m.id !== intentMessageId);
            const updatedThreads = currentThreads.map(t => {
                if (t.id === intentThread.id) {
                    return { ...t, messages: filteredMessages } as main.ChatThread;
                }
                return t;
            });
            setThreads(updatedThreads);
            await SaveChatHistory(updatedThreads);
            systemLog.info('[handleStickToOriginalWithData] Removed intent message from thread');
        }

        // Step 2: Clear all intent-related state
        clearIntentState();

        // Step 3: Send the original request with skipIntentUnderstanding=true
        // Note: handleSendMessage will set loading state internally, no need to set it here
        systemLog.info(`[handleStickToOriginalWithData] Sending original request: ${originalRequest}`);
        await handleSendMessage(originalRequest, threadId, undefined, undefined, true);

        systemLog.info('[handleStickToOriginalWithData] Completed');
    };

    /**
     * 清理所有意图相关状态
     * 清空 intentSuggestions、excludedIntentSuggestions、pendingMessage、pendingThreadId、intentMessageId
     * 同步更新 refs
     * 
     * Requirements: 8.2
     */
    const clearIntentState = (): void => {
        systemLog.info('[clearIntentState] Clearing all intent-related state');

        // Clear all state
        setIntentSuggestions([]);
        setExcludedIntentSuggestions([]);
        setPendingMessage('');
        setPendingThreadId('');
        setIntentMessageId('');
        setIsGeneratingIntent(false);

        // Sync refs immediately (avoid closure issues)
        intentSuggestionsRef.current = [];
        excludedIntentSuggestionsRef.current = [];
        pendingMessageRef.current = '';
        pendingThreadIdRef.current = '';
        intentMessageIdRef.current = '';

        systemLog.info('[clearIntentState] All intent state cleared');
    };

    // Update refs on every render to ensure they always have the latest function references
    // This is critical for the start-new-chat event listener which has empty dependencies
    handleCreateThreadRef.current = handleCreateThread;
    handleSendMessageRef.current = handleSendMessage;

    // 恢复被中止的分析：直接调用后端 SendMessage，复用原消息 ID，不创建新消息
    const handleResumeCancelledAnalysis = async (message: main.ChatMessage) => {
        if (!message.content || !activeThreadId || !activeThread) return;

        const threadId = activeThreadId;
        systemLog.info(`[ResumeCancelled] Resuming analysis for message: ${message.id}, thread: ${threadId}`);

        // 设置加载状态
        setIsLoading(true);
        setLoadingThreadId(threadId);
        loadingStateManager.setLoading(threadId, true);
        loadingStateManager.updateProgress(threadId, {
            stage: 'initializing',
            progress: 5,
            message: t('stage_initializing'),
            step: 0,
            total: 0
        });

        try {
            const requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
            // 直接调用后端，传入原消息 ID，后端会检测到消息已存在而不会重复创建
            const response = await SendMessage(threadId, message.content, message.id, requestId);

            // 重新加载线程以获取后端附加的 chart_data 和 assistant 消息
            const reloadedThreads = await GetChatHistory();
            const reloadedThread = reloadedThreads?.find(t => t.id === threadId);

            if (reloadedThread) {
                // 检查后端是否已添加 assistant 消息
                const lastMsg = reloadedThread.messages[reloadedThread.messages.length - 1];
                const backendAddedAssistant = lastMsg && lastMsg.role === 'assistant' && lastMsg.content === response;

                if (!backendAddedAssistant) {
                    // 后端未添加 assistant 消息，手动添加
                    const assistantMsg = new main.ChatMessage();
                    assistantMsg.id = (Date.now() + 1).toString();
                    assistantMsg.role = 'assistant';
                    assistantMsg.content = response;
                    assistantMsg.timestamp = Math.floor(Date.now() / 1000);
                    reloadedThread.messages = [...(reloadedThread.messages || []), assistantMsg];
                }

                // 更新线程状态
                setThreads(prevThreads => {
                    const idx = (prevThreads || []).findIndex(t => t.id === threadId);
                    const newThreads = [...(prevThreads || [])];
                    if (idx !== -1) {
                        newThreads[idx] = reloadedThread;
                    }
                    return newThreads;
                });

                await SaveChatHistory(reloadedThreads.map(t =>
                    t.id === reloadedThread.id ? reloadedThread : t
                ));
            }

            systemLog.info(`[ResumeCancelled] Analysis completed for message: ${message.id}`);
        } catch (err) {
            systemLog.error(`[ResumeCancelled] Failed to resume analysis: ${err}`);
        } finally {
            setIsLoading(false);
            setLoadingThreadId(null);
            loadingStateManager.setLoading(threadId, false);

            // 触发仪表盘更新：查找已完成的消息并发出事件
            try {
                const updatedThread = threadsRef.current.find(t => t.id === threadId);
                if (updatedThread && updatedThread.messages) {
                    const userMessage = updatedThread.messages.find(msg =>
                        msg.role === 'user' && msg.id === message.id
                    );
                    if (userMessage) {
                        EventsEmit('user-message-clicked', {
                            threadId: updatedThread.id,
                            messageId: userMessage.id,
                            content: userMessage.content
                        });
                    }
                }
            } catch (autoUpdateError) {
                systemLog.error(`[ResumeCancelled] Failed to auto-update dashboard: ${autoUpdateError}`);
            }
        }
    };

    // 处理会话切换
    const handleThreadSwitch = (threadId: string) => {
        setActiveThreadId(threadId);
        // 分析结果的显示由useEffect自动处理
    };

    const handleUserMessageClick = async (message: main.ChatMessage) => {
        // 检查消息是否完成（有对应的助手回复或有chart_data）
        let isCompleted = false;

        if (activeThread) {
            const messageIndex = activeThread.messages.findIndex(msg => msg.id === message.id);
            if (messageIndex !== -1) {
                // 检查是否有对应的助手回复
                if (messageIndex < activeThread.messages.length - 1) {
                    const nextMessage = activeThread.messages[messageIndex + 1];
                    if (nextMessage.role === 'assistant') {
                        isCompleted = true;
                    }
                }

                // 或者检查是否有分析数据
                if (message.chart_data || (message as any).has_analysis_data) {
                    isCompleted = true;
                }
            }
        }

        // 如果消息未完成，不允许点击
        if (!isCompleted) {
            console.log("[ChatSidebar] Message not completed, ignoring click:", message.id);
            return;
        }

        const threadId = activeThread?.id || '';
        const messageId = message.id;

        systemLog.warn(`[handleUserMessageClick] Clicked: threadId=${threadId}, messageId=${messageId}`);

        // 直接使用 AnalysisResultManager 恢复数据，绕过 Wails 事件系统
        // 这避免了 EventsEmit/EventsOn 的异步序列化/反序列化可能导致的数据丢失
        const manager = getAnalysisResultManager();

        // 修复 Bug 3：先清除当前数据，确保即使后续加载失败也能看到状态变化
        // 这样用户至少能看到"加载中"或"空状态"，而不是"没有任何响应"
        manager.setLoading(true, undefined, messageId);

        // Load analysis_results on-demand from backend (they are stripped from LoadThreads for performance)
        try {
            const analysisData = await GetMessageAnalysisData(threadId, messageId);
            systemLog.warn(`[handleUserMessageClick] GetMessageAnalysisData returned: keys=${Object.keys(analysisData || {}).join(',')}`);

            const analysisResults = analysisData?.analysisResults;

            if (analysisResults && analysisResults.length > 0) {
                systemLog.warn(`[handleUserMessageClick] Loaded ${analysisResults.length} analysis_results, types: ${analysisResults.map((r: any) => r.type).join(',')}`);

                // 直接调用 manager.restoreResults，不通过事件系统
                const stats = manager.restoreResults(threadId, messageId, analysisResults);
                systemLog.warn(`[handleUserMessageClick] restoreResults completed: valid=${stats.validItems}, invalid=${stats.invalidItems}, total=${stats.totalItems}, byType=${JSON.stringify(stats.itemsByType)}`);
                
                // 验证数据确实被设置了
                const verifyResults = manager.getCurrentResults();
                systemLog.warn(`[handleUserMessageClick] VERIFY after restoreResults: getCurrentResults()=${verifyResults.length}, types=${verifyResults.map((r: any) => r.type).join(',')}, currentSession=${manager.getCurrentSession()}, currentMessage=${manager.getCurrentMessage()}`);

                if (stats.errors.length > 0) {
                    stats.errors.forEach((err: string, i: number) => {
                        systemLog.warn(`[handleUserMessageClick] restoreResults error[${i}]: ${err}`);
                    });
                }
            } else {
                // Fallback to legacy chart_data format (loaded from backend)
                const chartDataToUse = analysisData?.chartData;

                if (chartDataToUse && chartDataToUse.charts && chartDataToUse.charts.length > 0) {
                    const convertedItems = chartDataToUse.charts.map((chart: any, index: number) => ({
                        id: `legacy_${messageId}_${index}`,
                        type: chart.type || 'echarts',
                        data: chart.data,
                        metadata: {
                            sessionId: threadId,
                            messageId: messageId,
                            timestamp: Date.now()
                        },
                        source: 'restored'
                    }));

                    systemLog.warn(`[handleUserMessageClick] Converted ${convertedItems.length} legacy chart_data items`);
                    const stats = manager.restoreResults(threadId, messageId, convertedItems);
                    systemLog.warn(`[handleUserMessageClick] legacy restoreResults: valid=${stats.validItems}, invalid=${stats.invalidItems}`);
                } else {
                    // No results found - notify empty
                    systemLog.warn(`[handleUserMessageClick] No analysis results or chart data found`);
                    manager.restoreResults(threadId, messageId, []);
                }
            }
        } catch (err) {
            systemLog.error(`[handleUserMessageClick] Failed to load analysis data: ${err}`);
            // Fallback: restore empty so UI shows empty state
            manager.restoreResults(threadId, messageId, []);
        }

        // Emit event with message data for UI state update
        EventsEmit('user-message-clicked', {
            threadId: threadId,
            messageId: messageId,
            content: message.content
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

    // Clear conversation (current thread messages only)
    const handleClearConversation = () => {
        if (!activeThreadId) return;
        setShowClearConversationConfirm(activeThreadId);
    };

    const confirmClearConversation = async () => {
        const threadId = showClearConversationConfirm;
        if (!threadId) return;
        // Preserve QAP complete state so re-execute/show-all-results buttons survive the clear
        const wasQapComplete = qapCompleteThreads.has(threadId);
        try {
            await ClearThreadMessages(threadId);
            // Clear pending error ref if it targets this thread, to prevent
            // loadThreads from re-adding a stale error message to the cleared thread
            if (pendingErrorRef.current?.threadId === threadId) {
                pendingErrorRef.current = null;
            }
            // Reload threads via loadThreads (handles pending error merging, QAP state, etc.)
            await loadThreads();
            // Restore QAP complete state for this thread (messages are gone but pack info should persist)
            if (wasQapComplete) {
                setQapCompleteThreads(prev => {
                    const next = new Set(prev);
                    next.add(threadId);
                    return next;
                });
            }
            // Clear dashboard since messages (and their analysis results) are gone
            if (activeThreadId === threadId) {
                EventsEmit('clear-dashboard');
            }
            // Clear any pending intent state
            setIntentSuggestions([]);
            setPendingMessage('');
            setPendingThreadId('');
            setIntentMessageId('');
            intentSuggestionsRef.current = [];
            pendingMessageRef.current = '';
            pendingThreadIdRef.current = '';
            intentMessageIdRef.current = '';
            // Clear comprehensive report cache for this thread
            setPreparedComprehensiveReportId(null);
            setComprehensiveReportCached(false);
            setComprehensiveReportError(null);
        } catch (err) {
            console.error('Failed to clear conversation:', err);
            EventsEmit('show-message-modal', {
                type: 'error',
                title: t('clear_conversation'),
                message: String(err)
            });
        } finally {
            setShowClearConversationConfirm(null);
        }
    };

    const cancelClearConversation = () => {
        setShowClearConversationConfirm(null);
    };

    const handleCancelAnalysis = () => {
        setShowCancelConfirm(true);
    };

    const confirmCancelAnalysis = async () => {
        systemLog.info(`confirmCancelAnalysis called: activeThreadId=${activeThreadId}`);

        // 立即发出取消事件，通知 App.tsx 和 AnalysisResultManager 更新状态
        // 这是必要的，因为后端的取消可能需要时间才能生效
        EventsEmit('analysis-cancelled', {
            threadId: activeThreadId,
            message: t('analysis_cancelled')
        });
        systemLog.debug('analysis-cancelled event emitted');

        // 立即更新本地状态
        setShowCancelConfirm(false);
        setIsLoading(false);
        setLoadingThreadId(null); // 清除加载会话ID
        // 通知 LoadingStateManager
        if (activeThreadId) {
            loadingStateManager.setLoading(activeThreadId, false);
        }
        systemLog.debug('Local loading state cleared');

        try {
            await CancelAnalysis();
            systemLog.info('Analysis cancelled successfully via backend');
        } catch (err) {
            systemLog.error(`Failed to cancel analysis: ${err}`);
            // 即使后端取消失败，前端状态已经更新，用户可以继续操作
        }
    };

    const cancelCancelAnalysis = () => {
        setShowCancelConfirm(false);
    };

    return (
        <>


            <div
                data-testid="chat-sidebar"
                style={{ width: '100%' }}
                className={`relative h-full bg-white dark:bg-[#1e1e1e] flex overflow-hidden`}
            >
                {/* Thread List Sidebar - hidden, sessions managed in left Sidebar */}
                <div
                    style={{ width: 0 }}
                    className="bg-slate-50 border-r border-slate-200 flex flex-col transition-all duration-300 overflow-hidden relative flex-shrink-0"
                >
                    {/* Collapse button on the left edge of history panel */}
                    <button
                        onClick={onClose}
                        className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-1/2 z-50 bg-white dark:bg-[#252526] border border-slate-200 dark:border-[#3c3c3c] rounded-full p-1.5 shadow-lg hover:bg-slate-50 dark:hover:bg-[#2d2d30] text-slate-400 dark:text-[#808080] hover:text-blue-500 dark:hover:text-[#569cd6] transition-all hover:scale-110"
                        title={t('collapse_chat')}
                    >
                        <ChevronLeft className="w-4 h-4" />
                    </button>

                    <div className="p-4 border-b border-slate-200 dark:border-[#3c3c3c] flex items-center justify-between bg-white/50 dark:bg-[#252526]/50 backdrop-blur-sm sticky top-0 z-10"
                    >
                        <span className="font-bold text-slate-900 dark:text-[#d4d4d4] text-[11px] uppercase tracking-[0.1em]">{t('history')}</span>
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
                                        ? 'bg-white dark:bg-[#264f78] border-blue-200 dark:border-[#264f78] text-blue-700 dark:text-[#569cd6] shadow-sm ring-1 ring-blue-100 dark:ring-[#264f78]'
                                        : 'text-slate-600 dark:text-[#d4d4d4] hover:bg-white dark:hover:bg-[#2d2d30] hover:border-slate-200 dark:hover:border-[#3c3c3c] border-transparent'
                                        }`}
                                >
                                    <div className="flex items-center gap-2.5 truncate pr-1">
                                        {/* Loading spinner for sessions with ongoing analysis - Requirements: 2.1, 2.2, 2.3 */}
                                        {isThreadLoading(thread.id) ? (
                                            <Loader2 className={`w-4 h-4 flex-shrink-0 animate-spin ${activeThreadId === thread.id ? 'text-blue-500' : 'text-blue-400'}`} />
                                        ) : thread.is_replay_session ? (
                                            <Zap className={`w-4 h-4 flex-shrink-0 ${activeThreadId === thread.id ? 'text-amber-500' : 'text-amber-400'}`} />
                                        ) : (
                                            <MessageSquare className={`w-4 h-4 flex-shrink-0 ${activeThreadId === thread.id ? 'text-blue-500' : 'text-slate-400'}`} />
                                        )}
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
                                        className="opacity-0 group-hover:opacity-100 p-1.5 hover:text-red-500 transition-all rounded-lg hover:bg-red-50 dark:hover:bg-[#2e1e1e] text-slate-400 dark:text-[#808080]"
                                    >
                                        <Trash2 className="w-3.5 h-3.5" />
                                    </button>
                                </div>
                            );
                        })}
                        {(!threads || threads.length === 0) && (
                            <div className="text-center py-12 px-4">
                                <div className="w-10 h-10 bg-slate-100 dark:bg-[#252526] rounded-full flex items-center justify-center mx-auto mb-3">
                                    <MessageSquare className="w-5 h-5 text-slate-300 dark:text-[#808080]" />
                                </div>
                                <p className="text-[10px] text-slate-400 dark:text-[#808080] font-medium">{t('no_threads_yet')}</p>
                            </div>
                        )}
                    </div>

                    <div className="p-3 border-t border-slate-200 dark:border-[#3c3c3c] bg-white/50 dark:bg-[#252526]/50">
                        <button
                            onClick={handleClearHistory}
                            className="w-full flex items-center justify-center gap-2 py-2.5 text-[10px] font-bold text-slate-500 dark:text-[#808080] hover:text-red-600 dark:hover:text-[#f14c4c] transition-colors rounded-xl hover:bg-red-50 dark:hover:bg-[#2e1e1e]"
                        >
                            <Trash2 className="w-3.5 h-3.5" />
                            {t('clear_history')}
                        </button>
                    </div>

                    {/* History Resizer (Left Edge of History Panel) */}
                    <div
                        className="absolute left-0 top-0 bottom-0 w-1 hover:bg-blue-400 cursor-col-resize z-20 transition-colors"
                        onMouseDown={(e) => { e.preventDefault(); setIsResizingHistory(true); document.body.style.cursor = 'col-resize'; }}
                    />
                </div>

                {/* Main Chat Area */}
                <div className="flex-1 flex flex-col min-w-0 bg-white dark:bg-[#1e1e1e] relative">

                    <div className="px-4 pt-3 pb-2 border-b border-slate-100 dark:border-[#2d2d30] bg-white/80 dark:bg-[#1e1e1e]/80 backdrop-blur-md z-10 relative">
                        <div className="flex items-center gap-3 mb-2">
                            <div className="bg-gradient-to-br from-[#6b8db5] to-[#5b7a9d] p-2 rounded-xl shadow-md shadow-slate-300/50 dark:shadow-slate-900/30 flex-shrink-0">
                                <MessageSquare className="w-5 h-5 text-white" />
                            </div>
                            <div className="flex items-center gap-2 min-w-0">
                                <h3 className="font-bold text-slate-900 dark:text-[#d4d4d4] text-sm tracking-tight flex-shrink-0">{activeThread && !activeThread.data_source_id ? t('free_chat') : t('ai_assistant')}</h3>
                                <span className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse flex-shrink-0" />
                                {activeThread && activeThread.data_source_id && (
                                    <span className="text-xs text-slate-500 dark:text-[#808080] font-medium truncate border border-slate-200 dark:border-[#3c3c3c] rounded-md px-2 py-0.5 bg-slate-50 dark:bg-[#252526]">
                                        {activeThread.title}
                                    </span>
                                )}
                            </div>
                            {/* Clear Conversation Button */}
                            {activeThreadId && (
                                <button
                                    onClick={handleClearConversation}
                                    disabled={isLoading || isStreaming || isGeneratingComprehensiveReport}
                                    className="ml-auto flex items-center gap-1.5 px-2.5 py-1 text-[11px] font-medium text-slate-500 dark:text-[#808080] hover:text-red-600 dark:hover:text-[#f14c4c] transition-colors rounded-lg hover:bg-red-50 dark:hover:bg-[#2e1e1e] disabled:opacity-50 disabled:cursor-not-allowed flex-shrink-0"
                                    title={t('clear_conversation')}
                                >
                                    <Trash2 className="w-3.5 h-3.5" />
                                    <span>{t('clear_conversation')}</span>
                                </button>
                            )}
                            {/* Inline Comprehensive Report Progress / Export Button */}
                            {/* Show for normal sessions (activeDataSource) and replay sessions (is_replay_session) */}
                            {activeThread && activeThread.data_source_id && (activeDataSource || activeThread.is_replay_session) && (
                                <div className="flex items-center gap-2 comprehensive-report-dropdown-container">
                                    {isGeneratingComprehensiveReport ? (
                                        /* Inline progress indicator */
                                        <div className="flex items-center gap-2 px-3 py-1 bg-[#f0f4f8] dark:bg-[#1e1e2e] border border-[#b8cade] dark:border-[#3d3d5a] rounded-lg">
                                            <Loader2 className="w-3.5 h-3.5 text-[#5b7a9d] animate-spin" />
                                            <div className="flex flex-col gap-0.5">
                                                <span className="text-[10px] font-medium text-[#5b7a9d]">{t('comprehensive_report_generating')}</span>
                                                <div className="w-28 bg-[#dce5ef] rounded-full h-1 overflow-hidden">
                                                    <div className="h-full bg-gradient-to-r from-[#7b9bb8] via-[#5b7a9d] to-[#7b9bb8] rounded-full" style={{ width: '100%', animation: 'comprehensiveReportProgress 2s ease-in-out infinite' }}></div>
                                                </div>
                                            </div>
                                            <style>{`
                                                @keyframes comprehensiveReportProgress {
                                                    0% { transform: translateX(-100%); }
                                                    50% { transform: translateX(0%); }
                                                    100% { transform: translateX(100%); }
                                                }
                                            `}</style>
                                        </div>
                                    ) : (
                                        /* Report button / Export dropdown */
                                        <div className="relative">
                                            <button
                                                onClick={() => {
                                                    if (preparedComprehensiveReportId) {
                                                        setComprehensiveReportExportDropdownOpen(!comprehensiveReportExportDropdownOpen);
                                                    } else {
                                                        setComprehensiveReportError(null);
                                                        handleGenerateComprehensiveReport();
                                                    }
                                                }}
                                                className={`text-[10px] px-2.5 py-1 rounded-lg font-medium flex items-center gap-1.5 transition-all ${
                                                    preparedComprehensiveReportId
                                                        ? 'bg-green-50 border border-green-200 text-green-600 hover:bg-green-100'
                                                        : comprehensiveReportError
                                                            ? 'bg-red-50 border border-red-200 text-red-600 hover:bg-red-100'
                                                            : 'bg-indigo-50 border border-indigo-200 text-indigo-600 hover:bg-indigo-100'
                                                }`}
                                                title={preparedComprehensiveReportId 
                                                    ? (t('comprehensive_report_ready_title'))
                                                    : comprehensiveReportError
                                                        ? `${t('comprehensive_report_failed')}：${comprehensiveReportError}`
                                                        : t('comprehensive_report_button_title')}
                                            >
                                                <FileText className="w-3 h-3" />
                                                {preparedComprehensiveReportId 
                                                    ? (t('export_comprehensive_report'))
                                                    : comprehensiveReportError
                                                        ? (t('comprehensive_report_retry'))
                                                        : t('comprehensive_report')}
                                            </button>
                                            
                                            {/* Export Format Dropdown */}
                                            {comprehensiveReportExportDropdownOpen && preparedComprehensiveReportId && (
                                                <div className="absolute right-0 top-full mt-1 w-48 bg-white dark:bg-[#252526] rounded-lg shadow-xl border border-slate-200 dark:border-[#3c3c3c] py-1 z-50">
                                                    <div className="px-3 py-1 text-[9px] font-medium text-slate-400 dark:text-[#808080] uppercase tracking-wider">
                                                        {t('select_export_format')}
                                                    </div>
                                                    {comprehensiveReportCached && (
                                                        <div className="px-3 py-1 text-[9px] text-green-600 dark:text-[#6a9955] bg-green-50 dark:bg-[#1e2a1e] border-b border-slate-100 dark:border-[#3c3c3c]">
                                                            {t('comprehensive_report_using_cache')}
                                                        </div>
                                                    )}
                                                    <button
                                                        onClick={() => { setComprehensiveReportExportDropdownOpen(false); exportComprehensiveReportAs('word'); }}
                                                        className="w-full flex items-center gap-2 px-3 py-1.5 text-[10px] text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                                    >
                                                        <FileChartColumn size={12} className="flex-shrink-0 text-indigo-500" />
                                                        <span>{t('export_as_word')}</span>
                                                    </button>
                                                    <button
                                                        onClick={() => { setComprehensiveReportExportDropdownOpen(false); exportComprehensiveReportAs('pdf'); }}
                                                        className="w-full flex items-center gap-2 px-3 py-1.5 text-[10px] text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                                    >
                                                        <FileChartColumn size={12} className="flex-shrink-0 text-rose-500" />
                                                        <span>{t('export_as_pdf')}</span>
                                                    </button>
                                                    <div className="border-t border-slate-100 dark:border-[#3c3c3c] mt-1 pt-1">
                                                        <button
                                                            onClick={() => { 
                                                                setPreparedComprehensiveReportId(null); 
                                                                setComprehensiveReportExportDropdownOpen(false); 
                                                                handleGenerateComprehensiveReport(); 
                                                            }}
                                                            className="w-full flex items-center gap-2 px-3 py-1.5 text-[10px] text-slate-400 dark:text-[#808080] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                                                        >
                                                            <span>{t('regenerate_comprehensive_report')}</span>
                                                        </button>
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                        {activeDataSource ? (
                            <div className="rounded-lg border border-slate-200 dark:border-[#3c3c3c] bg-slate-50/80 dark:bg-[#252526] px-3 py-2.5">
                                <div className="flex items-center gap-2 flex-wrap">
                                    <span
                                        className="text-[11px] font-semibold text-blue-600 dark:text-[#569cd6]"
                                        title={activeDataSource.name}
                                    >
                                        {getDataSourceIcon(activeDataSource.type)} {activeDataSource.name}
                                    </span>
                                    <span className="text-[9px] px-1.5 py-0.5 bg-blue-50 dark:bg-[#1a2332] text-blue-600 dark:text-[#569cd6] rounded font-medium uppercase tracking-wide">
                                        {activeDataSource.type}
                                    </span>
                                    {activeDataSource.analysis?.schema?.length > 0 && (
                                        <button
                                            className="text-[9px] px-1.5 py-0.5 bg-white dark:bg-[#3c3c3c] text-blue-600 dark:text-[#569cd6] rounded border border-blue-200 dark:border-[#4d4d4d] flex items-center gap-1 cursor-pointer hover:bg-blue-50 dark:hover:bg-[#2a2d2e] hover:border-blue-300 dark:hover:border-[#569cd6] transition-colors font-medium"
                                            onClick={() => EventsEmit('open-data-browser', { sourceId: activeDataSource.id, sourceName: activeDataSource.name })}
                                            title={t('browse_data')}
                                        >
                                            <Database className="w-2.5 h-2.5" />
                                            {t('ds_tables_count').replace('{0}', String(activeDataSource.analysis.schema.length))}
                                        </button>
                                    )}
                                </div>
                                {activeDataSource.analysis?.summary && (
                                    <p className="text-[10px] text-slate-500 dark:text-[#808080] mt-1.5 leading-relaxed">
                                        {activeDataSource.analysis.summary}
                                    </p>
                                )}
                            </div>
                        ) : null}
                    </div>

                    {/* QAP Replay Session Metadata Banner (Requirements: 6.3, 5.7, 6.4) */}
                    {activeThread?.is_replay_session && activeThread.pack_metadata && (
                        <div className="px-4 py-3 border-b border-amber-200 dark:border-amber-800/40 bg-amber-50/80 dark:bg-amber-900/20">
                            <div className="flex items-center gap-2 mb-2">
                                <Zap className="w-4 h-4 text-amber-500" />
                                <span className="text-xs font-semibold text-amber-700 dark:text-amber-400">
                                    {t('replay_session_badge')}
                                </span>
                            </div>
                            <div className="grid grid-cols-3 gap-2 text-[11px]">
                                <div>
                                    <span className="text-slate-500 dark:text-[#808080]">{t('replay_session_author')}:</span>{' '}
                                    <span className="font-medium text-slate-700 dark:text-[#d4d4d4]">{activeThread.pack_metadata.author}</span>
                                </div>
                                <div>
                                    <span className="text-slate-500 dark:text-[#808080]">{t('replay_session_created_at')}:</span>{' '}
                                    <span className="font-medium text-slate-700 dark:text-[#d4d4d4]">
                                        {new Date(activeThread.pack_metadata.created_at).toLocaleString()}
                                    </span>
                                </div>
                                <div>
                                    <span className="text-slate-500 dark:text-[#808080]">{t('replay_session_source')}:</span>{' '}
                                    <span className="font-medium text-slate-700 dark:text-[#d4d4d4]">{activeThread.pack_metadata.source_name}</span>
                                </div>
                            </div>

                            {/* Pack license info display */}
                            {packLicense && (
                                <div className="mt-2 text-[11px]">
                                    {packLicense.blocked ? (
                                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400">
                                            ⚠ {t('pack_license_blocked')}
                                        </span>
                                    ) : packLicense.pricing_model === 'free' ? (
                                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400">
                                            ✓ {t('pack_license_free')}
                                        </span>
                                    ) : packLicense.pricing_model === 'per_use' ? (
                                        packLicense.remaining_uses > 0 ? (
                                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400">
                                                🎫 {t('pack_license_remaining').replace('{remaining}', String(packLicense.remaining_uses)).replace('{total}', String(packLicense.total_uses))}
                                            </span>
                                        ) : (
                                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400">
                                                ⚠ {t('pack_license_exhausted')}
                                            </span>
                                        )
                                    ) : (packLicense.pricing_model === 'subscription' || packLicense.pricing_model === 'time_limited') && packLicense.expires_at ? (
                                        new Date(packLicense.expires_at) > new Date() ? (
                                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400">
                                                📅 {t('pack_license_expires').replace('{date}', new Date(packLicense.expires_at).toLocaleDateString())}
                                            </span>
                                        ) : (
                                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400">
                                                ⚠ {t('pack_license_expired')}
                                            </span>
                                        )
                                    ) : null}
                                </div>
                            )}

                            {/* Progress bar during execution (Requirement 5.7) */}
                            {qapProgress && qapProgress.threadId === activeThreadId && (
                                <div className="mt-3">
                                    <div className="flex items-center justify-between text-[10px] text-slate-600 dark:text-[#b0b0b0] mb-1">
                                        <span>{t('replay_session_progress').replace('{current}', String(qapProgress.currentStep)).replace('{total}', String(qapProgress.totalSteps))}</span>
                                        <span>{Math.round((qapProgress.currentStep / qapProgress.totalSteps) * 100)}%</span>
                                    </div>
                                    <div className="w-full bg-amber-100 dark:bg-amber-900/30 rounded-full h-1.5 overflow-hidden">
                                        <div
                                            className="h-full bg-amber-500 dark:bg-amber-400 rounded-full transition-all duration-300"
                                            style={{ width: `${(qapProgress.currentStep / qapProgress.totalSteps) * 100}%` }}
                                        />
                                    </div>
                                    {qapProgress.description && (
                                        <p className="text-[10px] text-slate-500 dark:text-[#808080] mt-1 truncate">{qapProgress.description}</p>
                                    )}
                                </div>
                            )}

                            {/* Re-execute button after completion (Requirement 6.4) */}
                            {activeThreadId && qapCompleteThreads.has(activeThreadId) && !(qapProgress && qapProgress.threadId === activeThreadId) && (
                                <div className="mt-3 flex items-center gap-3 flex-wrap">
                                    <span className="text-[11px] text-green-600 dark:text-green-400 font-medium">✅ {t('replay_session_complete')}</span>
                                    <button
                                        onClick={async () => {
                                            if (!activeThreadId) return;
                                            setQapReExecuting(true);
                                            setQapCompleteThreads(prev => {
                                                const next = new Set(prev);
                                                next.delete(activeThreadId!);
                                                return next;
                                            });
                                            try {
                                                await ReExecuteQuickAnalysisPack(activeThreadId);
                                            } catch (err: any) {
                                                setQapReExecuting(false);
                                                // Restore the thread to complete set so the button reappears
                                                setQapCompleteThreads(prev => {
                                                    const next = new Set(prev);
                                                    next.add(activeThreadId!);
                                                    return next;
                                                });
                                                setToast({ message: err?.message || 'Re-execution failed', type: 'error' });
                                            }
                                        }}
                                        disabled={qapReExecuting}
                                        className="flex items-center gap-1.5 px-3 py-1 text-[11px] font-medium text-white bg-amber-500 hover:bg-amber-600 dark:bg-amber-600 dark:hover:bg-amber-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                    >
                                        {qapReExecuting ? (
                                            <Loader2 className="w-3 h-3 animate-spin" />
                                        ) : (
                                            <Play className="w-3 h-3" />
                                        )}
                                        {t('replay_session_re_execute')}
                                    </button>
                                    <button
                                        onClick={async () => {
                                            if (!activeThreadId) return;
                                            setQapShowingResults(true);
                                            try {
                                                await ShowAllSessionResults(activeThreadId);
                                                setToast({ message: t('show_result_success'), type: 'success' });
                                            } catch (err: any) {
                                                setToast({ message: err?.message || t('show_result_failed'), type: 'error' });
                                            } finally {
                                                setQapShowingResults(false);
                                            }
                                        }}
                                        disabled={qapShowingResults}
                                        className="flex items-center gap-1.5 px-3 py-1 text-[11px] font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                    >
                                        {qapShowingResults ? (
                                            <Loader2 className="w-3 h-3 animate-spin" />
                                        ) : (
                                            <BarChart3 className="w-3 h-3" />
                                        )}
                                        {t('show_all_results')}
                                    </button>

                                </div>
                            )}
                        </div>
                    )}

                    <div className="flex-1 overflow-y-auto p-6 space-y-8 bg-slate-50/10 dark:bg-transparent scrollbar-thin scrollbar-thumb-slate-200 dark:scrollbar-thumb-[#424242] scrollbar-track-transparent">
                        {activeThread?.messages.map((msg, index) => {
                            // 如果是空的助手消息，跳过渲染（避免显示空气泡）
                            if (msg.role === 'assistant' && !msg.content) {
                                return null;
                            }
                            
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

                            // 为用户消息找到对应的助手消息的 timingData
                            let timingDataForUser = null;
                            if (msg.role === 'user') {
                                // 查找下一条助手消息
                                for (let i = index + 1; i < activeThread.messages.length; i++) {
                                    if (activeThread.messages[i].role === 'assistant') {
                                        timingDataForUser = (activeThread.messages[i] as any).timing_data;
                                        break;
                                    }
                                }
                            }

                            // 检查用户消息是否完成
                            const isUserMessageCompleted = msg.role === 'user' && (() => {
                                const msgIndex = activeThread.messages.findIndex(m => m.id === msg.id);
                                if (msgIndex !== -1) {
                                    // 检查是否有对应的助手回复
                                    if (msgIndex < activeThread.messages.length - 1) {
                                        const nextMsg = activeThread.messages[msgIndex + 1];
                                        if (nextMsg.role === 'assistant') {
                                            return true;
                                        }
                                    }
                                    // 或者检查是否有分析数据（chart_data已被strip，使用has_analysis_data标志）
                                    if (msg.chart_data || (msg as any).has_analysis_data) {
                                        return true;
                                    }
                                }
                                return false;
                            })();

                            // 检查用户消息是否失败（没有对应的assistant回复，且不是最后一条消息，且不在加载中）
                            const isUserMessageFailed = msg.role === 'user' && (() => {
                                // 如果正在加载，不算失败
                                if (isLoading && loadingThreadId === activeThreadId) {
                                    return false;
                                }
                                const msgIndex = activeThread.messages.findIndex(m => m.id === msg.id);
                                if (msgIndex !== -1) {
                                    // 如果是最后一条消息，不算失败（可能还没开始处理）
                                    if (msgIndex === activeThread.messages.length - 1) {
                                        return false;
                                    }
                                    // 如果没有对应的助手回复，且没有分析数据，则算失败
                                    if (msgIndex < activeThread.messages.length - 1) {
                                        const nextMsg = activeThread.messages[msgIndex + 1];
                                        if (nextMsg.role !== 'assistant' && !msg.chart_data && !(msg as any).has_analysis_data) {
                                            return true;
                                        }
                                    }
                                }
                                return false;
                            })();

                            // 检查用户消息是否是被中止/取消的分析（最后一条用户消息，没有助手回复，不在加载中）
                            // 这种消息应该允许点击以继续分析
                            // 重要：只有当前活动会话的最后一条用户消息才算被中止，其他会话的历史消息不应该显示为被中止
                            const isUserMessageCancelled = msg.role === 'user' && !isUserMessageCompleted && !isUserMessageFailed && (() => {
                                // 如果正在加载当前会话，不算被中止
                                // Check both local isLoading state AND LoadingStateManager (sessionStatus)
                                // They can be out of sync — sessionStatus is the authoritative source
                                if (isLoading && loadingThreadId === activeThreadId) {
                                    return false;
                                }
                                if (sessionStatus.isLoading) {
                                    return false;
                                }
                                // 如果有其他会话正在加载，当前会话的未完成消息也不算被中止（因为用户已经切换到新会话）
                                if (isLoading && loadingThreadId && loadingThreadId !== activeThreadId) {
                                    return false;
                                }
                                const msgIndex = activeThread.messages.findIndex(m => m.id === msg.id);
                                if (msgIndex !== -1 && msgIndex === activeThread.messages.length - 1) {
                                    // 最后一条消息是用户消息，没有助手回复 = 被中止的分析
                                    return true;
                                }
                                return false;
                            })();

                            return (
                                <MessageBubble
                                    key={msg.id || index}
                                    role={msg.role as 'user' | 'assistant'}
                                    content={msg.content}
                                    messageId={msg.id}
                                    userMessageId={userMessageId || undefined}
                                    dataSourceId={activeThread?.data_source_id}
                                    threadId={activeThreadId || undefined}
                                    isFailed={isUserMessageFailed}
                                    onRetryAnalysis={() => {
                                        // 重新分析：直接复用原消息重新发起分析，不创建新消息
                                        handleResumeCancelledAnalysis(msg);
                                    }}
                                    onActionClick={(action) => {
                                        systemLog.info(`[onActionClick] START: id=${action.id}`);
                                        systemLog.info(`[onActionClick] Label length: ${action.label?.length || 0}`);
                                        systemLog.info(`[onActionClick] Label first 200 chars: ${action.label?.substring(0, 200)}`);
                                        systemLog.info(`[onActionClick] Label last 200 chars: ${action.label?.substring(Math.max(0, (action.label?.length || 0) - 200))}`);
                                        systemLog.info(`[onActionClick] Contains INTENT_RETRY_DATA: ${action.label?.includes('[INTENT_RETRY_DATA:')}`);
                                        systemLog.info(`[onActionClick] Contains INTENT_STICK_DATA: ${action.label?.includes('[INTENT_STICK_DATA:')}`);
                                        systemLog.info(`[onActionClick] Contains INTENT_SELECT: ${action.label?.includes('[INTENT_SELECT:')}`);

                                        // ========== 数据驱动方案：从按钮标签中解析嵌入的数据 ==========
                                        
                                        // 检测并解析重试按钮数据
                                        // 格式: [INTENT_RETRY_DATA:threadId:originalRequest_base64:suggestions_json_base64:intentMsgId]
                                        const retryDataMatch = action.label?.match(/\[INTENT_RETRY_DATA:([^:]*):([^:]*):([^:]*):([^\]]*)\]/);
                                        systemLog.info(`[onActionClick] retryDataMatch result: ${retryDataMatch ? 'MATCHED' : 'NO MATCH'}`);
                                        
                                        // 调试：显示是否匹配到重试数据
                                        if (action.label?.includes('[INTENT_RETRY_DATA:')) {
                                            setToast({
                                                message: `Retry button detected, regex match: ${retryDataMatch ? 'success' : 'failed'}`,
                                                type: retryDataMatch ? 'info' : 'error'
                                            });
                                        }
                                        
                                        if (retryDataMatch) {
                                            systemLog.info(`[onActionClick] retryDataMatch groups: [1]=${retryDataMatch[1]?.substring(0, 30)}, [2]=${retryDataMatch[2]?.substring(0, 30)}, [3]=${retryDataMatch[3]?.substring(0, 30)}, [4]=${retryDataMatch[4]?.substring(0, 30)}`);
                                        }
                                        if (retryDataMatch) {
                                            systemLog.info('[onActionClick] ✅ Retry button with embedded data detected');
                                            try {
                                                const threadId = retryDataMatch[1];
                                                const originalRequest = decodeURIComponent(atob(retryDataMatch[2]));
                                                const suggestionsJson = decodeURIComponent(atob(retryDataMatch[3]));
                                                const intentMessageId = retryDataMatch[4];
                                                const currentSuggestions = JSON.parse(suggestionsJson) as IntentSuggestion[];
                                                
                                                systemLog.info(`[onActionClick] Parsed retry data: threadId=${threadId}, originalRequest=${originalRequest?.substring(0, 30)}, suggestionsCount=${currentSuggestions.length}, intentMessageId=${intentMessageId}`);
                                                
                                                handleRetryIntentUnderstandingWithData({
                                                    threadId,
                                                    originalRequest,
                                                    currentSuggestions,
                                                    intentMessageId
                                                });
                                            } catch (parseError) {
                                                systemLog.error(`[onActionClick] Failed to parse retry data: ${parseError}`);
                                            }
                                            return;
                                        }

                                        // 检测并解析坚持原始请求按钮数据
                                        // 格式: [INTENT_STICK_DATA:threadId:originalRequest_base64:intentMsgId]
                                        const stickDataMatch = action.label?.match(/\[INTENT_STICK_DATA:([^:]*):([^:]*):([^\]]*)\]/);
                                        if (stickDataMatch) {
                                            systemLog.info('[onActionClick] ✅ Stick to original button with embedded data detected');
                                            try {
                                                const threadId = stickDataMatch[1];
                                                const originalRequest = decodeURIComponent(atob(stickDataMatch[2]));
                                                const intentMessageId = stickDataMatch[3];
                                                
                                                systemLog.info(`[onActionClick] Parsed stick data: threadId=${threadId}, originalRequest=${originalRequest?.substring(0, 30)}, intentMessageId=${intentMessageId}`);
                                                
                                                handleStickToOriginalWithData({
                                                    threadId,
                                                    originalRequest,
                                                    intentMessageId
                                                });
                                            } catch (parseError) {
                                                systemLog.error(`[onActionClick] Failed to parse stick data: ${parseError}`);
                                            }
                                            return;
                                        }

                                        // 检测并解析意图选择按钮数据
                                        // 格式: [INTENT_SELECT:index:query_base64]
                                        const selectDataMatch = action.label?.match(/\[INTENT_SELECT:(\d+):([^\]]+)\]/);
                                        if (selectDataMatch) {
                                            systemLog.info('[onActionClick] ✅ Intent select button with embedded data detected');
                                            try {
                                                const index = parseInt(selectDataMatch[1], 10);
                                                const query = decodeURIComponent(atob(selectDataMatch[2]));
                                                
                                                systemLog.info(`[onActionClick] Parsed select data: index=${index}, query=${query?.substring(0, 30)}`);
                                                
                                                // 直接发送查询，跳过意图理解
                                                const currentPendingThreadId = pendingThreadIdRef.current || activeThread?.id;
                                                if (currentPendingThreadId) {
                                                    clearIntentState();
                                                    handleSendMessage(query, currentPendingThreadId, undefined, undefined, true);
                                                }
                                            } catch (parseError) {
                                                systemLog.error(`[onActionClick] Failed to parse select data: ${parseError}`);
                                            }
                                            return;
                                        }

                                        // ========== 旧方案回退：基于关键词检测（仅用于意图选择） ==========
                                        
                                        // 使用 refs 获取最新的意图状态
                                        const currentIntentMessageId = intentMessageIdRef.current;
                                        const currentIntentSuggestions = intentSuggestionsRef.current;

                                        // 检查是否是意图建议点击
                                        const isIntentMessage = msg.id === currentIntentMessageId && currentIntentSuggestions.length > 0;

                                        // 如果是意图建议消息，尝试匹配意图选项
                                        if (isIntentMessage) {
                                            const matchedIndex = currentIntentSuggestions.findIndex(s =>
                                                action.label.includes(s.title) || action.value?.includes(s.title)
                                            );
                                            if (matchedIndex >= 0) {
                                                systemLog.info(`[onActionClick] Calling handleIntentSelect with index: ${matchedIndex}`);
                                                handleIntentSelect(matchedIndex);
                                                return;
                                            }
                                        }

                                        // Normal action click - skip intent understanding
                                        systemLog.info('[onActionClick] Normal action click, sending message');
                                        handleSendMessage(action.value || action.label, activeThread?.id, undefined, undefined, true);
                                    }}
                                    onClick={msg.role === 'user' && isUserMessageCompleted ? () => handleUserMessageClick(msg) : msg.role === 'user' && isUserMessageCancelled ? () => {
                                        // 被中止的分析：直接复用原消息重新发起分析
                                        handleResumeCancelledAnalysis(msg);
                                    } : undefined}
                                    hasChart={msg.role === 'user' && !!(msg.chart_data || (msg as any).has_analysis_data)}
                                    isDisabled={msg.role === 'user' && !isUserMessageCompleted && !isUserMessageCancelled}
                                    isCancelled={isUserMessageCancelled}
                                    timingData={msg.role === 'user' ? timingDataForUser : (msg as any).timing_data}
                                    isReplaySession={activeThread?.is_replay_session}
                                    onShowResult={handleShowStepResult}
                                />
                            );
                        })}
                        {/* 显示"生成建议分析"按钮 - 当自动分析关闭且会话为空时 */}
                        {activeThreadId && suggestionButtonSessions.has(activeThreadId) && activeThread && (!activeThread.messages || activeThread.messages.length === 0) && !(isLoading && loadingThreadId === activeThreadId) && (
                            <div className="flex flex-col items-center justify-center py-12 animate-in fade-in zoom-in-95 duration-300">
                                <div className="bg-gradient-to-br from-[#f0f4f8] to-[#eaeff5] dark:from-[#1a2332] dark:to-[#1e1e2e] p-5 rounded-[2rem] mb-5 shadow-inner ring-1 ring-white dark:ring-[#3c3c3c]">
                                    <Zap className="w-8 h-8 text-[#6b8db5]" />
                                </div>
                                <p className="text-sm text-slate-500 dark:text-[#808080] mb-4 text-center max-w-[280px]">
                                    {t('ask_about_sales')}
                                </p>
                                <button
                                    onClick={handleGenerateSuggestions}
                                    className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-all shadow-md active:scale-95 font-medium text-sm"
                                >
                                    <Zap className="w-4 h-4" />
                                    {t('generate_analysis_suggestions')}
                                </button>
                            </div>
                        )}
                        {!activeThread && !(isLoading && loadingThreadId === activeThreadId) && (
                            <div className="h-full flex flex-col items-center justify-center text-center px-8 animate-in fade-in zoom-in-95 duration-500">
                                <div className="bg-gradient-to-br from-[#f0f4f8] to-[#eaeff5] dark:from-[#1a2332] dark:to-[#1e1e2e] p-6 rounded-[2.5rem] mb-6 shadow-inner ring-1 ring-white dark:ring-[#3c3c3c]">
                                    <MessageSquare className="w-10 h-10 text-[#6b8db5]" />
                                </div>
                                <h4 className="text-slate-900 dark:text-[#d4d4d4] font-extrabold text-xl tracking-tight mb-3">{t('insights_at_fingertips')}</h4>
                                <p className="text-sm text-slate-500 dark:text-[#808080] max-w-[280px] leading-relaxed font-medium">
                                    {t('ask_about_sales')}
                                </p>
                            </div>
                        )}
                        {/* Analysis Status Indicator - Requirements: 1.1, 1.2, 1.3 */}
                        {/* Display loading state indicator in chat area before AI response */}
                        {/* 仅对有数据源的会话显示分析状态指示器 */}
                        {activeThreadId && activeThread?.data_source_id && sessionStatus.isLoading && (
                            <AnalysisStatusIndicator
                                threadId={activeThreadId}
                                variant="full"
                                showMessage={true}
                                showProgress={true}
                                showCancelButton={true}
                                onCancel={async () => {
                                    systemLog.info(`AnalysisStatusIndicator onCancel called: activeThreadId=${activeThreadId}`);
                                    
                                    // 发出取消事件，通知 App.tsx 和 AnalysisResultManager 更新状态
                                    EventsEmit('analysis-cancelled', {
                                        threadId: activeThreadId,
                                        message: t('analysis_cancelled')
                                    });
                                    
                                    // 更新本地状态
                                    setIsLoading(false);
                                    setLoadingThreadId(null);
                                    if (activeThreadId) {
                                        loadingStateManager.setLoading(activeThreadId, false);
                                    }
                                    
                                    try {
                                        await CancelAnalysis();
                                        systemLog.info('Analysis cancelled successfully via backend');
                                    } catch (err) {
                                        systemLog.error(`Failed to cancel analysis: ${err}`);
                                    }
                                }}
                                className="mx-auto max-w-md animate-in fade-in slide-in-from-bottom-2 duration-300"
                            />
                        )}
                        {/* Free Chat 搜索状态指示器 */}
                        {/* 仅当 Free Chat 会话正在进行网络搜索时显示，流式输出不需要指示器 */}
                        {activeThreadId && !activeThread?.data_source_id && isSearching && (
                            <div className="flex items-start gap-4 mx-auto max-w-md animate-in fade-in slide-in-from-bottom-2 duration-300">
                                {/* AI 助手图标 */}
                                <div className="flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center shadow-sm bg-gradient-to-br from-[#6b8db5] to-[#5b7a9d] text-white">
                                    <Loader2 className="w-5 h-5 animate-spin" />
                                </div>
                                
                                {/* 状态内容区域 */}
                                <div className="flex-1 flex flex-col gap-2 p-3 bg-white dark:bg-[#252526] border border-slate-100 dark:border-[#3c3c3c] rounded-2xl rounded-tl-none shadow-sm">
                                    <div className="flex items-center gap-2">
                                        <div className="w-4 h-4 border-2 border-blue-200 dark:border-[#264f78] border-t-blue-600 dark:border-t-[#5b7a9d] rounded-full animate-spin" />
                                        <span className="text-sm text-slate-700 dark:text-[#d4d4d4] font-medium">
                                            {t('searching_web')}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        )}
                        <div ref={messagesEndRef} />
                    </div>

                    {/* Message Input Area - hidden for Replay Sessions (Requirement 6.2) */}
                    {!activeThread?.is_replay_session && (
                    <div className="p-6 border-t border-slate-100 dark:border-[#2d2d30] bg-white dark:bg-[#1e1e1e]">
                        <div className="flex items-stretch gap-3 max-w-2xl mx-auto w-full">
                            <input
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' && !((isLoading && loadingThreadId === activeThreadId) || (isStreaming && streamingThreadId === activeThreadId))) {
                                        handleSendMessage();
                                    } else {
                                        e.stopPropagation();
                                    }
                                }}
                                placeholder={t('what_to_analyze')}
                                disabled={(isLoading && loadingThreadId === activeThreadId) || (isStreaming && streamingThreadId === activeThreadId)}
                                className="flex-1 bg-slate-50 dark:bg-[#3c3c3c] border border-slate-200 dark:border-[#4d4d4d] rounded-2xl px-6 py-1.5 text-sm font-normal text-slate-900 dark:text-[#d4d4d4] focus:ring-4 focus:ring-blue-100 dark:focus:ring-[#5b7a9d33] focus:bg-white dark:focus:bg-[#3c3c3c] focus:border-blue-300 dark:focus:border-[#5b8ab5] transition-all outline-none shadow-sm hover:border-slate-300 dark:hover:border-[#5a5a5a] disabled:bg-slate-100 dark:disabled:bg-[#2d2d30] disabled:text-slate-400 dark:disabled:text-[#808080] disabled:cursor-not-allowed"
                            />
                            <button
                                onClick={() => handleSendMessage()}
                                disabled={(isLoading && loadingThreadId === activeThreadId) || (isStreaming && streamingThreadId === activeThreadId) || !input.trim()}
                                className="aspect-square bg-blue-600 dark:bg-[#5b7a9d] text-white hover:bg-blue-700 dark:hover:bg-[#456a8a] rounded-2xl disabled:bg-slate-200 dark:disabled:bg-[#3c3c3c] disabled:text-slate-400 dark:disabled:text-[#808080] transition-all shadow-md active:scale-95 flex items-center justify-center"
                            >
                                <Send className="w-5 h-5" />
                            </button>
                        </div>
                        <div className="flex items-center justify-center gap-4 mt-4">
                            <p className="text-[10px] text-slate-400 dark:text-[#808080] font-medium flex items-center gap-1">
                                <span className="w-1 h-1 bg-slate-300 dark:bg-[#4d4d4d] rounded-full" />
                                {t('data_driven_reasoning')}
                            </p>
                            <p className="text-[10px] text-slate-400 dark:text-[#808080] font-medium flex items-center gap-1">
                                <span className="w-1 h-1 bg-slate-300 dark:bg-[#4d4d4d] rounded-full" />
                                {t('visualized_summaries')}
                            </p>
                        </div>
                    </div>
                    )}
                </div>
            </div>

            {/* Confirmation Modal */}
            {showClearConfirm && ReactDOM.createPortal(
                <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
                    <div className="bg-white dark:bg-[#252526] rounded-xl shadow-2xl p-6 w-[320px] transform transition-all animate-in zoom-in-95 duration-200">
                        <h3 className="text-lg font-bold text-slate-900 dark:text-[#d4d4d4] mb-2">{t('clear_history_confirm_title')}</h3>
                        <p className="text-sm text-slate-500 dark:text-[#808080] mb-6">
                            {t('clear_history_confirm_desc')}
                        </p>
                        <div className="flex justify-end gap-3">
                            <button
                                onClick={cancelClearHistory}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#3c3c3c] rounded-lg transition-colors"
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
                </div>,
                document.body
            )}

            {/* Clear Conversation Confirmation Modal */}
            {showClearConversationConfirm && ReactDOM.createPortal(
                <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
                    <div className="bg-white dark:bg-[#252526] rounded-xl shadow-2xl p-6 w-[320px] transform transition-all animate-in zoom-in-95 duration-200">
                        <h3 className="text-lg font-bold text-slate-900 dark:text-[#d4d4d4] mb-2">{t('clear_conversation_confirm_title')}</h3>
                        <p className="text-sm text-slate-500 dark:text-[#808080] mb-6">
                            {t('clear_conversation_confirm_desc')}
                        </p>
                        <div className="flex justify-end gap-3">
                            <button
                                onClick={cancelClearConversation}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#3c3c3c] rounded-lg transition-colors"
                            >
                                {t('cancel')}
                            </button>
                            <button
                                onClick={confirmClearConversation}
                                className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-lg shadow-sm transition-colors"
                            >
                                {t('confirm_clear_conversation')}
                            </button>
                        </div>
                    </div>
                </div>,
                document.body
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
                    autoIntentUnderstanding={autoIntentUnderstanding}
                    isReplaySession={contextMenu.isReplaySession}
                    isGeneratingComprehensiveReport={isGeneratingComprehensiveReport}
                />
            )}

            {exportPackThreadId && (
                <ExportPackDialog
                    isOpen={true}
                    onClose={() => setExportPackThreadId(null)}
                    onConfirm={() => {
                        setExportPackThreadId(null);
                        setToast({ message: t('export_pack_title') + ' ✓', type: 'success' });
                    }}
                    threadId={exportPackThreadId}
                />
            )}

            {blankAreaContextMenu && (
                <div
                    ref={blankMenuRef}
                    className="fixed bg-white dark:bg-[#252526] border border-slate-200 dark:border-[#3c3c3c] rounded-lg shadow-xl z-[9999] w-40 py-1 overflow-hidden"
                    style={{ top: blankAreaContextMenu.y, left: blankAreaContextMenu.x }}
                    onContextMenu={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                    }}
                >
                    <button
                        onClick={(e) => { e.stopPropagation(); handleStartFreeChat(); }}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                    >
                        <MessageCircle className="w-4 h-4 text-purple-500" />
                        {t('start_free_chat')}
                    </button>
                    <div className="h-px bg-slate-100 dark:bg-[#3c3c3c] my-1" />
                    <button
                        onClick={(e) => { e.stopPropagation(); handleImportAction(); }}
                        className="w-full text-left px-4 py-2 text-sm text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-50 dark:hover:bg-[#2d2d30] flex items-center gap-2"
                    >
                        <Upload className="w-4 h-4 text-slate-400 dark:text-[#808080]" />
                        Import
                    </button>
                </div>
            )}

            {/* Toast Notification */}
            {toast && (
                <Toast
                    message={toast.message}
                    type={toast.type}
                    onClose={() => setToast(null)}
                />
            )}

            <CancelConfirmationModal
                isOpen={showCancelConfirm}
                onClose={cancelCancelAnalysis}
                onConfirm={confirmCancelAnalysis}
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

            {/* Free mode comprehensive report dialog */}
            {showFreeModReportDialog && ReactDOM.createPortal(
                <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/40">
                    <div className="bg-white dark:bg-[#252526] rounded-xl shadow-2xl border border-slate-200 dark:border-[#3c3c3c] p-6 max-w-sm w-full mx-4">
                        <h3 className="text-base font-semibold text-slate-800 dark:text-[#d4d4d4] mb-3">
                            {t('comprehensive_report_free_mode_title')}
                        </h3>
                        <p className="text-sm text-slate-600 dark:text-[#a0a0a0] mb-5">
                            {t('comprehensive_report_free_mode_message')}
                        </p>
                        <div className="flex justify-end gap-2">
                            <button
                                onClick={() => setShowFreeModReportDialog(false)}
                                className="px-4 py-2 text-sm rounded-lg border border-slate-200 dark:border-[#3c3c3c] text-slate-600 dark:text-[#a0a0a0] hover:bg-slate-50 dark:hover:bg-[#2d2d30] transition-colors"
                            >
                                {t('comprehensive_report_free_mode_cancel')}
                            </button>
                            <button
                                onClick={() => {
                                    setShowFreeModReportDialog(false);
                                    BrowserOpenURL('https://vantagics.com/#deployment');
                                }}
                                className="px-4 py-2 text-sm rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 transition-colors"
                            >
                                {t('comprehensive_report_free_mode_confirm')}
                            </button>
                        </div>
                    </div>
                </div>,
                document.body
            )}

        </>
    );
};

export default ChatSidebar;