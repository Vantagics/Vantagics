import React, { useState, useRef, useEffect } from 'react';
import ReactDOM from 'react-dom';
import ReactMarkdown from 'react-markdown';
import MetricCard from './MetricCard';
import Chart from './Chart';
import DataTable from './DataTable';
import TimingAnalysisModal from './TimingAnalysisModal';
import { User, Bot, ZoomIn } from 'lucide-react';
import { EventsEmit } from '../../wailsjs/runtime/runtime';
import { GetSessionFileAsBase64 } from '../../wailsjs/go/main/App';
import { createLogger } from '../utils/systemLog';
import { useLanguage } from '../i18n';

const systemLog = createLogger('MessageBubble');

interface MessageBubbleProps {
    role: 'user' | 'assistant';
    content: string;
    payload?: string;
    onActionClick?: (action: any) => void;
    onClick?: () => void;
    hasChart?: boolean;
    messageId?: string;  // 新增：消息ID用于关联建议
    userMessageId?: string;  // 新增：关联的用户消息ID（用于assistant消息）
    dataSourceId?: string;  // 新增：当前会话的数据源ID
    isDisabled?: boolean;  // 新增：是否禁用点击（用于未完成的用户消息）
    timingData?: any;  // 新增：耗时数据
    threadId?: string;  // 新增：线程ID用于加载图片
    isFailed?: boolean;  // 新增：分析是否失败
    onRetryAnalysis?: () => void;  // 新增：重新分析回调
}

