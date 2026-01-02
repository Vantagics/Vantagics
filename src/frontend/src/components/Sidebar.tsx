import React from 'react';
import { useLanguage } from '../i18n';

interface SidebarProps {
    onOpenSettings: () => void;
    onToggleChat: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ onOpenSettings, onToggleChat }) => {
    const { t } = useLanguage();
    const dataSources = [
        { id: 1, name: 'Sales DB (PostgreSQL)', type: 'SQL' },
        { id: 2, name: 'User Logs (Mongo)', type: 'NoSQL' },
        { id: 3, name: 'Marketing CSV', type: 'Local' },
        { id: 4, name: 'Redis Cache', type: 'Cache' },
    ];

    return (
        <div className="w-64 bg-slate-100 border-r border-slate-200 flex flex-col h-full">
            <div 
                className="p-4 pt-8 border-b border-slate-200 bg-slate-50"
                style={{ '--wails-draggable': 'drag' } as any}
            >
                <h2 className="text-lg font-semibold text-slate-700">{t('data_sources')}</h2>
            </div>
            <div className="flex-1 overflow-y-auto p-2">
                <ul className="space-y-1">
                    {dataSources.map((source) => (
                        <li key={source.id} className="p-2 rounded-md hover:bg-blue-100 cursor-pointer text-sm text-slate-600 flex items-center gap-2 transition-colors">
                            <span className={`w-2 h-2 rounded-full ${source.type === 'SQL' ? 'bg-blue-500' : source.type === 'NoSQL' ? 'bg-green-500' : 'bg-gray-400'}`}></span>
                            {source.name}
                        </li>
                    ))}
                </ul>
            </div>
            <div className="p-4 border-t border-slate-200 flex flex-col gap-2">
                <button 
                    onClick={onToggleChat}
                    aria-label="Toggle chat"
                    className="w-full py-2 px-4 bg-blue-100 hover:bg-blue-200 text-blue-700 rounded-md text-sm font-medium transition-colors flex items-center justify-center gap-2"
                >
                    <span>💬</span> {t('chat_analysis')}
                </button>
                <button className="w-full py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium transition-colors">
                    {t('add_source')}
                </button>
                <button 
                    onClick={onOpenSettings}
                    aria-label="Settings"
                    className="w-full py-2 px-4 bg-white border border-slate-300 hover:bg-slate-50 text-slate-700 rounded-md text-sm font-medium transition-colors flex items-center justify-center gap-2"
                >
                    <span>⚙️</span> {t('settings')}
                </button>
            </div>
        </div>
    );
};

export default Sidebar;