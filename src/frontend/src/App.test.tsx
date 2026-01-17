import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import App from './App';
import * as AppBindings from '../wailsjs/go/main/App';
import * as WailsRuntime from '../wailsjs/runtime/runtime';
import { vi } from 'vitest';

// Mock the Wails bindings
vi.mock('../wailsjs/go/main/App', async (importOriginal) => {
    const actual = await importOriginal<any>();
    return {
        ...actual,
        GetDashboardData: vi.fn(),
        GetConfig: vi.fn(),
        GetDataSources: vi.fn(),
        Greet: vi.fn(),
        SaveConfig: vi.fn(),
        SendMessage: vi.fn(),
        SetChatOpen: vi.fn(),
        ExportAnalysisProcess: vi.fn(),
        ExportMessageToPDF: vi.fn(),
        TestLLMConnection: vi.fn().mockResolvedValue({ success: true }),
        GetChatHistory: vi.fn().mockResolvedValue([]),
        SaveChatHistory: vi.fn().mockResolvedValue(null),
        DeleteThread: vi.fn().mockResolvedValue(null),
        ClearHistory: vi.fn().mockResolvedValue(null),
        GetSessionFiles: vi.fn().mockResolvedValue([]),
    WriteSystemLog: vi.fn().mockResolvedValue(null),
    LoadMetricsJson: vi.fn().mockResolvedValue("[]"),
    SaveMetricsJson: vi.fn().mockResolvedValue(null),
    CheckSessionNameExists: vi.fn().mockResolvedValue(false),
    };
});

// Mock the runtime
vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(() => () => {}),
    EventsEmit: vi.fn(),
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
        (AppBindings.GetConfig as any).mockResolvedValue({ apiKey: 'test-key' });
        (AppBindings.GetDataSources as any).mockResolvedValue([]);

        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('Total Sales')).toBeInTheDocument();
            expect(screen.getByText('0,000')).toBeInTheDocument();
            expect(screen.getByText('Great progress!')).toBeInTheDocument();
        });
    });

    it('handles chat message flow', async () => {
        const mockSources = [{ id: 'ds1', name: 'Sales DB', type: 'sqlite' }];
        (AppBindings.SendMessage as any).mockResolvedValue("Hello! I am your AI assistant.");
        (AppBindings.GetConfig as any).mockResolvedValue({ apiKey: 'test-key' });
        (AppBindings.GetDashboardData as any).mockResolvedValue({ metrics: [], insights: [] });
        (AppBindings.GetDataSources as any).mockResolvedValue(mockSources);

        render(<App />);

        // Wait for app to be ready
        await waitFor(() => {
            expect(screen.getByText('Sales DB')).toBeInTheDocument();
        });

        // Select data source
        fireEvent.click(screen.getByText('Sales DB'));

        // Click Chat Analysis
        const chatToggle = screen.getByLabelText(/Chat Analysis/i);
        fireEvent.click(chatToggle);

        // Fill New Chat Modal
        await waitFor(() => {
            expect(screen.getByPlaceholderText(/e.g. Sales Analysis Q1/i)).toBeInTheDocument();
        });
        fireEvent.change(screen.getByPlaceholderText(/e.g. Sales Analysis Q1/i), { target: { value: 'Test Session' } });
        fireEvent.click(screen.getByRole('button', { name: /Start Chat/i }));

        // Now wait for chat input
        await waitFor(() => {
            expect(screen.getByPlaceholderText('Type a message...')).toBeInTheDocument();
        });

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
        (AppBindings.GetConfig as any).mockResolvedValue({ apiKey: 'test-key' });
        (AppBindings.GetDashboardData as any).mockResolvedValue({ metrics: [], insights: [] });
        (AppBindings.GetDataSources as any).mockResolvedValue([]);

        render(<App />);

        // Wait for app to be ready
        await waitFor(() => {
            expect(screen.getByLabelText(/Settings/i)).toBeInTheDocument();
        });

        // Assuming settings button opens modal with inputs
        const settingsButton = screen.getByLabelText(/Settings/i);
        fireEvent.click(settingsButton);

        // Switch to LLM tab
        await waitFor(() => {
            expect(screen.getByRole('button', { name: /LLM Configuration/i })).toBeInTheDocument();
        });
        fireEvent.click(screen.getByRole('button', { name: /LLM Configuration/i }));

        await waitFor(() => {
            const apiKeyInput = screen.getByLabelText(/API Key/i);
            fireEvent.contextMenu(apiKeyInput, { clientX: 100, clientY: 100 });
        });

        expect(screen.getByRole('menu')).toBeInTheDocument();
        expect(screen.getByText('Paste')).toBeInTheDocument();
    });

    it('shows custom context menu on right-click of chat input', async () => {
        const mockSources = [{ id: 'ds1', name: 'Sales DB', type: 'sqlite' }];
        (AppBindings.GetConfig as any).mockResolvedValue({ apiKey: 'test-key' });
        (AppBindings.GetDashboardData as any).mockResolvedValue({ metrics: [], insights: [] });
        (AppBindings.GetDataSources as any).mockResolvedValue(mockSources);

        render(<App />);

        // Wait for app to be ready
        await waitFor(() => {
            expect(screen.getByText('Sales DB')).toBeInTheDocument();
        });

        // Select and Start Chat
        fireEvent.click(screen.getByText('Sales DB'));
        fireEvent.click(screen.getByLabelText(/Chat Analysis/i));
        await waitFor(() => {
            expect(screen.getByPlaceholderText(/e.g. Sales Analysis Q1/i)).toBeInTheDocument();
        });
        fireEvent.change(screen.getByPlaceholderText(/e.g. Sales Analysis Q1/i), { target: { value: 'Test Session' } });
        fireEvent.click(screen.getByRole('button', { name: /Start Chat/i }));

        // Now wait for chat input
        await waitFor(() => {
            expect(screen.getByPlaceholderText('Type a message...')).toBeInTheDocument();
        });

        const chatInput = screen.getByPlaceholderText('Type a message...');
        fireEvent.contextMenu(chatInput, { clientX: 200, clientY: 200 });

        expect(screen.getByRole('menu')).toBeInTheDocument();
        expect(screen.getByText('Select All')).toBeInTheDocument();
    });

    it('pastes text into an input field via context menu', async () => {
        (AppBindings.GetConfig as any).mockResolvedValue({ apiKey: 'test-key' });
        (AppBindings.GetDashboardData as any).mockResolvedValue({ metrics: [], insights: [] });
        (AppBindings.GetDataSources as any).mockResolvedValue([]);
        (WailsRuntime.ClipboardGetText as any).mockResolvedValue('Copied Key');

        render(<App />);

        // Wait for app to be ready
        await waitFor(() => {
            expect(screen.getByLabelText(/Settings/i)).toBeInTheDocument();
        });

        // Open settings
        const settingsButton = screen.getByLabelText(/Settings/i);
        fireEvent.click(settingsButton);

        // Switch to LLM tab
        await waitFor(() => {
            expect(screen.getByRole('button', { name: /LLM Configuration/i })).toBeInTheDocument();
        });
        fireEvent.click(screen.getByRole('button', { name: /LLM Configuration/i }));

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

        
        
        
                
        
        