import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, vi } from 'vitest';
import HistoricalSessionsSection, {
    HistoricalSessionsSectionProps,
    SessionItem,
    sortSessionsReverseChronological,
    formatSessionDate,
} from './HistoricalSessionsSection';

describe('HistoricalSessionsSection Component', () => {
    const defaultProps: HistoricalSessionsSectionProps = {
        sessions: [],
        selectedId: null,
        onSelect: vi.fn(),
        onContextMenu: vi.fn(),
    };

    const now = Date.now() / 1000;

    const mockSessions: SessionItem[] = [
        {
            id: 'session1',
            title: 'Sales Analysis',
            data_source_id: 'ds1',
            created_at: now - 7200, // 2 hours ago
            dataSourceName: 'Sales DB',
        },
        {
            id: 'session2',
            title: 'Revenue Report',
            data_source_id: 'ds2',
            created_at: now - 3600, // 1 hour ago
            dataSourceName: 'Revenue Excel',
        },
        {
            id: 'session3',
            title: 'Latest Analysis',
            data_source_id: 'ds1',
            created_at: now, // now
            dataSourceName: 'Sales DB',
        },
    ];

    describe('Header rendering', () => {
        it('should render the "Historical Sessions" title', () => {
            render(<HistoricalSessionsSection {...defaultProps} />);
            expect(screen.getByText('Historical Sessions')).toBeInTheDocument();
        });
    });

    describe('Empty state', () => {
        it('should display empty state when no sessions exist', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={[]} />);
            expect(screen.getByText('No historical sessions')).toBeInTheDocument();
        });

        it('should not render a list when no sessions exist', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={[]} />);
            expect(screen.queryByRole('list')).not.toBeInTheDocument();
        });

        it('should have role="status" on empty state', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={[]} />);
            expect(screen.getByRole('status')).toBeInTheDocument();
        });
    });

    describe('Sessions list rendering', () => {
        it('should render all sessions as list items', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={mockSessions} />);
            expect(screen.getByText('Sales Analysis')).toBeInTheDocument();
            expect(screen.getByText('Revenue Report')).toBeInTheDocument();
            expect(screen.getByText('Latest Analysis')).toBeInTheDocument();
        });

        it('should render exactly one list item per session', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={mockSessions} />);
            const listItems = screen.getAllByRole('listitem');
            expect(listItems).toHaveLength(mockSessions.length);
        });

        it('should render a list container with role="list"', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={mockSessions} />);
            expect(screen.getByRole('list')).toBeInTheDocument();
        });

        it('should not display empty state when sessions exist', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={mockSessions} />);
            expect(screen.queryByText('No historical sessions')).not.toBeInTheDocument();
        });
    });

    describe('Reverse chronological ordering', () => {
        it('should display sessions in reverse chronological order (newest first)', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={mockSessions} />);
            const sessionTitles = screen.getAllByRole('listitem').map(
                (item) => item.querySelector('.session-title')?.textContent
            );
            expect(sessionTitles[0]).toBe('Latest Analysis');
            expect(sessionTitles[1]).toBe('Revenue Report');
            expect(sessionTitles[2]).toBe('Sales Analysis');
        });

        it('should handle sessions already in correct order', () => {
            const orderedSessions: SessionItem[] = [
                { id: 's1', title: 'Newest', data_source_id: 'ds1', created_at: now, dataSourceName: 'DB' },
                { id: 's2', title: 'Oldest', data_source_id: 'ds1', created_at: now - 3600, dataSourceName: 'DB' },
            ];
            render(<HistoricalSessionsSection {...defaultProps} sessions={orderedSessions} />);
            const sessionTitles = screen.getAllByRole('listitem').map(
                (item) => item.querySelector('.session-title')?.textContent
            );
            expect(sessionTitles[0]).toBe('Newest');
            expect(sessionTitles[1]).toBe('Oldest');
        });

        it('should handle sessions with same timestamp', () => {
            const sameTsSessions: SessionItem[] = [
                { id: 's1', title: 'Session A', data_source_id: 'ds1', created_at: now },
                { id: 's2', title: 'Session B', data_source_id: 'ds1', created_at: now },
            ];
            render(<HistoricalSessionsSection {...defaultProps} sessions={sameTsSessions} />);
            const listItems = screen.getAllByRole('listitem');
            expect(listItems).toHaveLength(2);
        });
    });

    describe('Session selection', () => {
        it('should call onSelect with the correct id when a session is clicked', () => {
            const onSelect = vi.fn();
            render(
                <HistoricalSessionsSection
                    {...defaultProps}
                    sessions={mockSessions}
                    onSelect={onSelect}
                />
            );
            fireEvent.click(screen.getByText('Sales Analysis'));
            expect(onSelect).toHaveBeenCalledWith('session1');
        });

        it('should highlight the selected session with "selected" class', () => {
            render(
                <HistoricalSessionsSection
                    {...defaultProps}
                    sessions={mockSessions}
                    selectedId="session2"
                />
            );
            const selectedItem = screen.getByText('Revenue Report').closest('.session-item');
            expect(selectedItem).toHaveClass('selected');
        });

        it('should not highlight unselected sessions', () => {
            render(
                <HistoricalSessionsSection
                    {...defaultProps}
                    sessions={mockSessions}
                    selectedId="session2"
                />
            );
            const unselectedItem = screen.getByText('Sales Analysis').closest('.session-item');
            expect(unselectedItem).not.toHaveClass('selected');
        });

        it('should set aria-selected on the selected item', () => {
            render(
                <HistoricalSessionsSection
                    {...defaultProps}
                    sessions={mockSessions}
                    selectedId="session1"
                />
            );
            const selectedItem = screen.getByText('Sales Analysis').closest('[role="listitem"]');
            expect(selectedItem).toHaveAttribute('aria-selected', 'true');
        });

        it('should set aria-selected=false on unselected items', () => {
            render(
                <HistoricalSessionsSection
                    {...defaultProps}
                    sessions={mockSessions}
                    selectedId="session1"
                />
            );
            const unselectedItem = screen.getByText('Revenue Report').closest('[role="listitem"]');
            expect(unselectedItem).toHaveAttribute('aria-selected', 'false');
        });
    });

    describe('Session metadata display', () => {
        it('should display session title', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={mockSessions} />);
            expect(screen.getByText('Sales Analysis')).toBeInTheDocument();
        });

        it('should display session date', () => {
            const session: SessionItem = {
                id: 's1',
                title: 'Test Session',
                data_source_id: 'ds1',
                created_at: now,
                dataSourceName: 'Test DB',
            };
            render(<HistoricalSessionsSection {...defaultProps} sessions={[session]} />);
            const expectedDate = new Date(now * 1000).toLocaleDateString();
            expect(screen.getByText(expectedDate)).toBeInTheDocument();
        });

        it('should display data source name when available', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={mockSessions} />);
            const sourceElements = screen.getAllByText('Sales DB');
            expect(sourceElements.length).toBeGreaterThan(0);
        });

        it('should not display data source name when not available', () => {
            const sessionWithoutSource: SessionItem[] = [
                {
                    id: 's1',
                    title: 'No Source Session',
                    data_source_id: 'ds1',
                    created_at: now,
                },
            ];
            render(<HistoricalSessionsSection {...defaultProps} sessions={sessionWithoutSource} />);
            const listItem = screen.getByRole('listitem');
            expect(listItem.querySelector('.session-source')).not.toBeInTheDocument();
        });
    });

    describe('Context menu', () => {
        it('should call onContextMenu with event and id on right-click', () => {
            const onContextMenu = vi.fn();
            render(
                <HistoricalSessionsSection
                    {...defaultProps}
                    sessions={mockSessions}
                    onContextMenu={onContextMenu}
                />
            );
            fireEvent.contextMenu(screen.getByText('Sales Analysis'));
            expect(onContextMenu).toHaveBeenCalledTimes(1);
            expect(onContextMenu).toHaveBeenCalledWith(expect.any(Object), 'session1');
        });

        it('should pass the correct session id for each session on right-click', () => {
            const onContextMenu = vi.fn();
            render(
                <HistoricalSessionsSection
                    {...defaultProps}
                    sessions={mockSessions}
                    onContextMenu={onContextMenu}
                />
            );
            fireEvent.contextMenu(screen.getByText('Revenue Report'));
            expect(onContextMenu).toHaveBeenCalledWith(expect.any(Object), 'session2');
        });
    });

    describe('Accessibility', () => {
        it('should have a region role with "Historical Sessions" label', () => {
            render(<HistoricalSessionsSection {...defaultProps} />);
            expect(screen.getByRole('region', { name: 'Historical Sessions' })).toBeInTheDocument();
        });

        it('should set data-session-id attribute on each item', () => {
            render(<HistoricalSessionsSection {...defaultProps} sessions={mockSessions} />);
            const items = screen.getAllByRole('listitem');
            // Items are sorted reverse chronologically, so session3 is first
            expect(items[0]).toHaveAttribute('data-session-id', 'session3');
            expect(items[1]).toHaveAttribute('data-session-id', 'session2');
            expect(items[2]).toHaveAttribute('data-session-id', 'session1');
        });
    });
});

