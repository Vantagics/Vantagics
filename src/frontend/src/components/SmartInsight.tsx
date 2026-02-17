import React, { useState, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { TrendingUp, UserCheck, AlertCircle, Star, Info, Lightbulb, BarChart3, Search, Zap, Target, Layers, PieChart, ArrowUpRight } from 'lucide-react';
import { GetSessionFileAsBase64 } from '../../wailsjs/go/main/App';

interface SmartInsightProps {
    text: string;
    icon: string;
    onClick?: () => void;
    threadId?: string;  // 用于加载 sandbox: 路径的图片
}

const iconMap: Record<string, React.ReactNode> = {
    'trending-up': <TrendingUp className="w-4 h-4 text-blue-600 dark:text-blue-400" />,
    'user-check': <UserCheck className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />,
    'alert-circle': <AlertCircle className="w-4 h-4 text-amber-600 dark:text-amber-400" />,
    'star': <Star className="w-4 h-4 text-purple-600 dark:text-purple-400" />,
    'info': <Info className="w-4 h-4 text-blue-600 dark:text-blue-400" />,
    'lightbulb': <Lightbulb className="w-4 h-4 text-amber-500 dark:text-amber-400" />,
    'bar-chart': <BarChart3 className="w-4 h-4 text-indigo-600 dark:text-indigo-400" />,
    'search': <Search className="w-4 h-4 text-slate-600 dark:text-slate-400" />,
    'zap': <Zap className="w-4 h-4 text-yellow-500 dark:text-yellow-400" />,
    'target': <Target className="w-4 h-4 text-rose-600 dark:text-rose-400" />,
    'layers': <Layers className="w-4 h-4 text-teal-600 dark:text-teal-400" />,
    'pie-chart': <PieChart className="w-4 h-4 text-violet-600 dark:text-violet-400" />,
    'arrow-up-right': <ArrowUpRight className="w-4 h-4 text-green-600 dark:text-green-400" />,
};

// 解析 JSON 表格数据 - 支持三种格式:
// 1. 2D array: [["col1","col2"], [val1, val2], ...]
// 2. Object with columns/data: { "columns": [...], "data": [[...], ...] }
// 3. Object array: [{"col1": val1}, ...]
const parseJsonTable = (jsonStr: string): string[][] | null => {
    try {
        const data = JSON.parse(jsonStr);
        // Format 1: 2D array
        if (Array.isArray(data) && data.length > 0 && Array.isArray(data[0])) {
            return data.map(row => row.map((cell: unknown) => String(cell)));
        }
        // Format 2: { columns: [...], data: [[...], ...] }
        if (data && !Array.isArray(data) && Array.isArray(data.columns) && Array.isArray(data.data)) {
            const headers = data.columns.map(String);
            const rows = data.data.map((row: unknown[]) => row.map((cell: unknown) => String(cell)));
            return [headers, ...rows];
        }
        // Format 3: Object array
        if (Array.isArray(data) && data.length > 0 && typeof data[0] === 'object' && data[0] !== null) {
            const headers = Object.keys(data[0]);
            const rows = data.map((obj: Record<string, unknown>) => headers.map(h => String(obj[h])));
            return [headers, ...rows];
        }
    } catch {
        // 解析失败，返回 null
    }
    return null;
};

// 渲染表格组件
const JsonTableRenderer: React.FC<{ data: string[][] }> = ({ data }) => {
    if (data.length === 0) return null;
    
    const headers = data[0];
    const rows = data.slice(1);
    
    return (
        <div className="overflow-x-auto my-3">
            <table className="min-w-full border-collapse text-sm">
                <thead>
                    <tr className="bg-blue-50 dark:bg-[#1a2332]">
                        {headers.map((header, idx) => (
                            <th 
                                key={idx} 
                                className="border border-slate-200 dark:border-[#3c3c3c] px-3 py-2 text-left font-semibold text-slate-700 dark:text-[#d4d4d4]"
                            >
                                {header}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {rows.map((row, rowIdx) => (
                        <tr key={rowIdx} className={rowIdx % 2 === 0 ? 'bg-white dark:bg-[#1e1e1e]' : 'bg-slate-50 dark:bg-[#252526]'}>
                            {row.map((cell, cellIdx) => (
                                <td 
                                    key={cellIdx} 
                                    className="border border-slate-200 dark:border-[#3c3c3c] px-3 py-2 text-slate-600 dark:text-[#d4d4d4]"
                                >
                                    {cell}
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};

// 图片组件，支持加载 sandbox: 路径的图片
const InsightImage: React.FC<{ src?: string; alt?: string; threadId?: string }> = ({ src, alt, threadId }) => {
    const [imageSrc, setImageSrc] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    useEffect(() => {
        if (!src) {
            setLoading(false);
            return;
        }

        // 如果是 data URL，直接使用
        if (src.startsWith('data:')) {
            setImageSrc(src);
            setLoading(false);
            return;
        }

        // 处理 sandbox: 路径
        if (src.startsWith('sandbox:')) {
            const loadImage = async () => {
                try {
                    // 从 sandbox: 路径提取文件名
                    const pathParts = src.replace('sandbox:', '').split('/');
                    const filename = pathParts[pathParts.length - 1];
                    
                    if (!threadId) {
                        console.error('[InsightImage] No threadId available for sandbox path:', src);
                        setError(true);
                        setLoading(false);
                        return;
                    }

                    console.log('[InsightImage] Loading image from sandbox path:', { threadId, filename, src });

                    const base64Data = await GetSessionFileAsBase64(threadId, filename);
                    setImageSrc(base64Data);
                    setLoading(false);
                } catch (err) {
                    console.error('[InsightImage] Failed to load image from sandbox path:', err);
                    setError(true);
                    setLoading(false);
                }
            };

            loadImage();
        } else if (src.startsWith('file://')) {
            // 处理 file:// 路径
            const loadImage = async () => {
                try {
                    const match = src.match(/files[\/\\]([^\/\\]+)$/);
                    if (!match) {
                        setError(true);
                        setLoading(false);
                        return;
                    }

                    const filename = match[1];
                    let extractedThreadId = threadId;
                    const threadMatch = src.match(/sessions[\/\\]([^\/\\]+)[\/\\]files/);
                    if (threadMatch) {
                        extractedThreadId = threadMatch[1];
                    }

                    if (!extractedThreadId) {
                        setError(true);
                        setLoading(false);
                        return;
                    }

                    const base64Data = await GetSessionFileAsBase64(extractedThreadId, filename);
                    setImageSrc(base64Data);
                    setLoading(false);
                } catch (err) {
                    console.error('[InsightImage] Failed to load image:', err);
                    setError(true);
                    setLoading(false);
                }
            };

            loadImage();
        } else if (src.startsWith('files/') || src.match(/^[^:\/]+\.(png|jpg|jpeg|gif|svg)$/i)) {
            // 处理相对路径
            const loadImage = async () => {
                try {
                    const filename = src.replace(/^files[\/\\]/, '');
                    
                    if (!threadId) {
                        setError(true);
                        setLoading(false);
                        return;
                    }

                    const base64Data = await GetSessionFileAsBase64(threadId, filename);
                    setImageSrc(base64Data);
                    setLoading(false);
                } catch (err) {
                    console.error('[InsightImage] Failed to load image:', err);
                    setError(true);
                    setLoading(false);
                }
            };

            loadImage();
        } else {
            // 其他 URL 直接使用
            setImageSrc(src);
            setLoading(false);
        }
    }, [src, threadId]);

    if (loading) {
        return (
            <div className="bg-slate-100 dark:bg-[#252526] rounded p-4 my-2 flex items-center justify-center">
                <div className="animate-pulse text-slate-400 dark:text-[#808080] text-sm">加载图片中...</div>
            </div>
        );
    }

    if (error || !imageSrc) {
        return (
            <div className="bg-slate-100 dark:bg-[#252526] border border-slate-200 dark:border-[#3c3c3c] rounded p-4 my-2 text-center text-slate-500 dark:text-[#808080] text-sm">
                <Info className="w-6 h-6 mx-auto mb-2 text-slate-400 dark:text-[#808080]" />
                <p>图表: {alt || '分析图表'}</p>
                <p className="text-xs text-slate-400 dark:text-[#808080] mt-1">无法加载图片</p>
            </div>
        );
    }

    return (
        <img 
            src={imageSrc} 
            alt={alt || '分析图表'} 
            className="max-w-full h-auto rounded my-2 shadow-sm" 
        />
    );
};

const SmartInsight: React.FC<SmartInsightProps> = ({ text, icon, onClick, threadId }) => {
    const IconComponent = iconMap[icon] || iconMap['info'];

    const handleClick = (e: React.MouseEvent) => {
        e.preventDefault();
        e.stopPropagation(); // 阻止事件冒泡，防止触发Dashboard的点击处理
        if (onClick) {
            onClick();
        }
    };

    return (
        <div 
            onClick={handleClick}
            className={`bg-white dark:bg-[#252526] rounded-xl shadow-sm p-4 flex items-start gap-4 border-l-4 border-blue-500 dark:border-[#007acc] hover:shadow-md transition-shadow duration-200 hover:bg-slate-50/50 dark:hover:bg-[#2d2d30] ${onClick ? 'cursor-pointer active:scale-[0.99] transition-transform' : ''}`}
        >
            <div className="insight-icon bg-blue-50 dark:bg-[#1a2332] p-2 rounded-md shrink-0 border border-blue-100 dark:border-[#2a3a4a]">
                {IconComponent}
            </div>
            <div className="text-slate-700 dark:text-[#d4d4d4] text-sm leading-relaxed pt-1 prose prose-sm max-w-none">
                <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    components={{
                        // 自定义markdown组件样式
                        p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                        strong: ({ children }) => <strong className="font-semibold text-slate-800 dark:text-[#e0e0e0]">{children}</strong>,
                        em: ({ children }) => <em className="italic">{children}</em>,
                        ul: ({ children }) => <ul className="list-disc list-inside mb-2">{children}</ul>,
                        ol: ({ children }) => <ol className="list-decimal list-inside mb-2">{children}</ol>,
                        li: ({ children }) => <li className="mb-1">{children}</li>,
                        h1: ({ children }) => <h1 className="text-lg font-bold text-slate-800 dark:text-[#e0e0e0] mb-2">{children}</h1>,
                        h2: ({ children }) => <h2 className="text-base font-bold text-slate-800 dark:text-[#e0e0e0] mb-2">{children}</h2>,
                        h3: ({ children }) => <h3 className="text-sm font-bold text-slate-800 dark:text-[#e0e0e0] mb-2">{children}</h3>,
                        h4: ({ children }) => <h4 className="text-sm font-semibold text-slate-700 dark:text-[#d4d4d4] mb-1">{children}</h4>,
                        // 处理代码块，特别是 json:table
                        code: ({ className, children, ...props }) => {
                            const match = /language-(\w+)/.exec(className || '');
                            const language = match ? match[1] : '';
                            const codeContent = String(children).replace(/\n$/, '');
                            
                            // 检查是否是 json:table 格式
                            if (language === 'json:table' || className?.includes('json:table')) {
                                const tableData = parseJsonTable(codeContent);
                                if (tableData) {
                                    return <JsonTableRenderer data={tableData} />;
                                }
                            }
                            
                            // 普通代码块
                            return (
                                <code className="bg-slate-100 dark:bg-[#1e1e1e] px-1 py-0.5 rounded text-xs font-mono" {...props}>
                                    {children}
                                </code>
                            );
                        },
                        // 处理 pre 标签（代码块容器）
                        pre: ({ children, ...props }) => {
                            // 检查子元素是否是已经渲染的表格
                            const child = React.Children.toArray(children)[0];
                            if (React.isValidElement(child)) {
                                const childProps = child.props as { className?: string; children?: React.ReactNode };
                                // 如果是 json:table，直接返回子元素（表格）
                                if (childProps.className?.includes('json:table')) {
                                    const codeContent = String(childProps.children || '').replace(/\n$/, '');
                                    const tableData = parseJsonTable(codeContent);
                                    if (tableData) {
                                        return <JsonTableRenderer data={tableData} />;
                                    }
                                }
                            }
                            return <pre className="bg-slate-100 dark:bg-[#1e1e1e] p-2 rounded text-xs overflow-x-auto my-2" {...props}>{children}</pre>;
                        },
                        // 渲染标准 Markdown 表格为真正的表格
                        table: ({ children }) => (
                            <div className="overflow-x-auto my-3">
                                <table className="min-w-full border-collapse text-sm">{children}</table>
                            </div>
                        ),
                        thead: ({ children }) => <thead>{children}</thead>,
                        tbody: ({ children }) => <tbody>{children}</tbody>,
                        tr: ({ children, ...props }) => {
                            // 判断是否在 tbody 中（通过检查是否有 td 子元素）
                            const node = (props as any).node;
                            const isBody = node?.children?.some((c: any) => c.tagName === 'td');
                            const rowIndex = node?.position?.start?.line || 0;
                            return (
                                <tr className={isBody ? (rowIndex % 2 === 0 ? 'bg-white dark:bg-[#1e1e1e]' : 'bg-slate-50 dark:bg-[#252526]') : 'bg-blue-50 dark:bg-[#1a2332]'}>
                                    {children}
                                </tr>
                            );
                        },
                        th: ({ children }) => (
                            <th className="border border-slate-200 dark:border-[#3c3c3c] px-3 py-2 text-left font-semibold text-slate-700 dark:text-[#d4d4d4]">{children}</th>
                        ),
                        td: ({ children }) => (
                            <td className="border border-slate-200 dark:border-[#3c3c3c] px-3 py-2 text-slate-600 dark:text-[#d4d4d4]">{children}</td>
                        ),
                        // 处理图片
                        img: ({ src, alt }) => {
                            return <InsightImage src={src} alt={alt} threadId={threadId} />;
                        },
                    }}
                >
                    {text}
                </ReactMarkdown>
            </div>
        </div>
    );
};

export default SmartInsight;
