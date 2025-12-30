import React, { useState, useEffect } from 'react';
import Sidebar from './components/Sidebar';
import Dashboard from './components/Dashboard';
import ContextPanel from './components/ContextPanel';
import PreferenceModal from './components/PreferenceModal';
import ChatSidebar from './components/ChatSidebar';
import ContextMenu from './components/ContextMenu';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { GetDashboardData } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import './App.css';

function App() {
    const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);
    const [isChatOpen, setIsChatOpen] = useState(false);
    const [dashboardData, setDashboardData] = useState<main.DashboardData | null>(null);

    // Context Menu State
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; target: HTMLElement } | null>(null);

    useEffect(() => {
        // Fetch dashboard data
        GetDashboardData().then(setDashboardData).catch(console.error);

        // Listen for menu event
        const unsubscribe = EventsOn("open-settings", () => {
            setIsPreferenceOpen(true);
        });

        // Global Context Menu Listener
        const handleContextMenu = (e: MouseEvent) => {
            const target = e.target as HTMLElement;
            if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
                e.preventDefault();
                setContextMenu({ x: e.clientX, y: e.clientY, target });
            }
        };

        window.addEventListener('contextmenu', handleContextMenu);

        return () => {
            window.removeEventListener('contextmenu', handleContextMenu);
        };
    }, []);

    return (
        <div className="flex h-screen w-screen bg-slate-50 overflow-hidden font-sans text-slate-900 relative">
            <Sidebar 
                onOpenSettings={() => setIsPreferenceOpen(true)} 
                onToggleChat={() => setIsChatOpen(!isChatOpen)}
            />
            <Dashboard data={dashboardData} />
            <ContextPanel />
            
            <ChatSidebar 
                isOpen={isChatOpen} 
                onClose={() => setIsChatOpen(false)} 
            />

            <PreferenceModal 
                isOpen={isPreferenceOpen} 
                onClose={() => setIsPreferenceOpen(false)} 
            />

            {contextMenu && (
                <ContextMenu 
                    position={{ x: contextMenu.x, y: contextMenu.y }}
                    target={contextMenu.target}
                    onClose={() => setContextMenu(null)}
                />
            )}
        </div>
    );
}

export default App;
