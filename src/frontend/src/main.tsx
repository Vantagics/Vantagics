import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'

const container = document.getElementById('root')

const root = createRoot(container!)

// Mock Wails runtime for browser development
// @ts-ignore
if (!window.runtime) {
    // @ts-ignore
    window.runtime = {
        EventsOnMultiple: () => {}, // Correct mock for EventsOn
        LogPrint: console.log,
        // Add other necessary runtime mocks as needed
    };
}
// @ts-ignore
if (!window.go) {
    // @ts-ignore
    window.go = {
        main: {
            App: {
                GetDashboardData: async () => ({ metrics: [], insights: [] }),
                GetConfig: async () => ({
                    llmProvider: 'OpenAI',
                    apiKey: '',
                    baseUrl: '',
                    modelName: '',
                    maxTokens: 4096,
                    darkMode: false,
                    localCache: true,
                    language: 'English',
                    claudeHeaderStyle: 'Anthropic',
                    dataCacheDir: '~/RapidBI',
                    pythonPath: ''
                }),
                SaveConfig: async () => {},
                TestLLMConnection: async () => ({ success: true, message: 'Mock Success' }),
                SelectDirectory: async () => '/mock/selected/path',
                GetPythonEnvironments: async () => [
                    { path: '/usr/bin/python3', version: '3.9.6', type: 'System', isRecommended: true },
                    { path: '/opt/conda/bin/python', version: '3.10.0', type: 'Conda', isRecommended: false }
                ],
                ValidatePython: async () => ({
                    valid: true,
                    version: '3.9.6',
                    missingPackages: [],
                    error: ''
                }),
                // Add other mocks as needed
            }
        }
    };
}

// Enable system context menu for inputs and textareas
window.addEventListener('contextmenu', (e) => {
    const target = e.target as HTMLElement;
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
        // Allow the event to proceed normally for editable elements
        return;
    }
    // Optionally prevent default for non-editable areas if desired, 
    // but the user only asked to enable it for inputs.
});

root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
)
