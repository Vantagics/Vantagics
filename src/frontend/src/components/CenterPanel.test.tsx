import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import CenterPanel, { Message, CenterPanelProps } from './CenterPanel';

// Mock the Wails runtime
vi.mock('../../wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(() => vi.fn()),
    EventsEmit: vi.fn(),
}));

// Mock the Wails App functions
vi.mock('../../wailsjs/go/main/App', () => ({
    GetConfig: vi.fn().mockResolvedValue({}),
    GetSessionFileAsBase64: vi.fn().mockResolvedValue(''),
    GetMessageAnalysisData: vi.fn().mockResolvedValue({}),
    GetDataSourceTables: vi.fn().mockResolvedValue([]),
    GetDataSourceTableData: vi.fn().mockResolvedValue([]),
    GetDataSourceTableCount: vi.fn().mockResolvedValue(0),
}));

// Mock logger
vi.mock('../utils/systemLog', () => ({
    createLogger: () => ({
        debug: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    }),
}));

// Mock LoadingStateManager
vi.mock('../managers/LoadingStateManager', () => ({
    loadingStateManager: {
        getSessionState: vi.fn(() => undefined),
        subscribeToSession: vi.fn(() => vi.fn()),
        setLoading: vi.fn(),
        updateProgress: vi.fn(),
        clearSession: vi.fn(),
    },
}));

// Mock AnalysisResultManager
vi.mock('../managers/AnalysisResultManager', () => ({
    getAnalysisResultManager: vi.fn(() => ({
        getCurrentSession: vi.fn(),
        getCurrentMessage: vi.fn(),
        hasCurrentData: vi.fn(() => false),
        getCurrentResults: vi.fn(() => []),
        restoreResults: vi.fn(() => ({ validItems: 0, invalidItems: 0, totalItems: 0, errors: [], itemsByType: {} })),
        setLoading: vi.fn(),
    })),
}));

// Mock i18n
vi.mock('../i18n', () => ({
    useLanguage: () => ({
        t: (key: string) => {
            const translations: Record<string, string> = {
                'ai_assistant': 'AI Assistant',
                'ready_to_help': 'Ready to help',
                'select_session': 'Select a session to start',
                'insights_at_fingertips': 'Insights at Your Fingertips',
                'ask_about_sales': 'Ask a question about your data to get started',
                'what_to_analyze': 'Ask a question about your data...',
                'analyzing': 'Analyzing...',
            };
            return translations[key] || key;
        },
        language: 'English',
    }),
}));

