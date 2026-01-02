import { useState, useEffect } from 'react';
import { GetConfig } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

export type Language = 'English' | '简体中文';

export const translations: Record<Language, Record<string, string>> = {
    'English': {
        'data_sources': 'Data Sources',
        'chat_analysis': 'Chat Analysis',
        'add_source': '+ Add Source',
        'settings': 'Settings',
        'smart_dashboard': 'Smart Dashboard',
        'welcome_back': "Welcome back! Here's what's happening with your data.",
        'key_metrics': 'Key Metrics',
        'automated_insights': 'Automated Insights',
        'loading_insights': 'Loading your insights...',
        'ai_assistant': 'AI Assistant',
        'ready_to_help': 'Ready to help',
        'history': 'History',
        'new_chat': 'New Chat',
        'clear_history': 'CLEAR HISTORY',
        'what_to_analyze': 'What would you like to analyze?',
        'data_driven_reasoning': 'Data-driven reasoning',
        'visualized_summaries': 'Visualized summaries',
        'insights_at_fingertips': 'Insights at your fingertips',
        'ask_about_sales': 'Ask about sales trends, customer behavior, or request a complex data analysis.',
        'start_new_analysis': 'Start New Analysis',
        'ai_thinking': 'AI is thinking...',
        'clear_history_confirm_title': 'Clear All History?',
        'clear_history_confirm_desc': 'This action cannot be undone. All your chat threads will be permanently deleted.',
        'cancel': 'Cancel',
        'clear': 'Clear History',
        'preferences': 'Preferences',
        'llm_config': 'LLM Configuration',
        'system_params': 'System Parameters',
        'drivers': 'Drivers',
        'run_env': 'Run Environment',
        'save_changes': 'Save Changes',
        'test_connection': 'Test Connection',
        'provider_type': 'Provider Type',
        'api_key': 'API Key',
        'base_url': 'Base URL (Optional)',
        'model_name': 'Model Name',
        'max_tokens': 'Max Tokens',
        'dark_mode': 'Dark Mode',
        'local_cache': 'Local Cache',
        'language': 'Language',
        'data_cache_dir': 'Data Cache Directory',
        'browse': 'Browse...',
    },
    '简体中文': {
        'data_sources': '数据源',
        'chat_analysis': '对话分析',
        'add_source': '+ 添加数据源',
        'settings': '设置',
        'smart_dashboard': '智能仪表盘',
        'welcome_back': '欢迎回来！这是您的最新数据概览。',
        'key_metrics': '核心指标',
        'automated_insights': '自动化洞察',
        'loading_insights': '正在加载您的洞察数据...',
        'ai_assistant': 'AI 助手',
        'ready_to_help': '随时准备为您提供帮助',
        'history': '历史记录',
        'new_chat': '新建对话',
        'clear_history': '清空历史记录',
        'what_to_analyze': '您想分析什么？',
        'data_driven_reasoning': '数据驱动推理',
        'visualized_summaries': '可视化摘要',
        'insights_at_fingertips': '洞察近在咫尺',
        'ask_about_sales': '您可以询问销售趋势、用户行为，或请求复杂的数据分析。',
        'start_new_analysis': '开始新分析',
        'ai_thinking': 'AI 正在思考...',
        'clear_history_confirm_title': '清空所有历史记录？',
        'clear_history_confirm_desc': '此操作无法撤销。您的所有聊天记录都将被永久删除。',
        'cancel': '取消',
        'clear': '清空历史',
        'preferences': '系统偏好设置',
        'llm_config': 'LLM 配置',
        'system_params': '系统参数',
        'drivers': '驱动程序',
        'run_env': '运行环境',
        'save_changes': '保存更改',
        'test_connection': '测试连接',
        'provider_type': '供应商类型',
        'api_key': 'API 密钥',
        'base_url': '基础 URL (可选)',
        'model_name': '模型名称',
        'max_tokens': '最大 Token 数',
        'dark_mode': '深色模式',
        'local_cache': '本地缓存',
        'language': '语言',
        'data_cache_dir': '数据缓存目录',
        'browse': '浏览...',
    }
};

export function useLanguage() {
    const [language, setLanguage] = useState<Language>('English');

    const updateLanguage = () => {
        GetConfig().then(config => {
            if (config.language === '简体中文' || config.language === 'English') {
                setLanguage(config.language as Language);
            }
        }).catch(console.error);
    };

    useEffect(() => {
        updateLanguage();
        // Listen for config changes
        const unsubscribe = EventsOn('config-updated', () => {
            updateLanguage();
        });
        return () => {
            if (unsubscribe) unsubscribe();
        };
    }, []);

    const t = (key: string) => {
        return translations[language][key] || key;
    };

    return { language, t };
}
