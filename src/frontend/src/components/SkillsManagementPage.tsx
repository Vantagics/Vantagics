import React, { useState, useEffect } from 'react';
import { X, Upload, RefreshCw, Package, AlertCircle, CheckCircle, Trash2, Search } from 'lucide-react';
import { ListSkills, InstallSkillsFromZip, EnableSkill, DisableSkill, DeleteSkill } from '../../wailsjs/go/main/App';
import { useLanguage } from '../i18n';
import SkillCard from './SkillCard';

interface Skill {
    name: string;
    description: string;
    content: string;
    path: string;
    installed_at: string;
    enabled: boolean;
}

interface SkillsManagementPageProps {
    isOpen: boolean;
    onClose: () => void;
}

const SkillsManagementPage: React.FC<SkillsManagementPageProps> = ({ isOpen, onClose }) => {
    const { t } = useLanguage();
    const [skills, setSkills] = useState<Skill[]>([]);
    const [loading, setLoading] = useState(false);
    const [installing, setInstalling] = useState(false);
    const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
    const [searchQuery, setSearchQuery] = useState('');
    const [deletingSkill, setDeletingSkill] = useState<string | null>(null);

    useEffect(() => {
        if (isOpen) {
            loadSkills();
        }
    }, [isOpen]);

    const loadSkills = async () => {
        setLoading(true);
        setMessage(null);
        try {
            const skillsList = await ListSkills();
            setSkills(skillsList || []);
        } catch (error) {
            console.error('Failed to load skills:', error);
            setMessage({ type: 'error', text: `加载失败: ${error}` });
        } finally {
            setLoading(false);
        }
    };

    const handleInstallSkills = async () => {
        setInstalling(true);
        setMessage(null);
        try {
            const installed = await InstallSkillsFromZip();
            if (installed && installed.length > 0) {
                setMessage({
                    type: 'success',
                    text: `成功安装 ${installed.length} 个Skills: ${installed.join(', ')}`
                });
                await loadSkills();
            } else {
                setMessage({ type: 'error', text: '未选择文件或安装失败' });
            }
        } catch (error) {
            console.error('Failed to install skills:', error);
            setMessage({ type: 'error', text: `安装失败: ${error}` });
        } finally {
            setInstalling(false);
        }
    };

    const handleToggleSkill = async (skillName: string, currentEnabled: boolean) => {
        try {
            if (currentEnabled) {
                await DisableSkill(skillName);
                setMessage({ type: 'success', text: `已禁用 ${skillName}` });
            } else {
                await EnableSkill(skillName);
                setMessage({ type: 'success', text: `已启用 ${skillName}` });
            }
            await loadSkills();
        } catch (error) {
            console.error('Failed to toggle skill:', error);
            const errorMsg = String(error);
            
            // Check if error is due to analysis in progress
            if (errorMsg.includes('analysis is in progress') || errorMsg.includes('分析正在进行')) {
                setMessage({ 
                    type: 'error', 
                    text: `无法修改 Skill 状态：当前有分析任务正在进行中。请等待分析完成后再试。` 
                });
            } else {
                setMessage({ type: 'error', text: `操作失败: ${error}` });
            }
        }
    };

    const handleDeleteSkill = async (skillName: string) => {
        // Confirm deletion
        if (!confirm(`确定要删除 Skill "${skillName}" 吗？此操作将删除 Skill 的所有文件和配置，且无法恢复。`)) {
            return;
        }

        setDeletingSkill(skillName);
        try {
            await DeleteSkill(skillName);
            setMessage({ type: 'success', text: `已删除 ${skillName}` });
            await loadSkills();
        } catch (error) {
            console.error('Failed to delete skill:', error);
            const errorMsg = String(error);
            
            // Check if error is due to analysis in progress
            if (errorMsg.includes('analysis is in progress') || errorMsg.includes('分析正在进行')) {
                setMessage({ 
                    type: 'error', 
                    text: `无法删除 Skill：当前有分析任务正在进行中。请等待分析完成后再试。` 
                });
            } else {
                setMessage({ type: 'error', text: `删除失败: ${error}` });
            }
        } finally {
            setDeletingSkill(null);
        }
    };

    // Filter skills based on search query
    const filteredSkills = skills.filter(skill => {
        if (!searchQuery) return true;
        const query = searchQuery.toLowerCase();
        return (
            skill.name.toLowerCase().includes(query) ||
            skill.description.toLowerCase().includes(query)
        );
    });

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-6xl mx-4 max-h-[90vh] flex flex-col">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
                    <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center">
                            <Package className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                        </div>
                        <div>
                            <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                                {t('skills_management') || 'Skills 管理'}
                            </h2>
                            <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
                                {skills.length} {t('skills_installed') || '个已安装'} · {skills.filter(s => s.enabled).length} {t('skills_enabled') || '个已启用'}
                                {searchQuery && ` · ${filteredSkills.length} 个匹配`}
                            </p>
                        </div>
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={loadSkills}
                            disabled={loading}
                            className="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
                            title={t('refresh') || '刷新'}
                        >
                            <RefreshCw className={`w-5 h-5 ${loading ? 'animate-spin' : ''}`} />
                        </button>
                        <button
                            onClick={handleInstallSkills}
                            disabled={installing}
                            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:bg-blue-400 disabled:cursor-not-allowed"
                        >
                            {installing ? (
                                <RefreshCw className="w-4 h-4 animate-spin" />
                            ) : (
                                <Upload className="w-4 h-4" />
                            )}
                            <span>{installing ? (t('installing') || '安装中...') : (t('install_skills') || '安装 Skills')}</span>
                        </button>
                        <button
                            onClick={onClose}
                            className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>
                </div>

                {/* Message */}
                {message && (
                    <div className={`mx-6 mt-4 p-4 rounded-lg flex items-start gap-3 ${
                        message.type === 'success'
                            ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
                            : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
                    }`}>
                        {message.type === 'success' ? (
                            <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
                        ) : (
                            <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                        )}
                        <div className="flex-1">
                            <p className={`text-sm ${
                                message.type === 'success'
                                    ? 'text-green-800 dark:text-green-200'
                                    : 'text-red-800 dark:text-red-200'
                            }`}>
                                {message.text}
                            </p>
                        </div>
                        <button
                            onClick={() => setMessage(null)}
                            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                        >
                            <X className="w-4 h-4" />
                        </button>
                    </div>
                )}

                {/* Search Bar */}
                {skills.length > 0 && (
                    <div className="mx-6 mt-4">
                        <div className="relative">
                            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                            <input
                                type="text"
                                placeholder={t('search_skills') || '搜索 Skills（按名称或描述）...'}
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            />
                            {searchQuery && (
                                <button
                                    onClick={() => setSearchQuery('')}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                                >
                                    <X className="w-4 h-4" />
                                </button>
                            )}
                        </div>
                    </div>
                )}

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6">
                    {loading ? (
                        <div className="flex items-center justify-center h-64">
                            <RefreshCw className="w-8 h-8 animate-spin text-blue-600" />
                        </div>
                    ) : skills.length === 0 ? (
                        <div className="flex flex-col items-center justify-center h-64 text-gray-400 dark:text-gray-500">
                            <Package className="w-16 h-16 mb-4 opacity-20" />
                            <p className="text-lg font-medium">{t('no_skills') || '暂无已安装的 Skills'}</p>
                            <p className="text-sm mt-2">{t('install_skills_hint') || '点击上方"安装 Skills"按钮来安装新的 Skills'}</p>
                        </div>
                    ) : filteredSkills.length === 0 ? (
                        <div className="flex flex-col items-center justify-center h-64 text-gray-400 dark:text-gray-500">
                            <Search className="w-16 h-16 mb-4 opacity-20" />
                            <p className="text-lg font-medium">未找到匹配的 Skills</p>
                            <p className="text-sm mt-2">尝试使用不同的搜索关键词</p>
                        </div>
                    ) : (
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                            {filteredSkills.map((skill) => (
                                <div key={skill.name} className="relative group">
                                    {/* Checkbox and Delete Button overlay */}
                                    <div className="absolute top-3 left-3 right-3 z-10 flex items-center justify-between">
                                        <label className="flex items-center cursor-pointer group/checkbox">
                                            <input
                                                type="checkbox"
                                                checked={skill.enabled}
                                                onChange={() => handleToggleSkill(skill.name, skill.enabled)}
                                                className="w-5 h-5 text-blue-600 bg-white border-2 border-gray-300 rounded focus:ring-2 focus:ring-blue-500 cursor-pointer transition-all hover:border-blue-500"
                                            />
                                            <span className="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300 opacity-0 group-hover/checkbox:opacity-100 transition-opacity">
                                                {skill.enabled ? '已启用' : '已禁用'}
                                            </span>
                                        </label>
                                        <button
                                            onClick={() => handleDeleteSkill(skill.name)}
                                            disabled={deletingSkill === skill.name}
                                            className="p-1.5 bg-red-500 hover:bg-red-600 text-white rounded-md opacity-0 group-hover:opacity-100 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                                            title="删除 Skill"
                                        >
                                            {deletingSkill === skill.name ? (
                                                <RefreshCw className="w-4 h-4 animate-spin" />
                                            ) : (
                                                <Trash2 className="w-4 h-4" />
                                            )}
                                        </button>
                                    </div>
                                    {/* Skill card with opacity based on enabled state */}
                                    <div className={skill.enabled ? '' : 'opacity-50'}>
                                        <SkillCard skill={skill} />
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="p-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
                    <div className="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
                        <div className="flex items-center gap-2">
                            <AlertCircle className="w-4 h-4" />
                            <span>{t('skills_info') || 'Skills 包必须包含 SKILL.md 文件'}</span>
                        </div>
                        <button
                            onClick={onClose}
                            className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg transition-colors"
                        >
                            {t('close') || '关闭'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default SkillsManagementPage;