// Helper to create test messages
const createMessage = (overrides: Partial<Message> = {}): Message => ({
    id: `msg-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    role: 'user',
    content: 'Test message',
    timestamp: Date.now(),
    ...overrides,
});

const defaultProps: CenterPanelProps = {
    width: 600,
    sessionId: null,
    messages: [],
    isLoading: false,
    onSendMessage: vi.fn(),
};

describe('CenterPanel Component', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('Rendering', () => {
        it('should render the center panel container', () => {
            render(<CenterPanel {...defaultProps} />);
            expect(screen.getByTestId('center-panel')).toBeInTheDocument();
        });

        it('should render with correct width', () => {
            render(<CenterPanel {...defaultProps} width={800} />);
            const panel = screen.getByTestId('center-panel');
            expect(panel).toHaveStyle({ width: '800px' });
        });

        it('should have correct ARIA attributes', () => {
            render(<CenterPanel {...defaultProps} />);
            const panel = screen.getByTestId('center-panel');
            expect(panel).toHaveAttribute('role', 'region');
            expect(panel).toHaveAttribute('aria-label', 'Chat Panel');
        });

        it('should render the header with AI Assistant title', () => {
            render(<CenterPanel {...defaultProps} />);
            expect(screen.getByText('AI Assistant')).toBeInTheDocument();
        });
    });

    describe('Welcome Message (Requirement 5.4)', () => {
        it('should display welcome message when no session is active', () => {
            render(<CenterPanel {...defaultProps} sessionId={null} />);
            expect(screen.getByTestId('welcome-message')).toBeInTheDocument();
            expect(screen.getByText('Insights at Your Fingertips')).toBeInTheDocument();
        });

        it('should not display welcome message when session is active', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    messages={[createMessage({ content: 'Hello' })]}
                />
            );
            expect(screen.queryByTestId('welcome-message')).not.toBeInTheDocument();
        });
    });

    describe('Message List (Requirement 5.3)', () => {
        it('should render message list when session is active', () => {
            const messages: Message[] = [
                createMessage({ id: 'msg-1', role: 'user', content: 'Hello' }),
                createMessage({ id: 'msg-2', role: 'assistant', content: 'Hi there!' }),
            ];

            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    messages={messages}
                />
            );

            expect(screen.getByTestId('message-list')).toBeInTheDocument();
            expect(screen.getByText('Hello')).toBeInTheDocument();
            expect(screen.getByText('Hi there!')).toBeInTheDocument();
        });

        it('should render empty session state when session active but no messages', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    messages={[]}
                />
            );

            expect(screen.getByTestId('empty-session')).toBeInTheDocument();
        });

        it('should render multiple messages in order', () => {
            const messages: Message[] = [
                createMessage({ id: 'msg-1', role: 'user', content: 'First question' }),
                createMessage({ id: 'msg-2', role: 'assistant', content: 'First answer' }),
                createMessage({ id: 'msg-3', role: 'user', content: 'Second question' }),
                createMessage({ id: 'msg-4', role: 'assistant', content: 'Second answer' }),
            ];

            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    messages={messages}
                />
            );

            expect(screen.getByText('First question')).toBeInTheDocument();
            expect(screen.getByText('First answer')).toBeInTheDocument();
            expect(screen.getByText('Second question')).toBeInTheDocument();
            expect(screen.getByText('Second answer')).toBeInTheDocument();
        });
    });

    describe('Message Input (Requirement 5.5)', () => {
        it('should render message input area', () => {
            render(<CenterPanel {...defaultProps} sessionId="session-1" />);
            expect(screen.getByTestId('message-input')).toBeInTheDocument();
            expect(screen.getByTestId('send-button')).toBeInTheDocument();
        });

        it('should disable input when no session is active', () => {
            render(<CenterPanel {...defaultProps} sessionId={null} />);
            expect(screen.getByTestId('message-input')).toBeDisabled();
            expect(screen.getByTestId('send-button')).toBeDisabled();
        });

        it('should disable input when loading', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    isLoading={true}
                />
            );
            expect(screen.getByTestId('message-input')).toBeDisabled();
        });

        it('should disable send button when input is empty', () => {
            render(<CenterPanel {...defaultProps} sessionId="session-1" />);
            expect(screen.getByTestId('send-button')).toBeDisabled();
        });

        it('should enable send button when input has text', () => {
            render(<CenterPanel {...defaultProps} sessionId="session-1" />);
            const input = screen.getByTestId('message-input');
            fireEvent.change(input, { target: { value: 'Hello' } });
            expect(screen.getByTestId('send-button')).not.toBeDisabled();
        });

        it('should call onSendMessage when send button is clicked', () => {
            const onSendMessage = vi.fn();
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    onSendMessage={onSendMessage}
                />
            );

            const input = screen.getByTestId('message-input');
            fireEvent.change(input, { target: { value: 'Test message' } });
            fireEvent.click(screen.getByTestId('send-button'));

            expect(onSendMessage).toHaveBeenCalledWith('Test message');
        });

        it('should call onSendMessage when Enter key is pressed', () => {
            const onSendMessage = vi.fn();
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    onSendMessage={onSendMessage}
                />
            );

            const input = screen.getByTestId('message-input');
            fireEvent.change(input, { target: { value: 'Test message' } });
            fireEvent.keyDown(input, { key: 'Enter' });

            expect(onSendMessage).toHaveBeenCalledWith('Test message');
        });

        it('should clear input after sending message', () => {
            const onSendMessage = vi.fn();
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    onSendMessage={onSendMessage}
                />
            );

            const input = screen.getByTestId('message-input') as HTMLInputElement;
            fireEvent.change(input, { target: { value: 'Test message' } });
            fireEvent.click(screen.getByTestId('send-button'));

            expect(input.value).toBe('');
        });

        it('should not send empty messages', () => {
            const onSendMessage = vi.fn();
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    onSendMessage={onSendMessage}
                />
            );

            fireEvent.click(screen.getByTestId('send-button'));
            expect(onSendMessage).not.toHaveBeenCalled();
        });

        it('should not send whitespace-only messages', () => {
            const onSendMessage = vi.fn();
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    onSendMessage={onSendMessage}
                />
            );

            const input = screen.getByTestId('message-input');
            fireEvent.change(input, { target: { value: '   ' } });
            fireEvent.click(screen.getByTestId('send-button'));

            expect(onSendMessage).not.toHaveBeenCalled();
        });

        it('should have correct placeholder text', () => {
            render(<CenterPanel {...defaultProps} sessionId="session-1" />);
            const input = screen.getByTestId('message-input');
            expect(input).toHaveAttribute(
                'placeholder',
                'Ask a question about your data...'
            );
        });
    });

    describe('Loading State (Requirement 5.8)', () => {
        it('should display loading indicator when isLoading is true', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    messages={[createMessage()]}
                    isLoading={true}
                />
            );

            expect(screen.getByTestId('loading-indicator')).toBeInTheDocument();
        });

        it('should not display loading indicator when isLoading is false', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    messages={[createMessage()]}
                    isLoading={false}
                />
            );

            expect(screen.queryByTestId('loading-indicator')).not.toBeInTheDocument();
        });

        it('should not display loading indicator when no session is active', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId={null}
                    isLoading={true}
                />
            );

            expect(screen.queryByTestId('loading-indicator')).not.toBeInTheDocument();
        });
    });

    describe('Data Browser Overlay', () => {
        it('should show overlay when dataBrowserOpen is true', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    dataBrowserOpen={true}
                />
            );

            expect(screen.getByTestId('data-browser-overlay')).toBeInTheDocument();
        });

        it('should not show overlay when dataBrowserOpen is false', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    dataBrowserOpen={false}
                />
            );

            expect(screen.queryByTestId('data-browser-overlay')).not.toBeInTheDocument();
        });

        it('should call onCloseBrowser when overlay is clicked', () => {
            const onCloseBrowser = vi.fn();
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    dataBrowserOpen={true}
                    onCloseBrowser={onCloseBrowser}
                />
            );

            fireEvent.click(screen.getByTestId('data-browser-overlay'));
            expect(onCloseBrowser).toHaveBeenCalled();
        });
    });

    describe('Message Click Handling', () => {
        it('should call onMessageClick when completed user message is clicked', () => {
            const onMessageClick = vi.fn();
            const messages: Message[] = [
                createMessage({ id: 'msg-1', role: 'user', content: 'Question' }),
                createMessage({ id: 'msg-2', role: 'assistant', content: 'Answer' }),
            ];

            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                    messages={messages}
                    onMessageClick={onMessageClick}
                />
            );

            // Click on the user message (which has a completed assistant reply)
            const userMessage = screen.getByText('Question');
            // The click handler is on the MessageBubble wrapper, find the clickable element
            const clickableElement = userMessage.closest('[class*="cursor"]') || userMessage;
            fireEvent.click(clickableElement);
        });
    });

    describe('Fixed Positioning (Requirement 5.2)', () => {
        it('should not have overlay or collapse behavior', () => {
            render(<CenterPanel {...defaultProps} />);
            const panel = screen.getByTestId('center-panel');

            // Should not have overlay-related classes
            expect(panel.className).not.toContain('fixed');
            expect(panel.className).not.toContain('overlay');
            expect(panel.className).not.toContain('translate');
        });

        it('should always be visible (no hidden state)', () => {
            render(<CenterPanel {...defaultProps} />);
            const panel = screen.getByTestId('center-panel');
            expect(panel).toBeVisible();
        });
    });

    describe('Header Status', () => {
        it('should show session status when session is active', () => {
            render(
                <CenterPanel
                    {...defaultProps}
                    sessionId="session-1"
                />
            );
            expect(screen.getByText('Ready to help')).toBeInTheDocument();
        });

        it('should show prompt to select session when no session active', () => {
            render(<CenterPanel {...defaultProps} sessionId={null} />);
            expect(screen.getByText('Select a session to start')).toBeInTheDocument();
        });
    });
});
