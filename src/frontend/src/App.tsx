import React, { useState, useEffect } from 'react';
import Sidebar from './components/Sidebar';
import ChatArea from './components/ChatArea';
import ContextPanel from './components/ContextPanel';
import PreferenceModal from './components/PreferenceModal';
import { EventsOn } from '../wailsjs/runtime/runtime';
import './App.css'; // Keep if it has necessary global resets, but we rely mostly on Tailwind

function App() {
    const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);

    useEffect(() => {
        // Listen for menu event
        const unsubscribe = EventsOn("open-settings", () => {
            setIsPreferenceOpen(true);
        });
        return () => {
             // Wails runtime cleanup if necessary, though EventsOn returns specific unsub in newer versions or just ignore
        };
    }, []);

    return (
        <div className="flex h-screen w-screen bg-slate-50 overflow-hidden font-sans text-slate-900 relative">
            <Sidebar onOpenSettings={() => setIsPreferenceOpen(true)} />
            <ChatArea />
            <ContextPanel />
            <PreferenceModal 
                isOpen={isPreferenceOpen} 
                onClose={() => setIsPreferenceOpen(false)} 
            />
        </div>
    );
}

export default App;