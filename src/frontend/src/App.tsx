import React, { useState, useEffect } from 'react';
import { ChevronLeft } from 'lucide-react';
import Sidebar from './components/Sidebar';
import Dashboard from './components/Dashboard';
import ContextPanel from './components/ContextPanel';
import PreferenceModal from './components/PreferenceModal';
import ChatSidebar from './components/ChatSidebar';
import ContextMenu from './components/ContextMenu';
import MessageModal from './components/MessageModal';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { GetDashboardData, GetConfig, TestLLMConnection, SetChatOpen } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import './App.css';

function App() {
    const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);
    const [isChatOpen, setIsChatOpen] = useState(false);
    const [dashboardData, setDashboardData] = useState<main.DashboardData | null>(null);
    const [activeChart, setActiveChart] = useState<{ type: 'echarts' | 'image', data: string } | null>(null);
    const [messageModal, setMessageModal] = useState<{ isOpen: boolean, type: 'info' | 'warning' | 'error', title: string, message: string }>({
        isOpen: false,
        type: 'info',
        title: '',
        message: ''
    });

    useEffect(() => {
        SetChatOpen(isChatOpen);
    }, [isChatOpen]);

    // Startup State
    const [isAppReady, setIsAppReady] = useState(false);
    const [startupStatus, setStartupStatus] = useState<"checking" | "failed">("checking");
    const [startupMessage, setStartupMessage] = useState("Initializing...");

    // Layout State
    const [sidebarWidth, setSidebarWidth] = useState(256);
    const [contextPanelWidth, setContextPanelWidth] = useState(384);
    const [isResizingSidebar, setIsResizingSidebar] = useState(false);
    const [isResizingContextPanel, setIsResizingContextPanel] = useState(false);

    // Context Menu State
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; target: HTMLElement } | null>(null);

    const checkLLM = async () => {
        setStartupStatus("checking");
        setStartupMessage("Checking LLM Configuration...");
        try {
            const config = await GetConfig();
            
            // Basic validation
            if (!config.apiKey && config.llmProvider !== 'OpenAI-Compatible' && config.llmProvider !== 'Claude-Compatible') {
                throw new Error("API Key is missing. Please configure LLM settings.");
            }
            
            setStartupMessage("Testing LLM Connection...");
            const result = await TestLLMConnection(config);
            
            if (result.success) {
                setIsAppReady(true);
                // Fetch dashboard data only after ready
                GetDashboardData().then(setDashboardData).catch(console.error);
            } else {
                throw new Error(`Connection Test Failed: ${result.message}`);
            }
        } catch (err: any) {
            console.error("Startup check failed:", err);
            setStartupStatus("failed");
            setStartupMessage(err.message || String(err));
            setIsPreferenceOpen(true);
        }
    };

    useEffect(() => {
        // Initial Check - only if not ready
        if (!isAppReady) {
            checkLLM();
        }

        // Listen for config updates to retry
        const unsubscribeConfig = EventsOn("config-updated", () => {
            if (!isAppReady) {
                checkLLM();
            }
        });

        // Listen for analysis events
        const unsubscribeAnalysisError = EventsOn("analysis-error", (msg: string) => {
            alert(`Analysis Error: ${msg}`);
        });
        const unsubscribeAnalysisWarning = EventsOn("analysis-warning", (msg: string) => {
            alert(`Analysis Warning: ${msg}`);
        });

        // Listen for menu event
        const unsubscribeSettings = EventsOn("open-settings", () => {
            setIsPreferenceOpen(true);
        });

        // Listen for dashboard chart updates
        const unsubscribeDashboardUpdate = EventsOn("dashboard-update", (payload: any) => {
            console.log("Dashboard Update Received:", payload);
            setActiveChart(payload);
        });

        const unsubscribeDashboardDataUpdate = EventsOn("dashboard-data-update", (data: main.DashboardData) => {
            console.log("Dashboard Data Update:", data);
            setDashboardData(data);
        });

        const unsubscribeAnalyzeInsight = EventsOn("analyze-insight", () => {
            setIsChatOpen(true);
        });

        const unsubscribeStartNewChat = EventsOn("start-new-chat", () => {
            setIsChatOpen(true);
        });

        const unsubscribeMessageModal = EventsOn("show-message-modal", (payload: any) => {
            setMessageModal({
                isOpen: true,
                type: payload.type,
                title: payload.title,
                message: payload.message
            });
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
            if (unsubscribeConfig) unsubscribeConfig();
            if (unsubscribeAnalysisError) unsubscribeAnalysisError();
            if (unsubscribeAnalysisWarning) unsubscribeAnalysisWarning();
            if (unsubscribeSettings) unsubscribeSettings();
            if (unsubscribeDashboardUpdate) unsubscribeDashboardUpdate();
            if (unsubscribeDashboardDataUpdate) unsubscribeDashboardDataUpdate();
            if (unsubscribeAnalyzeInsight) unsubscribeAnalyzeInsight();
            if (unsubscribeStartNewChat) unsubscribeStartNewChat();
            if (unsubscribeMessageModal) unsubscribeMessageModal();
            window.removeEventListener('contextmenu', handleContextMenu);
        };
    }, [isAppReady]);

    // Resize Handlers
    useEffect(() => {
        const handleMouseMove = (e: MouseEvent) => {
            if (isResizingSidebar) {
                const newWidth = e.clientX;
                if (newWidth > 150 && newWidth < 600) {
                    setSidebarWidth(newWidth);
                }
            } else if (isResizingContextPanel) {
                // Context Panel starts after sidebar. 
                // We can calculate its width as (currentX - sidebarWidth)
                // However, there might be a resizer width offset.
                const newWidth = e.clientX - sidebarWidth;
                if (newWidth > 200 && newWidth < 800) {
                    setContextPanelWidth(newWidth);
                }
            }
        };

        const handleMouseUp = () => {
            setIsResizingSidebar(false);
            setIsResizingContextPanel(false);
            document.body.style.cursor = 'default';
        };

        if (isResizingSidebar || isResizingContextPanel) {
            window.addEventListener('mousemove', handleMouseMove);
            window.addEventListener('mouseup', handleMouseUp);
        }

        return () => {
            window.removeEventListener('mousemove', handleMouseMove);
            window.removeEventListener('mouseup', handleMouseUp);
        };
    }, [isResizingSidebar, isResizingContextPanel, sidebarWidth]);

    const startResizingSidebar = () => {
        setIsResizingSidebar(true);
        document.body.style.cursor = 'col-resize';
    };

    const startResizingContextPanel = () => {
        setIsResizingContextPanel(true);
        document.body.style.cursor = 'col-resize';
    };

    if (!isAppReady) {
        return (
            <div className="flex h-screen w-screen bg-slate-50 items-center justify-center flex-col gap-6 relative">
                {/* Draggable Area for Startup Screen */}
                <div 
                    className="absolute top-0 left-0 right-0 h-10 z-[100]"
                    style={{ '--wails-draggable': 'drag' } as any}
                />

                <div className="w-16 h-16 border-4 border-blue-200 border-t-blue-600 rounded-full animate-spin"></div>
                
                <div className="text-center max-w-md px-6">
                    <h2 className="text-xl font-semibold text-slate-800 mb-2">System Startup</h2>
                    <p className={`text-sm ${startupStatus === 'failed' ? 'text-red-600' : 'text-slate-600'}`}>
                        {startupMessage}
                    </p>
                    
                    {startupStatus === 'failed' && (
                        <div className="mt-6 flex flex-col gap-3">
                            <button 
                                onClick={() => setIsPreferenceOpen(true)}
                                className="px-6 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 transition-colors shadow-sm"
                            >
                                Open Settings
                            </button>
                            <button 
                                onClick={checkLLM}
                                className="px-6 py-2 bg-white border border-slate-300 text-slate-700 text-sm font-medium rounded-md hover:bg-slate-50 transition-colors"
                            >
                                Retry Connection
                            </button>
                        </div>
                    )}
                </div>

                <PreferenceModal 
                    isOpen={isPreferenceOpen} 
                    onClose={() => setIsPreferenceOpen(false)} 
                />
            </div>
        );
    }

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
                width={sidebarWidth}
                onOpenSettings={() => setIsPreferenceOpen(true)} 
                onToggleChat={() => setIsChatOpen(!isChatOpen)}
            />
            
            {/* Sidebar Resizer */}
            <div
                className={`w-1 hover:bg-blue-400 cursor-col-resize z-50 transition-colors flex-shrink-0 ${isResizingSidebar ? 'bg-blue-600' : 'bg-transparent'}`}
                onMouseDown={startResizingSidebar}
            />

            <ContextPanel width={contextPanelWidth} />

            {/* Context Panel Resizer */}
            <div
                className={`w-1 hover:bg-blue-400 cursor-col-resize z-50 transition-colors flex-shrink-0 ${isResizingContextPanel ? 'bg-blue-600' : 'bg-transparent'}`}
                onMouseDown={startResizingContextPanel}
            />

            <div className="flex-1 flex flex-col min-w-0">
                <Dashboard data={dashboardData} activeChart={activeChart} />
            </div>
            
            <ChatSidebar 
                isOpen={isChatOpen} 
                onClose={() => setIsChatOpen(false)} 
            />

            <PreferenceModal 
                isOpen={isPreferenceOpen} 
                onClose={() => setIsPreferenceOpen(false)} 
            />

            <MessageModal
                isOpen={messageModal.isOpen}
                type={messageModal.type}
                title={messageModal.title}
                message={messageModal.message}
                onClose={() => setMessageModal(prev => ({ ...prev, isOpen: false }))}
            />

            {contextMenu && (
                <ContextMenu 
                    position={{ x: contextMenu.x, y: contextMenu.y }}
                    target={contextMenu.target}
                    onClose={() => setContextMenu(null)}
                />
            )}

            {!isChatOpen && (
                <button
                    onClick={() => setIsChatOpen(true)}
                    className="fixed right-0 top-1/2 -translate-y-1/2 z-[40] bg-white border border-slate-200 border-r-0 rounded-l-xl p-2 shadow-lg hover:bg-slate-50 text-blue-600 transition-transform hover:-translate-x-1 group"
                    title="Open Chat"
                >
                    <ChevronLeft className="w-5 h-5 group-hover:scale-110 transition-transform" />
                </button>
            )}
        </div>
    );
}

export default App;
