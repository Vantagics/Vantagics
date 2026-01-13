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
                                    <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 mb-4">
                                        <p className="text-xs text-blue-700 font-medium">🧠 Short-Term Memory (Working Memory)</p>
                                        <p className="text-xs text-blue-600 mt-1">The most recent messages the AI sees in full detail. This is the "active context" used for immediate reasoning.</p>
                                    </div>
                                    {memory?.short_term && memory.short_term.length > 0 ? (
                                        memory.short_term.map((item, idx) => (
                                            <div key={idx} className="p-3 bg-white border border-slate-200 rounded-lg text-sm text-slate-700 shadow-sm whitespace-pre-wrap">
                                                {item}
                                            </div>
                                        ))
                                    ) : (
                                        <div className="text-center text-slate-400 text-sm py-10">No conversation yet.</div>
                                    )}
                                </>
                            )}
                            {activeTab === 'medium' && (
                                <>
                                    <div className="bg-purple-50 border border-purple-200 rounded-lg p-3 mb-4">
                                        <p className="text-xs text-purple-700 font-medium">📚 Medium-Term Memory (Compressed History)</p>
                                        <p className="text-xs text-purple-600 mt-1">Older messages are summarized to save context space. The AI remembers the key topics and findings, not every word.</p>
                                    </div>
                                    {memory?.medium_term && memory.medium_term.length > 0 ? (
                                        memory.medium_term.map((item, idx) => (
                                            <div key={idx} className="p-3 bg-white border border-purple-100 rounded-lg text-sm text-slate-700 shadow-sm border-l-4 border-l-purple-400 whitespace-pre-wrap">
                                                {item}
                                            </div>
                                        ))
                                    ) : (
                                        <div className="text-center text-slate-400 text-sm py-10">No compressed history yet.</div>
                                    )}
                                </>
                            )}
                            {activeTab === 'long' && (
                                <>
                                    <div className="bg-green-50 border border-green-200 rounded-lg p-3 mb-4">
                                        <p className="text-xs text-green-700 font-medium">🌍 Long-Term Memory (Session Overview)</p>
                                        <p className="text-xs text-green-600 mt-1">High-level session information and persistent facts that span the entire conversation.</p>
                                    </div>
                                    {memory?.long_term && memory.long_term.length > 0 ? (
                                        memory.long_term.map((item, idx) => (
                                            <div key={idx} className="p-3 bg-white border border-green-100 rounded-lg text-sm text-slate-700 shadow-sm border-l-4 border-l-green-400 whitespace-pre-wrap">
                                                {item}
                                            </div>
                                        ))
                                    ) : (
                                        <div className="text-center text-slate-400 text-sm py-10">No session data yet.</div>
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
