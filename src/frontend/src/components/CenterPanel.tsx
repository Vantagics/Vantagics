import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Send, MessageSquare, Loader2 } from 'lucide-react';
import MessageBubble from './MessageBubble';
import DataBrowser from './DataBrowser';
import { useLanguage } from '../i18n';
import { AnalysisStatusIndicator } from './AnalysisStatusIndicator';
import { useSessionStatus } from '../hooks/useSessionStatus';
import { useTrapFocus } from '../hooks/useTrapFocus';
import './CenterPanel.css';

/**
 * Message type used by CenterPanel.
 * Compatible with main.ChatMessage but can also be a plain object.
 */
export interface Message {
    id: string;
    role: string;
    content: string;
    timestamp: number;
    chart_data?: any;
    has_analysis_data?: boolean;
    timing_data?: Record<string, any>;
}

/**
 * CenterPanel Props Interface
 * Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8
 */
export interface CenterPanelProps {
    /** Panel width in pixels */
    width: number;
    /** Active session ID, null when no session is active */
    sessionId: string | null;
    /** Messages for the current session */
    messages: Message[];
    /** Whether analysis is currently loading */
    isLoading: boolean;
    /** Callback when user sends a message */
    onSendMessage: (text: string) => void;
    /** Callback when user clicks a message */
    onMessageClick?: (messageId: string) => void;
    /** Whether the data browser overlay is open */
    dataBrowserOpen?: boolean;
    /** Data source ID for the data browser */
    dataBrowserSourceId?: string | null;
    /** Callback to close the data browser */
    onCloseBrowser?: () => void;
    /** Optional: data source ID for the active session */
    dataSourceId?: string;
    /** Optional: thread ID for the active session */
    threadId?: string;
    /** Optional: callback for cancelling analysis */
    onCancelAnalysis?: () => void;
}

/**
 * CenterPanel Component
 *
 * Fixed chat interface panel that replaces the overlay-based ChatSidebar.
 * Displays conversation history, message input, loading states, and welcome message.
 *
 * Requirements:
 * - 5.1: Display chat/conversation interface as primary content
 * - 5.2: Remain visible at all times (no overlay or collapse behavior)
 * - 5.3: Display conversation history when session is active
 * - 5.4: Display welcome message when no session is active
 * - 5.5: Include message input area at bottom
 * - 5.6: Display sent messages immediately
 * - 5.7: Auto-scroll to latest message
 * - 5.8: Display loading indicators during analysis
 */
