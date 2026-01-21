import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { GetConfig } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

interface DarkModeContextType {
    isDarkMode: boolean;
    setIsDarkMode: (value: boolean) => void;
}

const DarkModeContext = createContext<DarkModeContextType>({
    isDarkMode: false,
    setIsDarkMode: () => {},
});

export const useDarkMode = () => useContext(DarkModeContext);

interface DarkModeProviderProps {
    children: ReactNode;
}

export const DarkModeProvider: React.FC<DarkModeProviderProps> = ({ children }) => {
    const [isDarkMode, setIsDarkMode] = useState(false);

    // Load dark mode setting from config on mount
    useEffect(() => {
        const loadDarkMode = async () => {
            try {
                const config = await GetConfig();
                setIsDarkMode(config.darkMode || false);
            } catch (err) {
                console.error('Failed to load dark mode setting:', err);
            }
        };
        loadDarkMode();
    }, []);

    // Listen for config updates
    useEffect(() => {
        const unsubscribe = EventsOn('config-updated', async () => {
            try {
                const config = await GetConfig();
                setIsDarkMode(config.darkMode || false);
            } catch (err) {
                console.error('Failed to reload dark mode setting:', err);
            }
        });

        return () => {
            unsubscribe();
        };
    }, []);

    // Apply dark mode class to document
    useEffect(() => {
        if (isDarkMode) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
    }, [isDarkMode]);

    return (
        <DarkModeContext.Provider value={{ isDarkMode, setIsDarkMode }}>
            {children}
        </DarkModeContext.Provider>
    );
};

export default DarkModeContext;
