import React, { useState, useEffect } from 'react';
import Sidebar from './components/Sidebar';
import Dashboard from './components/Dashboard';
import ContextPanel from './components/ContextPanel';
import PreferenceModal from './components/PreferenceModal';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { GetDashboardData } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import './App.css';

function App() {
    const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);
    const [dashboardData, setDashboardData] = useState<main.DashboardData | null>(null);

    useEffect(() => {
        // Fetch dashboard data
        GetDashboardData().then(setDashboardData).catch(console.error);

        // Listen for menu event
        const unsubscribe = EventsOn("open-settings", () => {
            setIsPreferenceOpen(true);
        });
        return () => {
        };
    }, []);

    return (
        <div className="flex h-screen w-screen bg-slate-50 overflow-hidden font-sans text-slate-900 relative">
            <Sidebar onOpenSettings={() => setIsPreferenceOpen(true)} />
            <Dashboard data={dashboardData} />
            <ContextPanel />
            <PreferenceModal 
                isOpen={isPreferenceOpen} 
                onClose={() => setIsPreferenceOpen(false)} 
            />
        </div>
    );
}

export default App;