const CenterPanel: React.FC<CenterPanelProps> = ({
    width,
    sessionId,
    messages,
    isLoading,
    onSendMessage,
    onMessageClick,
    dataBrowserOpen = false,
    dataBrowserSourceId = null,
    onCloseBrowser,
    dataSourceId,
    threadId,
    onCancelAnalysis,
}) => {
    const { t } = useLanguage();
    const [inputText, setInputText] = useState('');
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const messagesContainerRef = useRef<HTMLDivElement>(null);
    const panelRef = useRef<HTMLDivElement>(null);

    // Focus trap for tab navigation within panel - Requirements: 11.7
    useTrapFocus(panelRef);

    // Data browser width state - Requirement 7.8: default 60-70% of center area
    const [dataBrowserWidth, setDataBrowserWidth] = useState(() =>
        Math.round(width * 0.65)
    );

    // Use session status hook for loading indicator
    const sessionStatus = useSessionStatus(sessionId);

    /**
     * Auto-scroll to the latest message when new messages arrive.
     * Requirement 5.7: Auto-scroll to show the latest message when new messages arrive
     */
    const scrollToBottom = useCallback(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, []);

    useEffect(() => {
        const timeoutId = setTimeout(scrollToBottom, 100);
        return () => clearTimeout(timeoutId);
    }, [messages, isLoading, scrollToBottom]);

    /**
     * Handle sending a message.
     * Requirement 5.6: Display sent messages immediately
     */
    const handleSendMessage = useCallback(() => {
        const trimmed = inputText.trim();
        if (!trimmed || isLoading) return;

        onSendMessage(trimmed);
        setInputText('');
    }, [inputText, isLoading, onSendMessage]);

    /**
     * Handle key down in the input field.
     * Enter key sends the message.
     */
    const handleKeyDown = useCallback(
        (e: React.KeyboardEvent<HTMLInputElement>) => {
            if (e.key === 'Enter' && !isLoading) {
                handleSendMessage();
            } else {
                e.stopPropagation();
            }
        },
        [handleSendMessage, isLoading]
    );

    /**
     * Handle clicking on a user message to view its analysis results.
     */
    const handleMessageClick = useCallback(
        (msg: Message) => {
            if (onMessageClick && msg.role === 'user') {
                onMessageClick(msg.id);
            }
        },
        [onMessageClick]
    );

    /**
     * Check if a user message has been completed (has an assistant reply).
     */
    const isMessageCompleted = useCallback(
        (msg: Message, index: number): boolean => {
            if (msg.role !== 'user') return false;

            // Check if there's a following assistant reply
            if (index < messages.length - 1) {
                const nextMsg = messages[index + 1];
                if (nextMsg.role === 'assistant') return true;
            }

            // Check if message has analysis data
            if (msg.chart_data || msg.has_analysis_data) return true;

            return false;
        },
        [messages]
    );

    /**
     * Render the welcome message when no session is active.
     * Requirement 5.4: Display welcome message when no session active
     */
    const renderWelcomeMessage = () => (
        <div
            data-testid="welcome-message"
            className="center-panel-welcome"
        >
            <div className="center-panel-welcome-icon">
                <MessageSquare className="w-10 h-10 text-blue-500" />
            </div>
            <h4 className="center-panel-welcome-title">
                {t('insights_at_fingertips')}
            </h4>
            <p className="center-panel-welcome-text">
                {t('ask_about_sales')}
            </p>
        </div>
    );

    /**
     * Render the message list.
     * Requirement 5.3: Display conversation history when session is active
     */
    const renderMessages = () => (
        <div
            ref={messagesContainerRef}
            data-testid="message-list"
            className="center-panel-messages"
            aria-live="polite"
            aria-relevant="additions"
        >
            {messages.map((msg, index) => {
                const completed = isMessageCompleted(msg, index);

                // Find timing data for user messages
                let timingData = null;
                if (msg.role === 'user' && index < messages.length - 1) {
                    const nextMsg = messages[index + 1];
                    if (nextMsg.role === 'assistant') {
                        timingData = nextMsg.timing_data;
                    }
                }

                return (
                    <MessageBubble
                        key={msg.id || index}
                        role={msg.role as 'user' | 'assistant'}
                        content={msg.content}
                        messageId={msg.id}
                        dataSourceId={dataSourceId}
                        threadId={threadId || sessionId || undefined}
                        onClick={
                            msg.role === 'user' && completed
                                ? () => handleMessageClick(msg)
                                : undefined
                        }
                        hasChart={
                            msg.role === 'user' &&
                            !!(msg.chart_data || msg.has_analysis_data)
                        }
                        isDisabled={msg.role === 'user' && !completed}
                        timingData={
                            msg.role === 'user' ? timingData : msg.timing_data
                        }
                    />
                );
            })}

            {/* Loading indicator - Requirement 5.8 */}
            {sessionId && isLoading && sessionStatus.isLoading && (
                <div data-testid="loading-indicator" className="center-panel-loading" role="status" aria-live="assertive" aria-busy="true">
                    <AnalysisStatusIndicator
                        threadId={sessionId}
                        variant="full"
                        showMessage={true}
                        showProgress={true}
                        showCancelButton={!!onCancelAnalysis}
                        onCancel={onCancelAnalysis}
                        className="mx-auto max-w-md animate-in fade-in slide-in-from-bottom-2 duration-300"
                    />
                </div>
            )}

            {/* Simple loading fallback when sessionStatus is not yet tracking */}
            {sessionId && isLoading && !sessionStatus.isLoading && (
                <div data-testid="loading-indicator" className="center-panel-loading-simple" role="status" aria-live="assertive" aria-busy="true">
                    <Loader2 className="w-6 h-6 text-blue-500 animate-spin" />
                    <span className="text-sm text-slate-500 ml-2">
                        {t('analyzing')}
                    </span>
                </div>
            )}

            <div ref={messagesEndRef} />
        </div>
    );

    return (
        <div
            ref={panelRef}
            data-testid="center-panel"
            className="center-panel"
            style={{ width: '100%' }}
            role="region"
            aria-label={t('chat_panel')}
            tabIndex={-1}
        >
            {/* Header */}
            <div className="center-panel-header">
                <div className="center-panel-header-icon">
                    <MessageSquare className="w-5 h-5 text-white" />
                </div>
                <div className="center-panel-header-info">
                    <h3 className="center-panel-header-title">
                        {sessionId && !dataSourceId ? (t('free_chat')) : (t('ai_assistant'))}
                    </h3>
                    <div className="center-panel-header-status">
                        <span className="center-panel-status-dot" />
                        <p className="center-panel-status-text">
                            {sessionId
                                ? t('ready_to_help')
                                : t('select_session')}
                        </p>
                    </div>
                </div>
            </div>

            {/* Content Area */}
            <div className="center-panel-content">
                {sessionId ? renderMessages() : renderWelcomeMessage()}

                {/* Empty session state - session active but no messages */}
                {sessionId && messages.length === 0 && !isLoading && (
                    <div
                        data-testid="empty-session"
                        className="center-panel-empty-session"
                    >
                        <div className="center-panel-empty-icon">
                            <MessageSquare className="w-8 h-8 text-blue-500" />
                        </div>
                        <p className="center-panel-empty-text">
                            {t('ask_about_sales') ||
                                'Ask a question about your data to get started'}
                        </p>
                    </div>
                )}
            </div>

            {/* Data Browser Overlay Dimming - Requirement 7.9 */}
            {dataBrowserOpen && (
                <div
                    data-testid="data-browser-overlay"
                    className="center-panel-overlay"
                    onClick={onCloseBrowser}
                />
            )}

            {/* Data Browser Slide-Out Panel - Requirements 7.1, 7.2, 7.3 */}
            <DataBrowser
                isOpen={dataBrowserOpen}
                sourceId={dataBrowserSourceId}
                onClose={onCloseBrowser || (() => {})}
                width={dataBrowserWidth}
                onWidthChange={setDataBrowserWidth}
            />

            {/* Message Input Area - Requirement 5.5 */}
            <div className="center-panel-input-area" data-testid="message-input-area">
                <div className="center-panel-input-container">
                    <input
                        type="text"
                        data-testid="message-input"
                        value={inputText}
                        onChange={(e) => setInputText(e.target.value)}
                        onKeyDown={handleKeyDown}
                        placeholder={
                            t('what_to_analyze') ||
                            'Ask a question about your data...'
                        }
                        disabled={isLoading || !sessionId}
                        className="center-panel-input"
                        aria-label={t('message_input')}
                    />
                    <button
                        data-testid="send-button"
                        onClick={handleSendMessage}
                        disabled={isLoading || !inputText.trim() || !sessionId}
                        className="center-panel-send-button"
                        aria-label={t('send_message')}
                    >
                        <Send className="w-5 h-5" />
                    </button>
                </div>
            </div>
        </div>
    );
};

export default CenterPanel;
