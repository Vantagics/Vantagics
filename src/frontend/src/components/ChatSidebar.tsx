import React, { useState, useEffect, useRef } from 'react';
import { X, MessageSquare, Plus, Trash2, Send, Loader2, ChevronLeft, ChevronRight, Settings, Upload, Zap, XCircle } from 'lucide-react';
import { GetChatHistory, SaveChatHistory, SendMessage, DeleteThread, ClearHistory, GetDataSources, CreateChatThread, UpdateThreadTitle, ExportSessionHTML, OpenSessionResultsDirectory, CancelAnalysis, GetConfig, GenerateIntentSuggestions, GenerateIntentSuggestionsWithExclusions } from '../../wailsjs/go/main/App';
import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import MessageBubble from './MessageBubble';
import { useLanguage } from '../i18n';
import DeleteConfirmationModal from './DeleteConfirmationModal';
import ChatThreadContextMenu from './ChatThreadContextMenu';
import MemoryViewModal from './MemoryViewModal';
import CancelConfirmationModal from './CancelConfirmationModal';
import Toast, { ToastType } from './Toast';
import { createLogger } from '../utils/systemLog';

const systemLog = createLogger('ChatSidebar');

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
    const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
    const [showClearConfirm, setShowClearConfirm] = useState(false);
    const [dataSources, setDataSources] = useState<any[]>([]);
    const [deleteThreadTarget, setDeleteThreadTarget] = useState<{ id: string, title: string } | null>(null);
    const [memoryModalTarget, setMemoryModalTarget] = useState<string | null>(null);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, threadId: string } | null>(null);
    const [blankAreaContextMenu, setBlankAreaContextMenu] = useState<{ x: number, y: number } | null>(null);
    const [progress, setProgress] = useState<ProgressUpdate | null>(null);
    const [showCancelConfirm, setShowCancelConfirm] = useState(false);
    const [toast, setToast] = useState<{ message: string; type: ToastType } | null>(null);
    const [showSuggestionButton, setShowSuggestionButton] = useState<string | null>(null); // 跟踪哪个会话需要显示建议按钮
    
    // Intent Understanding State
    const [intentSuggestions, setIntentSuggestions] = useState<IntentSuggestion[]>([]);
    const [excludedIntentSuggestions, setExcludedIntentSuggestions] = useState<IntentSuggestion[]>([]); // 累积所有被拒绝的意图建议
    const [isGeneratingIntent, setIsGeneratingIntent] = useState(false);
    const [pendingMessage, setPendingMessage] = useState<string>('');
    const [pendingThreadId, setPendingThreadId] = useState<string>('');
    const [intentMessageId, setIntentMessageId] = useState<string>(''); // 意图消息的ID

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
                            setShowSuggestionButton(thread.id);
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
            } catch (err: any) {
                // Handle error when creating thread (e.g., active analysis in progress)
                systemLog.error(`Failed to create thread: ${err}`);
                
                const errorMsg = err?.message || String(err);
                
                // Show user-friendly error message
                EventsEmit('show-message-modal', {
                    type: 'warning',
                    title: t('create_session_failed') || '创建会话失败',
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
        const unsubscribeLoadMessageData = EventsOn('load-message-data', (data: any) => {
            console.log('[ChatSidebar] load-message-data event received:', data);
            
            const { messageId, threadId } = data;
            if (!messageId) {
                console.log('[ChatSidebar] No messageId provided, ignoring');
                return;
            }
            
            // Find the message in the threads
            const targetThread = threads?.find(t => t.id === threadId || t.id === activeThreadId);
            if (!targetThread) {
                console.log('[ChatSidebar] Target thread not found');
                return;
            }
            
            const message = targetThread.messages?.find(m => m.id === messageId);
            if (!message) {
                console.log('[ChatSidebar] Message not found:', messageId);
                return;
            }
            
            console.log('[ChatSidebar] Loading message data:', messageId);
            console.log('[ChatSidebar] Has chart_data:', !!message.chart_data);
            
            // Find chartData from user message or next assistant message
            let chartDataToUse = message.chart_data;
            const messageIndex = targetThread.messages.findIndex(m => m.id === messageId);
            if (messageIndex !== -1 && messageIndex < targetThread.messages.length - 1) {
                const nextMessage = targetThread.messages[messageIndex + 1];
                if (nextMessage.role === 'assistant' && nextMessage.chart_data) {
                    console.log('[ChatSidebar] Using chart_data from assistant response');
                    chartDataToUse = nextMessage.chart_data;
                }
            }
            
            // Emit user-message-clicked with the loaded data
            EventsEmit('user-message-clicked', {
                messageId: message.id,
                content: message.content,
                chartData: chartDataToUse
            });
        });

        // Listen for analysis cancellation event
        const unsubscribeCancelled = EventsOn('analysis-cancelled', (data: any) => {
            console.log('[ChatSidebar] Analysis cancelled event received:', data);
            
            // Clear loading state
            setIsLoading(false);
            setLoadingThreadId(null);
            setProgress(null);
            setShowCancelConfirm(false);
            
            console.log('[ChatSidebar] Loading state cleared after cancellation');
        });

        return () => {
            if (unsubscribeOpen) unsubscribeOpen();
            if (unsubscribeUpdate) unsubscribeUpdate();
            if (unsubscribeLoading) unsubscribeLoading();
            if (unsubscribeProgress) unsubscribeProgress();
            if (unsubscribeChatMessage) unsubscribeChatMessage();
            if (unsubscribeSendMessageInSession) unsubscribeSendMessageInSession();
            if (unsubscribeLoadMessageData) unsubscribeLoadMessageData();
            if (unsubscribeCancelled) unsubscribeCancelled();
        };
    }, [threads]);

    // 监听活动会话变化，自动显示第一个分析结果
    useEffect(() => {
        if (activeThreadId && threads) {
            const activeThread = threads.find(t => t.id === activeThreadId);
            if (activeThread && activeThread.messages) {
                // 找到第一个有分析结果的用户消息
                // 判断标准：用户消息后有助手回复，或者有 chart_data
                let firstAnalysisMessage: main.ChatMessage | null = null;

                for (let i = 0; i < activeThread.messages.length; i++) {
                    const msg = activeThread.messages[i];

                    // 必须是用户消息
                    if (msg.role !== 'user') continue;

                    // 检查是否有 chart_data
                    if (msg.chart_data) {
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

                    // 自动触发显示该消息的分析结果
                    setTimeout(() => {
                        EventsEmit('user-message-clicked', {
                            messageId: firstAnalysisMessage!.id,
                            content: firstAnalysisMessage!.content,
                            chartData: firstAnalysisMessage!.chart_data
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
                setProgress(null);
            }

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
                                // 触发仪表盘更新
                                EventsEmit('user-message-clicked', {
                                    threadId: newActiveThread.id,
                                    messageId: msg.id,
                                    charts: msg.chart_data ? [msg.chart_data] : []
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
                title: '删除失败',
                message: `无法删除会话：${err}`
            });
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

    const handleContextAction = async (action: 'export' | 'view_memory' | 'view_results_directory', threadId: string) => {
        console.log(`Action ${action} on thread ${threadId}`);
        if (action === 'view_memory') {
            setMemoryModalTarget(threadId);
        } else if (action === 'export') {
            try {
                await ExportSessionHTML(threadId);
            } catch (e) {
                console.error("Export failed:", e);
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

    // 处理"生成建议分析"按钮点击
    const handleGenerateSuggestions = async () => {
        if (!activeThreadId || !activeThread) return;
        
        // 清除按钮显示状态
        setShowSuggestionButton(null);
        
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

    // Format intent suggestions as markdown with clickable buttons
    const formatIntentSuggestions = (suggestions: IntentSuggestion[]): string => {
        const header = t('select_your_intent') || '请选择您的分析意图';
        const desc = t('intent_selection_desc') || '系统理解了您的请求，请选择最符合您意图的分析方向';
        
        let content = `**${header}**\n\n${desc}\n\n`;
        
        suggestions.forEach((suggestion, index) => {
            content += `${index + 1}. ${suggestion.icon} **${suggestion.title}**\n   ${suggestion.description}\n\n`;
        });
        
        // 添加"重新理解"选项
        const retryIndex = suggestions.length + 1;
        const retryText = t('retry_intent_understanding') || '以上都不是我所想的，重新理解意图';
        content += `${retryIndex}. 🔄 **${retryText}**\n\n`;
        
        content += `\n*${t('click_suggestion_to_continue') || '点击上方建议继续分析'}*`;
        
        return content;
    };

    const handleSendMessage = async (text?: string, explicitThreadId?: string, explicitThread?: main.ChatThread, requestId?: string, skipIntentUnderstanding?: boolean) => {
        const msgText = text || input;

        // 使用 refs 获取最新的状态值（避免闭包问题）
        const currentIsLoading = isLoadingRef.current;
        const currentLoadingThreadId = loadingThreadIdRef.current;

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

        // If explicitThread is passed (auto-start), ignore isLoading check to ensure it fires.
        if (!msgText.trim() || (currentIsLoading && !explicitThread)) {
            console.log('[ChatSidebar] ❌ handleSendMessage early return:', {
                emptyText: !msgText.trim(),
                isLoadingAndNoExplicitThread: currentIsLoading && !explicitThread
            });
            return;
        }

        // 检查当前会话是否有分析正在进行（包括 explicitThread 的情况）
        // 确定目标会话ID
        const targetThreadId = explicitThread?.id || explicitThreadId || activeThreadId;

        console.log('[ChatSidebar] 🔍 Loading state check:', {
            currentIsLoading,
            currentLoadingThreadId,
            targetThreadId,
            matches: currentLoadingThreadId === targetThreadId,
            willBlock: currentIsLoading && currentLoadingThreadId === targetThreadId
        });

        if (currentIsLoading && currentLoadingThreadId === targetThreadId) {
            console.log('[ChatSidebar] ⚠️ Analysis in progress for target thread, blocking new request');
            // 显示Toast提示
            setToast({
                message: t('analysis_in_progress') || '分析进行中，请等待当前分析完成后再发起新的分析',
                type: 'warning'
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

        // Track if user message was already added (to prevent duplication)
        let userMessageAlreadyAdded = false;

        // Check if we need intent understanding
        // Only for user-initiated messages (not system-generated or action clicks)
        const isUserInitiated = !explicitThread && !requestId && !skipIntentUnderstanding && !text; // text参数表示是通过代码调用的（如点击建议）
        
        if (isUserInitiated) {
            try {
                const cfg = await GetConfig();
                
                if (cfg.autoIntentUnderstanding !== false) {
                    // Add user message first
                    setInput('');
                    
                    // Generate intent suggestions
                    setIsGeneratingIntent(true);
                    setPendingMessage(msgText);
                    setPendingThreadId(targetThreadId || '');
                    // 清理之前累积的排除项（新的意图理解流程开始）
                    setExcludedIntentSuggestions([]);
                    
                    // Create a temporary message ID for the intent message
                    const tempIntentMsgId = `intent_${Date.now()}`;
                    setIntentMessageId(tempIntentMsgId);
                    
                    // Add "generating intent" message to thread
                    let intentThread = threads.find(t => t.id === targetThreadId);
                    let actualThreadId = targetThreadId;
                    if (!intentThread) {
                        // Create new thread if needed
                        const title = msgText.slice(0, 30);
                        intentThread = await CreateChatThread('', title);
                        setThreads([intentThread, ...threads]);
                        setActiveThreadId(intentThread.id);
                        actualThreadId = intentThread.id;
                        setPendingThreadId(actualThreadId);
                    }
                    
                    // Add user message
                    const userMsg: main.ChatMessage = {
                        id: `user_${Date.now()}`,
                        role: 'user',
                        content: msgText,
                        timestamp: Date.now()
                    };
                    
                    // Add "understanding intent" system message
                    const intentMsg: main.ChatMessage = {
                        id: tempIntentMsgId,
                        role: 'assistant',
                        content: t('generating_intent') || '正在理解您的意图...',
                        timestamp: Date.now()
                    };
                    
                    intentThread.messages = [...(intentThread.messages || []), userMsg, intentMsg];
                    const updatedThreads = threads.map(t => t.id === intentThread!.id ? intentThread! : t);
                    setThreads(updatedThreads);
                    await SaveChatHistory(updatedThreads);
                    
                    // Mark that user message was added
                    userMessageAlreadyAdded = true;
                    
                    try {
                        const suggestions = await GenerateIntentSuggestions(actualThreadId || '', msgText);
                        
                        if (suggestions && suggestions.length > 0) {
                            // Replace "understanding" message with intent suggestions
                            const intentContent = formatIntentSuggestions(suggestions);
                            const updatedIntentMsg: main.ChatMessage = {
                                id: tempIntentMsgId,
                                role: 'assistant',
                                content: intentContent,
                                timestamp: Date.now()
                            };
                            
                            intentThread.messages = intentThread.messages.map(m => 
                                m.id === tempIntentMsgId ? updatedIntentMsg : m
                            );
                            const finalThreads = threads.map(t => t.id === intentThread!.id ? intentThread! : t);
                            setThreads(finalThreads);
                            await SaveChatHistory(finalThreads);
                            
                            setIntentSuggestions(suggestions);
                            setPendingThreadId(actualThreadId || '');
                            clearTimeout(timeoutId);
                            if (pendingActionRef.current === actionKey) {
                                pendingActionRef.current = null;
                            }
                            return; // Wait for user to click a suggestion
                        }
                    } catch (error) {
                        console.error('[Intent] Failed to generate suggestions:', error);
                        // Remove the intent message and continue with normal flow
                        intentThread.messages = intentThread.messages.filter(m => m.id !== tempIntentMsgId);
                        const cleanThreads = threads.map(t => t.id === intentThread!.id ? intentThread! : t);
                        setThreads(cleanThreads);
                        await SaveChatHistory(cleanThreads);
                        // User message is already added, so we should NOT add it again below
                    } finally {
                        setIsGeneratingIntent(false);
                        setIntentMessageId('');
                    }
                }
            } catch (error) {
                console.error('[Intent] Error checking config:', error);
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

        // Only add user message if it wasn't already added during intent understanding
        let userMsg: main.ChatMessage;
        if (!userMessageAlreadyAdded) {
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

            setInput('');
            
            // Await save before sending message to prevent race condition
            // Now passing the explicitly calculated updatedThreads
            await SaveChatHistory(updatedThreads);
        } else {
            console.log('[ChatSidebar] User message already added during intent understanding, skipping duplicate');
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

        setIsLoading(true);
        setLoadingThreadId(currentThreadId); // 记录正在加载的会话ID

        try {

            // Task 3.1: Pass requestId to backend for request tracking (Requirements 1.3, 4.3, 4.4)
            const response = await SendMessage(currentThreadId, msgText, userMsg.id, requestId || '');

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
                    console.log("[ChatSidebar] Backend already added assistant message with chart_data:", !!lastMessage.chart_data);
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
            console.log('[ChatSidebar] 🧹 Clearing loading state:', {
                wasLoading: isLoading,
                wasLoadingThreadId: loadingThreadId,
                currentThreadId
            });
            setIsLoading(false);
            setLoadingThreadId(null); // 清除加载会话ID
            setProgress(null);
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

                        // Find chart data from user message or assistant response
                        let chartDataToUse = userMessage.chart_data;

                        // Check if there's an assistant response with chart_data
                        const messageIndex = updatedThread.messages.findIndex(msg => msg.id === userMessage.id);
                        if (messageIndex !== -1 && messageIndex < updatedThread.messages.length - 1) {
                            const nextMessage = updatedThread.messages[messageIndex + 1];
                            if (nextMessage.role === 'assistant' && nextMessage.chart_data) {
                                console.log('[ChatSidebar] Using chart_data from assistant response for auto-update');
                                chartDataToUse = nextMessage.chart_data;
                            }
                        }

                        // Emit event to update dashboard (same as clicking the message)
                        EventsEmit('user-message-clicked', {
                            messageId: userMessage.id,
                            content: userMessage.content,
                            chartData: chartDataToUse
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
    const handleIntentSelect = async (suggestionIndex: number) => {
        console.log('[DEBUG] handleIntentSelect called:', { suggestionIndex, intentSuggestionsLength: intentSuggestions.length });
        
        // 检查是否是"重新理解"选项（最后一项）
        if (suggestionIndex === intentSuggestions.length) {
            console.log('[DEBUG] Retry intent understanding triggered');
            // 重新理解意图
            await handleRetryIntentUnderstanding();
            return;
        }
        
        if (suggestionIndex < 0 || suggestionIndex >= intentSuggestions.length) {
            console.log('[DEBUG] Invalid suggestion index');
            return;
        }
        
        console.log('[DEBUG] Normal intent selection, continuing with analysis');
        const suggestion = intentSuggestions[suggestionIndex];
        const selectedQuery = suggestion.query;
        
        // Remove the intent message from thread
        if (intentMessageId && pendingThreadId) {
            const thread = threads.find(t => t.id === pendingThreadId);
            if (thread) {
                thread.messages = thread.messages.filter(m => m.id !== intentMessageId);
                const updatedThreads = threads.map(t => t.id === thread.id ? thread : t);
                setThreads(updatedThreads);
                await SaveChatHistory(updatedThreads);
            }
        }
        
        // Continue with normal message sending flow using the refined query
        // Skip intent understanding since this is already a refined query
        await handleSendMessage(selectedQuery, pendingThreadId, undefined, undefined, true);
        
        // Clear all pending state including accumulated exclusions
        setPendingMessage('');
        setPendingThreadId('');
        setIntentSuggestions([]);
        setExcludedIntentSuggestions([]); // 清理累积的排除项
        setIntentMessageId('');
    };
    
    // Handle retry intent understanding
    const handleRetryIntentUnderstanding = async () => {
        if (!pendingMessage || !pendingThreadId) {
            return;
        }
        
        // Update the intent message to show "regenerating"
        if (intentMessageId) {
            const thread = threads.find(t => t.id === pendingThreadId);
            if (thread) {
                const regeneratingMsg: main.ChatMessage = {
                    id: intentMessageId,
                    role: 'assistant',
                    content: t('generating_intent') || '正在理解您的意图...',
                    timestamp: Date.now()
                };
                
                thread.messages = thread.messages.map(m => 
                    m.id === intentMessageId ? regeneratingMsg : m
                );
                const updatedThreads = threads.map(t => t.id === thread.id ? thread : t);
                setThreads(updatedThreads);
                await SaveChatHistory(updatedThreads);
            }
        }
        
        // Regenerate intent suggestions with exclusions
        setIsGeneratingIntent(true);
        try {
            // 累积所有被拒绝的意图建议（当前的 + 之前累积的）
            const allExcludedSuggestions = [...excludedIntentSuggestions, ...intentSuggestions];
            console.log('[Intent] Retry with exclusions:', { 
                currentSuggestions: intentSuggestions.length, 
                previouslyExcluded: excludedIntentSuggestions.length,
                totalExcluded: allExcludedSuggestions.length 
            });
            
            // Pass all excluded suggestions
            const suggestions = await GenerateIntentSuggestionsWithExclusions(
                pendingThreadId, 
                pendingMessage,
                allExcludedSuggestions
            );
            
            if (suggestions && suggestions.length > 0) {
                // Replace with new intent suggestions
                const intentContent = formatIntentSuggestions(suggestions);
                const updatedIntentMsg: main.ChatMessage = {
                    id: intentMessageId,
                    role: 'assistant',
                    content: intentContent,
                    timestamp: Date.now()
                };
                
                const thread = threads.find(t => t.id === pendingThreadId);
                if (thread) {
                    thread.messages = thread.messages.map(m => 
                        m.id === intentMessageId ? updatedIntentMsg : m
                    );
                    const finalThreads = threads.map(t => t.id === thread.id ? thread : t);
                    setThreads(finalThreads);
                    await SaveChatHistory(finalThreads);
                    
                    // 更新累积的排除项（将当前建议添加到排除列表）
                    setExcludedIntentSuggestions(allExcludedSuggestions);
                    // 设置新的建议
                    setIntentSuggestions(suggestions);
                }
            }
        } catch (error) {
            console.error('[Intent] Failed to regenerate suggestions:', error);
            // Remove the intent message and continue with normal flow
            const thread = threads.find(t => t.id === pendingThreadId);
            if (thread && intentMessageId) {
                thread.messages = thread.messages.filter(m => m.id !== intentMessageId);
                const cleanThreads = threads.map(t => t.id === thread.id ? thread : t);
                setThreads(cleanThreads);
                await SaveChatHistory(cleanThreads);
            }
            // Fallback to direct execution
            await handleSendMessage(pendingMessage, pendingThreadId, undefined, undefined, true);
            // 清理所有意图相关状态
            setPendingMessage('');
            setPendingThreadId('');
            setIntentSuggestions([]);
            setExcludedIntentSuggestions([]);
            setIntentMessageId('');
        } finally {
            setIsGeneratingIntent(false);
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

                // 或者检查是否有chart_data
                if (message.chart_data) {
                    isCompleted = true;
                }
            }
        }

        // 如果消息未完成，不允许点击
        if (!isCompleted) {
            console.log("[ChatSidebar] Message not completed, ignoring click:", message.id);
            return;
        }

        // Debug logging
        console.log("[ChatSidebar] User message clicked:", message.id);
        console.log("[ChatSidebar] Message content:", message.content?.substring(0, 100));
        console.log("[ChatSidebar] Has chart_data:", !!message.chart_data);
        console.log("[ChatSidebar] chart_data object:", message.chart_data);
        if (message.chart_data) {
            console.log("[ChatSidebar] chart_data.charts:", message.chart_data.charts);
            console.log("[ChatSidebar] Number of charts:", message.chart_data.charts?.length || 0);
        }

        // Find the corresponding assistant message (next message after this user message)
        let chartDataToUse = message.chart_data;

        if (activeThread) {
            const messageIndex = activeThread.messages.findIndex(msg => msg.id === message.id);
            if (messageIndex !== -1 && messageIndex < activeThread.messages.length - 1) {
                const nextMessage = activeThread.messages[messageIndex + 1];
                // If next message is assistant and has chart_data, use it (it's more complete)
                if (nextMessage.role === 'assistant' && nextMessage.chart_data) {
                    console.log("[ChatSidebar] Using chart_data from assistant response");
                    chartDataToUse = nextMessage.chart_data;
                }
            }
        }

        // Emit event with message data for dashboard update
        EventsEmit('user-message-clicked', {
            messageId: message.id,
            content: message.content,
            chartData: chartDataToUse
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
                className={`fixed inset-y-0 left-0 bg-white shadow-2xl transform transition-transform duration-300 ease-in-out z-50 flex overflow-hidden border-r border-slate-200 ${isOpen ? 'translate-x-0' : '-translate-x-full'}`}
            >
                {/* Sidebar Resizer (Right Edge) */}
                <div
                    className="absolute right-0 top-0 bottom-0 w-1 hover:bg-blue-400 cursor-col-resize z-[60] transition-colors"
                    onMouseDown={() => { setIsResizingSidebar(true); document.body.style.cursor = 'col-resize'; }}
                />

                {/* Thread List Sidebar */}
                <div
                    style={{ width: isSidebarCollapsed ? 0 : historyWidth }}
                    className="bg-slate-50 border-r border-slate-200 flex flex-col transition-all duration-300 overflow-hidden relative flex-shrink-0"
                >
                    {/* Collapse button on the left edge of history panel */}
                    <button
                        onClick={onClose}
                        className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-1/2 z-50 bg-white border border-slate-200 rounded-full p-1.5 shadow-lg hover:bg-slate-50 text-slate-400 hover:text-blue-500 transition-all hover:scale-110"
                        title={t('collapse_chat')}
                    >
                        <ChevronLeft className="w-4 h-4" />
                    </button>

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
                                <p className="text-[10px] text-slate-400 font-medium">{t('no_threads_yet')}</p>
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

                    {/* History Resizer (Left Edge of History Panel) */}
                    <div
                        className="absolute left-0 top-0 bottom-0 w-1 hover:bg-blue-400 cursor-col-resize z-20 transition-colors"
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
                                    // 或者检查是否有chart_data
                                    if (msg.chart_data) {
                                        return true;
                                    }
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
                                    onActionClick={(action) => {
                                        console.log('[DEBUG] Action clicked:', action);
                                        
                                        // 检查是否是意图建议点击或重新理解按钮
                                        const isIntentMessage = msg.id === intentMessageId && intentSuggestions.length > 0;
                                        const isRetryButton = action.label && (action.label.includes('重新理解') || action.label.includes('retry') || action.label.includes('Retry'));
                                        
                                        console.log('[DEBUG] Button check:', { isIntentMessage, isRetryButton, intentMessageId, msgId: msg.id, intentSuggestionsLength: intentSuggestions.length });
                                        
                                        if (isIntentMessage || isRetryButton) {
                                            // 从 label 提取索引
                                            const match = action.label.match(/^(\d+)\./);
                                            console.log('[DEBUG] Intent action match:', { label: action.label, match, intentMessageId, msgId: msg.id });
                                            if (match) {
                                                const index = parseInt(match[1]) - 1;
                                                console.log('[DEBUG] Calling handleIntentSelect with index:', index);
                                                handleIntentSelect(index);
                                                return;
                                            }
                                        }
                                        // Normal action click - skip intent understanding
                                        console.log('[DEBUG] Normal action click, sending message');
                                        handleSendMessage(action.value || action.label, activeThread?.id, undefined, undefined, true);
                                    }}
                                    onClick={msg.role === 'user' && isUserMessageCompleted ? () => handleUserMessageClick(msg) : undefined}
                                    hasChart={msg.role === 'user' && !!msg.chart_data}
                                    isDisabled={msg.role === 'user' && !isUserMessageCompleted}
                                    timingData={msg.role === 'user' ? timingDataForUser : (msg as any).timing_data}
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
                                                {progress?.message
                                                    ? (progress.message === 'progress.ai_processing'
                                                        ? t('progress.ai_processing', progress.step || 0)
                                                        : progress.message === 'progress.tool_completed'
                                                            ? t('progress.tool_completed', progress.tool_name || '')
                                                            : t(progress.message))
                                                    : t('ai_thinking')}
                                            </span>
                                        </div>
                                        <button
                                            onClick={handleCancelAnalysis}
                                            className="flex items-center gap-1 px-2 py-1 text-xs text-red-600 hover:bg-red-50 rounded transition-colors"
                                            title={t('cancel_analysis')}
                                        >
                                            <XCircle className="w-3.5 h-3.5" />
                                            <span>{t('cancel_analysis')}</span>
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
                        {/* 显示"生成建议分析"按钮 - 当自动分析关闭且会话为空时 */}
                        {showSuggestionButton === activeThreadId && activeThread && (!activeThread.messages || activeThread.messages.length === 0) && !isLoading && (
                            <div className="flex flex-col items-center justify-center py-12 animate-in fade-in zoom-in-95 duration-300">
                                <div className="bg-gradient-to-br from-blue-50 to-indigo-50 p-5 rounded-[2rem] mb-5 shadow-inner ring-1 ring-white">
                                    <Zap className="w-8 h-8 text-blue-500" />
                                </div>
                                <p className="text-sm text-slate-500 mb-4 text-center max-w-[280px]">
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

        </>
    );
};

export default ChatSidebar;