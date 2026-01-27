import React, { useState, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import { TrendingUp, UserCheck, AlertCircle, Star, Info } from 'lucide-react';
import { GetSessionFileAsBase64 } from '../../wailsjs/go/main/App';

interface SmartInsightProps {
    text: string;
    icon: string;
    onClick?: () => void;
    threadId?: string;  // 用于加载 sandbox: 路径的图片
}

const iconMap: Record<string, React.ReactNode> = {
    'trending-up': <TrendingUp className="w-5 h-5 text-blue-500" />,
    'user-check': <UserCheck className="w-5 h-5 text-green-500" />,
    'alert-circle': <AlertCircle className="w-5 h-5 text-amber-500" />,
    'star': <Star className="w-5 h-5 text-purple-500" />,
    'info': <Info className="w-5 h-5 text-slate-500" />,
};

// 解析 JSON 表格数据
const parseJsonTable = (jsonStr: string): string[][] | null => {
    try {
        const data = JSON.parse(jsonStr);
        if (Array.isArray(data) && data.length > 0 && Array.isArray(data[0])) {
            return data.map(row => row.map((cell: unknown) => String(cell)));
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
                    <tr className="bg-blue-50">
                        {headers.map((header, idx) => (
                            <th 
                                key={idx} 
                                className="border border-slate-200 px-3 py-2 text-left font-semibold text-slate-700"
                            >
                                {header}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {rows.map((row, rowIdx) => (
                        <tr key={rowIdx} className={rowIdx % 2 === 0 ? 'bg-white' : 'bg-slate-50'}>
                            {row.map((cell, cellIdx) => (
                                <td 
                                    key={cellIdx} 
                                    className="border border-slate-200 px-3 py-2 text-slate-600"
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
            <div className="bg-slate-100 rounded p-4 my-2 flex items-center justify-center">
                <div className="animate-pulse text-slate-400 text-sm">加载图片中...</div>
            </div>
        );
    }

    if (error || !imageSrc) {
        return (
            <div className="bg-slate-100 border border-slate-200 rounded p-4 my-2 text-center text-slate-500 text-sm">
                <Info className="w-6 h-6 mx-auto mb-2 text-slate-400" />
                <p>图表: {alt || '分析图表'}</p>
                <p className="text-xs text-slate-400 mt-1">无法加载图片</p>
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
            className={`bg-white rounded-xl shadow-sm p-4 flex items-start gap-4 border-l-4 border-blue-500 hover:shadow-md transition-shadow duration-200 hover:bg-slate-50/50 ${onClick ? 'cursor-pointer active:scale-[0.99] transition-transform' : ''}`}
        >
            <div className="insight-icon bg-gradient-to-br from-slate-50 to-slate-100 p-2 rounded-lg shrink-0 shadow-inner">
                {IconComponent}
            </div>
            <div className="text-slate-700 text-sm leading-relaxed pt-1 prose prose-sm max-w-none">
                <ReactMarkdown
                    components={{
                        // 自定义markdown组件样式
                        p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                        strong: ({ children }) => <strong className="font-semibold text-slate-800">{children}</strong>,
                        em: ({ children }) => <em className="italic">{children}</em>,
                        ul: ({ children }) => <ul className="list-disc list-inside mb-2">{children}</ul>,
                        ol: ({ children }) => <ol className="list-decimal list-inside mb-2">{children}</ol>,
                        li: ({ children }) => <li className="mb-1">{children}</li>,
                        h1: ({ children }) => <h1 className="text-lg font-bold text-slate-800 mb-2">{children}</h1>,
                        h2: ({ children }) => <h2 className="text-base font-bold text-slate-800 mb-2">{children}</h2>,
                        h3: ({ children }) => <h3 className="text-sm font-bold text-slate-800 mb-2">{children}</h3>,
                        h4: ({ children }) => <h4 className="text-sm font-semibold text-slate-700 mb-1">{children}</h4>,
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
                                <code className="bg-slate-100 px-1 py-0.5 rounded text-xs font-mono" {...props}>
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
                            return <pre className="bg-slate-100 p-2 rounded text-xs overflow-x-auto my-2" {...props}>{children}</pre>;
                        },
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
