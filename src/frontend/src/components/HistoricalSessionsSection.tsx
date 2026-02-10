import React from 'react';
import { Trash2, BarChart3, History, Loader2 } from 'lucide-react';
import { useLanguage } from '../i18n';

/**
 * Session item shape used by HistoricalSessionsSection.
 * Compatible with main.ChatThread from the Wails models.
 * Requirements: 3.1, 3.2, 3.5
 */
export interface SessionItem {
    id: string;
    title: string;
    data_source_id: string;
    created_at: number;
    dataSourceName?: string;
}

/**
 * Props interface for HistoricalSessionsSection component
 * Requirements: 3.1, 3.2, 3.3, 3.5, 3.6, 3.7, 3.8
 */
export interface HistoricalSessionsSectionProps {
    sessions: SessionItem[];
    selectedId: string | null;
    onSelect: (id: string) => void;
    onContextMenu: (e: React.MouseEvent, id: string) => void;
    onDelete?: (id: string, title: string) => void;
    freeChatThreadId?: string | null;
    /** Check whether a given session is currently running an analysis */
    isSessionLoading?: (id: string) => boolean;
}

/**
 * Sort sessions in reverse chronological order (newest first).
 * Requirements: 3.7 - Historical Sessions list SHALL display sessions in reverse chronological order
 * Validates: Property 5 (Session chronological ordering)
 */
export function sortSessionsReverseChronological(sessions: SessionItem[]): SessionItem[] {
    return [...sessions].sort((a, b) => b.created_at - a.created_at);
}

/**
 * Format a Unix timestamp (seconds) to a localized date string.
 */
export function formatSessionDate(timestampSeconds: number): string {
    return new Date(timestampSeconds * 1000).toLocaleDateString();
}

/**
 * HistoricalSessionsSection component
 *
 * Displays the list of previous analysis sessions with selection, context menu,
 * metadata display, and empty state support. Sessions are rendered in reverse
 * chronological order (newest first).
 *
 * Requirements: 3.1, 3.2, 3.3, 3.5, 3.6, 3.7, 3.8
 * Validates: Property 5 (Session chronological ordering), Property 7 (Session metadata completeness),
 *            Property 8 (Selection state consistency), Property 9 (Context menu trigger)
 */
const HistoricalSessionsSection: React.FC<HistoricalSessionsSectionProps> = ({
    sessions,
    selectedId,
    onSelect,
    onContextMenu,
    onDelete,
    freeChatThreadId,
    isSessionLoading,
}) => {
    const { t } = useLanguage();

    // Sort sessions in reverse chronological order, excluding free chat thread
    const sortedSessions = React.useMemo(() => {
        const filtered = freeChatThreadId
            ? sessions.filter(s => s.id !== freeChatThreadId)
            : sessions;
        return sortSessionsReverseChronological(filtered);
    }, [sessions, freeChatThreadId]);

    return (
        <div className="historical-sessions-section" role="region" aria-label={t('historical_sessions')}>
            {/* Header with title - Requirements: 3.1 */}
            <div className="section-header">
                <h3 style={{ display: 'flex', alignItems: 'center', gap: '6px' }}><History className="w-4 h-4 text-blue-500" />{t('historical_sessions')}</h3>
            </div>

            {/* Sessions list or empty state */}
            {sortedSessions.length === 0 ? (
                /* Empty state - Requirements: 3.8 */
                <div className="empty-state" role="status">
                    {t('no_historical_sessions')}
                </div>
            ) : (
                /* Scrollable list - Requirements: 3.2 */
                <div className="sessions-list" role="list">
                    {sortedSessions.map((session) => (
                        <div
                            key={session.id}
                            className={`session-item group ${selectedId === session.id ? 'selected' : ''}`}
                            onClick={() => onSelect(session.id)}
                            onContextMenu={(e) => {
                                e.preventDefault();
                                onContextMenu(e, session.id);
                            }}
                            role="listitem"
                            aria-selected={selectedId === session.id}
                            data-session-id={session.id}
                        >
                            <div className="session-item-content">
                                {/* Session name - Requirements: 3.5 */}
                                <div className="session-title" style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                    {isSessionLoading?.(session.id) ? (
                                        <Loader2 className="flex-shrink-0 w-3.5 h-3.5 text-blue-500 animate-spin" />
                                    ) : (
                                        <BarChart3 className="flex-shrink-0 w-3.5 h-3.5 text-blue-400" />
                                    )}
                                    <span className="truncate">
                                        {session.title}
                                        {session.dataSourceName && ` (${session.dataSourceName})`}
                                    </span>
                                </div>
                                {/* Session metadata (date, data source) - Requirements: 3.5 */}
                                <div className="session-metadata">
                                    <span className="session-date">
                                        {formatSessionDate(session.created_at)}
                                    </span>
                                    {session.dataSourceName && (
                                        <span className="session-source">{session.dataSourceName}</span>
                                    )}
                                </div>
                            </div>
                            {onDelete && (
                                <button
                                    onClick={(e) => {
                                        e.preventDefault();
                                        e.stopPropagation();
                                        onDelete(session.id, session.title);
                                    }}
                                    className="session-delete-btn"
                                    title={t('delete_session')}
                                    aria-label={`${t('delete_session')} ${session.title}`}
                                >
                                    <Trash2 className="w-3 h-3" />
                                </button>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

export default HistoricalSessionsSection;
