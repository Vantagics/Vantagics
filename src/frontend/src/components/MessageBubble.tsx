import React, { useState, useRef, useEffect } from 'react';
import ReactDOM from 'react-dom';
import ReactMarkdown from 'react-markdown';
import MetricCard from './MetricCard';
import Chart from './Chart';
import DataTable from './DataTable';
import { User, Bot, ZoomIn } from 'lucide-react';
import { EventsEmit } from '../../wailsjs/runtime/runtime';

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
}

const MessageBubble: React.FC<MessageBubbleProps> = ({ role, content, payload, onActionClick, onClick, hasChart, messageId, userMessageId, dataSourceId }) => {
    const isUser = role === 'user';
    const [enlargedImage, setEnlargedImage] = useState<string | null>(null);
    const [clickedActions, setClickedActions] = useState<Set<string>>(new Set());
    const pendingActionsRef = useRef<Set<string>>(new Set()); // 跟踪正在处理的按钮点击
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; selectedText: string } | null>(null);
    const messageContentRef = useRef<HTMLDivElement>(null);

    let parsedPayload: any = null;

    if (payload) {
        try {
            parsedPayload = JSON.parse(payload);
        } catch (e) {
            console.error("Failed to parse payload", e);
        }
    }

    // Helper function to check if content is in a suggestion/recommendation context
    const isSuggestionContext = (content: string): boolean => {
        const suggestionKeywords = [
            '建议', '分析建议', '推荐', '可以分析', '以下分析', '推荐分析',
            'suggest', 'recommendation', 'analysis suggestion', 'can analyze',
            'following analysis', 'recommended'
        ];
        const lowerContent = content.toLowerCase();
        return suggestionKeywords.some(kw => lowerContent.includes(kw.toLowerCase()));
    };

    // Helper function to check if text is an actionable analysis item
    const isActionableItem = (text: string): boolean => {
        const actionKeywords = [
            '分析', '查看', '对比', '统计', '趋势', '预测', '细分', '探索',
            '检查', '评估', '比较', '研究', '洞察',
            'analysis', 'analyze', 'compare', 'trend', 'forecast', 'segment',
            'explore', 'examine', 'assess', 'study', 'insight', 'review'
        ];
        const lowerText = text.toLowerCase();
        return actionKeywords.some(kw => lowerText.includes(kw.toLowerCase()));
    };

    // Helper function to check if text matches explanation patterns
    const isExplanationPattern = (text: string): boolean => {
        const exclusionPatterns = [
            /^(首先|然后|接着|最后|其次)/i,
            /^(first|then|next|finally|second)/i,
            /^(步骤|step|阶段|phase)/i,
            /^(我|I)\s+(分析|analyzed|发现|found|查看|looked)/i,
            /^(这|this|that)\s+(是|was|will)/i
        ];
        return exclusionPatterns.some(pattern => pattern.test(text.trim()));
    };

    // Auto-extract numbered list actions with intelligent filtering
    const extractedActions: any[] = [];
    const extractedInsights: string[] = []; // 提取的洞察建议

    if (!isUser && isSuggestionContext(content)) {
        const lines = content.split('\n');
        for (const line of lines) {
            // Match lines starting with "1. ", "2. ", etc.
            const match = line.match(/^(\d+)\.\s+(.*)$/);
            if (match) {
                const rawLabel = match[2].trim();
                // Only extract if:
                // 1. Length is reasonable (avoid very long text)
                // 2. Contains actionable keywords
                // 3. Doesn't match explanation patterns
                if (rawLabel.length > 0 && rawLabel.length < 100 &&
                    isActionableItem(rawLabel) &&
                    !isExplanationPattern(rawLabel)) {
                    extractedActions.push({
                        id: `auto_${match[1]}`,
                        label: rawLabel,
                        // Value should be clean text for the LLM input (no markdown)
                        value: rawLabel.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '')
                    });

                    // 同时添加到洞察建议中
                    extractedInsights.push(rawLabel.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, ''));
                }
            }
        }
    }

    // 注意：关键指标现在通过后端自动提取，不再需要在这里解析json:metrics代码块

    const allActions = [
        ...(parsedPayload && parsedPayload.type === 'actions' ? parsedPayload.actions : []),
        ...extractedActions
    ];

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
                    userMessageId: userMessageId,
                    data_source_id: dataSourceId  // 添加数据源ID，这样点击时会创建新会话
                }))
            });
        }
    }, [extractedInsights.length, isUser, userMessageId, dataSourceId]); // 添加dataSourceId到依赖项

    // 注意：关键指标现在通过后端自动提取，不再需要手动发送到Dashboard

    const handleContextMenu = (e: React.MouseEvent) => {
        e.stopPropagation();
        e.preventDefault();

        const selection = window.getSelection();
        const selectedText = selection?.toString().trim();

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

    // Close context menu when clicking outside
    useEffect(() => {
        const handleClickOutside = () => {
            if (contextMenu) {
                setContextMenu(null);
            }
        };

        if (contextMenu) {
            document.addEventListener('click', handleClickOutside);
            return () => document.removeEventListener('click', handleClickOutside);
        }
    }, [contextMenu]);

    const renderButtonLabel = (label: string) => {
        // Split by bold markers **text**
        const parts = label.split(/(\*\*.*?\*\*)/g);
        return parts.map((part, i) => {
            if (part.startsWith('**') && part.endsWith('**')) {
                return <strong key={i} className="font-black underline-offset-2">{part.slice(2, -2)}</strong>;
            }
            return part;
        });
    };

    // Handle action button clicks with deduplication
    const handleActionClick = (action: any) => {
        if (!onActionClick) return;

        // Create unique key for this action
        const actionKey = `${action.id}-${action.value || action.label}`;

        // Check if this action is already being processed
        if (pendingActionsRef.current.has(actionKey)) {
            console.log('[MessageBubble] Ignoring duplicate action click:', action.label?.substring(0, 50));
            return;
        }

        // Mark action as pending
        pendingActionsRef.current.add(actionKey);

        try {
            // Call the parent handler
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

    // First, preserve markdown images by replacing them with placeholders
    const imageRegex = /!\[([^\]]*)\]\((data:image\/[^;]+;base64,[A-Za-z0-9+/=]+)\)/g;
    const preservedImages: string[] = [];
    let contentWithPlaceholders = content.replace(imageRegex, (match) => {
        preservedImages.push(match);
        return `__IMAGE_PLACEHOLDER_${preservedImages.length - 1}__`;
    });

    // Now clean the content (removing code blocks and standalone base64 data)
    const cleanedContent = contentWithPlaceholders
        .replace(/```[ \t]*json:dashboard[\s\S]*?```/g, '')
        .replace(/```[ \t]*json:echarts[\s\S]*?```/g, '') // 隐藏ECharts代码
        .replace(/```[ \t]*json:table[\s\S]*?```/g, '') // 隐藏Table代码，在仪表盘显示
        .replace(/```[ \t]*json:metrics[\s\S]*?```/g, '') // 隐藏Metrics代码，在仪表盘显示
        .replace(/```[ \t]*(sql|SQL)[\s\S]*?```/g, '')
        .replace(/```[ \t]*(python|Python|py)[\s\S]*?```/g, '')
        // Remove standalone base64 data URLs (now safe since markdown images are preserved)
        .replace(/data:image\/[^;]+;base64,[A-Za-z0-9+/=]+/g, '')
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
                        ? `bg-blue-600 text-white rounded-tr-none ${onClick && hasChart ? 'cursor-pointer hover:bg-blue-700 hover:shadow-lg hover:scale-[1.02] transition-all duration-200' : ''}`
                        : 'bg-white border border-slate-100 text-slate-700 rounded-tl-none ring-1 ring-slate-50'
                        }`}
                    onClick={onClick && isUser ? onClick : undefined}
                    onContextMenu={handleContextMenu}
                    style={onClick && hasChart && isUser ? { cursor: 'pointer' } : undefined}
                    title={onClick && hasChart && isUser ? 'Click to view analysis results on dashboard' : undefined}
                >
                    {isUser && hasChart && (
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
                                    const { src, alt, ...rest } = props;
                                    return (
                                        <div className="relative group my-4">
                                            <img
                                                src={src}
                                                alt={alt || 'Chart'}
                                                {...rest}
                                                className="rounded-lg shadow-md max-w-full cursor-pointer hover:shadow-xl transition-shadow"
                                                onClick={() => setEnlargedImage(src || null)}
                                            />
                                            <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                                <div className="bg-black/60 text-white px-2 py-1 rounded text-xs flex items-center gap-1">
                                                    <ZoomIn className="w-3 h-3" />
                                                    Click to enlarge
                                                </div>
                                            </div>
                                        </div>
                                    );
                                },
                                code(props) {
                                    const { children, className, node, ...rest } = props;
                                    const match = /language-(\w+)/.exec(className || '');
                                    // Handle specific custom languages (formats: language-json:echarts or just json:echarts if passed directly)
                                    // ReactMarkdown usually prefixes with language-

                                    const isECharts = className?.includes('json:echarts');
                                    const isTable = className?.includes('json:table');

                                    // Hide SQL, Python, ECharts, and Table code blocks (non-programmers don't need to see them)
                                    // ECharts and Tables are shown on dashboard instead
                                    const isSql = className?.includes('sql') || className?.includes('SQL');
                                    const isPython = className?.includes('python') || className?.includes('Python') || className?.includes('py');

                                    if (isSql || isPython || isECharts || isTable) {
                                        // Return null to hide these technical code blocks
                                        return null;
                                    }

                                    return <code {...rest} className={className}>{children}</code>;
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
                        <div className="mt-4 flex flex-wrap gap-2">
                            {allActions.map((action: any) => (
                                <button
                                    key={action.id}
                                    onClick={() => handleActionClick(action)}
                                    className={`px-4 py-1.5 rounded-full text-xs font-medium transition-all border ${isUser
                                        ? 'bg-white/20 border-white/30 text-white hover:bg-white/30'
                                        : 'bg-blue-50 border-blue-100 text-blue-600 hover:bg-blue-100'
                                        } shadow-sm hover:shadow-md active:scale-95`}
                                >
                                    {renderButtonLabel(action.label)}
                                </button>
                            ))}
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
                            alt="Enlarged chart"
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
        </>
    );
};

export default MessageBubble;