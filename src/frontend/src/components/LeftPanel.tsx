import React, { useState, useEffect, useRef } from 'react';
import { GetChatHistory } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import { createLogger } from '../utils/systemLog';
import NewSessionButton from './NewSessionButton';
import HistoricalSessionsSection from './HistoricalSessionsSection';
import { useTrapFocus } from '../hooks/useTrapFocus';
import { useLoadingState } from '../hooks/useLoadingState';
import { useLanguage } from '../i18n';
import './LeftPanel.css';

const logger = createLogger('LeftPanel');

/**
 * Props interface for LeftPanel component
 * Requirements: 2.3, 4.1
 */
interface LeftPanelProps {
    width: number;                                      // Panel width in pixels
    onSessionSelect: (sessionId: string) => void;      // Callback when session is selected
    onNewSession: () => void;                          // Callback when new session button is clicked
    selectedSessionId: string | null;                  // Currently selected session ID
}

/**
 * Context menu state
 */
interface ContextMenuState {
    x: number;
    y: number;
    type: 'session';
    targetId: string;
}

/**
 * LeftPanel container component
 * 
 * This is the leftmost panel in the three-panel layout.
 * Contains NewSessionButton (top) and HistoricalSessionsSection (bottom).
 * 
 * Requirements: 1.1, 2.1, 2.2, 2.3, 4.1
 */
const LeftPanel: React.FC<LeftPanelProps> = ({
    width,
    onSessionSelect,
    onNewSession,
    selectedSessionId
}) => {
    // Ref for focus trap - Requirements: 11.7
    const panelRef = useRef<HTMLDivElement>(null);
    useTrapFocus(panelRef);
    const { t } = useLanguage();
    const { isLoading: isSessionAnalysisLoading } = useLoadingState();

    // State management
    const [sessions, setSessions] = useState<main.ChatThread[]>([]);
    const [isLoadingSessions, setIsLoadingSessions] = useState(false);
    const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);

    /**
     * Fetch historical sessions from backend
     * Requirements: 3.1, 3.2
     */
    const fetchSessions = async () => {
        setIsLoadingSessions(true);
        try {
            logger.debug('Fetching chat history');
            const history = await GetChatHistory();
            // Sort sessions in reverse chronological order (newest first)
            // Requirements: 3.7 - Validates Property 5
            const sortedSessions = (history || []).sort((a, b) => b.created_at - a.created_at);
            setSessions(sortedSessions);
            logger.debug(`Loaded ${sortedSessions.length} sessions`);
        } catch (error) {
            logger.error('Failed to fetch sessions:', error);
            setSessions([]);
        } finally {
            setIsLoadingSessions(false);
        }
    };

    /**
     * Handle session context menu (right-click)
     * Requirements: 3.6
     */
    const handleSessionContextMenu = (e: React.MouseEvent, sessionId: string) => {
        e.preventDefault();
        e.stopPropagation();
        logger.debug(`Context menu opened for session: ${sessionId}`);
        setContextMenu({
            x: e.clientX,
            y: e.clientY,
            type: 'session',
            targetId: sessionId
        });
    };

    /**
     * Handle new session button click
     * Requirements: 4.1, 4.2
     */
    const handleNewSessionClick = () => {
        logger.debug('New session button clicked');
        onNewSession();
    };

    // Initial data loading
    useEffect(() => {
        fetchSessions();
    }, []);

    // Listen for session events
    useEffect(() => {
        const unsubscribeCreated = EventsOn('chat-thread-created', () => {
            logger.debug('Chat thread created event received');
            fetchSessions();
        });

        const unsubscribeDeleted = EventsOn('chat-thread-deleted', () => {
            logger.debug('Chat thread deleted event received');
            fetchSessions();
        });

        const unsubscribeUpdated = EventsOn('chat-thread-updated', () => {
            logger.debug('Chat thread updated event received');
            fetchSessions();
        });

        return () => {
            if (unsubscribeCreated) unsubscribeCreated();
            if (unsubscribeDeleted) unsubscribeDeleted();
            if (unsubscribeUpdated) unsubscribeUpdated();
        };
    }, []);

    // Close context menu when clicking outside
    useEffect(() => {
        const handleClickOutside = () => {
            if (contextMenu) {
                setContextMenu(null);
            }
        };

        if (contextMenu) {
            document.addEventListener('click', handleClickOutside);
            return () => document.removeEventListener('click', handleClickOutside);
        }
    }, [contextMenu]);

    return (
        <div 
            ref={panelRef}
            data-testid="left-panel"
            className="left-panel"
            style={{ width: `${width}px` }}
            role="region"
            aria-label={t('data_sources_panel')}
            tabIndex={-1}
        >
            {/* NewSessionButton component - Requirements: 2.3, 4.1 */}
            <NewSessionButton
                onClick={handleNewSessionClick}
            />

            {/* HistoricalSessionsSection component - Requirements: 3.1, 3.2, 3.3, 3.5, 3.6, 3.7, 3.8 */}
            {isLoadingSessions ? (
                <div className="historical-sessions-section">
                    <div className="section-header">
                        <h3>{t('historical_sessions')}</h3>
                    </div>
                    <div className="loading">{t('loading_sessions')}</div>
                </div>
            ) : (
                <HistoricalSessionsSection
                    sessions={sessions.map(session => ({
                        id: session.id,
                        title: session.title,
                        data_source_id: session.data_source_id,
                        created_at: session.created_at,
                    }))}
                    selectedId={selectedSessionId}
                    onSelect={onSessionSelect}
                    onContextMenu={handleSessionContextMenu}
                    isSessionLoading={isSessionAnalysisLoading}
                />
            )}

            {/* Context Menu */}
            {contextMenu && (
                <div
                    className="context-menu"
                    style={{
                        position: 'fixed',
                        left: `${contextMenu.x}px`,
                        top: `${contextMenu.y}px`,
                        zIndex: 1000
                    }}
                >
                    <div className="context-menu-item">{t('context_menu_rename')}</div>
                    <div className="context-menu-item">{t('context_menu_delete')}</div>
                    <div className="context-menu-item">{t('context_menu_export')}</div>
                </div>
            )}
        </div>
    );
};

export default LeftPanel;
