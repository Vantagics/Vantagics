import React, { useRef } from 'react';
import DraggableDashboard from './DraggableDashboard';
import { main } from '../../wailsjs/go/models';
import { useTrapFocus } from '../hooks/useTrapFocus';
import { useLanguage } from '../i18n';
import './RightPanel.css';

/**
 * RightPanel Props Interface
 * Requirements: 6.1, 6.2, 6.5
 */
export interface RightPanelProps {
    /** Panel width in pixels */
    width: number;
    /** Callback when width changes */
    onWidthChange: (width: number) => void;
    /** Dashboard data with metrics and insights */
    dashboardData: main.DashboardData | null;
    /** Currently active chart data */
    activeChart: { type: 'echarts' | 'image' | 'table' | 'csv'; data: any; chartData?: main.ChartData } | null;
    /** Files associated with the current session */
    sessionFiles: main.SessionFile[];
    /** Currently selected message ID */
    selectedMessageId: string | null;
    /** Callback when an insight is clicked */
    onInsightClick: (insight: string) => void;
    /** Active thread/session ID */
    activeThreadId?: string | null;
    /** Whether the chat is open (legacy compatibility) */
    isChatOpen?: boolean;
    /** User request text for display */
    userRequestText?: string | null;
}

/**
 * RightPanel Component
 *
 * Thin wrapper around DraggableDashboard that provides fixed positioning
 * on the right side of the three-panel layout. Displays metrics, charts,
 * insights, and analysis results.
 *
 * Requirements:
 * - 6.1: Display the dashboard with metrics, charts, insights, and analysis results
 * - 6.2: Remain visible at all times (no overlay or collapse behavior)
 * - 6.5: Be scrollable when content exceeds available height
 */
const RightPanel: React.FC<RightPanelProps> = ({
    width,
    onWidthChange,
    dashboardData,
    activeChart,
    sessionFiles,
    selectedMessageId,
    onInsightClick,
    activeThreadId = null,
    isChatOpen = false,
    userRequestText = null,
}) => {
    const panelRef = useRef<HTMLDivElement>(null);
    const { t } = useLanguage();

    // Focus trap for tab navigation within panel - Requirements: 11.7
    useTrapFocus(panelRef);

    return (
        <div
            ref={panelRef}
            data-testid="right-panel"
            className="right-panel"
            style={{ width: `${width}px` }}
            role="region"
            aria-label={t('dashboard_panel')}
            tabIndex={-1}
        >
            <div className="right-panel-content">
                <DraggableDashboard
                    data={dashboardData}
                    activeChart={activeChart}
                    userRequestText={userRequestText}
                    isChatOpen={isChatOpen}
                    activeThreadId={activeThreadId}
                    sessionFiles={sessionFiles}
                    selectedMessageId={selectedMessageId}
                    onInsightClick={onInsightClick}
                />
            </div>
        </div>
    );
};

export default RightPanel;
