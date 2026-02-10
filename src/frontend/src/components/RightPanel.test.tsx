import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import RightPanel, { RightPanelProps } from './RightPanel';

// Mock DraggableDashboard since RightPanel is a thin wrapper
// We test the wrapper behavior, not the dashboard itself
vi.mock('./DraggableDashboard', () => ({
    default: (props: any) => (
        <div data-testid="draggable-dashboard">
            {props.sessionFiles?.length > 0 && (
                <span data-testid="session-files-count">{props.sessionFiles.length}</span>
            )}
            {props.selectedMessageId && (
                <span data-testid="selected-message-id">{props.selectedMessageId}</span>
            )}
            {props.onInsightClick && (
                <button
                    data-testid="insight-click-trigger"
                    onClick={() => props.onInsightClick('test insight')}
                >
                    Trigger Insight
                </button>
            )}
            {props.activeThreadId && (
                <span data-testid="active-thread-id">{props.activeThreadId}</span>
            )}
        </div>
    ),
}));

const defaultProps: RightPanelProps = {
    width: 384,
    onWidthChange: vi.fn(),
    dashboardData: null,
    activeChart: null,
    sessionFiles: [],
    selectedMessageId: null,
    onInsightClick: vi.fn(),
};

describe('RightPanel Component', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('Rendering', () => {
        it('should render the right panel container', () => {
            render(<RightPanel {...defaultProps} />);
            expect(screen.getByTestId('right-panel')).toBeInTheDocument();
        });

        it('should render with correct width', () => {
            render(<RightPanel {...defaultProps} width={500} />);
            const panel = screen.getByTestId('right-panel');
            expect(panel).toHaveStyle({ width: '500px' });
        });

        it('should render DraggableDashboard inside', () => {
            render(<RightPanel {...defaultProps} />);
            expect(screen.getByTestId('draggable-dashboard')).toBeInTheDocument();
        });

        it('should have correct ARIA attributes', () => {
            render(<RightPanel {...defaultProps} />);
            const panel = screen.getByTestId('right-panel');
            expect(panel).toHaveAttribute('role', 'region');
            expect(panel).toHaveAttribute('aria-label', 'Dashboard Panel');
        });
    });

    describe('Fixed Positioning (Requirement 6.2)', () => {
        it('should always be visible (no hidden or overlay state)', () => {
            render(<RightPanel {...defaultProps} />);
            const panel = screen.getByTestId('right-panel');
            expect(panel).toBeVisible();
        });

        it('should not have overlay or collapse classes', () => {
            render(<RightPanel {...defaultProps} />);
            const panel = screen.getByTestId('right-panel');
            expect(panel.className).not.toContain('overlay');
            expect(panel.className).not.toContain('collapse');
            expect(panel.className).not.toContain('hidden');
        });
    });

    describe('Scrollable Content (Requirement 6.5)', () => {
        it('should have a scrollable content wrapper', () => {
            render(<RightPanel {...defaultProps} />);
            const panel = screen.getByTestId('right-panel');
            const contentWrapper = panel.querySelector('.right-panel-content');
            expect(contentWrapper).toBeInTheDocument();
        });

        it('should contain DraggableDashboard within the scrollable wrapper', () => {
            render(<RightPanel {...defaultProps} />);
            const panel = screen.getByTestId('right-panel');
            const contentWrapper = panel.querySelector('.right-panel-content');
            const dashboard = screen.getByTestId('draggable-dashboard');
            expect(contentWrapper).toContainElement(dashboard);
        });
    });

    describe('Dashboard Props Passthrough (Requirement 6.1)', () => {
        it('should pass sessionFiles to DraggableDashboard', () => {
            const sessionFiles = [
                { name: 'file1.csv', path: '/path/file1.csv', type: 'csv', size: 1024, created_at: Date.now() },
            ] as any[];

            render(<RightPanel {...defaultProps} sessionFiles={sessionFiles} />);
            expect(screen.getByTestId('session-files-count')).toHaveTextContent('1');
        });

        it('should pass selectedMessageId to DraggableDashboard', () => {
            render(<RightPanel {...defaultProps} selectedMessageId="msg-123" />);
            expect(screen.getByTestId('selected-message-id')).toHaveTextContent('msg-123');
        });

        it('should pass onInsightClick to DraggableDashboard', () => {
            const onInsightClick = vi.fn();
            render(<RightPanel {...defaultProps} onInsightClick={onInsightClick} />);

            screen.getByTestId('insight-click-trigger').click();
            expect(onInsightClick).toHaveBeenCalledWith('test insight');
        });

        it('should pass activeThreadId to DraggableDashboard', () => {
            render(<RightPanel {...defaultProps} activeThreadId="thread-456" />);
            expect(screen.getByTestId('active-thread-id')).toHaveTextContent('thread-456');
        });
    });

    describe('Width Changes', () => {
        it('should update width when prop changes', () => {
            const { rerender } = render(<RightPanel {...defaultProps} width={384} />);
            expect(screen.getByTestId('right-panel')).toHaveStyle({ width: '384px' });

            rerender(<RightPanel {...defaultProps} width={500} />);
            expect(screen.getByTestId('right-panel')).toHaveStyle({ width: '500px' });
        });

        it('should render correctly at minimum width', () => {
            render(<RightPanel {...defaultProps} width={280} />);
            expect(screen.getByTestId('right-panel')).toHaveStyle({ width: '280px' });
        });

        it('should render correctly at maximum width', () => {
            render(<RightPanel {...defaultProps} width={600} />);
            expect(screen.getByTestId('right-panel')).toHaveStyle({ width: '600px' });
        });
    });
});
