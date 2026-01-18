import React, { useState, useEffect } from 'react';
import { X, BookOpen, Power, PowerOff, RefreshCw, Search, Filter, Tag, Zap, Check, Settings } from 'lucide-react';
import { GetSkills, EnableSkill, DisableSkill, ReloadSkills } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import { useLanguage } from '../i18n';

interface SkillInfo {
    id: string;
    name: string;
    description: string;
    version: string;
    author: string;
    category: string;
    keywords: string[];
    required_columns: string[];
    tools: string[];
    enabled: boolean;
    icon: string;
    tags: string[];
}

interface SkillsPageProps {
    isOpen: boolean;
    onClose: () => void;
    onSelectSkill?: (skillId: string) => void;
}

const SkillsPage: React.FC<SkillsPageProps> = ({ isOpen, onClose, onSelectSkill }) => {
    const { t } = useLanguage();
    const [skills, setSkills] = useState<SkillInfo[]>([]);
    const [filteredSkills, setFilteredSkills] = useState<SkillInfo[]>([]);
    const [selectedCategory, setSelectedCategory] = useState<string>('all');
    const [searchQuery, setSearchQuery] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [selectedSkill, setSelectedSkill] = useState<SkillInfo | null>(null);

    useEffect(() => {
        if (isOpen) {
            loadSkills();
        }
    }, [isOpen]);

    useEffect(() => {
        filterSkills();
    }, [skills, selectedCategory, searchQuery]);

    const loadSkills = async () => {
        setIsLoading(true);
        try {
            const loadedSkills = await GetSkills() as unknown as SkillInfo[];
            setSkills(loadedSkills || []);
        } catch (error) {
            console.error('Failed to load skills:', error);
        } finally {
            setIsLoading(false);
        }
    };

    const filterSkills = () => {
        let filtered = [...(skills || [])];

        // Filter by category
        if (selectedCategory !== 'all') {
            filtered = filtered.filter(s => s.category === selectedCategory);
        }

        // Filter by search query
        if (searchQuery.trim()) {
            const query = searchQuery.toLowerCase();
            filtered = filtered.filter(s =>
                s.name.toLowerCase().includes(query) ||
                s.description.toLowerCase().includes(query) ||
                s.keywords.some(k => k.toLowerCase().includes(query)) ||
                s.tags.some(t => t.toLowerCase().includes(query))
            );
        }

        setFilteredSkills(filtered);
    };

    const handleToggleSkill = async (skillId: string, currentlyEnabled: boolean) => {
        try {
            if (currentlyEnabled) {
                await DisableSkill(skillId);
            } else {
                await EnableSkill(skillId);
            }
            await loadSkills();
        } catch (error) {
            console.error('Failed to toggle skill:', error);
        }
    };

    const handleReloadSkills = async () => {
        setIsLoading(true);
        try {
            await ReloadSkills();
            await loadSkills();
        } catch (error) {
            console.error('Failed to reload skills:', error);
        } finally {
            setIsLoading(false);
        }
    };

    const categories = ['all', ...Array.from(new Set((skills || []).map(s => s.category)))];
    const enabledCount = (skills || []).filter(s => s.enabled).length;

    const getCategoryIcon = (category: string) => {
        const icons: { [key: string]: string } = {
            user_analytics: '👥',
            sales_analytics: '💰',
            marketing: '📢',
            product: '📦',
            custom: '🔧',
            all: '📚'
        };
        return icons[category] || '📊';
    };

    const getIconComponent = (iconName: string) => {
        const icons: { [key: string]: React.ReactNode } = {
            users: <Tag className="w-5 h-5" />,
            filter: <Filter className="w-5 h-5" />,
            zap: <Zap className="w-5 h-5" />,
            chart: <BookOpen className="w-5 h-5" />,
        };
        return icons[iconName] || <BookOpen className="w-5 h-5" />;
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div className="bg-white rounded-2xl shadow-2xl w-full max-w-6xl h-[85vh] flex flex-col overflow-hidden">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-200 bg-gradient-to-r from-blue-50 to-indigo-50">
                    <div className="flex items-center gap-3">
                        <div className="bg-gradient-to-br from-blue-500 to-indigo-600 p-3 rounded-xl">
                            <Zap className="w-6 h-6 text-white" />
                        </div>
                        <div>
                            <h2 className="text-2xl font-bold text-slate-900">Skills 插件库</h2>
                            <p className="text-sm text-slate-600 mt-0.5">
                                {skills.length} 个插件 · {enabledCount} 个已启用
                            </p>
                        </div>
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={handleReloadSkills}
                            disabled={isLoading}
                            className="p-2 hover:bg-white/80 rounded-lg transition-colors text-slate-600 hover:text-blue-600"
                            title="重新加载Skills"
                        >
                            <RefreshCw className={`w-5 h-5 ${isLoading ? 'animate-spin' : ''}`} />
                        </button>
                        <button
                            onClick={onClose}
                            className="p-2 hover:bg-white/80 rounded-lg transition-colors text-slate-400 hover:text-slate-600"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>
                </div>

                {/* Toolbar */}
                <div className="p-4 border-b border-slate-200 bg-slate-50/50">
                    <div className="flex flex-col sm:flex-row gap-3">
                        {/* Search */}
                        <div className="flex-1 relative">
                            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                            <input
                                type="text"
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                placeholder="搜索 Skills (名称、关键词、标签...)"
                                className="w-full pl-10 pr-4 py-2 bg-white border border-slate-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                            />
                        </div>

                        {/* Category Filter */}
                        <div className="flex gap-2 overflow-x-auto scrollbar-hide">
                            {categories.map(cat => (
                                <button
                                    key={cat}
                                    onClick={() => setSelectedCategory(cat)}
                                    className={`px-3 py-2 rounded-lg text-sm font-medium transition-all whitespace-nowrap flex items-center gap-1.5 ${
                                        selectedCategory === cat
                                            ? 'bg-blue-600 text-white shadow-sm'
                                            : 'bg-white text-slate-600 hover:bg-slate-100 border border-slate-200'
                                    }`}
                                >
                                    <span>{getCategoryIcon(cat)}</span>
                                    <span className="capitalize">{cat === 'all' ? '全部' : cat}</span>
                                </button>
                            ))}
                        </div>
                    </div>
                </div>

                {/* Skills Grid */}
                <div className="flex-1 overflow-y-auto p-6">
                    {isLoading ? (
                        <div className="flex items-center justify-center h-64">
                            <RefreshCw className="w-8 h-8 animate-spin text-blue-600" />
                        </div>
                    ) : filteredSkills.length === 0 ? (
                        <div className="flex flex-col items-center justify-center h-64 text-slate-400">
                            <Zap className="w-16 h-16 mb-4 opacity-20" />
                            <p className="text-lg font-medium">未找到匹配的 Skills</p>
                            <p className="text-sm mt-1">尝试调整搜索条件或分类筛选</p>
                        </div>
                    ) : (
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                            {filteredSkills.map(skill => (
                                <div
                                    key={skill.id}
                                    className={`group bg-white border rounded-xl p-5 hover:shadow-lg transition-all cursor-pointer ${
                                        skill.enabled
                                            ? 'border-blue-200 hover:border-blue-300'
                                            : 'border-slate-200 hover:border-slate-300 opacity-60'
                                    }`}
                                    onClick={() => setSelectedSkill(skill)}
                                >
                                    {/* Header */}
                                    <div className="flex items-start justify-between mb-3">
                                        <div className="flex items-center gap-3">
                                            <div className={`p-2 rounded-lg ${
                                                skill.enabled ? 'bg-blue-100 text-blue-600' : 'bg-slate-100 text-slate-400'
                                            }`}>
                                                {getIconComponent(skill.icon)}
                                            </div>
                                            <div className="flex-1">
                                                <h3 className="font-bold text-slate-900 text-sm leading-tight">
                                                    {skill.name}
                                                </h3>
                                                <p className="text-xs text-slate-500 mt-0.5">v{skill.version}</p>
                                            </div>
                                        </div>
                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                handleToggleSkill(skill.id, skill.enabled);
                                            }}
                                            className={`p-1.5 rounded-lg transition-all ${
                                                skill.enabled
                                                    ? 'bg-green-100 text-green-600 hover:bg-green-200'
                                                    : 'bg-slate-100 text-slate-400 hover:bg-slate-200'
                                            }`}
                                            title={skill.enabled ? '禁用' : '启用'}
                                        >
                                            {skill.enabled ? <Power className="w-4 h-4" /> : <PowerOff className="w-4 h-4" />}
                                        </button>
                                    </div>

                                    {/* Description */}
                                    <p className="text-xs text-slate-600 mb-3 line-clamp-2 leading-relaxed">
                                        {skill.description}
                                    </p>

                                    {/* Tags */}
                                    <div className="flex flex-wrap gap-1.5 mb-3">
                                        {skill.tags.slice(0, 3).map(tag => (
                                            <span
                                                key={tag}
                                                className="px-2 py-0.5 bg-slate-100 text-slate-600 rounded text-xs font-medium"
                                            >
                                                {tag}
                                            </span>
                                        ))}
                                        {skill.tags.length > 3 && (
                                            <span className="px-2 py-0.5 bg-slate-100 text-slate-400 rounded text-xs">
                                                +{skill.tags.length - 3}
                                            </span>
                                        )}
                                    </div>

                                    {/* Footer */}
                                    <div className="flex items-center justify-between pt-3 border-t border-slate-100">
                                        <div className="flex items-center gap-1 text-xs text-slate-500">
                                            {skill.tools.map(tool => (
                                                <span
                                                    key={tool}
                                                    className="px-1.5 py-0.5 bg-slate-50 rounded border border-slate-200 font-mono"
                                                >
                                                    {tool}
                                                </span>
                                            ))}
                                        </div>
                                        {onSelectSkill && skill.enabled && (
                                            <button
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    onSelectSkill(skill.id);
                                                    onClose();
                                                }}
                                                className="px-2 py-1 bg-blue-600 text-white rounded text-xs font-medium hover:bg-blue-700 transition-colors opacity-0 group-hover:opacity-100"
                                            >
                                                使用
                                            </button>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Detail Modal */}
                {selectedSkill && (
                    <SkillDetailModal
                        skill={selectedSkill}
                        onClose={() => setSelectedSkill(null)}
                        onToggle={(enabled) => handleToggleSkill(selectedSkill.id, enabled)}
                        onUse={onSelectSkill}
                    />
                )}
            </div>
        </div>
    );
};

interface SkillDetailModalProps {
    skill: SkillInfo;
    onClose: () => void;
    onToggle: (currentlyEnabled: boolean) => void;
    onUse?: (skillId: string) => void;
}

const SkillDetailModal: React.FC<SkillDetailModalProps> = ({ skill, onClose, onToggle, onUse }) => {
    return (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-[60] flex items-center justify-center p-4">
            <div className="bg-white rounded-2xl shadow-2xl w-full max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
                {/* Header */}
                <div className="p-6 border-b border-slate-200 bg-gradient-to-r from-blue-50 to-indigo-50">
                    <div className="flex items-start justify-between">
                        <div className="flex items-center gap-4">
                            <div className={`p-3 rounded-xl ${
                                skill.enabled ? 'bg-blue-600 text-white' : 'bg-slate-300 text-slate-600'
                            }`}>
                                <Zap className="w-6 h-6" />
                            </div>
                            <div>
                                <h2 className="text-2xl font-bold text-slate-900">{skill.name}</h2>
                                <p className="text-sm text-slate-600 mt-1">
                                    v{skill.version} · by {skill.author}
                                </p>
                            </div>
                        </div>
                        <button
                            onClick={onClose}
                            className="p-2 hover:bg-white/80 rounded-lg transition-colors text-slate-400 hover:text-slate-600"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6">
                    {/* Description */}
                    <div className="mb-6">
                        <h3 className="text-sm font-bold text-slate-900 mb-2 uppercase tracking-wider">描述</h3>
                        <p className="text-slate-700 leading-relaxed">{skill.description}</p>
                    </div>

                    {/* Required Columns */}
                    <div className="mb-6">
                        <h3 className="text-sm font-bold text-slate-900 mb-3 uppercase tracking-wider">数据要求</h3>
                        <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
                            <p className="text-xs text-slate-600 mb-2">需要以下类型的数据列：</p>
                            <div className="flex flex-wrap gap-2">
                                {skill.required_columns.map(col => (
                                    <span
                                        key={col}
                                        className="px-3 py-1.5 bg-white border border-slate-300 rounded-lg text-sm font-mono text-slate-700"
                                    >
                                        {col}
                                    </span>
                                ))}
                            </div>
                        </div>
                    </div>

                    {/* Keywords */}
                    <div className="mb-6">
                        <h3 className="text-sm font-bold text-slate-900 mb-3 uppercase tracking-wider">触发关键词</h3>
                        <div className="flex flex-wrap gap-2">
                            {skill.keywords.map(keyword => (
                                <span
                                    key={keyword}
                                    className="px-3 py-1.5 bg-blue-50 text-blue-700 rounded-lg text-sm font-medium border border-blue-200"
                                >
                                    "{keyword}"
                                </span>
                            ))}
                        </div>
                        <p className="text-xs text-slate-500 mt-2">
                            在对话中使用这些关键词可以自动触发此 Skill
                        </p>
                    </div>

                    {/* Tags */}
                    <div className="mb-6">
                        <h3 className="text-sm font-bold text-slate-900 mb-3 uppercase tracking-wider">标签</h3>
                        <div className="flex flex-wrap gap-2">
                            {skill.tags.map(tag => (
                                <span
                                    key={tag}
                                    className="px-3 py-1.5 bg-slate-100 text-slate-600 rounded-lg text-sm"
                                >
                                    #{tag}
                                </span>
                            ))}
                        </div>
                    </div>

                    {/* Tools */}
                    <div className="mb-6">
                        <h3 className="text-sm font-bold text-slate-900 mb-3 uppercase tracking-wider">使用工具</h3>
                        <div className="flex gap-2">
                            {skill.tools.map(tool => (
                                <span
                                    key={tool}
                                    className="px-4 py-2 bg-gradient-to-r from-purple-50 to-pink-50 text-purple-700 rounded-lg text-sm font-bold border border-purple-200"
                                >
                                    {tool.toUpperCase()}
                                </span>
                            ))}
                        </div>
                    </div>

                    {/* Category */}
                    <div>
                        <h3 className="text-sm font-bold text-slate-900 mb-3 uppercase tracking-wider">分类</h3>
                        <span className="inline-block px-4 py-2 bg-slate-100 text-slate-700 rounded-lg text-sm font-medium">
                            {skill.category}
                        </span>
                    </div>
                </div>

                {/* Footer */}
                <div className="p-6 border-t border-slate-200 bg-slate-50 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-slate-600">状态:</span>
                        <span className={`px-3 py-1 rounded-full text-xs font-bold ${
                            skill.enabled
                                ? 'bg-green-100 text-green-700'
                                : 'bg-slate-200 text-slate-600'
                        }`}>
                            {skill.enabled ? '✓ 已启用' : '✗ 已禁用'}
                        </span>
                    </div>
                    <div className="flex gap-3">
                        <button
                            onClick={() => onToggle(skill.enabled)}
                            className={`px-4 py-2 rounded-lg font-medium transition-all ${
                                skill.enabled
                                    ? 'bg-slate-200 text-slate-700 hover:bg-slate-300'
                                    : 'bg-green-600 text-white hover:bg-green-700'
                            }`}
                        >
                            {skill.enabled ? '禁用' : '启用'}
                        </button>
                        {onUse && skill.enabled && (
                            <button
                                onClick={() => {
                                    onUse(skill.id);
                                    onClose();
                                }}
                                className="px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-all"
                            >
                                立即使用
                            </button>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
};

export default SkillsPage;
