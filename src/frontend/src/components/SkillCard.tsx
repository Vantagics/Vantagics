import React, { useState } from 'react';
import { BookOpen, ChevronDown, ChevronUp } from 'lucide-react';
import ReactMarkdown from 'react-markdown';

interface SkillCardProps {
    skill: {
        name: string;
        description: string;
        content: string;
        path: string;
        installed_at: string;
    };
}

const SkillCard: React.FC<SkillCardProps> = ({ skill }) => {
    const [isExpanded, setIsExpanded] = useState(false);

    return (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm hover:shadow-md transition-shadow">
            {/* Card Header */}
            <div className="p-4 border-b border-gray-200 dark:border-gray-700">
                <div className="flex items-start gap-3">
                    <div className="flex-shrink-0 mt-1">
                        <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center">
                            <BookOpen className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                        </div>
                    </div>
                    <div className="flex-1 min-w-0">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white truncate">
                            {skill.name}
                        </h3>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1 line-clamp-2">
                            {skill.description}
                        </p>
                    </div>
                </div>
            </div>

            {/* Card Body */}
            <div className="p-4">
                {/* Expandable Content */}
                <button
                    onClick={() => setIsExpanded(!isExpanded)}
                    className="w-full flex items-center justify-between text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
                >
                    <span>{isExpanded ? '收起详情' : '查看详情'}</span>
                    {isExpanded ? (
                        <ChevronUp className="w-4 h-4" />
                    ) : (
                        <ChevronDown className="w-4 h-4" />
                    )}
                </button>

                {isExpanded && (
                    <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                        <div className="prose prose-sm dark:prose-invert max-w-none">
                            <ReactMarkdown>{skill.content}</ReactMarkdown>
                        </div>
                    </div>
                )}
            </div>

            {/* Card Footer */}
            <div className="px-4 py-3 bg-gray-50 dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 rounded-b-lg">
                <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                    <span>安装时间: {new Date(skill.installed_at).toLocaleDateString('zh-CN')}</span>
                    <span className="px-2 py-1 bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 rounded">
                        已安装
                    </span>
                </div>
            </div>
        </div>
    );
};

export default SkillCard;
