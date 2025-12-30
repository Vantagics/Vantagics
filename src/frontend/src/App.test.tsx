import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import App from './App';
import * as AppBindings from '../wailsjs/go/main/App';
import * as WailsRuntime from '../wailsjs/runtime/runtime';
import { vi } from 'vitest';

// Mock the Wails bindings
vi.mock('../wailsjs/go/main/App', () => ({
    GetDashboardData: vi.fn(),
    GetConfig: vi.fn(),
    Greet: vi.fn(),
    SaveConfig: vi.fn(),
    SendMessage: vi.fn(),
}));

// Mock the runtime
vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(() => () => {}),
    ClipboardGetText: vi.fn(),
}));

// ... (existing code snippet markers or just full content if small)
// describe block starts here
describe('App Integration', () => {
    it('fetches and displays dashboard data on mount', async () => {
        const mockData = {
            metrics: [
                { title: 'Total Sales', value: '0,000', change: '+10%' },
            ],
            insights: [
                { text: 'Great progress!', icon: 'star' },
            ],
        };

        (AppBindings.GetDashboardData as any).mockResolvedValue(mockData);
        (AppBindings.GetConfig as any).mockResolvedValue({});

        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('Total Sales')).toBeInTheDocument();
            expect(screen.getByText('0,000')).toBeInTheDocument();
            expect(screen.getByText('Great progress!')).toBeInTheDocument();
        });
    });

    it('handles chat message flow', async () => {
        (AppBindings.SendMessage as any).mockResolvedValue("Hello! I am your AI assistant.");
        (AppBindings.GetConfig as any).mockResolvedValue({});
        (AppBindings.GetDashboardData as any).mockResolvedValue({ metrics: [], insights: [] });

        render(<App />);

        // Assuming there will be a "Chat" button
        const chatToggle = screen.getByLabelText('Toggle chat');
        fireEvent.click(chatToggle);

        const input = screen.getByPlaceholderText('Type a message...');
        const sendButton = screen.getByLabelText('Send message');

        fireEvent.change(input, { target: { value: 'How is business?' } });
        fireEvent.click(sendButton);

        expect(screen.getByText('How is business?')).toBeInTheDocument();
        
        await waitFor(() => {
            expect(screen.getByText('Hello! I am your AI assistant.')).toBeInTheDocument();
        });
    });

    it('shows custom context menu on right-click of input', async () => {
        (AppBindings.GetConfig as any).mockResolvedValue({});
        (AppBindings.GetDashboardData as any).mockResolvedValue({ metrics: [], insights: [] });

        render(<App />);

        // Assuming settings button opens modal with inputs
        const settingsButton = screen.getByLabelText(/Settings/i);
        fireEvent.click(settingsButton);

        await waitFor(() => {
            const apiKeyInput = screen.getByLabelText(/API Key/i);
            fireEvent.contextMenu(apiKeyInput, { clientX: 100, clientY: 100 });
        });

        expect(screen.getByRole('menu')).toBeInTheDocument();
        expect(screen.getByText('Paste')).toBeInTheDocument();
    });

    it('shows custom context menu on right-click of chat input', async () => {
        (AppBindings.GetConfig as any).mockResolvedValue({});
        (AppBindings.GetDashboardData as any).mockResolvedValue({ metrics: [], insights: [] });

        render(<App />);

        // Toggle chat to see input
        const chatToggle = screen.getByLabelText('Toggle chat');
        fireEvent.click(chatToggle);

        const chatInput = screen.getByPlaceholderText('Type a message...');
        fireEvent.contextMenu(chatInput, { clientX: 200, clientY: 200 });

        expect(screen.getByRole('menu')).toBeInTheDocument();
        expect(screen.getByText('Select All')).toBeInTheDocument();
    });

    it('pastes text into an input field via context menu', async () => {
        (AppBindings.GetConfig as any).mockResolvedValue({});
        (AppBindings.GetDashboardData as any).mockResolvedValue({ metrics: [], insights: [] });
        (WailsRuntime.ClipboardGetText as any).mockResolvedValue('Copied Key');

        render(<App />);

        // Open settings
        const settingsButton = screen.getByLabelText(/Settings/i);
        fireEvent.click(settingsButton);

        await waitFor(() => {
            const apiKeyInput = screen.getByLabelText(/API Key/i) as HTMLInputElement;
            fireEvent.contextMenu(apiKeyInput, { clientX: 100, clientY: 100 });
            
            const pasteButton = screen.getByText('Paste');
            fireEvent.click(pasteButton);
            
            expect(WailsRuntime.ClipboardGetText).toHaveBeenCalled();
            expect(apiKeyInput.value).toBe('Copied Key');
        });
    });
});

        
        
        
                
        
        