describe('sortSessionsReverseChronological', () => {
    it('should sort sessions newest first', () => {
        const sessions: SessionItem[] = [
            { id: 's1', title: 'Old', data_source_id: 'ds1', created_at: 100 },
            { id: 's2', title: 'New', data_source_id: 'ds1', created_at: 300 },
            { id: 's3', title: 'Mid', data_source_id: 'ds1', created_at: 200 },
        ];
        const sorted = sortSessionsReverseChronological(sessions);
        expect(sorted[0].id).toBe('s2');
        expect(sorted[1].id).toBe('s3');
        expect(sorted[2].id).toBe('s1');
    });

    it('should return empty array for empty input', () => {
        expect(sortSessionsReverseChronological([])).toEqual([]);
    });

    it('should not mutate the original array', () => {
        const sessions: SessionItem[] = [
            { id: 's1', title: 'Old', data_source_id: 'ds1', created_at: 100 },
            { id: 's2', title: 'New', data_source_id: 'ds1', created_at: 300 },
        ];
        const original = [...sessions];
        sortSessionsReverseChronological(sessions);
        expect(sessions).toEqual(original);
    });

    it('should handle single session', () => {
        const sessions: SessionItem[] = [
            { id: 's1', title: 'Only', data_source_id: 'ds1', created_at: 100 },
        ];
        const sorted = sortSessionsReverseChronological(sessions);
        expect(sorted).toHaveLength(1);
        expect(sorted[0].id).toBe('s1');
    });
});

describe('formatSessionDate', () => {
    it('should format a Unix timestamp to a localized date string', () => {
        const timestamp = 1700000000; // Nov 14, 2023 in UTC
        const result = formatSessionDate(timestamp);
        // The exact format depends on locale, but it should be a non-empty string
        expect(result).toBeTruthy();
        expect(typeof result).toBe('string');
    });

    it('should handle timestamp of 0 (epoch)', () => {
        const result = formatSessionDate(0);
        expect(result).toBeTruthy();
    });
});