const MessageBubble: React.FC<MessageBubbleProps> = ({ role, content, payload, onActionClick, onClick, hasChart, messageId, userMessageId, dataSourceId, isDisabled, timingData, threadId, isFailed, onRetryAnalysis }) => {
    const { t } = useLanguage();
    const isUser = role === 'user';
    const [enlargedImage, setEnlargedImage] = useState<string | null>(null);
    const [clickedActions, setClickedActions] = useState<Set<string>>(new Set());
    const pendingActionsRef = useRef<Set<string>>(new Set()); // 跟踪正在处理的按钮点击
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; selectedText: string } | null>(null);
    const [exportMenu, setExportMenu] = useState<{ x: number; y: number } | null>(null);
    const [timingModalOpen, setTimingModalOpen] = useState(false);
    const messageContentRef = useRef<HTMLDivElement>(null);

    // Image component that loads base64 data for file:// URLs
    const MessageImage: React.FC<{ src?: string; alt?: string; threadId?: string; [key: string]: any }> = ({ src, alt, threadId, ...rest }) => {
        const [imageSrc, setImageSrc] = useState<string | null>(null);
        const [loading, setLoading] = useState(true);
        const [error, setError] = useState(false);

        useEffect(() => {
            if (!src) {
                setLoading(false);
                return;
            }

            // If it's already a data URL, use it directly
            if (src.startsWith('data:')) {
                setImageSrc(src);
                setLoading(false);
                return;
            }

            // If it's a file:// URL, extract the filename and load via API
            if (src.startsWith('file://')) {
                const loadImage = async () => {
                    try {
                        // Extract filename from file:// URL
                        // Format: file:///path/to/sessions/{threadId}/files/{filename}
                        // Or: file:///path/to/files/{filename} (relative path)
                        const match = src.match(/files[\/\\]([^\/\\]+)$/);
                        if (!match) {
                            console.error('[MessageImage] Could not extract filename from:', src);
                            setError(true);
                            setLoading(false);
                            return;
                        }

                        const filename = match[1];
                        
                        // Try to extract threadId from the path first
                        let extractedThreadId = threadId; // Use prop as fallback
                        const threadMatch = src.match(/sessions[\/\\]([^\/\\]+)[\/\\]files/);
                        if (threadMatch) {
                            extractedThreadId = threadMatch[1];
                        }

                        if (!extractedThreadId) {
                            console.error('[MessageImage] No threadId available (neither in path nor prop)');
                            setError(true);
                            setLoading(false);
                            return;
                        }

                        console.log('[MessageImage] Loading image:', { threadId: extractedThreadId, filename, src });

                        const base64Data = await GetSessionFileAsBase64(extractedThreadId, filename);
                        setImageSrc(base64Data);
                        setLoading(false);
                    } catch (err) {
                        console.error('[MessageImage] Failed to load image:', err);
                        setError(true);
                        setLoading(false);
                    }
                };

                loadImage();
            } else if (src.startsWith('sandbox:')) {
                // Handle sandbox: paths (OpenAI code interpreter format)
                // Format: sandbox:/mnt/data/chart.png
                const loadImage = async () => {
                    try {
                        // Extract filename from sandbox: path
                        const pathParts = src.replace('sandbox:', '').split('/');
                        const filename = pathParts[pathParts.length - 1];
                        
                        if (!threadId) {
                            console.error('[MessageImage] No threadId available for sandbox path:', src);
                            setError(true);
                            setLoading(false);
                            return;
                        }

                        console.log('[MessageImage] Loading image from sandbox path:', { threadId, filename, src });

                        const base64Data = await GetSessionFileAsBase64(threadId, filename);
                        setImageSrc(base64Data);
                        setLoading(false);
                    } catch (err) {
                        console.error('[MessageImage] Failed to load image from sandbox path:', err);
                        setError(true);
                        setLoading(false);
                    }
                };

                loadImage();
            } else if (src.startsWith('files/') || src.match(/^[^:\/]+\.(png|jpg|jpeg|gif|svg)$/i)) {
                // Handle relative paths like "files/chart_xxx.png" or "chart_xxx.png"
                const loadImage = async () => {
                    try {
                        // Extract filename
                        const filename = src.replace(/^files[\/\\]/, ''); // Remove "files/" prefix if present
                        
                        if (!threadId) {
                            console.error('[MessageImage] No threadId available for relative path:', src);
                            setError(true);
                            setLoading(false);
                            return;
                        }

                        console.log('[MessageImage] Loading image from relative path:', { threadId, filename, src });

                        const base64Data = await GetSessionFileAsBase64(threadId, filename);
                        setImageSrc(base64Data);
                        setLoading(false);
                    } catch (err) {
                        console.error('[MessageImage] Failed to load image from relative path:', err);
                        setError(true);
                        setLoading(false);
                    }
                };

                loadImage();
            } else {
                // For other URLs (http, https), use directly
                setImageSrc(src);
                setLoading(false);
            }
        }, [src, threadId]);

        if (loading) {
            return (
                <div className="relative group my-4 bg-slate-100 rounded-lg p-8 flex items-center justify-center">
                    <div className="animate-pulse text-slate-400 text-sm">{t('loading_preview')}</div>
                </div>
            );
        }

        if (error || !imageSrc) {
            return (
                <div className="relative group my-4 bg-blue-50 border border-blue-200 rounded-lg p-8 flex items-center justify-center">
                    <div className="text-blue-600 text-sm">{t('failed_load_image')}</div>
                </div>
            );
        }

        return (
            <div className="relative group my-4">
                <img
                    src={imageSrc}
                    alt={alt || 'Chart'}
                    {...rest}
                    className="rounded-lg shadow-md max-w-full cursor-pointer hover:shadow-xl transition-shadow"
                    onClick={() => setEnlargedImage(imageSrc)}
                />
                <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <div className="bg-black/60 text-white px-2 py-1 rounded text-xs flex items-center gap-1">
                        <ZoomIn className="w-3 h-3" />
                        Click to enlarge
                    </div>
                </div>
            </div>
        );
    };

    let parsedPayload: any = null;

    if (payload) {
        try {
            parsedPayload = JSON.parse(payload);
        } catch (e) {
            console.error("Failed to parse payload", e);
        }
    }

    // Helper function to check if content is in a suggestion/recommendation context
    // 使用标签检测意图理解消息，避免多语言问题
    const isSuggestionContext = (content: string): boolean => {
        // 首先检查是否有意图理解标记（最可靠）
        if (content.includes('[INTENT_SUGGESTIONS]')) {
            return true;
        }
        // 检查是否包含意图数据标记（也是可靠的标识）
        if (content.includes('[INTENT_SELECT:') || content.includes('[INTENT_RETRY_DATA:') || content.includes('[INTENT_STICK_DATA:')) {
            return true;
        }
        // 回退到关键词检测（用于其他建议类消息）
        const suggestionKeywords = [
            '建议', '分析建议', '推荐', '可以分析', '以下分析', '推荐分析', '可以进行',
            'suggest', 'recommendation', 'analysis suggestion', 'can analyze',
            'following analysis', 'recommended', 'you can', 'you could', 'consider'
        ];
        const lowerContent = content.toLowerCase();
        return suggestionKeywords.some(kw => lowerContent.includes(kw.toLowerCase()));
    };

    // Helper function to check if text is an actionable analysis item
    const isActionableItem = (text: string): boolean => {
        const actionKeywords = [
            '分析', '查看', '对比', '统计', '趋势', '预测', '细分', '探索',
            '检查', '评估', '比较', '研究', '洞察', '计算', '识别', '发现',
            '重新', '理解', '意图', // 添加意图理解相关关键词
            'analysis', 'analyze', 'compare', 'trend', 'forecast', 'segment',
            'explore', 'examine', 'assess', 'study', 'insight', 'review',
            'calculate', 'identify', 'discover', 'investigate', 'evaluate',
            'retry', 'understand', 'intent' // 英文意图关键词
        ];
        const lowerText = text.toLowerCase();
        return actionKeywords.some(kw => lowerText.includes(kw.toLowerCase()));
    };

    // Helper function to check if text matches explanation patterns (should be excluded)
    const isExplanationPattern = (text: string): boolean => {
        const exclusionPatterns = [
            /^(首先|然后|接着|最后|其次|另外|此外)/i,
            /^(first|then|next|finally|second|also|additionally)/i,
            /^(步骤|step|阶段|phase)/i,
            /^(我|I)\s+(分析|analyzed|发现|found|查看|looked)/i,
            /^(这|this|that)\s+(是|was|will|can|could)/i,
            /^(通过|by|via|using)/i,
            /^(为了|to|in order to)/i
        ];
        return exclusionPatterns.some(pattern => pattern.test(text.trim()));
    };

    // Enhanced: Extract actions from various formats
    const extractedActions: any[] = [];
    const extractedInsights: string[] = []; // 提取的洞察建议

    if (!isUser) {
        const lines = content.split('\n');
        let inSuggestionSection = false;
        let suggestionCount = 0;

        // Check if this is a suggestion response
        const hasSuggestionContext = isSuggestionContext(content);
        
        systemLog.info(`[EXTRACT] Starting extraction: hasSuggestionContext=${hasSuggestionContext}, linesCount=${lines.length}`);
        systemLog.info(`[EXTRACT] Content preview: ${content.substring(0, 300)}`);

        for (let i = 0; i < lines.length; i++) {
            const line = lines[i].trim();
            if (!line) continue;
            
            // Log each line for debugging
            if (line.includes('INTENT_') || line.includes('重新') || line.includes('理解')) {
                systemLog.info(`[EXTRACT] Processing line ${i}: ${line.substring(0, 150)}`);
            }

            // Detect suggestion section headers
            if (/^(分析建议|推荐分析|建议|suggestions?|recommendations?|you (can|could|might))[:：]/i.test(line)) {
                inSuggestionSection = true;
                continue;
            }

            // Extract from numbered lists: "1. ", "1) ", "1、"
            const numberedMatch = line.match(/^(\d+)[.、)]\s+(.+)$/);
            if (numberedMatch) {
                const rawLabel = numberedMatch[2].trim();
                
                // Log raw label for debugging intent buttons
                if (rawLabel.includes('INTENT_') || rawLabel.includes('重新') || rawLabel.includes('理解')) {
                    systemLog.info(`[EXTRACT] Found intent-related numbered item: rawLabel=${rawLabel.substring(0, 200)}`);
                }
                
                // 计算显示长度时移除嵌入的数据标记（这些标记不会显示给用户）
                const displayLabel = rawLabel
                    .replace(/\s*\[INTENT_RETRY_DATA:[^\]]*\]/g, '')
                    .replace(/\s*\[INTENT_STICK_DATA:[^\]]*\]/g, '')
                    .replace(/\s*\[INTENT_SELECT:[^\]]*\]/g, '');
                const displayLength = displayLabel.length;
                
                systemLog.debug(`Found numbered item: rawLabel=${rawLabel.substring(0, 100)}, displayLength=${displayLength}, hasSuggestionContext=${hasSuggestionContext}, inSuggestionSection=${inSuggestionSection}, isActionable=${isActionableItem(displayLabel)}`);

                // More lenient extraction in suggestion context
                if (hasSuggestionContext || inSuggestionSection) {
                    if (displayLength > 5 && displayLength < 200) {
                        suggestionCount++;
                        const cleanValue = displayLabel.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '').trim();
                        extractedActions.push({
                            id: `auto_${suggestionCount}`,
                            label: rawLabel,  // 保留原始标签（包含嵌入数据）用于点击处理
                            value: cleanValue
                        });
                        extractedInsights.push(cleanValue);
                        systemLog.info(`✅ Extracted action (suggestion context): id=auto_${suggestionCount}, label=${displayLabel.substring(0, 80)}`);
                    } else {
                        systemLog.debug(`❌ Skipped (length check): displayLength=${displayLength}`);
                    }
                } else if (isActionableItem(displayLabel) && !isExplanationPattern(displayLabel)) {
                    // Stricter filtering outside suggestion context
                    if (displayLength > 5 && displayLength < 200) {
                        suggestionCount++;
                        const cleanValue = displayLabel.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '').trim();
                        extractedActions.push({
                            id: `auto_${suggestionCount}`,
                            label: rawLabel,  // 保留原始标签（包含嵌入数据）用于点击处理
                            value: cleanValue
                        });
                        extractedInsights.push(cleanValue);
                        systemLog.info(`✅ Extracted action (actionable): id=auto_${suggestionCount}, label=${displayLabel.substring(0, 80)}`);
                    }
                } else {
                    systemLog.debug(`❌ Skipped (not actionable or is explanation)`);
                }
                continue;
            }

            // Extract from bullet points: "- ", "* ", "• "
            const bulletMatch = line.match(/^[-*•]\s+(.+)$/);
            if (bulletMatch && (hasSuggestionContext || inSuggestionSection)) {
                const rawLabel = bulletMatch[1].trim();
                if (rawLabel.length > 5 && rawLabel.length < 150 && isActionableItem(rawLabel)) {
                    suggestionCount++;
                    extractedActions.push({
                        id: `auto_bullet_${suggestionCount}`,
                        label: rawLabel,
                        value: rawLabel.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '').trim()
                    });
                    extractedInsights.push(rawLabel.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '').trim());
                }
            }
        }
        
        systemLog.info(`[EXTRACT] Extraction complete: extractedActionsCount=${extractedActions.length}`);
        if (extractedActions.length > 0) {
            extractedActions.forEach((a, i) => {
                const hasRetryData = a.label?.includes('[INTENT_RETRY_DATA:');
                const hasStickData = a.label?.includes('[INTENT_STICK_DATA:');
                const hasSelectData = a.label?.includes('[INTENT_SELECT:');
                systemLog.info(`[EXTRACT] Action ${i}: id=${a.id}, labelLen=${a.label?.length}, hasRetryData=${hasRetryData}, hasStickData=${hasStickData}, hasSelectData=${hasSelectData}`);
                if (hasRetryData || hasStickData) {
                    systemLog.info(`[EXTRACT] Action ${i} full label: ${a.label}`);
                }
            });
        }
        console.log('[MessageBubble] Extraction complete:', { 
            extractedActionsCount: extractedActions.length,
            actions: extractedActions.map(a => a.label.substring(0, 50))
        });

        // Fallback: If no actions extracted but content suggests analysis, extract from sentences
        if (extractedActions.length === 0 && hasSuggestionContext) {
            const sentences = content.split(/[。.！!？?]/);
            for (const sentence of sentences) {
                const trimmed = sentence.trim();
                if (trimmed.length > 10 && trimmed.length < 150 &&
                    isActionableItem(trimmed) &&
                    !isExplanationPattern(trimmed)) {
                    suggestionCount++;
                    if (suggestionCount <= 5) { // Limit to 5 suggestions
                        extractedActions.push({
                            id: `auto_sentence_${suggestionCount}`,
                            label: trimmed,
                            value: trimmed.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '').trim()
                        });
                        extractedInsights.push(trimmed.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '').trim());
                    }
                }
            }
        }
    }

    // 注意：关键指标现在通过后端自动提取，不再需要在这里解析json:metrics代码块

    const allActions = [
        ...(parsedPayload && parsedPayload.type === 'actions' ? parsedPayload.actions : []),
        ...extractedActions
    ];

    // Debug: Log when allActions changes - 使用 useRef 来避免重复日志
    const prevAllActionsLengthRef = useRef(0);
    useEffect(() => {
        if (allActions.length !== prevAllActionsLengthRef.current) {
            systemLog.info(`[RENDER] allActions changed: prevCount=${prevAllActionsLengthRef.current}, newCount=${allActions.length}`);
            if (allActions.length > 0) {
                systemLog.info(`[RENDER] allActions content: ${allActions.map(a => `${a.id}:${a.label?.substring(0, 30)}`).join(' | ')}`);
            }
            prevAllActionsLengthRef.current = allActions.length;
        }
    });
    
    // Debug: Log on every render to track if buttons should be visible
    useEffect(() => {
        if (allActions.length > 0 && !isUser) {
            systemLog.debug(`[RENDER] MessageBubble: role=${role}, allActionsLength=${allActions.length}, hasOnActionClick=${!!onActionClick}, messageId=${messageId}`);
        }
    });

    // 发送洞察建议到Dashboard（仅在有新建议时）
    useEffect(() => {
        if (extractedInsights && extractedInsights.length > 0 && !isUser && userMessageId) {
            // 发送洞察建议到Dashboard的自动洞察区域，关联到特定的用户消息
            EventsEmit('update-dashboard-insights', {
                userMessageId: userMessageId,  // 关联的用户消息ID
                insights: extractedInsights.map((insight, index) => ({
                    text: insight,
                    icon: 'star', // 使用星形图标表示LLM建议
                    source: 'llm_suggestion',
                    id: `llm_${userMessageId}_${index}`,
                    userMessageId: userMessageId
                    // 不添加 data_source_id，因为LLM建议应该在当前会话中继续，而不是创建新会话
                }))
            });
        }
    }, [extractedInsights.length, isUser, userMessageId]); // 移除dataSourceId依赖项

    // 注意：关键指标现在通过后端自动提取，不再需要手动发送到Dashboard

    const handleContextMenu = (e: React.MouseEvent) => {
        e.stopPropagation();
        e.preventDefault();

        const selection = window.getSelection();
        const selectedText = selection?.toString().trim();

        // If text is selected, show text selection menu
        if (selectedText && selectedText.length > 0) {
            // Wails应用中，使用pageX/pageY可能更准确
            // 对于fixed定位，需要使用clientX/Y（视口坐标）
            let x = e.clientX;
            let y = e.clientY;

            const menuWidth = 220;
            const menuHeight = 120;

            // 获取视口尺寸
            const viewportWidth = window.innerWidth;
            const viewportHeight = window.innerHeight;

            // 计算菜单位置：在鼠标位置稍微偏移
            x = x + 5;
            y = y + 5;

            // 边界检查：确保菜单不超出视口
            if (x + menuWidth > viewportWidth) {
                x = e.clientX - menuWidth - 5;
            }
            if (y + menuHeight > viewportHeight) {
                y = e.clientY - menuHeight - 5;
            }

            // 确保不会出现负坐标
            x = Math.max(10, x);
            y = Math.max(10, y);

            setContextMenu({
                x: x,
                y: y,
                selectedText: selectedText
            });
        } else if ((isUser && hasChart) || !isUser) {
            // Show export menu for:
            // 1. User messages with analysis results
            // 2. Assistant messages (for PDF export)
            setExportMenu({
                x: e.clientX,
                y: e.clientY
            });
        }
    };

    // Handle context menu action - request analysis
    const handleRequestAnalysis = () => {
        if (contextMenu && onActionClick) {
            onActionClick({
                id: 'selected_text_analysis',
                label: contextMenu.selectedText,
                value: contextMenu.selectedText
            });
        }
        setContextMenu(null);
    };

    // Handle export analysis action
    const handleExportAnalysis = async () => {
        if (!messageId) {
            EventsEmit('show-message-modal', {
                type: 'error',
                title: '导出失败',
                message: '无法获取消息ID'
            });
            setExportMenu(null);
            return;
        }

        try {
            console.log('[EXPORT] Starting export for message:', messageId);

            // Import the function dynamically to avoid build errors
            const { ExportAnalysisProcess } = await import('../../wailsjs/go/main/App');

            console.log('[EXPORT] Calling ExportAnalysisProcess...');
            await ExportAnalysisProcess(messageId);

            console.log('[EXPORT] Export completed successfully');

            // Show success message
            EventsEmit('show-message-modal', {
                type: 'info',
                title: '导出成功',
                message: '分析过程已导出'
            });
        } catch (err) {
            console.error('[EXPORT] Export analysis failed:', err);

            // Show error with details
            EventsEmit('show-message-modal', {
                type: 'error',
                title: '导出失败',
                message: err instanceof Error ? err.message : String(err)
            });
        } finally {
            setExportMenu(null);
        }
    };

    // Handle export PDF action
    const handleExportPDF = async () => {
        try {
            console.log('[EXPORT PDF] Starting PDF export for message:', messageId);

            // Import the function dynamically
            const { ExportMessageToPDF } = await import('../../wailsjs/go/main/App');

            await ExportMessageToPDF(content, messageId || '');

            // Show success toast
            import('../contexts/ToastContext').then(({ useToast }) => {
                // Can't use hook here, emit event instead
                EventsEmit('show-toast', {
                    type: 'success',
                    message: 'PDF导出成功',
                    title: '导出完成'
                });
            });
        } catch (err) {
            console.error('[EXPORT PDF] PDF export failed:', err);
            EventsEmit('show-toast', {
                type: 'error',
                message: err instanceof Error ? err.message : String(err),
                title: 'PDF导出失败'
            });
        } finally {
            setExportMenu(null);
        }
    };

    // Handle retry analysis action
    const handleRetryAnalysis = () => {
        setExportMenu(null);
        if (onRetryAnalysis) {
            onRetryAnalysis();
        }
    };

    // Close context menu when clicking outside
    useEffect(() => {
        const handleClickOutside = () => {
            if (contextMenu) {
                setContextMenu(null);
            }
            if (exportMenu) {
                setExportMenu(null);
            }
        };

        if (contextMenu || exportMenu) {
            document.addEventListener('click', handleClickOutside);
            return () => document.removeEventListener('click', handleClickOutside);
        }
    }, [contextMenu, exportMenu]);

    const renderButtonLabel = (label: string) => {
        // Remove all embedded data markers from display (but they're still in action.label for detection)
        const displayLabel = label
            .replace(/\s*\[INTENT_RETRY_BUTTON\]/g, '')
            .replace(/\s*\[INTENT_STICK_ORIGINAL\]/g, '')
            .replace(/\s*\[INTENT_RETRY_DATA:[^\]]*\]/g, '')  // 移除重试数据标记
            .replace(/\s*\[INTENT_STICK_DATA:[^\]]*\]/g, '')  // 移除坚持原始请求数据标记
            .replace(/\s*\[INTENT_SELECT:[^\]]*\]/g, '');     // 移除意图选择数据标记
        
        // Split by bold markers **text**
        const parts = displayLabel.split(/(\*\*.*?\*\*)/g);
        return parts.map((part, i) => {
            if (part.startsWith('**') && part.endsWith('**')) {
                return <strong key={i} className="font-black underline-offset-2">{part.slice(2, -2)}</strong>;
            }
            return part;
        });
    };

    // Handle action button clicks with deduplication
    const handleActionClick = (action: any) => {
        systemLog.info(`handleActionClick called: id=${action.id}, label=${action.label?.substring(0, 80)}, hasOnActionClick=${!!onActionClick}`);
        
        if (!onActionClick) {
            systemLog.warn('handleActionClick: onActionClick is not defined, returning');
            return;
        }

        // Create unique key for this action
        const actionKey = `${action.id}-${action.value || action.label}`;

        // Check if this action is already being processed
        if (pendingActionsRef.current.has(actionKey)) {
            systemLog.debug(`Ignoring duplicate action click: ${action.label?.substring(0, 50)}`);
            return;
        }

        // Mark action as pending
        pendingActionsRef.current.add(actionKey);
        systemLog.debug(`Action marked as pending: ${actionKey}`);

        try {
            // Call the parent handler
            systemLog.info(`Calling parent onActionClick handler`);
            onActionClick(action);
        } finally {
            // Clear the pending flag after a delay to prevent rapid clicking
            setTimeout(() => {
                pendingActionsRef.current.delete(actionKey);
            }, 1000); // 1 second delay
        }
    };

    // Remove technical code blocks that non-programmers don't need to see
    // Keep: json:dashboard (for dashboard display)
    // Hide: json:echarts, json:table, json:metrics (shown on dashboard instead), SQL queries, Python code
    // Hide: Raw base64 data URLs that aren't part of markdown images

    // First, preserve ALL markdown images by replacing them with placeholders
    // This regex matches markdown images with various URL formats:
    // - data:image/...;base64,... (inline base64)
    // - sandbox:/mnt/data/... (OpenAI code interpreter format)
    // - file://... (local file paths)
    // - files/... (relative paths)
    // - http(s)://... (remote URLs)
    // - any other path format
    const imageRegex = /!\[([^\]]*)\]\(([^)]+)\)/g;
    const preservedImages: string[] = [];
    let contentWithPlaceholders = content.replace(imageRegex, (match) => {
        preservedImages.push(match);
        return `__IMAGE_PLACEHOLDER_${preservedImages.length - 1}__`;
    });

    // Now clean the content (removing code blocks and standalone base64 data)
    const cleanedContent = contentWithPlaceholders
        .replace(/```[ \t]*json:dashboard[\s\S]*?```/g, '')
        .replace(/```[ \t]*json:echarts[\s\S]*?```/g, '') // 隐藏ECharts代码
        // json:table 保留，在 ReactMarkdown 中渲染为表格
        .replace(/```[ \t]*json:metrics[\s\S]*?```/g, '') // 隐藏Metrics代码，在仪表盘显示
        .replace(/```[ \t]*(sql|SQL)[\s\S]*?```/g, '')
        .replace(/```[ \t]*(python|Python|py)[\s\S]*?```/g, '')
        // Remove standalone base64 data URLs (now safe since markdown images are preserved)
        .replace(/data:image\/[^;]+;base64,[A-Za-z0-9+/=]+/g, '')
        // Remove intent suggestions marker (used for detection, not display)
        .replace(/\[INTENT_SUGGESTIONS\]\s*/g, '')
        // Remove all embedded data markers from display (but keep in action.label for detection)
        .replace(/\s*\[INTENT_RETRY_BUTTON\]/g, '')
        .replace(/\s*\[INTENT_STICK_ORIGINAL\]/g, '')
        .replace(/\s*\[INTENT_RETRY_DATA:[^\]]*\]/g, '')  // 移除重试数据标记
        .replace(/\s*\[INTENT_STICK_DATA:[^\]]*\]/g, '')  // 移除坚持原始请求数据标记
        .replace(/\s*\[INTENT_SELECT:[^\]]*\]/g, '')      // 移除意图选择数据标记
        // Restore preserved markdown images
        .replace(/__IMAGE_PLACEHOLDER_(\d+)__/g, (match, index) => preservedImages[parseInt(index)])
        .trim();

    return (
        <>
            <div className={`flex items-start gap-4 ${isUser ? 'flex-row-reverse' : 'flex-row'} animate-in fade-in slide-in-from-bottom-2 duration-300`}>
                <div className={`flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center shadow-sm ${isUser
                    ? 'bg-slate-200 text-slate-600'
                    : 'bg-gradient-to-br from-blue-500 to-indigo-600 text-white'
                    }`}>
                    {isUser ? <User className="w-5 h-5" /> : <Bot className="w-5 h-5" />}
                </div>

                <div
                    ref={messageContentRef}
                    className={`max-w-[85%] rounded-2xl px-5 py-3.5 shadow-sm ${isUser
                        ? `bg-blue-600 text-white rounded-tr-none ${isDisabled
                            ? 'opacity-50 cursor-not-allowed'
                            : onClick && hasChart
                                ? 'cursor-pointer hover:bg-blue-700 hover:shadow-lg hover:scale-[1.02] transition-all duration-200'
                                : ''
                        }`
                        : 'bg-white border border-slate-100 text-slate-700 rounded-tl-none ring-1 ring-slate-50'
                        }`}
                    onClick={onClick && isUser && !isDisabled ? onClick : undefined}
                    onContextMenu={handleContextMenu}
                    style={onClick && hasChart && isUser && !isDisabled ? { cursor: 'pointer' } : isDisabled ? { cursor: 'not-allowed' } : undefined}
                    title={
                        isDisabled
                            ? 'Analysis in progress or incomplete - cannot view yet'
                            : onClick && hasChart && isUser
                                ? 'Click to view analysis results on dashboard'
                                : undefined
                    }
                >
                    {isUser && hasChart && !isDisabled && (
                        <div className="mb-2 flex items-center gap-2 text-xs opacity-70">
                            <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                                <path d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zM3 10a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H4a1 1 0 01-1-1v-6zM14 9a1 1 0 00-1 1v6a1 1 0 001 1h2a1 1 0 001-1v-6a1 1 0 00-1-1h-2z" />
                            </svg>
                            <span>Has visualization - Click to view</span>
                        </div>
                    )}
                    <div
                        className={`prose prose-sm font-normal leading-relaxed ${isUser ? 'prose-invert text-white' : 'text-slate-700'} max-w-none`}
                        onContextMenu={handleContextMenu}
                    >
                        <ReactMarkdown
                            components={{
                                img(props) {
                                    return <MessageImage {...props} threadId={threadId} />;
                                },
                                code(props) {
                                    const { children, className, node, ...rest } = props;
                                    const match = /language-(\w+)/.exec(className || '');
                                    // Handle specific custom languages (formats: language-json:echarts or just json:echarts if passed directly)
                                    // ReactMarkdown usually prefixes with language-

                                    const isECharts = className?.includes('json:echarts');
                                    const isTable = className?.includes('json:table');

                                    // Hide SQL, Python, ECharts code blocks (non-programmers don't need to see them)
                                    // ECharts are shown on dashboard instead
                                    const isSql = className?.includes('sql') || className?.includes('SQL');
                                    const isPython = className?.includes('python') || className?.includes('Python') || className?.includes('py');

                                    if (isSql || isPython || isECharts) {
                                        // Return null to hide these technical code blocks
                                        return null;
                                    }

                                    // Render json:table as actual table
                                    if (isTable) {
                                        const codeContent = String(children).replace(/\n$/, '');
                                        try {
                                            const tableData = JSON.parse(codeContent);
                                            if (Array.isArray(tableData) && tableData.length > 0 && Array.isArray(tableData[0])) {
                                                const headers = tableData[0];
                                                const rows = tableData.slice(1);
                                                return (
                                                    <div className="overflow-x-auto my-3">
                                                        <table className="min-w-full border-collapse text-sm">
                                                            <thead>
                                                                <tr className="bg-blue-50">
                                                                    {headers.map((header: string, idx: number) => (
                                                                        <th 
                                                                            key={idx} 
                                                                            className="border border-slate-200 px-3 py-2 text-left font-semibold text-slate-700"
                                                                        >
                                                                            {String(header)}
                                                                        </th>
                                                                    ))}
                                                                </tr>
                                                            </thead>
                                                            <tbody>
                                                                {rows.map((row: any[], rowIdx: number) => (
                                                                    <tr key={rowIdx} className={rowIdx % 2 === 0 ? 'bg-white' : 'bg-slate-50'}>
                                                                        {row.map((cell: any, cellIdx: number) => (
                                                                            <td 
                                                                                key={cellIdx} 
                                                                                className="border border-slate-200 px-3 py-2 text-slate-600"
                                                                            >
                                                                                {String(cell)}
                                                                            </td>
                                                                        ))}
                                                                    </tr>
                                                                ))}
                                                            </tbody>
                                                        </table>
                                                    </div>
                                                );
                                            }
                                        } catch (e) {
                                            // JSON parse failed, show as code
                                            console.error('[MessageBubble] Failed to parse json:table:', e);
                                        }
                                        return null; // Hide if parse fails
                                    }

                                    return <code {...rest} className={className}>{children}</code>;
                                },
                                // Handle pre tag to properly render json:table
                                pre(props) {
                                    const { children, ...rest } = props;
                                    // Check if child is a code element with json:table
                                    const child = React.Children.toArray(children)[0];
                                    if (React.isValidElement(child)) {
                                        const childProps = child.props as { className?: string; children?: React.ReactNode };
                                        if (childProps.className?.includes('json:table')) {
                                            const codeContent = String(childProps.children || '').replace(/\n$/, '');
                                            try {
                                                const tableData = JSON.parse(codeContent);
                                                if (Array.isArray(tableData) && tableData.length > 0 && Array.isArray(tableData[0])) {
                                                    const headers = tableData[0];
                                                    const rows = tableData.slice(1);
                                                    return (
                                                        <div className="overflow-x-auto my-3">
                                                            <table className="min-w-full border-collapse text-sm">
                                                                <thead>
                                                                    <tr className="bg-blue-50">
                                                                        {headers.map((header: string, idx: number) => (
                                                                            <th 
                                                                                key={idx} 
                                                                                className="border border-slate-200 px-3 py-2 text-left font-semibold text-slate-700"
                                                                            >
                                                                                {String(header)}
                                                                            </th>
                                                                        ))}
                                                                    </tr>
                                                                </thead>
                                                                <tbody>
                                                                    {rows.map((row: any[], rowIdx: number) => (
                                                                        <tr key={rowIdx} className={rowIdx % 2 === 0 ? 'bg-white' : 'bg-slate-50'}>
                                                                            {row.map((cell: any, cellIdx: number) => (
                                                                                <td 
                                                                                    key={cellIdx} 
                                                                                    className="border border-slate-200 px-3 py-2 text-slate-600"
                                                                                >
                                                                                    {String(cell)}
                                                                                </td>
                                                                            ))}
                                                                        </tr>
                                                                    ))}
                                                                </tbody>
                                                            </table>
                                                        </div>
                                                    );
                                                }
                                            } catch (e) {
                                                // Parse failed
                                            }
                                            return null;
                                        }
                                    }
                                    return <pre {...rest}>{children}</pre>;
                                }
                            }}
                        >
                            {cleanedContent}
                        </ReactMarkdown>
                    </div>

                    {parsedPayload && parsedPayload.type === 'visual_insight' && (
                        <div className="mt-4 pt-4 border-t border-slate-100">
                            <MetricCard
                                title={parsedPayload.data.title}
                                value={parsedPayload.data.value}
                                change={parsedPayload.data.change}
                            />
                        </div>
                    )}

                    {parsedPayload && parsedPayload.type === 'echarts' && (
                        <div className="mt-4 pt-4 border-t border-slate-100">
                            <Chart options={parsedPayload.data} />
                        </div>
                    )}

                    {allActions.length > 0 && (
                        <div className="mt-4 flex flex-wrap gap-2" style={{ pointerEvents: 'auto' }}>
                            {allActions.map((action: any) => {
                                const isRetryBtn = action.label?.includes('[INTENT_RETRY_DATA:') || action.label?.includes('[INTENT_RETRY_BUTTON]');
                                const hasIntentData = action.label?.includes('[INTENT_RETRY_DATA:') || action.label?.includes('[INTENT_STICK_DATA:') || action.label?.includes('[INTENT_SELECT:');
                                return (
                                    <button
                                        key={action.id}
                                        type="button"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            e.preventDefault();
                                            const hasRetryData = action.label?.includes('[INTENT_RETRY_DATA:');
                                            const hasStickData = action.label?.includes('[INTENT_STICK_DATA:');
                                            const hasSelectData = action.label?.includes('[INTENT_SELECT:');
                                            systemLog.info(`[BUTTON_CLICK] id=${action.id}, labelLen=${action.label?.length}, isRetry=${isRetryBtn}, hasRetryData=${hasRetryData}, hasStickData=${hasStickData}, hasSelectData=${hasSelectData}`);
                                            if (hasRetryData || hasStickData) {
                                                systemLog.info(`[BUTTON_CLICK] Full label: ${action.label}`);
                                            }
                                            console.log('[BUTTON_CLICK]', action.id, action.label?.substring(0, 80), 'isRetry:', isRetryBtn, 'hasIntentData:', hasIntentData);
                                            handleActionClick(action);
                                        }}
                                        onMouseDown={(e) => {
                                            systemLog.debug(`[BUTTON_MOUSEDOWN] id=${action.id}, isRetry=${isRetryBtn}`);
                                        }}
                                        className={`px-4 py-1.5 rounded-full text-xs font-medium transition-all border ${isUser
                                            ? 'bg-white/20 border-white/30 text-white hover:bg-white/30'
                                            : isRetryBtn
                                                ? 'bg-orange-50 border-orange-200 text-orange-600 hover:bg-orange-100'
                                                : 'bg-blue-50 border-blue-100 text-blue-600 hover:bg-blue-100'
                                            } shadow-sm hover:shadow-md active:scale-95`}
                                        style={{ pointerEvents: 'auto', position: 'relative', zIndex: 100, cursor: 'pointer' }}
                                    >
                                        {renderButtonLabel(action.label)}
                                    </button>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>

            {/* Image Enlargement Modal */}
            {enlargedImage && (
                <div
                    className="fixed inset-0 z-[200] flex items-center justify-center bg-black/90 backdrop-blur-sm animate-in fade-in duration-200"
                    onClick={() => setEnlargedImage(null)}
                >
                    <div className="absolute top-4 right-4 z-[210]">
                        <button
                            onClick={() => setEnlargedImage(null)}
                            className="p-2 bg-white/10 hover:bg-red-500/80 rounded-full text-white transition-colors"
                        >
                            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                            </svg>
                        </button>
                    </div>
                    <div
                        className="relative z-[205] max-w-[95vw] max-h-[95vh] bg-white rounded-xl p-4 shadow-2xl"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <img
                            src={enlargedImage}
                            alt={t('enlarged_chart')}
                            className="max-w-full max-h-[90vh] object-contain rounded-lg"
                        />
                    </div>
                </div>
            )}

            {/* Text Selection Context Menu - 使用Portal渲染到body */}
            {contextMenu && ReactDOM.createPortal(
                <div
                    style={{
                        position: 'fixed',
                        left: `${contextMenu.x}px`,
                        top: `${contextMenu.y}px`,
                        zIndex: 99999999,
                        backgroundColor: 'white',
                        border: '2px solid #3b82f6',
                        borderRadius: '8px',
                        boxShadow: '0 4px 20px rgba(0,0,0,0.15)',
                        minWidth: '200px',
                        overflow: 'hidden'
                    }}
                    onClick={(e) => {
                        e.stopPropagation();
                        e.preventDefault();
                    }}
                    onMouseDown={(e) => e.stopPropagation()}
                >
                    <div style={{
                        padding: '10px 12px',
                        borderBottom: '1px solid #e2e8f0',
                        backgroundColor: '#f8fafc'
                    }}>
                        <div style={{ fontSize: '10px', color: '#64748b', fontWeight: 600, marginBottom: '4px' }}>
                            选中的文本
                        </div>
                        <div style={{
                            fontSize: '12px',
                            color: '#1e293b',
                            maxWidth: '280px',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                            fontWeight: 600
                        }}>
                            {contextMenu.selectedText}
                        </div>
                    </div>
                    <button
                        onClick={(e) => {
                            e.stopPropagation();
                            e.preventDefault();
                            handleRequestAnalysis();
                        }}
                        style={{
                            width: '100%',
                            padding: '10px 12px',
                            textAlign: 'left',
                            fontSize: '13px',
                            color: '#1e293b',
                            backgroundColor: 'white',
                            border: 'none',
                            cursor: 'pointer',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px',
                            fontWeight: 500,
                            transition: 'background-color 0.15s'
                        }}
                        onMouseEnter={(e) => {
                            e.currentTarget.style.backgroundColor = '#eff6ff';
                            e.currentTarget.style.color = '#2563eb';
                        }}
                        onMouseLeave={(e) => {
                            e.currentTarget.style.backgroundColor = 'white';
                            e.currentTarget.style.color = '#1e293b';
                        }}
                    >
                        <svg style={{ width: '14px', height: '14px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                        </svg>
                        <span>请求分析</span>
                    </button>
                </div>,
                document.body
            )}

            {/* Export Analysis Context Menu - Portal to body */}
            {exportMenu && ReactDOM.createPortal(
                <div
                    style={{
                        position: 'fixed',
                        left: `${exportMenu.x}px`,
                        top: `${exportMenu.y}px`,
                        zIndex: 99999999,
                        backgroundColor: 'white',
                        border: '1px solid #e2e8f0',
                        borderRadius: '8px',
                        boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
                        minWidth: '160px',
                        overflow: 'hidden'
                    }}
                    onClick={(e) => {
                        e.stopPropagation();
                        e.preventDefault();
                    }}
                    onMouseDown={(e) => e.stopPropagation()}
                >
                    <button
                        onClick={(e) => {
                            e.stopPropagation();
                            e.preventDefault();
                            if (isUser) {
                                handleExportAnalysis();
                            } else {
                                handleExportPDF();
                            }
                        }}
                        style={{
                            width: '100%',
                            padding: '10px 12px',
                            textAlign: 'left',
                            fontSize: '13px',
                            color: '#1e293b',
                            backgroundColor: 'white',
                            border: 'none',
                            cursor: 'pointer',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px'
                        }}
                        onMouseEnter={(e) => e.currentTarget.style.backgroundColor = '#f1f5f9'}
                        onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'white'}
                    >
                        <span style={{ fontSize: '16px' }}>📋</span>
                        <span style={{ fontWeight: 500 }}>{isUser ? '导出分析过程' : '导出为PDF'}</span>
                    </button>

                    {/* Retry Analysis Option - Only show for failed user messages */}
                    {isUser && isFailed && onRetryAnalysis && (
                        <button
                            onClick={(e) => {
                                e.stopPropagation();
                                e.preventDefault();
                                handleRetryAnalysis();
                            }}
                            style={{
                                width: '100%',
                                padding: '10px 12px',
                                textAlign: 'left',
                                fontSize: '13px',
                                color: '#dc2626',
                                backgroundColor: 'white',
                                border: 'none',
                                borderTop: '1px solid #e2e8f0',
                                cursor: 'pointer',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '8px'
                            }}
                            onMouseEnter={(e) => {
                                e.currentTarget.style.backgroundColor = '#fef2f2';
                                e.currentTarget.style.color = '#b91c1c';
                            }}
                            onMouseLeave={(e) => {
                                e.currentTarget.style.backgroundColor = 'white';
                                e.currentTarget.style.color = '#dc2626';
                            }}
                        >
                            <span style={{ fontSize: '16px' }}>🔄</span>
                            <span style={{ fontWeight: 500 }}>重新分析</span>
                        </button>
                    )}

                    {/* Timing Analysis Option - Only show if timing data exists */}
                    {timingData && (
                        <button
                            onClick={(e) => {
                                e.stopPropagation();
                                e.preventDefault();
                                setExportMenu(null);
                                setTimingModalOpen(true);
                            }}
                            style={{
                                width: '100%',
                                padding: '10px 12px',
                                textAlign: 'left',
                                fontSize: '13px',
                                color: '#1e293b',
                                backgroundColor: 'white',
                                border: 'none',
                                borderTop: '1px solid #e2e8f0',
                                cursor: 'pointer',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '8px'
                            }}
                            onMouseEnter={(e) => e.currentTarget.style.backgroundColor = '#f1f5f9'}
                            onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'white'}
                        >
                            <span style={{ fontSize: '16px' }}>⏱️</span>
                            <span style={{ fontWeight: 500 }}>耗时分析</span>
                        </button>
                    )}
                </div>,
                document.body
            )}

            {/* Timing Analysis Modal */}
            <TimingAnalysisModal
                isOpen={timingModalOpen}
                onClose={() => setTimingModalOpen(false)}
                timingData={timingData}
                messageContent={content}
            />
        </>
    );
};

export default MessageBubble;