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
            {/* Draggable Title Bar Area */}
            <div 
                className="absolute top-0 left-0 right-0 h-10 z-[100] flex"
                style={{ '--wails-draggable': 'drag' } as any}
            >
                {/* Traffic Lights Area - clickable area for macOS buttons */}
                <div className="w-24 h-full" style={{ '--wails-draggable': 'no-drag' } as any} />
                
                {/* Drag Area - the rest of the top bar */}
                <div className="flex-1 h-full" />
            </div>

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
