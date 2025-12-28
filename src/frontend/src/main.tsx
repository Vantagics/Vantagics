import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'

const container = document.getElementById('root')

const root = createRoot(container!)

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
