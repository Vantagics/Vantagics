import React, { useState, useEffect } from 'react';
import { useLanguage } from '../i18n';
import { X, Brain, Clock, History } from 'lucide-react';
import { GetAgentMemory } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

interface MemoryViewModalProps {
    isOpen: boolean;
    threadId: string;
    onClose: () => void;
}

const MemoryViewModal: React.FC<MemoryViewModalProps> = ({ isOpen, threadId, onClose }) => {
    const { t } = useLanguage();
    const [activeTab, setActiveTab] = useState<'short' | 'medium' | 'long'>('short');
    const [memory, setMemory] = useState<main.AgentMemoryView | null>(null);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (isOpen && threadId) {
            setLoading(true);
            GetAgentMemory(threadId)
                .then(data => setMemory(data))
                .catch(console.error)
                .finally(() => setLoading(false));
        }
    }, [isOpen, threadId]);

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-white w-[600px] h-[500px] rounded-xl shadow-2xl flex flex-col overflow-hidden text-slate-900">
                {/* Header */}
                <div className="p-4 border-b border-slate-200 flex justify-between items-center bg-slate-50">
                    <div className="flex items-center gap-2">
                        <Brain className="w-5 h-5 text-blue-600" />
                        <h2 className="text-lg font-bold text-slate-800">{t('agent_memory')}</h2>
                    </div>
                    <button onClick={onClose} className="text-slate-500 hover:text-slate-700">
                        <X className="w-5 h-5" />
                    </button>
                </div>

                {/* Tabs */}
                <div className="flex border-b border-slate-200">
                    <button
                        onClick={() => setActiveTab('short')}
                        className={`flex-1 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
                            activeTab === 'short' ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50/50' : 'text-slate-600 hover:bg-slate-50'
                        }`}
                    >
                        <Clock className="w-4 h-4" />
                        {t('short_term_memory')}
                    </button>
                    <button
                        onClick={() => setActiveTab('medium')}
                        className={`flex-1 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
                            activeTab === 'medium' ? 'text-purple-600 border-b-2 border-purple-600 bg-purple-50/50' : 'text-slate-600 hover:bg-slate-50'
                        }`}
                    >
                        <History className="w-4 h-4" />
                        {t('medium_term_memory')}
                    </button>
                    <button
                        onClick={() => setActiveTab('long')}
                        className={`flex-1 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
                            activeTab === 'long' ? 'text-green-600 border-b-2 border-green-600 bg-green-50/50' : 'text-slate-600 hover:bg-slate-50'
                        }`}
                    >
                        <Brain className="w-4 h-4" />
                        {t('long_term_memory')}
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6 bg-slate-50/30">
                    {loading ? (
                        <div className="flex justify-center items-center h-full text-slate-400 text-sm">Loading...</div>
                    ) : (
                        <div className="space-y-2">
                            {activeTab === 'short' && (
                                <>
                                    <p className="text-xs text-slate-500 mb-4 italic">Recent context from the current conversation (last 10 messages).</p>
                                    {memory?.short_term && memory.short_term.length > 0 ? (
                                        memory.short_term.map((item, idx) => (
                                            <div key={idx} className="p-3 bg-white border border-slate-200 rounded-lg text-sm text-slate-700 shadow-sm">
                                                {item}
                                            </div>
                                        ))
                                    ) : (
                                        <div className="text-center text-slate-400 text-sm py-10">No short-term memory found.</div>
                                    )}
                                </>
                            )}
                            {activeTab === 'medium' && (
                                <>
                                    <p className="text-xs text-slate-500 mb-4 italic">Important facts derived from this and recent interactions.</p>
                                    {memory?.medium_term && memory.medium_term.length > 0 ? (
                                        memory.medium_term.map((item, idx) => (
                                            <div key={idx} className="p-3 bg-white border border-purple-100 rounded-lg text-sm text-slate-700 shadow-sm border-l-4 border-l-purple-400">
                                                {item}
                                            </div>
                                        ))
                                    ) : (
                                        <div className="text-center text-slate-400 text-sm py-10">No medium-term memory found.</div>
                                    )}
                                </>
                            )}
                            {activeTab === 'long' && (
                                <>
                                    <p className="text-xs text-slate-500 mb-4 italic">Global knowledge and long-standing facts.</p>
                                    {memory?.long_term && memory.long_term.length > 0 ? (
                                        memory.long_term.map((item, idx) => (
                                            <div key={idx} className="p-3 bg-white border border-green-100 rounded-lg text-sm text-slate-700 shadow-sm border-l-4 border-l-green-400">
                                                {item}
                                            </div>
                                        ))
                                    ) : (
                                        <div className="text-center text-slate-400 text-sm py-10">No long-term memory found.</div>
                                    )}
                                </>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default MemoryViewModal;
