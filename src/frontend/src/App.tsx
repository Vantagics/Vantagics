import React, { useState, useEffect } from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import Sidebar from './components/Sidebar';
import DashboardTest from './components/DashboardTest';
import DraggableDashboard from './components/DraggableDashboard';
import ContextPanel from './components/ContextPanel';
import PreferenceModal from './components/PreferenceModal';
import AboutModal from './components/AboutModal';
import ChatSidebar from './components/ChatSidebar';
import ContextMenu from './components/ContextMenu';
import MessageModal from './components/MessageModal';
import SkillsManagementPage from './components/SkillsManagementPage';
import { EventsOn, EventsEmit } from '../wailsjs/runtime/runtime';
import { GetDashboardData, GetConfig, TestLLMConnection, SetChatOpen, CanStartNewAnalysis } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import { createLogger } from './utils/systemLog';
import { useLanguage } from './i18n';
import { ToastProvider, useToast } from './contexts/ToastContext';
import './App.css';

const logger = createLogger('App');

// Inner component that has access to ToastContext
function AppContent() {
    const { t } = useLanguage();
    const { showToast } = useToast();
    const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);
    const [isAboutOpen, setIsAboutOpen] = useState(false);
    const [isSkillsOpen, setIsSkillsOpen] = useState(false);
    const [isChatOpen, setIsChatOpen] = useState(false);
    const [dashboardData, setDashboardData] = useState<main.DashboardData | null>(null);
    const [activeChart, setActiveChart] = useState<{ type: 'echarts' | 'image' | 'table' | 'csv', data: any, chartData?: main.ChartData } | null>(null);
    const [sessionCharts, setSessionCharts] = useState<{ [sessionId: string]: { type: 'echarts' | 'image' | 'table' | 'csv', data: any, chartData?: main.ChartData } }>({});
    const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
    const [selectedUserRequest, setSelectedUserRequest] = useState<string | null>(null);
    const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null);  // 存储当前选中的消息ID
    const [sessionInsights, setSessionInsights] = useState<{ [messageId: string]: any[] }>({});  // 存储每个用户消息对应的LLM建议
    const [sessionMetrics, setSessionMetrics] = useState<{ [messageId: string]: any[] }>({});  // 存储每个用户消息对应的关键指标
    const [originalSystemInsights, setOriginalSystemInsights] = useState<any[]>([]);  // 存储系统初始化的洞察
    const [originalSystemMetrics, setOriginalSystemMetrics] = useState<any[]>([]);  // 存储系统初始化的指标
    const [sessionFiles, setSessionFiles] = useState<main.SessionFile[]>([]);  // 存储当前会话的文件列表
    const [messageModal, setMessageModal] = useState<{ isOpen: boolean, type: 'info' | 'warning' | 'error', title: string, message: string }>({
        isOpen: false,
        type: 'info',
        title: '',
        message: ''
    });

    // Analysis loading state
    const [isAnalysisLoading, setIsAnalysisLoading] = useState(false);
    const [loadingThreadId, setLoadingThreadId] = useState<string | null>(null);

    // Request tracking state (Requirements 2.1, 4.1)
    const [pendingRequestId, setPendingRequestId] = useState<string | null>(null);
    const [lastCompletedRequestId, setLastCompletedRequestId] = useState<string | null>(null);
    
    // Task 6.1: Timeout tracking for request timeout handling (Requirement 2.4)
    const [requestTimeouts, setRequestTimeouts] = useState<Map<string, number>>(new Map());

    // Generate unique request ID for tracking analysis requests (Requirements 2.1, 4.1)
    const generateRequestId = (): string => {
        return `req_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
    };

    // Handle insight click - Requirements 1.1, 1.2, 2.1, 2.2, 4.2, 5.3, 5.4
    // insight can be either a string (backward compatibility) or an object with data_source_id
    const handleInsightClick = async (insight: any) => {
        // Extract insight text - handle both string and object formats
        const insightText = typeof insight === 'string' ? insight : (insight.text || '');
        const dataSourceId = typeof insight === 'object' ? insight.data_source_id : null;
        const sourceName = typeof insight === 'object' ? insight.source_name : null;
        
        logger.debug(`Insight clicked: ${insightText.substring(0, 50)}`);
        logger.debug(`dataSourceId: ${dataSourceId}, sourceName: ${sourceName}, activeSessionId: ${activeSessionId}`);
        
        // 判断逻辑：
        // 1. 如果有 data_source_id 且没有 activeSessionId -> 系统初始状态，创建新会话
        // 2. 如果有 activeSessionId -> 会话分析结果状态，在当前会话继续
        
        if (dataSourceId && !activeSessionId) {
            // 场景1：系统初始状态，点击数据源洞察 -> 创建新会话
            logger.info(`System initial state: Creating new session for data source: ${dataSourceId}`);
            
            // Check if we can start a new analysis (concurrent limit check)
            try {
                const [canStart, errorMessage] = await CanStartNewAnalysis();
                if (!canStart) {
                    logger.warn(`Cannot start new analysis: ${errorMessage}`);
                    setMessageModal({
                        isOpen: true,
                        type: 'warning',
                        title: t('warning') || '警告',
                        message: errorMessage
                    });
                    return;
                }
            } catch (error) {
                logger.error(`Error checking concurrent limit: ${error}`);
                // Continue anyway if check fails
            }
            
            const sessionName = `分析会话: ${sourceName || insightText}`;
            logger.info(`Session name will be: ${sessionName}`);
            
            const eventData = {
                dataSourceId: dataSourceId,
                sessionName: sessionName,
                keepChatOpen: true,
                initialMessage: insightText
            };
            
            logger.info(`Emitting start-new-chat event with data: ${JSON.stringify(eventData)}`);
            
            // 打开聊天区域
            setIsChatOpen(true);
            // 折叠数据浏览器
            setIsContextPanelCollapsed(true);
            
            // 发送创建新会话事件
            EventsEmit('start-new-chat', eventData);
            return;
        }
        
        // 场景2：会话分析结果状态，在当前会话继续分析
        // Task 4.1: Check if there's a pending request and cancel it (Requirements 5.3, 5.4)
        if (pendingRequestId) {
            logger.info(`Canceling previous request ${pendingRequestId} - new insight clicked`);
            // Clear the timeout for the previous request
            const prevTimeout = requestTimeouts.get(pendingRequestId);
            if (prevTimeout) {
                clearTimeout(prevTimeout);
                setRequestTimeouts(prev => {
                    const newMap = new Map(prev);
                    newMap.delete(pendingRequestId);
                    return newMap;
                });
            }
            // The previous request will be ignored when its result arrives
            // because the requestId won't match the new pendingRequestId
        }
        
        // Generate unique request ID for tracking
        const requestId = generateRequestId();
        logger.debug(`Generated requestId: ${requestId}`);
        
        // Set loading state and pending request ID
        // CRITICAL: Do NOT modify dashboardData - keep it stable during loading
        // Task 4.1: Update pendingRequestId to the new request, effectively canceling the old one
        setPendingRequestId(requestId);
        setIsAnalysisLoading(true);
        
        // Task 6.1: Set 30-second timeout for the request (Requirement 2.4)
        const timeoutId = setTimeout(() => {
            // Check if this request is still pending
            setPendingRequestId(currentPendingId => {
                if (currentPendingId === requestId) {
                    logger.warn(`Request ${requestId} timed out after 30 seconds`);
                    
                    // Clear loading state but keep dashboard data (Requirement 2.4)
                    setIsAnalysisLoading(false);
                    
                    // Show timeout error message
                    showToast('error', '分析请求超时（30秒），请重试', '分析超时');
                    
                    // Clear the timeout from the map
                    setRequestTimeouts(prev => {
                        const newMap = new Map(prev);
                        newMap.delete(requestId);
                        return newMap;
                    });
                    
                    return null; // Clear pending request ID
                }
                return currentPendingId; // Keep current pending ID if it's different
            });
        }, 30000); // 30 seconds timeout
        
        // Store the timeout ID for potential cancellation
        setRequestTimeouts(prev => {
            const newMap = new Map(prev);
            newMap.set(requestId, timeoutId);
            return newMap;
        });
        
        // If there's an active session, send the analysis request with requestId
        if (activeSessionId) {
            logger.debug(`Sending analysis request in session ${activeSessionId} with requestId ${requestId}`);
            
            // Ensure chat sidebar is open so user can see the message
            if (!isChatOpen) {
                setIsChatOpen(true);
            }
            
            EventsEmit('chat-send-message-in-session', {
                text: insightText,
                threadId: activeSessionId,
                requestId: requestId
            });
        } else {
            // No active session - open chat and send message
            logger.debug('No active session, opening chat and sending message');
            setIsChatOpen(true);
            
            // Delay to ensure chat sidebar is mounted
            setTimeout(() => {
                EventsEmit('chat-send-message', insightText);
            }, 150);
        }
    };

    useEffect(() => {
        SetChatOpen(isChatOpen);
    }, [isChatOpen]);

    // Startup State
    const [isAppReady, setIsAppReady] = useState(false);
    const [startupStatus, setStartupStatus] = useState<"checking" | "failed">("checking");
    const [startupMessage, setStartupMessage] = useState(t('initializing'));

    // Layout State
    const [sidebarWidth, setSidebarWidth] = useState(256);
    const [contextPanelWidth, setContextPanelWidth] = useState(384);
    const [isResizingSidebar, setIsResizingSidebar] = useState(false);
    const [isResizingContextPanel, setIsResizingContextPanel] = useState(false);
    const [isContextPanelCollapsed, setIsContextPanelCollapsed] = useState(false); // 数据浏览器折叠状态

    // Context Menu State
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; target: HTMLElement } | null>(null);

    const checkLLM = async () => {
        setStartupStatus("checking");
        
        // Check LLM configuration
        setStartupMessage(t('checking_llm_config'));
        try {
            const config = await GetConfig();

            // Basic validation
            if (!config.apiKey && config.llmProvider !== 'OpenAI-Compatible' && config.llmProvider !== 'Claude-Compatible') {
                throw new Error(t('api_key_missing'));
            }

            setStartupMessage(t('testing_llm_connection'));
            const result = await TestLLMConnection(config);

            if (result.success) {
                setIsAppReady(true);
                // Fetch dashboard data only after ready
                GetDashboardData().then(data => {
                    setDashboardData(data);
                    // 保存系统初始化的洞察和指标，用于后续恢复
                    if (data && data.insights) {
                        setOriginalSystemInsights(Array.isArray(data.insights) ? data.insights : []);
                    }
                    if (data && data.metrics) {
                        setOriginalSystemMetrics(Array.isArray(data.metrics) ? data.metrics : []);
                    }
                }).catch(console.error);
            } else {
                throw new Error(t('connection_test_failed') + `: ${result.message}`);
            }
        } catch (err: any) {
            console.error("Startup check failed:", err);
            setStartupStatus("failed");
            setStartupMessage(err.message || String(err));
            setIsPreferenceOpen(true);
        }
    };

    useEffect(() => {
        // Initial Check - only if not ready
        if (!isAppReady) {
            checkLLM();
        }

        // Listen for config updates to retry
        const unsubscribeConfig = EventsOn("config-updated", async () => {
            logger.info("Configuration updated, reinitializing services...");

            if (!isAppReady) {
                // If app is not ready, retry initialization
                checkLLM();
            } else {
                // If app is ready, configuration has been updated
                logger.info("Configuration updated successfully");
            }
        });

        // Listen for analysis events
        // Task 6.2: Enhanced analysis error handling (Requirement 2.4)
        const unsubscribeAnalysisError = EventsOn("analysis-error", (payload: any) => {
            logger.debug(`Analysis error event received: ${JSON.stringify(payload)}`);
            
            // Support both old format (string) and new format (object with requestId)
            let errorMessage: string;
            let requestId: string | null = null;
            
            if (typeof payload === 'string') {
                // Old format: just error message string
                errorMessage = payload;
                logger.debug('Old format error message (no requestId)');
            } else if (payload && typeof payload === 'object') {
                // New format: object with requestId and error message
                errorMessage = payload.error || payload.message || 'Unknown error';
                requestId = payload.requestId || null;
                logger.debug(`New format error with requestId: ${requestId}`);
            } else {
                errorMessage = 'Unknown error';
                logger.warn('Invalid error payload format');
            }
            
            // Task 6.2: Verify requestId matches if provided (Requirement 2.4)
            if (requestId) {
                // Check if this error is for the current pending request
                if (requestId !== pendingRequestId) {
                    logger.info(`Ignoring stale error - requestId mismatch: received=${requestId}, expected=${pendingRequestId}`);
                    return; // Ignore errors for old requests
                }
                
                logger.debug(`RequestId matched: ${requestId}, processing error`);
                
                // Clear the timeout for this request
                const timeoutId = requestTimeouts.get(requestId);
                if (timeoutId) {
                    clearTimeout(timeoutId);
                    setRequestTimeouts(prev => {
                        const newMap = new Map(prev);
                        newMap.delete(requestId);
                        return newMap;
                    });
                    logger.debug(`Cleared timeout for failed request ${requestId}`);
                }
                
                // Task 6.2: Clear loading state but keep dashboard data (Requirement 2.4)
                setPendingRequestId(null);
                setIsAnalysisLoading(false);
                logger.debug('Loading state cleared, dashboard data preserved');
            } else {
                // Backward compatibility: if no requestId, clear loading state anyway
                logger.debug('No requestId in error payload, clearing loading state (backward compatibility)');
                setPendingRequestId(null);
                setIsAnalysisLoading(false);
            }
            
            // Task 6.2: Show error toast message (Requirement 2.4)
            showToast('error', errorMessage, '分析失败');
            logger.warn(`Analysis failed: ${errorMessage}`);
            
            // Note: dashboardData is NOT modified - existing data is preserved
        });
        const unsubscribeAnalysisWarning = EventsOn("analysis-warning", (msg: string) => {
            alert(`Analysis Warning: ${msg}`);
        });

        // Listen for loading state from ChatSidebar
        const unsubscribeLoading = EventsOn('chat-loading', (data: any) => {
            if (typeof data === 'boolean') {
                // 向后兼容：如果是布尔值，应用到当前活动会话
                if (activeSessionId) {
                    setIsAnalysisLoading(data);
                    if (data) {
                        setLoadingThreadId(activeSessionId);
                    } else {
                        setLoadingThreadId(null);
                    }
                }
            } else if (data && typeof data === 'object') {
                // 新格式：包含threadId的对象
                setIsAnalysisLoading(data.loading);
                if (data.loading) {
                    setLoadingThreadId(data.threadId);
                } else {
                    setLoadingThreadId(null);
                }
            }
        });

        // Listen for menu event
        const unsubscribeSettings = EventsOn("open-settings", () => {
            setIsPreferenceOpen(true);
        });

        // Listen for about menu event
        const unsubscribeAbout = EventsOn("open-about", () => {
            setIsAboutOpen(true);
        });

        // Listen for dashboard chart updates (with session ID)
        const unsubscribeDashboardUpdate = EventsOn("dashboard-update", (payload: any) => {
            // Log received event for debugging
            logger.debug(`Dashboard update received: ${JSON.stringify(payload).substring(0, 100)}`);
            
            // Validate payload structure
            if (!payload) {
                logger.warn("Dashboard update received with null/undefined payload");
                return;
            }

            // Validate required fields: type and data
            if (!payload.type) {
                logger.warn(`Dashboard update received without type field`);
                return;
            }

            if (payload.data === undefined || payload.data === null) {
                logger.warn(`Dashboard update received with type=${payload.type} but no data`);
                return;
            }

            // 合并新数据到现有的chartData.charts数组中
            const updateActiveChart = () => {
                setActiveChart(prevChart => {
                    const newChartItem = {
                        type: payload.type,
                        data: payload.data
                    };
                    
                    // 如果之前没有chartData，创建新的
                    if (!prevChart || !prevChart.chartData) {
                        logger.debug(`Creating new chartData with type: ${payload.type}`);
                        return {
                            type: payload.type as 'echarts' | 'image' | 'table' | 'csv',
                            data: payload.data,
                            chartData: {
                                charts: [newChartItem]
                            } as any
                        };
                    }
                    
                    // 合并到现有的charts数组中
                    const existingCharts = prevChart.chartData.charts || [];
                    // 检查是否已存在相同类型的数据，如果存在则更新，否则添加
                    const existingIndex = existingCharts.findIndex((c: any) => c.type === payload.type);
                    let updatedCharts;
                    if (existingIndex >= 0) {
                        // 更新现有的
                        updatedCharts = [...existingCharts];
                        updatedCharts[existingIndex] = newChartItem;
                        logger.debug(`Updated existing chart at index ${existingIndex}, type: ${payload.type}`);
                    } else {
                        // 添加新的
                        updatedCharts = [...existingCharts, newChartItem];
                        logger.debug(`Added new chart, type: ${payload.type}, total charts: ${updatedCharts.length}`);
                    }
                    
                    return {
                        type: payload.type as 'echarts' | 'image' | 'table' | 'csv',
                        data: payload.data,
                        chartData: {
                            charts: updatedCharts
                        } as any
                    };
                });
            };

            // Handle session ID validation and filtering
            if (payload.sessionId) {
                // Check if this event is for the current active session
                setActiveSessionId(currentSessionId => {
                    // If sessionId doesn't match current active session, silently ignore
                    if (currentSessionId && currentSessionId !== payload.sessionId) {
                        return currentSessionId;
                    }
                    
                    // Session ID matches - update chart outside of this callback
                    return currentSessionId;
                });
                
                // 直接更新 activeChart，不在 setActiveSessionId 回调中
                updateActiveChart();
                logger.debug(`Active chart updated for session ${payload.sessionId}, type: ${payload.type}`);
            } else {
                // Fallback: sessionId not provided - update anyway
                updateActiveChart();
                logger.debug(`Active chart updated (no sessionId provided, using fallback)`);
            }
        });

        // Listen for session switch to update dashboard
        const unsubscribeSessionSwitch = EventsOn("session-switched", async (sessionId: string) => {
            logger.debug(`Session switched: ${sessionId}`);
            
            // Task 6.3: Cancel pending requests when switching sessions (Requirement 2.4)
            // If there's a pending request from the previous session, cancel it
            if (pendingRequestId) {
                logger.info(`Canceling pending request ${pendingRequestId} due to session switch`);
                
                // Clear the timeout for the pending request
                const timeoutId = requestTimeouts.get(pendingRequestId);
                if (timeoutId) {
                    clearTimeout(timeoutId);
                    setRequestTimeouts(prev => {
                        const newMap = new Map(prev);
                        newMap.delete(pendingRequestId);
                        return newMap;
                    });
                    logger.debug(`Cleared timeout for request ${pendingRequestId}`);
                }
                
                // Clear loading state
                setPendingRequestId(null);
                setIsAnalysisLoading(false);
                logger.debug('Loading state cleared due to session switch');
            }
            
            setActiveSessionId(sessionId);

            // 从 sessionCharts 中加载该会话的图表
            setSessionCharts(charts => {
                const chart = charts[sessionId];
                setActiveChart(chart || null);
                return charts;
            });

            // 加载该会话的文件列表
            try {
                const { GetSessionFiles } = await import('../wailsjs/go/main/App');
                const files = await GetSessionFiles(sessionId);
                logger.debug(`Loaded ${files?.length || 0} session files for session ${sessionId}`);
                setSessionFiles(files || []);
            } catch (err) {
                logger.error(`Failed to load session files: ${err}`);
                setSessionFiles([]);
            }

            // ChatSidebar 会自动加载第一个分析结果（通过 activeThreadId 的 useEffect）
        });

        const unsubscribeDashboardDataUpdate = EventsOn("dashboard-data-update", (data: main.DashboardData) => {
            logger.debug("Dashboard data update received");
            setDashboardData(data);
            // 更新系统原始洞察和指标（如果当前没有显示LLM内容）
            if (data && data.insights) {
                const hasLLMInsights = Array.isArray(data.insights) &&
                    data.insights.some((insight: any) => insight.source === 'llm_suggestion');

                if (!hasLLMInsights) {
                    // 如果当前没有LLM建议，更新原始系统洞察
                    setOriginalSystemInsights(Array.isArray(data.insights) ? data.insights : []);
                }
            }

            if (data && data.metrics) {
                const hasLLMMetrics = Array.isArray(data.metrics) &&
                    data.metrics.some((metric: any) => metric.source === 'llm_analysis');

                if (!hasLLMMetrics) {
                    // 如果当前没有LLM指标，更新原始系统指标
                    setOriginalSystemMetrics(Array.isArray(data.metrics) ? data.metrics : []);
                }
            }
        });

        // Listen for clear dashboard data event (when thread is deleted or history is cleared)
        const unsubscribeClearDashboardData = EventsOn("clear-dashboard-data", (payload: any) => {
            logger.debug("Clear dashboard data event received");

            // Clear all dashboard data
            setDashboardData(prevData => {
                if (!prevData) return prevData;
                return main.DashboardData.createFrom({
                    ...prevData,
                    insights: [],
                    metrics: [],
                });
            });

            // Clear active chart
            setActiveChart(null);

            // Clear original system data
            setOriginalSystemInsights([]);
            setOriginalSystemMetrics([]);

            logger.info(`Dashboard cleared: ${payload?.reason || 'unknown reason'}`);
        });

        const unsubscribeAnalyzeInsight = EventsOn("analyze-insight", (text: string) => {
            logger.debug(`analyze-insight event received: ${text.substring(0, 50)}`);
            logger.debug(`Current isChatOpen state: ${isChatOpen}`);

            // First, open the chat sidebar
            setIsChatOpen(true);
            logger.debug('Set isChatOpen to true');

            // Then, after a small delay to allow sidebar to mount, send the message
            // Use setTimeout to ensure the sidebar component has mounted and initialized
            setTimeout(() => {
                logger.debug(`Sending chat-send-message event: ${text.substring(0, 50)}`);
                EventsEmit('chat-send-message', text);
            }, 150); // 150ms delay to ensure sidebar is fully mounted
        });

        const unsubscribeAnalyzeInsightInSession = EventsOn("analyze-insight-in-session", (data: any) => {
            logger.debug(`analyze-insight-in-session event received: ${JSON.stringify(data).substring(0, 100)}`);
            logger.debug(`Current isChatOpen state: ${isChatOpen}`);

            // First, open the chat sidebar
            setIsChatOpen(true);
            logger.debug('Set isChatOpen to true');

            // Then, after a small delay to allow sidebar to mount, send the message with session context
            setTimeout(() => {
                logger.debug(`Sending chat-send-message-in-session event`);
                EventsEmit('chat-send-message-in-session', data);
            }, 150); // 150ms delay to ensure sidebar is fully mounted
        });

        const unsubscribeStartNewChat = EventsOn("start-new-chat", (data: any) => {
            setIsChatOpen(true);
            // 启动分析会话时折叠数据浏览器区域
            setIsContextPanelCollapsed(true);
            // If keepChatOpen is true, don't auto-hide the chat area
            if (data && data.keepChatOpen) {
                logger.debug('start-new-chat with keepChatOpen=true, keeping chat area open');
                // Additional logic could be added here if needed
            }
        });

        const unsubscribeEnsureChatOpen = EventsOn("ensure-chat-open", () => {
            logger.debug('ensure-chat-open event received, ensuring chat is open');
            setIsChatOpen(true);
        });

        const unsubscribeOpenDevTools = EventsOn("open-dev-tools", () => {
            // Show instructions to user on how to open developer tools
            setMessageModal({
                isOpen: true,
                type: 'info',
                title: '打开开发者工具',
                message: '请按 F12 键或右键点击页面选择"检查元素"来打开开发者工具控制台。'
            });
        });

        const unsubscribeClearDashboard = EventsOn("clear-dashboard", async () => {
            logger.debug("Clearing dashboard - resetting to initial state");
            // 清空所有会话相关的状态
            setSelectedUserRequest(null);
            setActiveChart(null);
            setSessionCharts({});

            // 重新获取并显示系统初始的仪表盘数据（数据源统计和自动洞察）
            logger.debug("Reloading original system dashboard data");
            try {
                const freshData = await GetDashboardData();
                logger.debug(`Fresh dashboard data loaded: ${JSON.stringify(freshData)}`);
                setDashboardData(freshData);

                // 更新保存的初始数据
                if (freshData && freshData.insights) {
                    setOriginalSystemInsights(Array.isArray(freshData.insights) ? freshData.insights : []);
                }
                if (freshData && freshData.metrics) {
                    setOriginalSystemMetrics(Array.isArray(freshData.metrics) ? freshData.metrics : []);
                }
            } catch (err) {
                logger.error(`Failed to reload dashboard data: ${err}`);
                // 如果获取失败，尝试使用保存的数据
                setDashboardData(prevData => {
                    if (!prevData) return prevData;

                    return main.DashboardData.createFrom({
                        ...prevData,
                        insights: originalSystemInsights,
                        metrics: originalSystemMetrics
                    });
                });
            }
        });

        const unsubscribeAnalysisCompleted = EventsOn("analysis-completed", (payload: any) => {
            logger.debug(`Analysis completed event received: ${JSON.stringify(payload)}`);

            const { threadId, userMessageId, assistantMsgId, hasChartData, requestId } = payload;

            // Task 3.1: Verify requestId matches pending request (Requirements 1.3, 4.3, 4.4)
            if (requestId) {
                // Check if this result matches the current pending request
                if (requestId !== pendingRequestId) {
                    logger.info(`Ignoring stale analysis result - requestId mismatch: received=${requestId}, expected=${pendingRequestId}`);
                    return; // Ignore outdated results
                }
                
                logger.debug(`RequestId matched: ${requestId}, proceeding with dashboard update`);
                
                // Task 6.1: Clear the timeout for this request (Requirement 2.4)
                const timeoutId = requestTimeouts.get(requestId);
                if (timeoutId) {
                    clearTimeout(timeoutId);
                    setRequestTimeouts(prev => {
                        const newMap = new Map(prev);
                        newMap.delete(requestId);
                        return newMap;
                    });
                    logger.debug(`Cleared timeout for request ${requestId}`);
                }
                
                // Task 3.2: Clear loading state when result matches (Requirements 2.3)
                setPendingRequestId(null);
                setLastCompletedRequestId(requestId);
                setIsAnalysisLoading(false);
            } else {
                // Backward compatibility: if no requestId in payload, process anyway
                logger.debug('No requestId in payload, processing without validation (backward compatibility)');
                
                // Task 3.2: Clear loading state for backward compatibility (Requirements 2.3)
                setPendingRequestId(null);
                setIsAnalysisLoading(false);
            }

            // Set active session ID so that clicking insights will continue in this session
            if (threadId) {
                logger.debug(`Setting activeSessionId to: ${threadId}`);
                setActiveSessionId(threadId);
            }

            // 清除仪表盘所有内容，准备显示新的分析结果
            logger.debug('Clearing dashboard for new analysis results');
            setDashboardData(prevData => {
                if (!prevData) return prevData;

                return main.DashboardData.createFrom({
                    ...prevData,
                    insights: [],  // 清除所有洞察
                    metrics: []    // 清除所有指标
                });
            });

            // 注意：不清除 activeChart，因为 dashboard-update 事件已经设置了正确的数据
            // 如果清除会导致实时更新的数据丢失
            // setActiveChart(null);

            // 延迟加载新的分析结果（确保清除操作完成）
            setTimeout(() => {
                logger.debug(`Auto-loading analysis results for message: ${userMessageId}`);

                // 触发 user-message-clicked 事件来加载完整的分析结果
                // 注意：不传递 chartData，让 ChatSidebar 从消息历史中加载
                // ChatSidebar 会监听这个事件并从消息的 chart_data 字段加载数据
                EventsEmit('load-message-data', {
                    messageId: userMessageId,
                    threadId: threadId
                });
            }, 150); // 150ms 延迟确保清除完成
        });

        const unsubscribeMessageModal = EventsOn("show-message-modal", (data: any) => {
            setMessageModal({
                isOpen: true,
                type: data.type || 'info',
                title: data.title || '',
                message: data.message || ''
            });
        });

        const unsubscribeUserMessageClick = EventsOn("user-message-clicked", async (payload: any) => {
            logger.debug(`User message clicked: ${payload.messageId}`);
            logger.debug(`Has chartData: ${!!payload.chartData}`);
            if (payload.chartData) {
                logger.debug(`ChartData.charts length: ${payload.chartData.charts?.length || 0}`);
            }

            setSelectedUserRequest(payload.content);
            setSelectedMessageId(payload.messageId);  // 保存当前选中的消息ID

            // 获取会话文件列表
            if (activeSessionId) {
                try {
                    const { GetSessionFiles } = await import('../wailsjs/go/main/App');
                    const files = await GetSessionFiles(activeSessionId);
                    logger.debug(`Loaded ${files?.length || 0} session files for active session ${activeSessionId}`);
                    if (files && files.length > 0) {
                        logger.debug(`Session files: ${JSON.stringify(files.map(f => ({ name: f.name, type: f.type, size: f.size, message_id: f.message_id })))}`);
                    }
                    setSessionFiles(files || []);
                } catch (err) {
                    logger.error(`Failed to load session files: ${err}`);
                    setSessionFiles([]);
                }
            } else {
                logger.warn('No active session ID, cannot load session files');
            }

            // 加载与此用户消息关联的LLM建议和指标
            if (payload.messageId) {
                logger.debug(`Loading insights and metrics for message: ${payload.messageId}`);

                // 首先尝试从后端加载保存的指标JSON
                EventsEmit('load-metrics-json', { messageId: payload.messageId });

                // 处理洞察和指标
                setSessionInsights(currentInsights => {
                    const messageInsights = currentInsights[payload.messageId];

                    setSessionMetrics(currentMetrics => {
                        const messageMetrics = currentMetrics[payload.messageId];

                        // 更新 Dashboard 数据
                        setDashboardData(prevData => {
                            if (!prevData) return prevData;

                            const hasInsights = messageInsights && messageInsights.length > 0;
                            const hasMetrics = messageMetrics && messageMetrics.length > 0;

                            logger.debug(`Message insights: ${hasInsights ? messageInsights.length : 0}`);
                            logger.debug(`Message metrics: ${hasMetrics ? messageMetrics.length : 0}`);
                            logger.debug(`Current insights: ${prevData.insights?.length || 0}`);
                            logger.debug(`Current metrics: ${prevData.metrics?.length || 0}`);

                            // 策略：
                            // 1. 如果有该消息的 insights/metrics，使用它们
                            // 2. 如果没有，清空显示（不保留之前的内容）
                            return main.DashboardData.createFrom({
                                ...prevData,
                                insights: hasInsights ? messageInsights : [],
                                metrics: hasMetrics ? messageMetrics : []
                            });
                        });

                        return currentMetrics;
                    });

                    return currentInsights;
                });
            } else {
                // 没有messageId时，保持当前状态不变
                logger.debug(`No messageId provided, keeping current dashboard state`);
            }

            if (payload.chartData) {
                // Check if this is the new format (with charts array) or old format (direct type/data)
                if (payload.chartData.charts && Array.isArray(payload.chartData.charts) && payload.chartData.charts.length > 0) {
                    // New format: ChartData with charts array
                    const firstChart = payload.chartData.charts[0];
                    logger.debug(`New format detected - Chart count: ${payload.chartData.charts.length}`);

                    if (firstChart && firstChart.type && firstChart.data) {
                        setActiveChart({
                            type: firstChart.type,
                            data: firstChart.data,
                            chartData: payload.chartData // Store full ChartData for multi-chart support
                        });
                        logger.info(`Active chart set with ${payload.chartData.charts.length} charts`);
                    } else {
                        logger.warn(`Invalid first chart in array`);
                        // 不清空，保留之前的数据
                    }
                } else if (payload.chartData.type && payload.chartData.data) {
                    // Old format: Direct type and data fields (backward compatibility)
                    logger.debug(`Old format detected - Chart type: ${payload.chartData.type}`);

                    // Convert old format to new format
                    const convertedChartData = {
                        charts: [{
                            type: payload.chartData.type,
                            data: payload.chartData.data
                        }]
                    };

                    setActiveChart({
                        type: payload.chartData.type,
                        data: payload.chartData.data,
                        chartData: convertedChartData as any // Convert to new format for consistency
                    });

                    logger.info(`Active chart set (converted from old format)`);
                } else {
                    logger.debug(`Invalid chartData format - keeping current activeChart`);
                    // 不清空，保留之前的数据
                }
            } else {
                // No chart data in payload - keep current activeChart
                // This is important because dashboard-update events may have already set the data
                logger.debug(`No chartData in payload - keeping current activeChart`);
            }
        });

        // 监听Dashboard洞察更新事件
        const unsubscribeUpdateDashboardInsights = EventsOn("update-dashboard-insights", (payload: any) => {
            logger.debug(`Dashboard insights update received: ${payload?.insights?.length || 0} insights`);
            if (payload && payload.insights && Array.isArray(payload.insights) && payload.userMessageId) {
                // 存储与特定用户消息关联的建议
                setSessionInsights(prev => ({
                    ...prev,
                    [payload.userMessageId]: payload.insights
                }));

                // 显示新的LLM建议时，清除所有现有洞察（包括系统初始化的内容），但保留metrics
                setDashboardData(prevData => {
                    if (!prevData) return prevData;

                    // 转换新的洞察格式
                    const newInsights = payload.insights.map((insight: any) => ({
                        text: insight.text,
                        icon: insight.icon || 'star',
                        source: insight.source || 'llm_suggestion',
                        userMessageId: insight.userMessageId
                    }));

                    return main.DashboardData.createFrom({
                        ...prevData,
                        insights: newInsights,  // 完全替换所有洞察，清除系统初始化内容
                        metrics: prevData.metrics || []  // 明确保留metrics
                    });
                });
            }
        });

        // 监听Dashboard指标更新事件
        const unsubscribeUpdateDashboardMetrics = EventsOn("update-dashboard-metrics", (payload: any) => {
            logger.debug(`Dashboard metrics update received: ${payload?.metrics?.length || 0} metrics`);
            if (payload && payload.metrics && Array.isArray(payload.metrics) && payload.userMessageId) {
                // 存储与特定用户消息关联的指标
                setSessionMetrics(prev => ({
                    ...prev,
                    [payload.userMessageId]: payload.metrics
                }));

                // 显示新的LLM指标时，完全替换所有现有指标，但保留insights
                setDashboardData(prevData => {
                    if (!prevData) return prevData;

                    // 转换新的指标格式
                    const newMetrics = payload.metrics.map((metric: any) => ({
                        title: metric.title,
                        value: metric.value,
                        change: metric.change || '',
                        source: metric.source || 'llm_analysis',
                        userMessageId: metric.userMessageId
                    }));

                    return main.DashboardData.createFrom({
                        ...prevData,
                        metrics: newMetrics,  // 完全替换所有指标
                        insights: prevData.insights || []  // 明确保留insights
                    });
                });
            }
        });

        // 监听指标提取开始事件
        const unsubscribeMetricsExtracting = EventsOn("metrics-extracting", (messageId: string) => {
            logger.debug(`Metrics extraction started for message: ${messageId}`);
            // 可以在这里显示提取状态指示器
        });

        // 监听指标提取完成事件
        const unsubscribeMetricsExtracted = EventsOn("metrics-extracted", (payload: any) => {
            logger.debug(`Metrics extracted: ${payload?.metrics?.length || 0} metrics for message ${payload?.messageId}`);
            logger.debug(`Current activeChart state: ${activeChart ? activeChart.type : 'null'}`);

            if (payload && payload.messageId && payload.metrics) {
                // 转换为Dashboard格式
                const formattedMetrics = payload.metrics.map((metric: any, index: number) => {
                    const cleanName = String(metric.name || '').trim();
                    const cleanValue = String(metric.value || '').trim();
                    const cleanUnit = metric.unit ? String(metric.unit).trim() : '';

                    // 格式化显示值
                    const formattedValue = cleanUnit ? `${cleanValue}${cleanUnit}` : cleanValue;

                    // 计算变化趋势
                    let change = '';
                    if (cleanValue.includes('+')) {
                        change = '↗️ 上升';
                    } else if (cleanValue.includes('-')) {
                        change = '↘️ 下降';
                    } else if (cleanUnit === '%') {
                        const numValue = parseFloat(cleanValue.replace(/[+\-,]/g, ''));
                        if (!isNaN(numValue) && numValue > 10) {
                            change = '📈 良好';
                        }
                    } else if (cleanUnit && (cleanUnit.includes('次/') || cleanUnit.includes('率'))) {
                        change = '🔄 周期';
                    }

                    return {
                        title: cleanName,
                        value: formattedValue,
                        change: change,
                        source: 'llm_auto_extracted',
                        id: `auto_metric_${payload.messageId}_${index}`,
                        userMessageId: payload.messageId
                    };
                });

                // 存储到sessionMetrics中
                setSessionMetrics(prev => ({
                    ...prev,
                    [payload.messageId]: formattedMetrics
                }));

                // 更新Dashboard显示 - 只更新metrics，保留insights和其他数据
                setDashboardData(prevData => {
                    if (!prevData) return prevData;

                    logger.debug(`Before metrics update - insights count: ${prevData.insights?.length || 0}`);

                    const newData = main.DashboardData.createFrom({
                        ...prevData,
                        metrics: formattedMetrics,
                        // 明确保留insights，防止被清除
                        insights: prevData.insights || []
                    });

                    logger.debug(`After metrics update - insights count: ${newData.insights?.length || 0}`);
                    return newData;
                });

                logger.info(`Auto-extracted metrics displayed, insights preserved`);
            }
        });

        // 监听保存指标JSON事件（保留现有功能作为备用）
        const unsubscribeSaveMetricsJson = EventsOn("save-metrics-json", async (payload: any) => {
            logger.debug(`Save metrics JSON request for message: ${payload?.messageId}`);
            if (payload && payload.messageId && payload.metrics) {
                try {
                    // 调用后端API保存指标JSON
                    const { SaveMetricsJson } = await import('../wailsjs/go/main/App');
                    await SaveMetricsJson(payload.messageId, JSON.stringify(payload.metrics));
                    logger.info(`Metrics JSON saved successfully for message: ${payload.messageId}`);
                } catch (error) {
                    logger.error(`Failed to save metrics JSON: ${error}`);
                }
            }
        });

        // 监听加载指标JSON事件
        const unsubscribeLoadMetricsJson = EventsOn("load-metrics-json", async (payload: any) => {
            console.log("[DEBUG] Load metrics JSON request:", payload);
            if (payload && payload.messageId) {
                try {
                    // 调用后端API加载指标JSON
                    const { LoadMetricsJson } = await import('../wailsjs/go/main/App');
                    const metricsJson = await LoadMetricsJson(payload.messageId);

                    console.log("[DEBUG] Raw metrics JSON:", metricsJson);

                    // 清理JSON字符串，移除可能的JavaScript函数
                    let cleanedJson = metricsJson;
                    if (typeof metricsJson === 'string') {
                        // 移除JavaScript函数定义
                        cleanedJson = metricsJson
                            .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                            .replace(/,?\s*"?matter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                            .replace(/,?\s*[a-zA-Z_$][a-zA-Z0-9_$]*\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
                            .replace(/,(\s*[}\]])/g, '$1')
                            .replace(/(\{\s*),/g, '$1');

                        logger.debug(`Cleaned metrics JSON, length: ${cleanedJson.length}`);
                    }

                    const metricsData = JSON.parse(cleanedJson);

                    logger.debug(`Metrics JSON loaded successfully: ${metricsData.length} metrics`);

                    // 转换为Dashboard格式并更新显示
                    const formattedMetrics = metricsData.map((metric: any, index: number) => {
                        const cleanName = String(metric.name || '').trim();
                        const cleanValue = String(metric.value || '').trim();
                        const cleanUnit = metric.unit ? String(metric.unit).trim() : '';

                        // 格式化显示值
                        const formattedValue = cleanUnit ? `${cleanValue}${cleanUnit}` : cleanValue;

                        // 计算变化趋势
                        let change = '';
                        if (cleanValue.includes('+')) {
                            change = '↗️ 上升';
                        } else if (cleanValue.includes('-')) {
                            change = '↘️ 下降';
                        } else if (cleanUnit === '%') {
                            const numValue = parseFloat(cleanValue.replace(/[+\-,]/g, ''));
                            if (!isNaN(numValue) && numValue > 10) {
                                change = '📈 良好';
                            }
                        } else if (cleanUnit && (cleanUnit.includes('次/') || cleanUnit.includes('率'))) {
                            change = '🔄 周期';
                        }

                        return {
                            title: cleanName,
                            value: formattedValue,
                            change: change,
                            source: 'llm_json_metrics',
                            id: `loaded_metric_${payload.messageId}_${index}`,
                            userMessageId: payload.messageId
                        };
                    });

                    // 存储到sessionMetrics中
                    setSessionMetrics(prev => ({
                        ...prev,
                        [payload.messageId]: formattedMetrics
                    }));

                    // 更新Dashboard显示 - 保留insights
                    setDashboardData(prevData => {
                        if (!prevData) return prevData;

                        return main.DashboardData.createFrom({
                            ...prevData,
                            metrics: formattedMetrics,
                            insights: prevData.insights || []  // 明确保留insights
                        });
                    });

                } catch (error) {
                    logger.error(`Failed to load metrics JSON: ${error}`);
                    // 如果加载失败，可能是文件不存在，这是正常情况
                    logger.debug(`No saved metrics found for message: ${payload.messageId}`);
                }
            }
        });

        // Global Context Menu Listener
        const handleContextMenu = (e: MouseEvent) => {
            const target = e.target as HTMLElement;
            if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
                // Don't prevent default for input fields - allow native context menu
                // e.preventDefault();
                setContextMenu({ x: e.clientX, y: e.clientY, target });
            }
        };

        window.addEventListener('contextmenu', handleContextMenu);

        return () => {
            if (unsubscribeConfig) unsubscribeConfig();
            if (unsubscribeAnalysisError) unsubscribeAnalysisError();
            if (unsubscribeAnalysisWarning) unsubscribeAnalysisWarning();
            if (unsubscribeLoading) unsubscribeLoading();
            if (unsubscribeSettings) unsubscribeSettings();
            if (unsubscribeAbout) unsubscribeAbout();
            if (unsubscribeDashboardUpdate) unsubscribeDashboardUpdate();
            if (unsubscribeSessionSwitch) unsubscribeSessionSwitch();
            if (unsubscribeDashboardDataUpdate) unsubscribeDashboardDataUpdate();
            if (unsubscribeClearDashboardData) unsubscribeClearDashboardData();
            if (unsubscribeAnalyzeInsight) unsubscribeAnalyzeInsight();
            if (unsubscribeAnalyzeInsightInSession) unsubscribeAnalyzeInsightInSession();
            if (unsubscribeStartNewChat) unsubscribeStartNewChat();
            if (unsubscribeEnsureChatOpen) unsubscribeEnsureChatOpen();
            if (unsubscribeOpenDevTools) unsubscribeOpenDevTools();
            if (unsubscribeClearDashboard) unsubscribeClearDashboard();
            if (unsubscribeAnalysisCompleted) unsubscribeAnalysisCompleted();
            if (unsubscribeMessageModal) unsubscribeMessageModal();
            if (unsubscribeUserMessageClick) unsubscribeUserMessageClick();
            if (unsubscribeUpdateDashboardInsights) unsubscribeUpdateDashboardInsights();
            if (unsubscribeUpdateDashboardMetrics) unsubscribeUpdateDashboardMetrics();
            if (unsubscribeMetricsExtracting) unsubscribeMetricsExtracting();
            if (unsubscribeMetricsExtracted) unsubscribeMetricsExtracted();
            if (unsubscribeSaveMetricsJson) unsubscribeSaveMetricsJson();
            if (unsubscribeLoadMetricsJson) unsubscribeLoadMetricsJson();
            window.removeEventListener('contextmenu', handleContextMenu);
        };
    }, [isAppReady]);

    // Resize Handlers
    useEffect(() => {
        const handleMouseMove = (e: MouseEvent) => {
            if (isResizingSidebar) {
                const newWidth = e.clientX;
                if (newWidth > 150 && newWidth < 600) {
                    setSidebarWidth(newWidth);
                }
            } else if (isResizingContextPanel) {
                // Context Panel starts after sidebar. 
                // We can calculate its width as (currentX - sidebarWidth)
                // However, there might be a resizer width offset.
                const newWidth = e.clientX - sidebarWidth;
                if (newWidth > 200 && newWidth < 800) {
                    setContextPanelWidth(newWidth);
                }
            }
        };

        const handleMouseUp = () => {
            setIsResizingSidebar(false);
            setIsResizingContextPanel(false);
            document.body.style.cursor = 'default';
        };

        if (isResizingSidebar || isResizingContextPanel) {
            window.addEventListener('mousemove', handleMouseMove);
            window.addEventListener('mouseup', handleMouseUp);
        }

        return () => {
            window.removeEventListener('mousemove', handleMouseMove);
            window.removeEventListener('mouseup', handleMouseUp);
        };
    }, [isResizingSidebar, isResizingContextPanel, sidebarWidth]);

    const startResizingSidebar = () => {
        setIsResizingSidebar(true);
        document.body.style.cursor = 'col-resize';
    };

    const startResizingContextPanel = () => {
        setIsResizingContextPanel(true);
        document.body.style.cursor = 'col-resize';
    };

    if (!isAppReady) {
        return (
            <div className="flex h-screen w-screen bg-slate-50 items-center justify-center flex-col gap-6 relative">
                {/* Removed draggable area - using system window border for dragging */}

                <div className="w-16 h-16 border-4 border-blue-200 border-t-blue-600 rounded-full animate-spin"></div>

                {/* Normal startup UI */}
                <div className="text-center max-w-md px-6">
                    <h2 className="text-xl font-semibold text-slate-800 mb-2">{t('system_startup')}</h2>
                    <p className={`text-sm ${startupStatus === 'failed' ? 'text-red-600' : 'text-slate-600'}`}>
                        {startupMessage}
                    </p>

                    {startupStatus === 'failed' && (
                        <div className="mt-6 flex flex-col gap-3">
                            <button
                                onClick={() => setIsPreferenceOpen(true)}
                                className="px-6 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 transition-colors shadow-sm"
                            >
                                {t('open_settings')}
                            </button>
                            <button
                                onClick={checkLLM}
                                className="px-6 py-2 bg-white border border-slate-300 text-slate-700 text-sm font-medium rounded-md hover:bg-slate-50 transition-colors"
                            >
                                {t('retry_connection')}
                            </button>
                        </div>
                    )}
                </div>

                <PreferenceModal
                    isOpen={isPreferenceOpen}
                    onClose={() => setIsPreferenceOpen(false)}
                />
            </div>
        );
    }

    return (
            <div className="flex h-screen overflow-hidden bg-slate-100 font-sans text-slate-900 relative">
                {/* Removed draggable title bar - using system window border for dragging */}

                <Sidebar
                    width={sidebarWidth}
                    onOpenSettings={() => setIsPreferenceOpen(true)}
                    onToggleChat={() => setIsChatOpen(!isChatOpen)}
                    isChatOpen={isChatOpen}
                    isAnalysisLoading={isAnalysisLoading}
                />

                {/* Sidebar Resizer */}
                <div
                    className={`w-1 hover:bg-blue-400 cursor-col-resize z-50 transition-colors flex-shrink-0 ${isResizingSidebar ? 'bg-blue-600' : 'bg-transparent'}`}
                    onMouseDown={startResizingSidebar}
                />

                {/* 数据浏览器区域 - 可折叠 */}
                {!isContextPanelCollapsed ? (
                    <>
                        <ContextPanel
                            width={contextPanelWidth}
                            onContextPanelClick={() => {
                                if (isChatOpen) {
                                    setIsChatOpen(false);
                                }
                            }}
                            onCollapse={() => setIsContextPanelCollapsed(true)}
                        />

                        {/* Context Panel Resizer */}
                        <div
                            className={`w-1 hover:bg-blue-400 cursor-col-resize z-50 transition-colors flex-shrink-0 ${isResizingContextPanel ? 'bg-blue-600' : 'bg-transparent'}`}
                            onMouseDown={startResizingContextPanel}
                        />
                    </>
                ) : (
                    /* 折叠状态下显示展开按钮 */
                    <div className="relative flex-shrink-0">
                        <button
                            onClick={() => setIsContextPanelCollapsed(false)}
                            className="absolute left-0 top-1/2 -translate-y-1/2 z-10 bg-blue-500 hover:bg-blue-600 text-white px-1 py-3 rounded-r-md shadow-lg transition-colors"
                            title="展开数据浏览器"
                        >
                            <ChevronRight className="w-4 h-4" />
                        </button>
                    </div>
                )}

                <div className="flex-1 flex flex-col min-w-0">
                    <DraggableDashboard
                        data={dashboardData}
                        activeChart={activeChart}
                        userRequestText={selectedUserRequest}
                        isChatOpen={isChatOpen}
                        activeThreadId={activeSessionId}
                        isAnalysisLoading={isAnalysisLoading}
                        loadingThreadId={loadingThreadId}
                        sessionFiles={sessionFiles}
                        selectedMessageId={selectedMessageId}
                        onInsightClick={handleInsightClick}
                        onDashboardClick={() => {
                            if (isChatOpen) {
                                setIsChatOpen(false);
                            }
                        }}
                    />
                </div>

                <ChatSidebar
                    isOpen={isChatOpen}
                    onClose={() => {
                        logger.debug('ChatSidebar onClose called');
                        setIsChatOpen(false);
                    }}
                />

                <PreferenceModal
                    isOpen={isPreferenceOpen}
                    onClose={() => setIsPreferenceOpen(false)}
                    onOpenSkills={() => {
                        setIsPreferenceOpen(false);
                        setIsSkillsOpen(true);
                    }}
                />

                <SkillsManagementPage
                    isOpen={isSkillsOpen}
                    onClose={() => setIsSkillsOpen(false)}
                />

                <AboutModal
                    isOpen={isAboutOpen}
                    onClose={() => setIsAboutOpen(false)}
                />

                <MessageModal
                    isOpen={messageModal.isOpen}
                    type={messageModal.type}
                    title={messageModal.title}
                    message={messageModal.message}
                    onClose={() => setMessageModal(prev => ({ ...prev, isOpen: false }))}
                />

                {contextMenu && (
                    <ContextMenu
                        position={{ x: contextMenu.x, y: contextMenu.y }}
                        target={contextMenu.target}
                        onClose={() => setContextMenu(null)}
                    />
                )}

                {!isChatOpen && (
                    <button
                        onClick={() => setIsChatOpen(true)}
                        className="fixed left-0 top-1/2 -translate-y-1/2 z-[40] bg-white border border-slate-200 border-l-0 rounded-r-xl p-2 shadow-lg hover:bg-slate-50 text-blue-600 transition-transform hover:translate-x-1 group"
                        title="Open Chat"
                    >
                        <ChevronRight className="w-5 h-5 group-hover:scale-110 transition-transform" />
                    </button>
                )}
            </div>
    );
}

// Main App component that provides ToastContext
function App() {
    return (
        <ToastProvider>
            <AppContent />
        </ToastProvider>
    );
}

export default App;
