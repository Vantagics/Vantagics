import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import LeftPanel from './LeftPanel';
import { GetChatHistory } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Mock the Wails functions
vi.mock('../../wailsjs/go/main/App', () => ({
    GetChatHistory: vi.fn(),
    GetConfig: vi.fn(() => Promise.resolve({ language: 'English' })),
}));

// Mock EventsOn
vi.mock('../../wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(() => vi.fn()),
}));

// Mock i18n
vi.mock('../i18n', () => ({
    useLanguage: () => ({
        language: 'English',
        t: (key: string) => {
            const translations: Record<string, string> = {
                'historical_sessions': 'Historical Sessions',
                'no_historical_sessions': 'No historical sessions',
                'new_session': 'New Session',
                'loading_sessions': 'Loading sessions...',
                'data_sources_panel': 'Data Sources Panel',
                'delete_session': 'Delete session',
                'context_menu_rename': 'Rename',
                'context_menu_delete': 'Delete',
                'context_menu_export': 'Export',
            };
            return translations[key] || key;
        },
    }),
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

describe('LeftPanel Component', () => {
    const mockProps = {
        width: 256,
        onSessionSelect: vi.fn(),
        onNewSession: vi.fn(),
        selectedSessionId: null,
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('New Session Button', () => {
        it('should render new session button', async () => {
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.getByText('New Session')).toBeInTheDocument();
            });
        });

        it('should call onNewSession when clicked', async () => {
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                const button = screen.getByText('New Session').closest('button');
                expect(button).not.toBeDisabled();
            });

            const button = screen.getByText('New Session').closest('button')!;
            fireEvent.click(button);

            expect(mockProps.onNewSession).toHaveBeenCalled();
        });
    });

    describe('Historical Sessions Section', () => {
        it('should render historical sessions section with header', async () => {
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            expect(screen.getByText('Historical Sessions')).toBeInTheDocument();
        });

        it('should display loading state while fetching sessions', () => {
            (GetChatHistory as any).mockImplementation(() => new Promise(() => {}));

            render(<LeftPanel {...mockProps} />);

            expect(screen.getByText('Loading sessions...')).toBeInTheDocument();
        });

        it('should display empty state when no sessions exist', async () => {
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.getByText('No historical sessions')).toBeInTheDocument();
            });
        });

        it('should render sessions list when sessions exist', async () => {
            const mockSessions: main.ChatThread[] = [
                {
                    id: 'session1',
                    title: 'Analysis Session 1',
                    data_source_id: 'ds1',
                    created_at: Date.now() / 1000,
                    messages: [],
                },
                {
                    id: 'session2',
                    title: 'Analysis Session 2',
                    data_source_id: 'ds1',
                    created_at: Date.now() / 1000 - 3600,
                    messages: [],
                },
            ];

            (GetChatHistory as any).mockResolvedValue(mockSessions);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.getByText('Analysis Session 1')).toBeInTheDocument();
                expect(screen.getByText('Analysis Session 2')).toBeInTheDocument();
            });
        });

        it('should sort sessions in reverse chronological order', async () => {
            const now = Date.now() / 1000;
            const mockSessions: main.ChatThread[] = [
                {
                    id: 'session1',
                    title: 'Older Session',
                    data_source_id: 'ds1',
                    created_at: now - 7200, // 2 hours ago
                    messages: [],
                },
                {
                    id: 'session2',
                    title: 'Newer Session',
                    data_source_id: 'ds1',
                    created_at: now - 3600, // 1 hour ago
                    messages: [],
                },
                {
                    id: 'session3',
                    title: 'Newest Session',
                    data_source_id: 'ds1',
                    created_at: now, // now
                    messages: [],
                },
            ];

            (GetChatHistory as any).mockResolvedValue(mockSessions);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                const sessionTitles = screen.getAllByText(/Session/).filter(el => 
                    el.className === 'session-title'
                );
                expect(sessionTitles[0]).toHaveTextContent('Newest Session');
                expect(sessionTitles[1]).toHaveTextContent('Newer Session');
                expect(sessionTitles[2]).toHaveTextContent('Older Session');
            });
        });

        it('should call onSessionSelect when session is clicked', async () => {
            const mockSessions: main.ChatThread[] = [
                {
                    id: 'session1',
                    title: 'Test Session',
                    data_source_id: 'ds1',
                    created_at: Date.now() / 1000,
                    messages: [],
                },
            ];

            (GetChatHistory as any).mockResolvedValue(mockSessions);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.getByText('Test Session')).toBeInTheDocument();
            });

            fireEvent.click(screen.getByText('Test Session'));

            expect(mockProps.onSessionSelect).toHaveBeenCalledWith('session1');
        });

        it('should highlight selected session', async () => {
            const mockSessions: main.ChatThread[] = [
                {
                    id: 'session1',
                    title: 'Test Session',
                    data_source_id: 'ds1',
                    created_at: Date.now() / 1000,
                    messages: [],
                },
            ];

            (GetChatHistory as any).mockResolvedValue(mockSessions);

            const { rerender } = render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.getByText('Test Session')).toBeInTheDocument();
            });

            // Rerender with selected session
            rerender(<LeftPanel {...mockProps} selectedSessionId="session1" />);

            const selectedItem = screen.getByText('Test Session').closest('.session-item');
            expect(selectedItem).toHaveClass('selected');
        });
    });

    describe('Data Sources Removal', () => {
        it('should not render DataSourcesSection', async () => {
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.queryByText('Data Sources')).not.toBeInTheDocument();
            });
        });

        it('should not show data source loading state', async () => {
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            expect(screen.queryByText('Loading data sources...')).not.toBeInTheDocument();
        });

        it('should not render Browse Data context menu item', async () => {
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.queryByText('Browse Data')).not.toBeInTheDocument();
            });
        });
    });

    describe('Context Menu', () => {
        it('should show session context menu with Rename, Delete, Export options', async () => {
            const mockSessions: main.ChatThread[] = [
                {
                    id: 'session1',
                    title: 'Test Session',
                    data_source_id: 'ds1',
                    created_at: Date.now() / 1000,
                    messages: [],
                },
            ];

            (GetChatHistory as any).mockResolvedValue(mockSessions);

            render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.getByText('Test Session')).toBeInTheDocument();
            });

            fireEvent.contextMenu(screen.getByText('Test Session'));

            await waitFor(() => {
                expect(screen.getByText('Rename')).toBeInTheDocument();
                expect(screen.getByText('Delete')).toBeInTheDocument();
                expect(screen.getByText('Export')).toBeInTheDocument();
            });
        });
    });

    describe('QAP Event Subscription', () => {
        it('should subscribe to qap-session-created event', async () => {
            const { EventsOn } = await import('../../wailsjs/runtime/runtime');
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            // Verify EventsOn was called with 'qap-session-created'
            const calls = (EventsOn as any).mock.calls;
            const eventNames = calls.map((call: any[]) => call[0]);
            expect(eventNames).toContain('qap-session-created');
        });

        it('should subscribe to all required session events', async () => {
            const { EventsOn } = await import('../../wailsjs/runtime/runtime');
            (GetChatHistory as any).mockResolvedValue([]);

            render(<LeftPanel {...mockProps} />);

            const calls = (EventsOn as any).mock.calls;
            const eventNames = calls.map((call: any[]) => call[0]);
            expect(eventNames).toContain('chat-thread-created');
            expect(eventNames).toContain('chat-thread-deleted');
            expect(eventNames).toContain('chat-thread-updated');
            expect(eventNames).toContain('qap-session-created');
        });

        it('should pass is_replay_session to HistoricalSessionsSection', async () => {
            const mockSessions = [
                {
                    id: 'qap1',
                    title: 'QAP Session',
                    data_source_id: 'ds1',
                    created_at: Date.now() / 1000,
                    messages: [],
                    is_replay_session: true,
                },
                {
                    id: 'normal1',
                    title: 'Normal Session',
                    data_source_id: 'ds1',
                    created_at: Date.now() / 1000 - 3600,
                    messages: [],
                    is_replay_session: false,
                },
            ];

            (GetChatHistory as any).mockResolvedValue(mockSessions);

            const { container } = render(<LeftPanel {...mockProps} />);

            await waitFor(() => {
                expect(screen.getByText('QAP Session')).toBeInTheDocument();
                expect(screen.getByText('Normal Session')).toBeInTheDocument();
            });

            // QAP session should have amber icon, normal should have blue
            const amberIcons = container.querySelectorAll('.text-amber-500');
            const blueIcons = container.querySelectorAll('.session-title .text-blue-400');
            expect(amberIcons).toHaveLength(1);
            expect(blueIcons).toHaveLength(1);
        });
    });
});
