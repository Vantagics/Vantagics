import React, { useState } from 'react';

const ChatArea: React.FC = () => {
    const [messages, setMessages] = useState([
        { id: 1, role: 'ai', text: 'Hello! I am your AI Business Intelligence assistant. How can I help you analyze your data today?' },
        { id: 2, role: 'user', text: 'Show me the sales performance for the last quarter.' },
        { id: 3, role: 'ai', text: 'Sure. I have pulled the data from the Sales DB. You can see the summary in the panel to the right.' }
    ]);
    const [input, setInput] = useState('');

    const handleSend = () => {
        if (!input.trim()) return;
        setMessages([...messages, { id: Date.now(), role: 'user', text: input }]);
        setInput('');
        // Mock AI response
        setTimeout(() => {
             setMessages(prev => [...prev, { id: Date.now() + 1, role: 'ai', text: 'I am processing your request...' }]);
        }, 600);
    };

    return (
        <div className="flex-1 flex flex-col min-w-0 bg-white relative">
            {/* Header (Optional) */}
            <div 
                className="h-20 border-b border-slate-200 flex items-center px-4 pt-6 bg-white/80 backdrop-blur-sm z-10 sticky top-0"
                style={{ WebkitAppRegion: 'drag' } as any}
            >
                <span className="font-semibold text-slate-700">Chat Analysis</span>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4 bg-slate-50/30">
                {messages.map((msg) => (
                    <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                        <div className={`max-w-[80%] rounded-2xl px-4 py-3 text-sm shadow-sm ${
                            msg.role === 'user' 
                                ? 'bg-blue-600 text-white rounded-br-none' 
                                : 'bg-white border border-slate-200 text-slate-700 rounded-bl-none'
                        }`}>
                            {msg.text}
                        </div>
                    </div>
                ))}
            </div>

            {/* Input Area */}
            <div className="p-4 border-t border-slate-200 bg-white">
                <div className="flex gap-2">
                    <input 
                        type="text" 
                        className="flex-1 border border-slate-300 rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent placeholder-slate-400"
                        placeholder="Ask a question about your data..."
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                                handleSend();
                            } else {
                                e.stopPropagation();
                            }
                        }}
                    />
                    <button 
                        onClick={handleSend}
                        className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
                    >
                        Send
                    </button>
                </div>
            </div>
        </div>
    );
};

export default ChatArea;
